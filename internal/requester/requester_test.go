package requester

import (
	"testing"
	"net/http"
	"net/http/httptest"
	"context"
	"time"
)

func TestRequester_Do(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	}))
	defer srv.Close()

	req    := NewRequester(5 * time.Second)
	result := req.Do(context.Background(), srv.URL)

	if result.StatusCode != 200 {
		t.Errorf("expected 200, got %d", result.StatusCode)
	}

	if result.Error != "" {
		t.Error("expected no error, got", result.Error)
	}

	if result.Bytes == 0 {
		t.Error("expected non-zero body length")
	}

	if result.Duration == 0 {
		t.Error("expected non-zero duration")
	}

	if result.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestRequester_Do_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	req    := NewRequester(5 * time.Second)
	result := req.Do(context.Background(), srv.URL)

	if result.StatusCode != 500 {
		t.Errorf("expected 500, got %d", result.StatusCode)
	}

	if result.Bytes != 0 {
		t.Error("expected a zero-length body")
	}

	if result.Error != "" {
		t.Errorf("expected no error, got %v", result.Error)
	}

	if result.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestRequester_Do_ConnectionClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// left empty because closing the server early won't let it execute anyways
	}))
	srv.Close()

	req    := NewRequester(5 * time.Second)
	result := req.Do(context.Background(), srv.URL)

	if result.StatusCode != 0 {
		t.Errorf("expected 0, got %d", result.StatusCode)
	}

	if result.Error == "" {
		t.Errorf("expected error, got none")
	}

	if result.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestRequester_Do_ContextAlreadyCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// left empty because cancelling the context early won't let it execute anyways
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req    := NewRequester(5 * time.Second)
	result := req.Do(ctx, srv.URL)

	if result.Error == "" {
		t.Errorf("expected error, got none")
	}

	if result.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}
