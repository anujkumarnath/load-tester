package worker

import (
	"context"
	"sync"

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
}

func NewPool(
	cfg   config.Config,
	req   *requester.Requester,
	stats *stats.Stats,
) *Pool {
	return &Pool{
		config    : cfg,
		requester : req,
		stats     : stats,
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
			result := p.requester.Do(reqCtx, p.config.URL)
			// Explicit call is required, defer will cause leak till the end of the loop
			cancel()
			p.stats.Record(result)
		}
	}
}
