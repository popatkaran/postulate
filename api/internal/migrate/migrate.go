// Package migrate provides the database migration runner for the Postulate API.
// Migration files are embedded into the binary and applied automatically at startup.
package migrate

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx stdlib driver for database/sql
)

// migrator is the interface used by runMigrator, allowing injection of a fake in tests.
type migrator interface {
	Up() error
	Close() (error, error)
}

// poolConfiger is the minimal interface Run needs from the pool.
type poolConfiger interface {
	Config() *pgxpool.Config
}

// Run applies all pending migrations from the embedded migration files.
// It returns nil when migrations are applied successfully or when there are no
// pending migrations. It returns an error if any migration fails.
func Run(ctx context.Context, pool poolConfiger, logger *slog.Logger) error {
	src, err := iofs.New(MigrationFiles, "migrations")
	if err != nil {
		return err
	}
	return runWithSource(ctx, pool, src, logger)
}

// runWithSource is separated so tests can inject a pre-built source driver.
func runWithSource(_ context.Context, pool poolConfiger, src source.Driver, logger *slog.Logger) error {
	connStr := pool.Config().ConnString()

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck

	driver, err := pgxmigrate.WithInstance(db, &pgxmigrate.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		return err
	}

	return runMigrator(m, logger)
}

// RunMigrator is exported for unit testing. Production code uses Run.
func RunMigrator(m migrator, logger *slog.Logger) error {
	return runMigrator(m, logger)
}

// Separated so unit tests can inject a fake migrator.
func runMigrator(m migrator, logger *slog.Logger) error {
	defer m.Close() //nolint:errcheck

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("database schema up to date")
			return nil
		}
		logger.Error("database migration failed", "error", err)
		return err
	}

	logger.Info("database migrations applied")
	return nil
}
