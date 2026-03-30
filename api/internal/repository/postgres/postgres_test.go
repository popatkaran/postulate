package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/popatkaran/postulate/api/internal/domain"
	"github.com/popatkaran/postulate/api/internal/repository/postgres"
)

// TestMapErr_ErrNoRows_ReturnsDomainErrNotFound verifies the sentinel mapping
// without a real database by exercising the exported error path via FindByID.
// We use a nil UUID which will fail to connect — but we test mapErr directly
// through the exported helper.
func TestContextWithTx_RoundTrip(t *testing.T) {
	tx := &fakeTx{}
	ctx := postgres.ContextWithTx(context.Background(), tx)
	got := postgres.TxFromContext(ctx)
	if got != tx {
		t.Error("expected TxFromContext to return the stored tx")
	}
}

func TestTxFromContext_NilWhenAbsent(t *testing.T) {
	got := postgres.TxFromContext(context.Background())
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

// fakeTx satisfies pgx.Tx minimally for the context round-trip test.
type fakeTx struct{ pgx.Tx }

// --- domain error sentinel tests via mapErr (exported for testing) ---

func TestMapErr_Nil_ReturnsNil(t *testing.T) {
	if err := postgres.MapErr(nil); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestMapErr_ErrNoRows_ReturnsErrNotFound(t *testing.T) {
	if err := postgres.MapErr(pgx.ErrNoRows); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMapErr_UniqueViolation_ReturnsErrConflict(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505"}
	if err := postgres.MapErr(pgErr); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestMapErr_OtherError_PassesThrough(t *testing.T) {
	orig := errors.New("some db error")
	if err := postgres.MapErr(orig); !errors.Is(err, orig) {
		t.Errorf("expected original error, got %v", err)
	}
}

func TestMapErr_OtherPgError_PassesThrough(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "42P01"} // undefined table
	if err := postgres.MapErr(pgErr); errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected passthrough, got sentinel: %v", err)
	}
}

// --- compile-time interface satisfaction checks ---

var _ interface {
	Create(context.Context, *domain.User) error
} = (*postgres.UserRepo)(nil)
var _ interface {
	Create(context.Context, *domain.Session) error
} = (*postgres.SessionRepo)(nil)
var _ interface {
	Create(context.Context, *domain.RefreshToken) error
} = (*postgres.RefreshTokenRepo)(nil)
var _ interface {
	WithTransaction(context.Context, func(context.Context) error) error
} = (*postgres.PostgresTransactor)(nil)

// --- uuid sentinel: ensure domain types use google/uuid ---
var _ uuid.UUID = domain.User{}.ID
var _ uuid.UUID = domain.Session{}.UserID
var _ uuid.UUID = domain.RefreshToken{}.SessionID
