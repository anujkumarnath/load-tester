package stats

import (
	"sync"
	"slices"
	"maps"
	"math"
	"time"
	"fmt"

	"load-tester/internal/requester"
)

// Stats collects request results from multiple concurrent goroutines.
// All fields are protected by mu — never read or write them without holding the lock.
type Stats struct {
	mu          sync.Mutex
	latency     []time.Duration
	statusCodes map[int]int
	errCount    int
	// startTime is set on the first recorded result and used to calculate
	// wall-clock throughput in Report(). Using result.Timestamp ensures accuracy
	// even if Record() is called with a delay after the request completes.
	startTime   time.Time
}

func NewStats() *Stats {
	return &Stats{
		statusCodes: map[int]int{},
	}
}

func (s *Stats) Total() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.latency)
}

// Record is safe for concurrent use. Called from every worker goroutine.
func (s *Stats) Record(result requester.Result) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.latency) == 0 {
		s.startTime = result.Timestamp
	}

	s.latency = append(s.latency, result.Duration)
	s.statusCodes[result.StatusCode]++
	if result.Error != "" {
		s.errCount++
	}
}

// Report prints the final summary. Should be called after all workers have
// stopped — it holds the lock for its entire duration, blocking any concurrent
// Record() calls while printing.
func (s *Stats) Report() {
	s.mu.Lock()
	defer s.mu.Unlock()

	totalRequests := len(s.latency)
	var errorRate, throughput float64

	if totalRequests != 0 {
		errorRate = float64(s.errCount) / float64(totalRequests) * 100
		totalTime := time.Since(s.startTime)
		throughput = float64(totalRequests) / totalTime.Seconds()
	}

	fmt.Println("============ Report ============")
	fmt.Printf("Total Requests  :  %d\n",         totalRequests)
	fmt.Printf("Throughput      :  %.2f req/s\n", throughput)
	fmt.Printf("Error Rate      :  %.2f\n",       errorRate)
	fmt.Println()

	fmt.Println("Latency")
	fmt.Printf("  p50  :  %dms\n", percentile(s.latency, 50).Milliseconds())
	fmt.Printf("  p95  :  %dms\n", percentile(s.latency, 95).Milliseconds())
	fmt.Printf("  p99  :  %dms\n", percentile(s.latency, 99).Milliseconds())
	fmt.Printf("  max  :  %dms\n", percentile(s.latency, 100).Milliseconds())
	fmt.Println()

	fmt.Println("Status Codes")
	for k, v := range s.statusCodes {
		fmt.Printf("  %d  :%4d\n", k, v)
	}
	fmt.Println("================================")
}

// latencies, errorCount, statusCodeCounts exist for test access only.
// They return safe copies under the lock so tests don't touch internal fields directly.
func (s *Stats) latencies() []time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.latency)
}

func (s *Stats) errorCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.errCount
}

func (s *Stats) statusCodeCounts() map[int]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return maps.Clone(s.statusCodes)
}

// percentile clones data before sorting to avoid mutating the caller's slice.
// p=100 returns the maximum value.
func percentile(data []time.Duration, p float64) time.Duration {
	size := len(data)

	if size == 0 {
		return 0
	}

	clone := slices.Clone(data)
	slices.Sort(clone)

	index := int(math.Ceil(float64(size) * p / 100)) - 1
	return clone[index]
}
