package server_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"log/slog"
	"os"

	"github.com/popatkaran/postulate/api/internal/config"
	"github.com/popatkaran/postulate/api/internal/server"
)

// slowHandler sleeps for the given duration before responding 200.
func slowHandler(delay time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(http.StatusOK)
	})
}

// startTestServer starts a server on a random port and returns the server,
// its base URL, and a function to wait for it to be ready.
func startTestServer(t *testing.T, handler http.Handler) (*server.Server, string) {
	t.Helper()

	// Pick a free port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	cfg := config.ServerConfig{Port: port, ShutdownTimeoutSeconds: 5}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	srv := server.New(cfg, handler, logger)

	go func() { _ = srv.Start() }()

	// Wait until the port is accepting connections.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	return srv, "http://" + addr
}

func TestShutdown_InFlightRequestCompletesSuccessfully(t *testing.T) {
	// Arrange — handler takes 200 ms; shutdown is initiated after 50 ms.
	srv, base := startTestServer(t, slowHandler(200*time.Millisecond))

	// Issue the slow request in the background.
	type result struct {
		status int
		err    error
	}
	done := make(chan result, 1)
	go func() {
		resp, err := http.Get(base + "/") //nolint:noctx
		if err != nil {
			done <- result{err: err}
			return
		}
		_ = resp.Body.Close()
		done <- result{status: resp.StatusCode}
	}()

	// Give the request time to reach the handler, then begin shutdown.
	time.Sleep(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	// Assert — the in-flight request must have completed with 200.
	res := <-done
	if res.err != nil {
		t.Fatalf("in-flight request failed: %v", res.err)
	}
	if res.status != http.StatusOK {
		t.Errorf("expected 200, got %d", res.status)
	}
}

func TestShutdown_NewRequestAfterShutdownFails(t *testing.T) {
	// Arrange
	srv, base := startTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Shut down the server.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	// Act — attempt a new request after shutdown.
	_, err := http.Get(base + "/") //nolint:noctx

	// Assert — connection must be refused.
	if err == nil {
		t.Error("expected connection error after shutdown, got nil")
	}
	var netErr *net.OpError
	if !errors.As(err, &netErr) {
		t.Errorf("expected net.OpError, got %T: %v", err, err)
	}
}
