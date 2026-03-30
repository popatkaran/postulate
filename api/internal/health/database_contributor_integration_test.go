//go:build integration

package health_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/popatkaran/postulate/api/internal/config"
	"github.com/popatkaran/postulate/api/internal/database"
	"github.com/popatkaran/postulate/api/internal/health"
)

var integrationLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

func testPool(t *testing.T) interface{ Close() } {
	t.Helper()
	cfg := config.DatabaseConfig{
		Host: "localhost", Port: 5432, Name: "postulate_test",
		User: "postulate_dev", Password: "postulate_dev", SSLMode: "disable",
		MaxOpenConns: 5, MaxIdleConns: 1, ConnMaxLifetimeSeconds: 60,
	}
	pool, err := database.New(context.Background(), cfg, integrationLogger)
	if err != nil {
		t.Fatalf("pool creation failed: %v", err)
	}
	return pool
}

func TestDatabaseContributor_Integration_ReturnsHealthy(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host: "localhost", Port: 5432, Name: "postulate_test",
		User: "postulate_dev", Password: "postulate_dev", SSLMode: "disable",
		MaxOpenConns: 5, MaxIdleConns: 1, ConnMaxLifetimeSeconds: 60,
	}
	pool, err := database.New(context.Background(), cfg, integrationLogger)
	if err != nil {
		t.Fatalf("pool creation failed: %v", err)
	}
	defer pool.Close()

	c := health.NewDatabaseContributor(pool)
	result := c.Check(context.Background())

	if result.Status != health.StatusHealthy {
		t.Errorf("expected healthy, got %s: %s", result.Status, result.Message)
	}
	if result.Extensions == nil {
		t.Error("expected Extensions to be non-nil on healthy result")
	}
}
