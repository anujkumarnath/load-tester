package main

import (
	"context"
	"fmt"
	"os"

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

	fmt.Printf("Config: %+v\n", cfg)

	st   := stats.NewStats()
	req  := requester.NewRequester(cfg.Timeout)
	pool := worker.NewPool(cfg, req, st)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
	defer cancel()

	pool.Run(ctx)
	req.Close()

	st.Report()
}
