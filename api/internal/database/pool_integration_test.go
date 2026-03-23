//go:build integration

package database_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/popatkaran/postulate/api/internal/config"
	"github.com/popatkaran/postulate/api/internal/database"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

func testDBConfig() config.DatabaseConfig {
	return config.DatabaseConfig{
		Host:                   "localhost",
		Port:                   5432,
		Name:                   "postulate_test",
		User:                   "postulate_dev",
		Password:               "postulate_dev",
		SSLMode:                "disable",
		MaxOpenConns:           10,
		MaxIdleConns:           2,
		ConnMaxLifetimeSeconds: 60,
	}
}

func TestNew_ReturnsPoolWhenDatabaseReachable(t *testing.T) {
	pool, err := database.New(context.Background(), testDBConfig(), testLogger)
	if err != nil {
		t.Fatalf("expected non-nil pool, got error: %v", err)
	}
	defer pool.Close()

	if pool == nil {
		t.Fatal("expected non-nil pool")
	}
}

func TestNew_PingReturnsNil(t *testing.T) {
	pool, err := database.New(context.Background(), testDBConfig(), testLogger)
	if err != nil {
		t.Fatalf("pool creation failed: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("expected Ping to return nil, got: %v", err)
	}
}

func TestNew_StatsMaxConnsMatchesConfig(t *testing.T) {
	cfg := testDBConfig()
	pool, err := database.New(context.Background(), cfg, testLogger)
	if err != nil {
		t.Fatalf("pool creation failed: %v", err)
	}
	defer pool.Close()

	if got := pool.Stat().MaxConns(); int(got) != cfg.MaxOpenConns {
		t.Errorf("expected MaxConns=%d, got %d", cfg.MaxOpenConns, got)
	}
}

func TestNew_CloseCompletesWithoutError(t *testing.T) {
	pool, err := database.New(context.Background(), testDBConfig(), testLogger)
	if err != nil {
		t.Fatalf("pool creation failed: %v", err)
	}
	// Close must not panic or block.
	pool.Close()
}
