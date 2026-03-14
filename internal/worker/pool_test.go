package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
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
	perReqTimout  := 5 * time.Second

	cfg := config.Config{
		URL         : srv.URL,
		Concurrency : numWorkers,
		Timeout     : perReqTimout,
	}

	requester := requester.NewRequester(cfg.Timeout)
	stats     := stats.NewStats()

	pool := NewPool(cfg, requester, stats)

	ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
	defer cancel()

	before := runtime.NumGoroutine()
	pool.Run(ctx)

	// Lower bound: each worker should complete at least 10 req/s against a local
	// httptest server with no artificial delay. This catches silent worker failures
	// while staying well below realistic throughput (~1000s req/s per worker).
	minExpectedTotal := numWorkers * int(globalTimeout.Seconds()) * 10
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
