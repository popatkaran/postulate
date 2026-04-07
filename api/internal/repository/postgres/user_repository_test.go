package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/popatkaran/postulate/api/internal/domain"
	"github.com/popatkaran/postulate/api/internal/repository/postgres"
)

// newMock creates a pgxmock pool for unit tests.
func newMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	return mock
}

// --- UserRepo ---

func TestUserRepo_Create_HappyPath(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	id := uuid.New()
	now := time.Now()
	hash := "hash"
	u := &domain.User{
		Email: "test@example.com", EmailVerified: false,
		PasswordHash: &hash, FullName: "Test User",
		Role: domain.RolePlatformMember, Status: domain.StatusActive,
	}

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(u.Email, u.EmailVerified, u.PasswordHash, u.FullName, u.Role, u.Status).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(id, now, now))

	repo := postgres.NewUserRepo(mock)
	if err := repo.Create(context.Background(), u); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != id {
		t.Errorf("expected ID %v, got %v", id, u.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestUserRepo_Create_UniqueViolation_ReturnsErrConflict(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	hash := "h"
	u := &domain.User{Email: "dup@example.com", PasswordHash: &hash, FullName: "X", Role: domain.RolePlatformMember, Status: domain.StatusActive}
	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(u.Email, u.EmailVerified, u.PasswordHash, u.FullName, u.Role, u.Status).
		WillReturnError(&pgconn.PgError{Code: "23505"})

	repo := postgres.NewUserRepo(mock)
	err := repo.Create(context.Background(), u)
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestUserRepo_FindByID_Found(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	id := uuid.New()
	now := time.Now()
	hash := "h"
	mock.ExpectQuery(`SELECT .* FROM users WHERE id`).
		WithArgs(id).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "email", "email_verified", "password_hash", "full_name",
			"role", "status", "created_at", "updated_at", "deleted_at",
		}).AddRow(id, "a@b.com", false, &hash, "Name", "platform_member", "active", now, now, nil))

	repo := postgres.NewUserRepo(mock)
	u, err := repo.FindByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != id {
		t.Errorf("expected ID %v, got %v", id, u.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestUserRepo_FindByID_NotFound_ReturnsErrNotFound(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	id := uuid.New()
	mock.ExpectQuery(`SELECT .* FROM users WHERE id`).
		WithArgs(id).
		WillReturnError(pgx.ErrNoRows)

	repo := postgres.NewUserRepo(mock)
	_, err := repo.FindByID(context.Background(), id)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserRepo_FindByEmail_Found(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	id := uuid.New()
	now := time.Now()
	hash := "h"
	mock.ExpectQuery(`SELECT .* FROM users WHERE email`).
		WithArgs("a@b.com").
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "email", "email_verified", "password_hash", "full_name",
			"role", "status", "created_at", "updated_at", "deleted_at",
		}).AddRow(id, "a@b.com", false, &hash, "Name", "platform_member", "active", now, now, nil))

	repo := postgres.NewUserRepo(mock)
	u, err := repo.FindByEmail(context.Background(), "a@b.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Email != "a@b.com" {
		t.Errorf("expected email a@b.com, got %v", u.Email)
	}
}

func TestUserRepo_FindByEmail_NotFound_ReturnsErrNotFound(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	mock.ExpectQuery(`SELECT .* FROM users WHERE email`).
		WithArgs("missing@b.com").
		WillReturnError(pgx.ErrNoRows)

	repo := postgres.NewUserRepo(mock)
	_, err := repo.FindByEmail(context.Background(), "missing@b.com")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserRepo_Update(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	hash := "h"
	u := &domain.User{ID: uuid.New(), Email: "a@b.com", PasswordHash: &hash, FullName: "N", Role: domain.RolePlatformMember, Status: domain.StatusActive}
	mock.ExpectExec(`UPDATE users SET`).
		WithArgs(u.Email, u.EmailVerified, u.PasswordHash, u.FullName, u.Role, u.Status, u.ID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := postgres.NewUserRepo(mock)
	if err := repo.Update(context.Background(), u); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestUserRepo_SoftDelete(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	id := uuid.New()
	mock.ExpectExec(`UPDATE users SET deleted_at`).
		WithArgs(id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := postgres.NewUserRepo(mock)
	if err := repo.SoftDelete(context.Background(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestUserRepo_CountAll(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(int64(3)))

	repo := postgres.NewUserRepo(mock)
	n, err := repo.CountAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3, got %d", n)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// ── OAuthAccountRepo ──────────────────────────────────────────────────────────

func TestOAuthAccountRepo_Upsert_HappyPath(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	id := uuid.New()
	userID := uuid.New()
	now := time.Now()
	at := "access"
	rt := "refresh"
	expiry := now.Add(time.Hour)

	a := &domain.OAuthAccount{
		UserID: userID, Provider: "google", ProviderUID: "uid-1",
		Email: "a@b.com", AccessToken: &at, RefreshToken: &rt, TokenExpiry: &expiry,
	}

	mock.ExpectQuery(`INSERT INTO oauth_accounts`).
		WithArgs(a.UserID, a.Provider, a.ProviderUID, a.Email, a.AccessToken, a.RefreshToken, a.TokenExpiry).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(id, now, now))

	repo := postgres.NewOAuthAccountRepo(mock)
	if err := repo.Upsert(context.Background(), a); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if a.ID != id {
		t.Errorf("expected ID %v, got %v", id, a.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestOAuthAccountRepo_FindByProvider_Found(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	id := uuid.New()
	userID := uuid.New()
	now := time.Now()
	expiry := now.Add(time.Hour)

	mock.ExpectQuery(`SELECT .* FROM oauth_accounts WHERE provider`).
		WithArgs("google", "uid-1").
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "user_id", "provider", "provider_uid", "email",
			"access_token", "refresh_token", "token_expiry", "created_at", "updated_at",
		}).AddRow(id, userID, "google", "uid-1", "a@b.com", nil, nil, &expiry, now, now))

	repo := postgres.NewOAuthAccountRepo(mock)
	a, err := repo.FindByProvider(context.Background(), "google", "uid-1")
	if err != nil {
		t.Fatalf("FindByProvider: %v", err)
	}
	if a.ID != id {
		t.Errorf("expected ID %v, got %v", id, a.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestOAuthAccountRepo_FindByProvider_NotFound(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	mock.ExpectQuery(`SELECT .* FROM oauth_accounts WHERE provider`).
		WithArgs("google", "missing").
		WillReturnError(pgx.ErrNoRows)

	repo := postgres.NewOAuthAccountRepo(mock)
	_, err := repo.FindByProvider(context.Background(), "google", "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestOAuthAccountRepo_FindByUserID_ReturnsAccounts(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	userID := uuid.New()
	id1 := uuid.New()
	now := time.Now()
	expiry := now.Add(time.Hour)

	mock.ExpectQuery(`SELECT .* FROM oauth_accounts WHERE user_id`).
		WithArgs(userID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "user_id", "provider", "provider_uid", "email",
			"access_token", "refresh_token", "token_expiry", "created_at", "updated_at",
		}).AddRow(id1, userID, "google", "uid-1", "a@b.com", nil, nil, &expiry, now, now))

	repo := postgres.NewOAuthAccountRepo(mock)
	accounts, err := repo.FindByUserID(context.Background(), userID)
	if err != nil {
		t.Fatalf("FindByUserID: %v", err)
	}
	if len(accounts) != 1 {
		t.Errorf("expected 1 account, got %d", len(accounts))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestOAuthAccountRepo_FindByUserID_Empty(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	userID := uuid.New()
	mock.ExpectQuery(`SELECT .* FROM oauth_accounts WHERE user_id`).
		WithArgs(userID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "user_id", "provider", "provider_uid", "email",
			"access_token", "refresh_token", "token_expiry", "created_at", "updated_at",
		}))

	repo := postgres.NewOAuthAccountRepo(mock)
	accounts, err := repo.FindByUserID(context.Background(), userID)
	if err != nil {
		t.Fatalf("FindByUserID: %v", err)
	}
	if len(accounts) != 0 {
		t.Errorf("expected 0 accounts, got %d", len(accounts))
	}
}

func TestTransactor_BeginError_ReturnsError(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	tx := postgres.NewTransactor(mock)
	err := tx.WithTransaction(context.Background(), func(_ context.Context) error { return nil })
	if err == nil {
		t.Fatal("expected error when BeginTx fails")
	}
}
