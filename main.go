package main

import (
	"os"
	"fmt"

	"load-tester/internal/config"
)

func main() {
	var cfg config.Config

	cfg.Parse()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Config: %+v", cfg)
}
