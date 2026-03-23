//go:build integration

package startup_test

import (
	"context"
	"testing"

	"github.com/popatkaran/postulate/api/internal/config"
	"github.com/popatkaran/postulate/api/internal/database"
	"github.com/popatkaran/postulate/api/internal/startup"
)

// TestCheckDatabase_ReturnsNilWhenDatabaseIsReachable requires a running
// PostgreSQL instance with the postulate_test database.
// Run with: go test -tags integration ./api/internal/startup/...
func TestCheckDatabase_ReturnsNilWhenDatabaseIsReachable(t *testing.T) {
	// Arrange
	cfg := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Name:     "postulate_test",
		User:     "postulate_dev",
		Password: "postulate_dev",
		SSLMode:  "disable",
	}

	pool, err := database.New(context.Background(), cfg, nopLogger)
	if err != nil {
		t.Fatalf("pool creation failed: %v", err)
	}
	defer pool.Close()

	// Act
	err = startup.CheckDatabase(context.Background(), pool, "development", nopLogger)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error for reachable database, got %v", err)
	}
}
