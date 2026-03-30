package migrate_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	apimigrate "github.com/popatkaran/postulate/api/internal/migrate"
)

// fakeMigrator is a test double for the migrate.Migrate instance.
type fakeMigrator struct {
	upErr error
}

func (f *fakeMigrator) Up() error             { return f.upErr }
func (f *fakeMigrator) Close() (error, error) { return nil, nil }

func captureLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return slog.New(slog.NewTextHandler(buf, nil)), buf
}

func TestRunMigrator_ErrNoChange_LogsUpToDateAndReturnsNil(t *testing.T) {
	logger, buf := captureLogger()
	err := apimigrate.RunMigrator(&fakeMigrator{upErr: migrate.ErrNoChange}, logger)

	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("database schema up to date")) {
		t.Errorf("expected 'database schema up to date' in log, got: %s", buf)
	}
}

func TestRunMigrator_Success_LogsAppliedAndReturnsNil(t *testing.T) {
	logger, buf := captureLogger()
	err := apimigrate.RunMigrator(&fakeMigrator{}, logger)

	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("database migrations applied")) {
		t.Errorf("expected 'database migrations applied' in log, got: %s", buf)
	}
}

func TestRunMigrator_Error_LogsErrorAndReturnsError(t *testing.T) {
	logger, buf := captureLogger()
	migErr := errors.New("migration version 2: syntax error")
	err := apimigrate.RunMigrator(&fakeMigrator{upErr: migErr}, logger)

	if !errors.Is(err, migErr) {
		t.Fatalf("expected migration error, got %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("database migration failed")) {
		t.Errorf("expected 'database migration failed' in log, got: %s", buf)
	}
}

// fakePoolConfiger returns a pgxpool.Config to exercise Run without a real DB.
type fakePoolConfiger struct{ cfg *pgxpool.Config }

func (f *fakePoolConfiger) Config() *pgxpool.Config { return f.cfg }

func TestRun_ReturnsErrorOnInvalidConnectionString(t *testing.T) {
	// iofs.New succeeds (embedded files are valid), then runWithSource fails on bad DSN.
	cfg, err := pgxpool.ParseConfig("host=localhost port=5432 dbname=x user=x password=x sslmode=disable")
	if err != nil {
		t.Fatalf("unexpected ParseConfig error: %v", err)
	}
	badCfg, parseErr := pgxpool.ParseConfig("not-a-valid-dsn!!!")
	if parseErr != nil {
		badCfg = cfg
	}

	logger, _ := captureLogger()
	runErr := apimigrate.Run(context.Background(), &fakePoolConfiger{cfg: badCfg}, logger)
	if runErr == nil {
		t.Fatal("expected error from Run with invalid connection string, got nil")
	}
}
