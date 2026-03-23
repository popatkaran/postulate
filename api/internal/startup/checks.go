// Package startup contains pre-flight checks run before the server accepts requests.
package startup

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/popatkaran/postulate/api/internal/config"
)

// CheckDatabase opens a temporary connection pool, pings PostgreSQL, and closes
// the pool. On failure it logs a structured error and returns a non-nil error.
// The hint field is only emitted when environment is "development".
func CheckDatabase(ctx context.Context, cfg config.DatabaseConfig, env string, logger *slog.Logger) error {
	dsn := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Name, cfg.User, cfg.Password, cfg.SSLMode,
	)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		logDBError(logger, cfg, env, err)
		return fmt.Errorf("database unreachable at startup: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logDBError(logger, cfg, env, err)
		return fmt.Errorf("database unreachable at startup: %w", err)
	}

	return nil
}

func logDBError(logger *slog.Logger, cfg config.DatabaseConfig, env string, err error) {
	args := []any{
		"host", cfg.Host,
		"port", cfg.Port,
		"name", cfg.Name,
		"error", err,
	}
	if env == "development" {
		args = append(args, "hint", "run 'make db-start' to start the local PostgreSQL service")
	}
	logger.Error("database unreachable at startup — ensure PostgreSQL is running", args...)
}
