package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestMain_SIGINTPrintsReport(t *testing.T) {
	bin   := t.TempDir() + "/load-tester"
	build := exec.Command("go", "build", "-o", bin, ".")

	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s", out)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cmd := exec.Command(bin, "-url", srv.URL, "-c", "5", "-d", "30s")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	cmd.Process.Signal(syscall.SIGINT)

	cmd.Wait()

	output := stdout.String()
	if !strings.Contains(output, "Report") {
		t.Errorf("report not printed after SIGINT\nstdout:\n%s", output)
	}
}
