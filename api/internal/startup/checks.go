// Package startup contains pre-flight checks run before the server accepts requests.
package startup

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CheckDatabase pings the provided pool to verify the database is reachable.
// On failure it logs a structured error and returns a non-nil error.
func CheckDatabase(ctx context.Context, pool *pgxpool.Pool, env string, logger *slog.Logger) error {
	if err := pool.Ping(ctx); err != nil {
		args := []any{"error", err}
		if env == "development" {
			args = append(args, "hint", "run 'make db-start' to start the local PostgreSQL service")
		}
		logger.Error("database unreachable at startup — ensure PostgreSQL is running", args...)
		return fmt.Errorf("database unreachable at startup: %w", err)
	}
	return nil
}
