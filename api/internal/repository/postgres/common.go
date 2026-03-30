// Package postgres contains pgx-backed implementations of the repository interfaces.
// All pgx and pgconn types are confined to this package.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/popatkaran/postulate/api/internal/database"
	"github.com/popatkaran/postulate/api/internal/domain"
)

// pgxQuerier is the common interface satisfied by both database.Pool and pgx.Tx.
type pgxQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type txKey struct{}

// ContextWithTx stores an active pgx.Tx in the context.
func ContextWithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// TxFromContext retrieves a pgx.Tx from the context, or nil if none is present.
func TxFromContext(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txKey{}).(pgx.Tx)
	return tx
}

// querier returns the active transaction if one is present in ctx, otherwise the pool.
func querier(ctx context.Context, pool database.Pool) pgxQuerier {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return pool
}

// mapErr converts pgx/pgconn errors to domain sentinels.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrConflict
	}
	return err
}
