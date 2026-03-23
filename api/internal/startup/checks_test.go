package startup_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/popatkaran/postulate/api/internal/config"
	"github.com/popatkaran/postulate/api/internal/startup"
)

var nopLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

func TestCheckDatabase_ReturnsErrorWhenHostIsUnreachable(t *testing.T) {
	// Arrange — use an invalid host to guarantee connection failure without a real database.
	cfg := config.DatabaseConfig{
		Host:     "127.0.0.1",
		Port:     19999, // port nothing is listening on
		Name:     "postulate_test",
		User:     "postulate_dev",
		Password: "postulate_dev",
		SSLMode:  "disable",
	}

	// Act
	err := startup.CheckDatabase(context.Background(), cfg, "development", nopLogger)

	// Assert
	if err == nil {
		t.Fatal("expected error for unreachable host, got nil")
	}
}
