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
	u := &domain.User{
		Email: "test@example.com", EmailVerified: false,
		PasswordHash: "hash", FullName: "Test User",
		Role: domain.RoleMember, Status: domain.StatusActive,
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

	u := &domain.User{Email: "dup@example.com", PasswordHash: "h", FullName: "X", Role: domain.RoleMember, Status: domain.StatusActive}
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
	mock.ExpectQuery(`SELECT .* FROM users WHERE id`).
		WithArgs(id).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "email", "email_verified", "password_hash", "full_name",
			"role", "status", "created_at", "updated_at", "deleted_at",
		}).AddRow(id, "a@b.com", false, "h", "Name", "member", "active", now, now, nil))

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
	mock.ExpectQuery(`SELECT .* FROM users WHERE email`).
		WithArgs("a@b.com").
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "email", "email_verified", "password_hash", "full_name",
			"role", "status", "created_at", "updated_at", "deleted_at",
		}).AddRow(id, "a@b.com", false, "h", "Name", "member", "active", now, now, nil))

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

	u := &domain.User{ID: uuid.New(), Email: "a@b.com", PasswordHash: "h", FullName: "N", Role: domain.RoleMember, Status: domain.StatusActive}
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
