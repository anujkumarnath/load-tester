package config

import (
	"time"
	"flag"
	"net/url"
	"errors"
)

type Config struct {
	URL          string
	Concurrency  int
	Duration     time.Duration
	RPS          int
	Timeout      time.Duration
}

func (c *Config) Parse() {
	targetURL   := flag.String  ("url",     "",              "target url")
	concurrency := flag.Int     ("c",       1,               "no. of concurrent requests")
	duration    := flag.Duration("d",       time.Minute,     "how long to run (e.g. 10s, 1m)")
	rps         := flag.Int     ("rps",     1,               "max requests per second (0 = unlimited)")
	timeout     := flag.Duration("timeout", 3 * time.Second, "per-request HTTP timeout (e.g. 10s, 1m)")

	flag.Parse()

	c.URL         = *targetURL
	c.Concurrency = *concurrency
	c.Duration    = *duration
	c.RPS         = *rps
	c.Timeout     = *timeout
}

func (c *Config) Validate() error {
	_, err := url.ParseRequestURI(c.URL)
	if err != nil {
		return errors.New(err.Error())
	}

	if c.Concurrency <= 0 {
		return errors.New("concurrency value should be > 0")
	}

	if c.Duration <= 0 {
		return errors.New("duration should be > 0")
	}

	if c.RPS < 0 {
		return errors.New("rps should be >= 0")
	}

	if c.Timeout <= 0 {
		return errors.New("timeout should be >= 0")
	}

	return nil
}
