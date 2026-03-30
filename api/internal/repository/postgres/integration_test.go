//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/popatkaran/postulate/api/internal/config"
	"github.com/popatkaran/postulate/api/internal/database"
	"github.com/popatkaran/postulate/api/internal/domain"
	apimigrate "github.com/popatkaran/postulate/api/internal/migrate"
	"github.com/popatkaran/postulate/api/internal/repository/postgres"
)

var iLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg := config.DatabaseConfig{
		Host: "localhost", Port: 5432, Name: "postulate_test",
		User: "postulate_dev", Password: "postulate_dev", SSLMode: "disable",
		MaxOpenConns: 5, MaxIdleConns: 1, ConnMaxLifetimeSeconds: 60,
	}
	pool, err := database.New(context.Background(), cfg, iLogger)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	if err := apimigrate.Run(context.Background(), pool, iLogger); err != nil {
		pool.Close()
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

func newUser(email string) *domain.User {
	return &domain.User{
		Email:        email,
		PasswordHash: "hash",
		FullName:     "Test User",
		Role:         domain.RoleMember,
		Status:       domain.StatusActive,
	}
}

// ── UserRepository ────────────────────────────────────────────────────────────

func TestUserRepo_Create_And_FindByEmail(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	repo := postgres.NewUserRepo(pool)
	ctx := context.Background()

	u := newUser("create@example.com")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if u.ID == (uuid.UUID{}) {
		t.Error("expected ID to be populated after Create")
	}

	found, err := repo.FindByEmail(ctx, u.Email)
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}
	if found.ID != u.ID {
		t.Errorf("ID mismatch: got %v want %v", found.ID, u.ID)
	}

	t.Cleanup(func() { pool.Exec(ctx, "DELETE FROM users WHERE id=$1", u.ID) }) //nolint:errcheck
}

func TestUserRepo_Create_DuplicateEmail_ReturnsErrConflict(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	repo := postgres.NewUserRepo(pool)
	ctx := context.Background()

	u := newUser("dup@example.com")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	err := repo.Create(ctx, newUser("dup@example.com"))
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, "DELETE FROM users WHERE email='dup@example.com'") }) //nolint:errcheck
}

func TestUserRepo_FindByID_NotFound_ReturnsErrNotFound(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	repo := postgres.NewUserRepo(pool)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserRepo_SoftDelete_SetsDeletedAt(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	repo := postgres.NewUserRepo(pool)
	ctx := context.Background()

	u := newUser("softdel@example.com")
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.SoftDelete(ctx, u.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	found, err := repo.FindByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("FindByID after soft delete: %v", err)
	}
	if found.DeletedAt == nil {
		t.Error("expected deleted_at to be set")
	}
	t.Cleanup(func() { pool.Exec(ctx, "DELETE FROM users WHERE id=$1", u.ID) }) //nolint:errcheck
}

// ── SessionRepository ─────────────────────────────────────────────────────────

func TestSessionRepo_Create_And_Revoke(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	ctx := context.Background()
	userRepo := postgres.NewUserRepo(pool)
	sessionRepo := postgres.NewSessionRepo(pool)

	u := newUser("session@example.com")
	if err := userRepo.Create(ctx, u); err != nil {
		t.Fatalf("Create user: %v", err)
	}

	s := &domain.Session{
		UserID:       u.ID,
		TokenHash:    "session-hash-" + uuid.NewString(),
		UserAgent:    "test",
		LastActiveAt: time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
	}
	if err := sessionRepo.Create(ctx, s); err != nil {
		t.Fatalf("Create session: %v", err)
	}
	if err := sessionRepo.Revoke(ctx, s.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	found, err := sessionRepo.FindByTokenHash(ctx, s.TokenHash)
	if err != nil {
		t.Fatalf("FindByTokenHash: %v", err)
	}
	if found.RevokedAt == nil {
		t.Error("expected revoked_at to be set")
	}
	t.Cleanup(func() { pool.Exec(ctx, "DELETE FROM users WHERE id=$1", u.ID) }) //nolint:errcheck
}

func TestSessionRepo_RevokeAllForUser(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	ctx := context.Background()
	userRepo := postgres.NewUserRepo(pool)
	sessionRepo := postgres.NewSessionRepo(pool)

	u := newUser("revokeall@example.com")
	if err := userRepo.Create(ctx, u); err != nil {
		t.Fatalf("Create user: %v", err)
	}
	for i := 0; i < 2; i++ {
		s := &domain.Session{
			UserID: u.ID, TokenHash: "hash-" + uuid.NewString(),
			LastActiveAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour),
		}
		if err := sessionRepo.Create(ctx, s); err != nil {
			t.Fatalf("Create session: %v", err)
		}
	}
	if err := sessionRepo.RevokeAllForUser(ctx, u.ID); err != nil {
		t.Fatalf("RevokeAllForUser: %v", err)
	}
	sessions, err := sessionRepo.FindByUserID(ctx, u.ID)
	if err != nil {
		t.Fatalf("FindByUserID: %v", err)
	}
	for _, s := range sessions {
		if s.RevokedAt == nil {
			t.Error("expected all sessions to be revoked")
		}
	}
	t.Cleanup(func() { pool.Exec(ctx, "DELETE FROM users WHERE id=$1", u.ID) }) //nolint:errcheck
}

func TestSessionRepo_DeleteExpired(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	ctx := context.Background()
	userRepo := postgres.NewUserRepo(pool)
	sessionRepo := postgres.NewSessionRepo(pool)

	u := newUser("expired@example.com")
	if err := userRepo.Create(ctx, u); err != nil {
		t.Fatalf("Create user: %v", err)
	}
	s := &domain.Session{
		UserID: u.ID, TokenHash: "expired-" + uuid.NewString(),
		LastActiveAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt:    time.Now().Add(-time.Hour),
	}
	if err := sessionRepo.Create(ctx, s); err != nil {
		t.Fatalf("Create session: %v", err)
	}
	n, err := sessionRepo.DeleteExpired(ctx, time.Now())
	if err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}
	if n < 1 {
		t.Errorf("expected at least 1 deleted, got %d", n)
	}
	t.Cleanup(func() { pool.Exec(ctx, "DELETE FROM users WHERE id=$1", u.ID) }) //nolint:errcheck
}

// ── RefreshTokenRepository ────────────────────────────────────────────────────

func TestRefreshTokenRepo_Create_MarkUsed_DeleteBySession(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	ctx := context.Background()
	userRepo := postgres.NewUserRepo(pool)
	sessionRepo := postgres.NewSessionRepo(pool)
	tokenRepo := postgres.NewRefreshTokenRepo(pool)

	u := newUser("token@example.com")
	if err := userRepo.Create(ctx, u); err != nil {
		t.Fatalf("Create user: %v", err)
	}
	s := &domain.Session{
		UserID: u.ID, TokenHash: "sess-" + uuid.NewString(),
		LastActiveAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := sessionRepo.Create(ctx, s); err != nil {
		t.Fatalf("Create session: %v", err)
	}
	tok := &domain.RefreshToken{
		SessionID: s.ID, UserID: u.ID,
		TokenHash: "tok-" + uuid.NewString(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := tokenRepo.Create(ctx, tok); err != nil {
		t.Fatalf("Create token: %v", err)
	}

	usedAt := time.Now()
	if err := tokenRepo.MarkUsed(ctx, tok.ID, usedAt); err != nil {
		t.Fatalf("MarkUsed: %v", err)
	}
	found, err := tokenRepo.FindByTokenHash(ctx, tok.TokenHash)
	if err != nil {
		t.Fatalf("FindByTokenHash: %v", err)
	}
	if found.UsedAt == nil {
		t.Error("expected used_at to be set")
	}

	if err := tokenRepo.DeleteBySessionID(ctx, s.ID); err != nil {
		t.Fatalf("DeleteBySessionID: %v", err)
	}
	_, err = tokenRepo.FindByTokenHash(ctx, tok.TokenHash)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, "DELETE FROM users WHERE id=$1", u.ID) }) //nolint:errcheck
}

// ── Transactor ────────────────────────────────────────────────────────────────

func TestTransactor_Commit(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	ctx := context.Background()
	tx := postgres.NewTransactor(pool)
	userRepo := postgres.NewUserRepo(pool)

	u := newUser("tx-commit@example.com")
	err := tx.WithTransaction(ctx, func(txCtx context.Context) error {
		return userRepo.Create(txCtx, u)
	})
	if err != nil {
		t.Fatalf("WithTransaction: %v", err)
	}
	_, err = userRepo.FindByID(ctx, u.ID)
	if err != nil {
		t.Errorf("expected committed user to be found, got %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, "DELETE FROM users WHERE id=$1", u.ID) }) //nolint:errcheck
}

func TestTransactor_Rollback(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	ctx := context.Background()
	tx := postgres.NewTransactor(pool)
	userRepo := postgres.NewUserRepo(pool)

	u := newUser("tx-rollback@example.com")
	_ = tx.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := userRepo.Create(txCtx, u); err != nil {
			return err
		}
		return errors.New("force rollback")
	})

	_, err := userRepo.FindByEmail(ctx, u.Email)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected rolled-back user to be absent, got %v", err)
	}
}
