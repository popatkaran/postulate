//go:build integration

package migrate_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/popatkaran/postulate/api/internal/config"
	"github.com/popatkaran/postulate/api/internal/database"
	apimigrate "github.com/popatkaran/postulate/api/internal/migrate"
)

var integrationLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

func testDBConfig() config.DatabaseConfig {
	return config.DatabaseConfig{
		Host: "localhost", Port: 5432, Name: "postulate_test",
		User: "postulate_dev", Password: "postulate_dev", SSLMode: "disable",
		MaxOpenConns: 5, MaxIdleConns: 1, ConnMaxLifetimeSeconds: 60,
	}
}

func TestRun_Integration_AppliesMigrationsAndReturnsNil(t *testing.T) {
	pool, err := database.New(context.Background(), testDBConfig(), integrationLogger)
	if err != nil {
		t.Fatalf("pool creation failed: %v", err)
	}
	defer pool.Close()

	if err := apimigrate.Run(context.Background(), pool, integrationLogger); err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

func TestRun_Integration_IdempotentOnSecondCall(t *testing.T) {
	pool, err := database.New(context.Background(), testDBConfig(), integrationLogger)
	if err != nil {
		t.Fatalf("pool creation failed: %v", err)
	}
	defer pool.Close()

	// First run — apply or no-change.
	_ = apimigrate.Run(context.Background(), pool, integrationLogger)

	// Second run — must return nil (ErrNoChange is handled internally).
	if err := apimigrate.Run(context.Background(), pool, integrationLogger); err != nil {
		t.Fatalf("expected nil on second run, got: %v", err)
	}
}
