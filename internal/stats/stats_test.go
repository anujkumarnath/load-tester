package stats

import (
	"testing"
	"sync"
	"time"

	"load-tester/internal/requester"
)

// TestStats_ConcurrentRecord is designed to be run with -race.
// The test itself only checks the count, but the race detector will catch
// any missing or incorrect lock usage across the 100 concurrent goroutines.
func TestStats_ConcurrentRecord(t *testing.T) {
	s := NewStats()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Record(requester.Result{StatusCode: 200, Duration: 100 * time.Millisecond})
		}()
	}
	wg.Wait()
	if s.Total() != 100 {
		t.Errorf("expected 100, got %d", s.Total())
	}
}

// TestStats_percentile uses sequential 1-100ms inputs so expected
// percentile values are exact and predictable without any approximation.
func TestStats_percentile(t *testing.T) {
	s := NewStats()
	for i := 1; i <= 100; i++ {
		s.Record(requester.Result{StatusCode: 200, Duration: time.Duration(i) * time.Millisecond})
	}

	// call latencies() once and reuse the snapshot — avoids acquiring the lock
	// three times and ensures all percentiles operate on the same data.
	latencies := s.latencies()

	p50 := percentile(latencies, 50).Milliseconds()
	p95 := percentile(latencies, 95).Milliseconds()
	p99 := percentile(latencies, 99).Milliseconds()

	if p50 != 50 {
		t.Errorf("expected p50 latency to be 50, got %d", p50)
	}

	if p95 != 95 {
		t.Errorf("expected p95 latency to be 95, got %d", p95)
	}

	if p99 != 99 {
		t.Errorf("expected p99 latency to be 99, got %d", p99)
	}
}

// TestStats_EmptyStatsReport uses defer+recover to catch a panic rather
// than letting the test binary crash. Report() on empty stats should be a no-op.
func TestStats_EmptyStatsReport(t *testing.T) {
	s := NewStats()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic on Report() on emtpy stats")
		}
	}()

	s.Report()
}

func TestStats_ErrorCount(t *testing.T) {
	s := NewStats()
	for i := 0; i < 10; i++ {
		s.Record(requester.Result{StatusCode: 200, Error: "error"})
	}
	for i := 0; i < 20; i++ {
		s.Record(requester.Result{StatusCode: 200})
	}

	errCount := s.ErrorCount()

	if errCount != 10 {
		t.Errorf("expected 10 errors, found: %d", errCount)
	}
}

func TestStats_StatusCodes(t *testing.T) {
	s := NewStats()

	for i := 0; i < 10; i++ {
		s.Record(requester.Result{StatusCode: 200})
	}

	for i := 0; i < 20; i++ {
		s.Record(requester.Result{StatusCode: 500})
	}

	for i := 0; i < 15; i++ {
		s.Record(requester.Result{StatusCode: 400})
	}

	statusCodes := s.statusCodeCounts()

	if statusCodes[200] != 10 {
		t.Errorf("expected 10 response with status code 200, found: %d", statusCodes[200])
	}

	if statusCodes[500] != 20 {
		t.Errorf("expected 20 response with status code 500, found: %d", statusCodes[500])
	}

	if statusCodes[400] != 15 {
		t.Errorf("expected 15 response with status code 400, found: %d", statusCodes[400])
	}
}
