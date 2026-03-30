// Package database provides the pgxpool connection pool and its lifecycle management.
package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/popatkaran/postulate/api/internal/config"
)

// Pool is the interface the application uses for database access.
// It exposes only the operations required by the application, preventing
// pgxpool internals from leaking into business logic packages.
// *pgxpool.Pool satisfies this interface.
type Pool interface {
	Acquire(ctx context.Context) (*pgxpool.Conn, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Ping(ctx context.Context) error
	Stat() *pgxpool.Stat
	Config() *pgxpool.Config
	Close()
}

// BuildDSN constructs a PostgreSQL connection string from individual config fields.
// The result must never be logged — it contains the database password.
func BuildDSN(cfg config.DatabaseConfig) string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Name, cfg.User, cfg.Password, cfg.SSLMode,
	)
}

// buildPoolConfig constructs a pgxpool.Config from the given DatabaseConfig,
// applying defaults for zero-value pool size and lifetime settings.
// Exported for unit testing.
func buildPoolConfig(cfg config.DatabaseConfig) (*pgxpool.Config, error) {
	poolCfg, err := pgxpool.ParseConfig(BuildDSN(cfg))
	if err != nil {
		return nil, fmt.Errorf("invalid database config: %w", err)
	}

	maxConns := cfg.MaxOpenConns
	if maxConns == 0 {
		maxConns = 25
	}
	minConns := cfg.MaxIdleConns
	if minConns == 0 {
		minConns = 5
	}
	lifetime := cfg.ConnMaxLifetimeSeconds
	if lifetime == 0 {
		lifetime = 300
	}

	poolCfg.MaxConns = int32(maxConns)
	poolCfg.MinConns = int32(minConns)
	poolCfg.MaxConnLifetime = time.Duration(lifetime) * time.Second

	return poolCfg, nil
}

// New creates and validates a pgxpool connection pool from the given config.
// It pings the database to confirm reachability before returning.
// On failure it logs a structured error (without the password) and returns an error.
func New(ctx context.Context, cfg config.DatabaseConfig, logger *slog.Logger) (Pool, error) {
	poolCfg, err := buildPoolConfig(cfg)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		logger.Error("failed to create database connection pool",
			"host", cfg.Host, "port", cfg.Port, "name", cfg.Name, "error", err)
		return nil, fmt.Errorf("failed to create database connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		logger.Error("database unreachable at startup — ensure PostgreSQL is running",
			"host", cfg.Host, "port", cfg.Port, "name", cfg.Name, "error", err)
		return nil, fmt.Errorf("database unreachable at startup: %w", err)
	}

	logger.Info("database connection pool established",
		"max_conns", poolCfg.MaxConns, "host", cfg.Host, "name", cfg.Name)

	return pool, nil
}
