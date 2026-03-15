package worker

import (
	"context"
	"sync"

	"golang.org/x/time/rate"

	"load-tester/internal/config"
	"load-tester/internal/requester"
	"load-tester/internal/stats"
)

type Pool struct {
	// config is a value — it's plain data, no mutex, no shared state, cheap to copy
	config    config.Config
	// requester and stats are pointers — they manage shared resources and contain
	// or wrap types that must not be copied (http.Client, sync.Mutex)
	requester *requester.Requester
	stats     *stats.Stats
	limiter   *rate.Limiter
}

func NewPool(
	cfg   config.Config,
	req   *requester.Requester,
	stats *stats.Stats,
) *Pool {
	var limiter *rate.Limiter
	if cfg.RPS > 0 {
		limiter = rate.NewLimiter(rate.Limit(cfg.RPS), cfg.Concurrency)
	}

	return &Pool{
		config    : cfg,
		requester : req,
		stats     : stats,
		limiter   : limiter,
	}
}

func (p *Pool) Run(ctx context.Context) {
	numWorkers := p.config.Concurrency
	var wg sync.WaitGroup

	for range numWorkers {
		wg.Add(1)
		go p.doWork(ctx, &wg)
	}

	wg.Wait()
}

func (p *Pool) doWork(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	reqTimeout := p.config.Timeout

	for {
		select {
		case <-ctx.Done():
			return
		default:
			reqCtx, cancel := context.WithTimeout(ctx, reqTimeout)

			if p.limiter != nil {
				if err := p.limiter.Wait(reqCtx); err != nil {
					cancel()
					return
				}
			}

			result := p.requester.Do(reqCtx, p.config.URL)
			// Explicit call is required, defer will cause leak till the end of the loop
			cancel()
			p.stats.Record(result)
		}
	}
}
