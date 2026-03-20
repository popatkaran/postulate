package server_test

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/popatkaran/postulate/api/internal/config"
	"github.com/popatkaran/postulate/api/internal/server"
)

func TestShutdown_ReturnsNilOnRunningServer(t *testing.T) {
	// Arrange
	cfg := config.ServerConfig{Port: 0, ShutdownTimeoutSeconds: 5, Environment: "test"}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	srv := server.New(cfg, http.NewServeMux(), logger)

	// Start the server in the background; port 0 lets the OS pick a free port.
	// We use a real listener here because httptest.Server is for handler testing;
	// this test verifies the lifecycle method, not the handler.
	started := make(chan struct{})
	go func() {
		close(started)
		_ = srv.Start()
	}()
	<-started
	// Give the goroutine a moment to bind.
	time.Sleep(10 * time.Millisecond)

	// Act
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := srv.Shutdown(ctx)

	// Assert
	if err != nil {
		t.Fatalf("expected Shutdown to return nil, got %v", err)
	}
}
