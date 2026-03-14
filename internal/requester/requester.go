package requester

import (
	"net/http"
	"context"
	"time"
	"io"
)

type Requester struct {
	client *http.Client
}

// NewRequester returns a pointer because Requester is a shared resource handle
// intended to be reused across requests, not copied.
func NewRequester(timeout time.Duration) *Requester {
	return &Requester{
		client: &http.Client{Timeout: timeout},
	}
}

func (r *Requester) Close() {
	r.client.CloseIdleConnections()
}

func (r *Requester) Do(ctx context.Context, url string) Result {
	startTime := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{ Error: err.Error(), Timestamp: startTime }
	}

	res, err := r.client.Do(req)
	if err != nil {
		return Result{ Error: err.Error(), Timestamp: startTime }
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Result{ Error: err.Error(), Timestamp: startTime }
	}

	return Result {
		Duration   : time.Since(startTime),
		StatusCode : res.StatusCode,
		Bytes      : len(body),
		Timestamp  : startTime,
	}
}
