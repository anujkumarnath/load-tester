package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"load-tester/internal/config"
	"load-tester/internal/requester"
	"load-tester/internal/stats"
)

func TestPool(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	defer srv.Close()

	numWorkers    := 10
	globalTimeout := 10 * time.Second
	perReqTimeout := 5 * time.Second

	cfg := config.Config{
		URL         : srv.URL,
		Concurrency : numWorkers,
		Timeout     : perReqTimeout,
	}

	requester := requester.NewRequester(cfg.Timeout)
	stats     := stats.NewStats()

	pool := NewPool(cfg, requester, stats)

	ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
	defer cancel()

	before := runtime.NumGoroutine()
	pool.Run(ctx)

	// Worst case: every request takes the full per-request timeout.
	// Floor = numWorkers * (globalTimeout / perReqTimeout).
	minExpectedTotal := numWorkers * int(globalTimeout.Seconds() / perReqTimeout.Seconds())

	if stats.Total() < minExpectedTotal {
		t.Errorf(
			"total requests less than minimum expected requests (%d)",
			minExpectedTotal,
		)
	}

	// Close idle keepalive connections held by the HTTP client transport.
	// Without this, server-side goroutines stay alive waiting on open connections,
	// causing a false positive in the goroutine leak check below.
	requester.Close()

	// Some inflight requests might be there when the context is cancelled
	// this is needed to allow go runtime to stop all running goroutines
	time.Sleep(1 * time.Second)
	after := runtime.NumGoroutine()

	if after > before {
		t.Errorf("%d goroutine leaked", after - before)
	}
}

func TestPool_RateLimited(t *testing.T) {
	var count atomic.Int64
	ctx, cancel := context.WithCancel(context.Background())

	srv := httptest.NewServer(
		http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			if count.Add(1) >= 10 {
				cancel()
			}
			w.WriteHeader(http.StatusOK)
		}))
	defer srv.Close()

	numWorkers   := 1
	perReqTimout := 5 * time.Second

	cfg := config.Config{
		URL         : srv.URL,
		Concurrency : numWorkers,
		Timeout     : perReqTimout,
		RPS         : 5,
	}

	requester := requester.NewRequester(cfg.Timeout)
	stats     := stats.NewStats()

	pool := NewPool(cfg, requester, stats)

	startTime := time.Now()
	pool.Run(ctx)
	requester.Close()
	timeTaken := time.Since(startTime)

	// Workers check ctx.Done() before each request, not during — a worker that has
	// already passed the select check will complete its in-flight request even after
	// cancellation. So total may exceed 10 by at most (numWorkers - 1).
	if stats.Total() < 10 {
		t.Errorf("expected at least 10 requests, got %d", stats.Total())
	}

	// 5 RPS = 200ms per request. First request is instant (pre-filled token),
	// remaining 9 require one token each: 9 * 200ms = 1.8s minimum.
	if timeTaken < 1800 * time.Millisecond {
		t.Errorf("time taken should be >= 1.8s, got %s", timeTaken)
	}
}

func TestPool_CancelDuringWait(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

	defer srv.Close()

	numWorkers    := 1
	globalTimeout := 500 * time.Millisecond
	perReqTimeout := 5 * time.Second
	RPS           := 1

	cfg := config.Config{
		URL         : srv.URL,
		Concurrency : numWorkers,
		Timeout     : perReqTimeout,
		RPS         : RPS,
	}

	requester := requester.NewRequester(cfg.Timeout)
	stats     := stats.NewStats()

	pool := NewPool(cfg, requester, stats)

	ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
	defer cancel()

	pool.Run(ctx)
	requester.Close()

	// 1 RPS = 1s per request. First request is instant (pre-filled token),
	// Past request 1, any new request has to wait for 1s for a new token.
	// If globalTimeout expires before that, the Wait() fails, so no more requests
	if stats.Total() > 1 {
		t.Errorf("Total requests should not be more than 1, found: %d", stats.Total())
	}
}

func TestPool_NoRateLimit(t *testing.T) {
	srv := httptest.NewServer(
		http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	defer srv.Close()

	numWorkers    := 5
	globalTimeout := 10 * time.Second
	perReqTimout  := 5 * time.Second

	cfg := config.Config{
		URL         : srv.URL,
		Concurrency : numWorkers,
		Timeout     : perReqTimout,
		RPS         : 0,
	}

	requester := requester.NewRequester(cfg.Timeout)
	stats     := stats.NewStats()

	pool := NewPool(cfg, requester, stats)

	// RPS = 0 skips the limiter path entirely and
	// this guards against a nil pointer panic if that branch is broken
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("pool ran into a panic with 0 RPS")
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
	defer cancel()

	pool.Run(ctx)
	requester.Close()

	// Worst case: every request takes the full per-request timeout.
	// Floor = numWorkers * (globalTimeout / perReqTimeout).
	minReqs := numWorkers * int(globalTimeout.Seconds() / perReqTimout.Seconds())
	if stats.Total() < minReqs {
		t.Errorf("Expected at least %d requests, got %d", minReqs, stats.Total())
	}
}
