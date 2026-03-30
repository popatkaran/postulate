package repository

import "context"

// Transactor executes a function within a database transaction.
// The transaction is committed if fn returns nil, rolled back otherwise.
type Transactor interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
