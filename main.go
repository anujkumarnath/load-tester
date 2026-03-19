package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"load-tester/internal/config"
	"load-tester/internal/requester"
	"load-tester/internal/stats"
	"load-tester/internal/worker"
)

func main() {
	var cfg config.Config

	cfg.Parse()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	st   := stats.NewStats()
	req  := requester.NewRequester(cfg.Timeout)
	pool := worker.NewPool(cfg, req, st)

	shutdownCtx, stopSignal := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignal()

	ctx, cancel := context.WithTimeout(shutdownCtx, cfg.Duration)
	defer cancel()

	fmt.Println("Running load test...")
	fmt.Printf("  Target      : %s\n",       cfg.URL)
	fmt.Printf("  Workers     : %d\n",       cfg.Concurrency)
	fmt.Printf("  Duration    : %v\n",       cfg.Duration)
	fmt.Printf("  Rate Limit  : %d req/s\n", cfg.RPS)
	fmt.Println()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	go func() {
		var tick int
		var lastTotal int

		for {
			select {
			case <-ticker.C:
				tick++
				total   := st.Total()
				errors  := st.ErrorCount()
				rps     := float64(total - lastTotal)
				lastTotal = total

				fmt.Printf(
					"[%4ds] %7d reqs | %7.1f rps | %7d errors\n",
					tick,
					total,
					rps,
					errors,
				)
			case <-ctx.Done():
				return
			}
		}
	}()

	// pool.Run blocks until all workers exit — by the time it returns, the context
	// is already done. No need to wait on shutdownCtx.Done() separately.
	pool.Run(ctx)
	fmt.Println()

	req.Close()
	st.Report()
}
