# load-tester

A CLI HTTP load testing tool. Hammers an endpoint with concurrent workers, collects latency data, and prints a report. Like `hey` or `wrk`.

## Install

```bash
git clone <repo>
cd load-tester
go build -o load-tester .
```

Or run directly:

```bash
go run . [flags]
```

## Usage

```
Flags:
  -url      string    target URL (required)
  -c        int       number of concurrent workers (default 1)
  -d        duration  how long to run, e.g. 10s, 1m (default 1m0s)
  -rps      int       max requests per second, 0 = unlimited (default 1)
  -timeout  duration  per-request HTTP timeout (default 3s)
```

## Examples

### Rate-limited run — 10 workers, 50 req/s for 10 seconds

```bash
$ go run . -url http://localhost:8080 -c 10 -d 10s -rps 50

Running load test...
  Target      : http://localhost:8080
  Workers     : 10
  Duration    : 10s
  Rate Limit  : 50 req/s

[   1s]      57 reqs |    57.0 rps |       0 errors
[   2s]     109 reqs |    52.0 rps |       0 errors
[   3s]     159 reqs |    50.0 rps |       0 errors
[   4s]     209 reqs |    50.0 rps |       0 errors
[   5s]     259 reqs |    50.0 rps |       0 errors
[   6s]     309 reqs |    50.0 rps |       0 errors
[   7s]     359 reqs |    50.0 rps |       0 errors
[   8s]     409 reqs |    50.0 rps |       0 errors
[   9s]     459 reqs |    50.0 rps |       0 errors

============ Report ============
Total Requests  :  509
Throughput      :  50.99 req/s
Error Rate      :  0.000%

Latency
  p50  :  1ms
  p95  :  1ms
  p99  :  2ms
  max  :  1010ms

Status Codes
  500  :    509
================================
```

### Unlimited rate — 5 workers, no rate limit

```bash
$ go run . -url http://localhost:8080 -c 5 -d 5s -rps 0

Running load test...
  Target      : http://localhost:8080
  Workers     : 5
  Duration    : 5s
  Rate Limit  : 0 req/s

[   1s]    4098 reqs |  4098.0 rps |       0 errors
[   2s]    7810 reqs |  3712.0 rps |       0 errors
[   3s]   11542 reqs |  3732.0 rps |       0 errors
[   4s]   15445 reqs |  3903.0 rps |       0 errors

============ Report ============
Total Requests  :  19377
Throughput      :  3872.38 req/s
Error Rate      :  0.021%

Latency
  p50  :  1ms
  p95  :  1ms
  p99  :  2ms
  max  :  6ms

Status Codes
  500  :  19373
================================
```

### Single worker — 1 worker, 5 req/s for 5 seconds

```bash
$ go run . -url http://localhost:8080 -c 1 -d 5s -rps 5

Running load test...
  Target      : http://localhost:8080
  Workers     : 1
  Duration    : 5s
  Rate Limit  : 5 req/s

[   1s]       5 reqs |     5.0 rps |       0 errors
[   2s]      10 reqs |     5.0 rps |       0 errors
[   3s]      15 reqs |     5.0 rps |       0 errors
[   4s]      20 reqs |     5.0 rps |       0 errors

============ Report ============
Total Requests  :  25
Throughput      :  5.21 req/s
Error Rate      :  0.000%

Latency
  p50  :  1ms
  p95  :  1ms
  p99  :  2ms
  max  :  2ms

Status Codes
  500  :     25
================================
```

## Report fields

**Total Requests** — all requests fired, including transport errors.

**Throughput** — requests per second over the actual test window.

**Error Rate** — transport-level failures only (connection refused, timeout, etc.). HTTP error codes like 500 are not counted as errors — they appear in Status Codes.

**Latency** — p50/p95/p99/max of successful requests only.

**Status Codes** — breakdown of HTTP response codes from successful transport.

## Notes

- `-rps 0` disables rate limiting entirely. The default is `-rps 1`.
- The rate limiter uses a token bucket with burst equal to `-c`. This means up to `-c` requests can fire immediately at startup before the rate limit kicks in.
- The report always prints — even on Ctrl+C.
- Transport errors (e.g. connections dropped at shutdown) may cause a small discrepancy between total request count and the sum of status code counts. The difference equals the error count.

## Running tests

```bash
go test ./... -race -v
```

Zero race conditions, zero real network calls — all tests use `httptest.NewServer`.

## Project structure

```
load-tester/
├── main.go
├── internal/
│   ├── config/      # flag parsing and validation
│   ├── requester/   # single HTTP request execution
│   ├── stats/       # thread-safe result collection and report
│   └── worker/      # worker pool, rate limiting, graceful shutdown
├── go.mod
└── README.md
```
