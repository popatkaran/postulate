package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/popatkaran/postulate/api/internal/repository/postgres"
)

// --- PostgresTransactor ---

func TestTransactor_WithTransaction_CommitsOnSuccess(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	mock.ExpectBeginTx(pgx.TxOptions{})
	mock.ExpectCommit()

	tr := postgres.NewTransactor(mock)
	err := tr.WithTransaction(context.Background(), func(_ context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestTransactor_WithTransaction_RollsBackOnError(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	fnErr := errors.New("fn failed")
	mock.ExpectBeginTx(pgx.TxOptions{})
	mock.ExpectRollback()

	tr := postgres.NewTransactor(mock)
	err := tr.WithTransaction(context.Background(), func(_ context.Context) error {
		return fnErr
	})
	if !errors.Is(err, fnErr) {
		t.Errorf("expected fnErr, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
