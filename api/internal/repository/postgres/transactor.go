package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/popatkaran/postulate/api/internal/database"
)

// PostgresTransactor implements repository.Transactor using database.Pool.
type PostgresTransactor struct{ pool database.Pool }

// NewTransactor constructs a PostgresTransactor.
func NewTransactor(pool database.Pool) *PostgresTransactor {
	return &PostgresTransactor{pool: pool}
}

// WithTransaction begins a pgx transaction, injects it into ctx, calls fn,
// commits on nil return, and rolls back on any error.
func (t *PostgresTransactor) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	txCtx := ContextWithTx(ctx, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}
