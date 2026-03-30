package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/popatkaran/postulate/api/internal/domain"
	"github.com/popatkaran/postulate/api/internal/repository/postgres"
)

// --- SessionRepo ---

func TestSessionRepo_Create_HappyPath(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	id := uuid.New()
	now := time.Now()
	s := &domain.Session{
		UserID: uuid.New(), TokenHash: "hash", IPAddress: "127.0.0.1",
		UserAgent: "test", LastActiveAt: now, ExpiresAt: now.Add(time.Hour),
	}

	mock.ExpectQuery(`INSERT INTO sessions`).
		WithArgs(s.UserID, s.TokenHash, s.IPAddress, s.UserAgent, s.LastActiveAt, s.ExpiresAt).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at"}).AddRow(id, now))

	repo := postgres.NewSessionRepo(mock)
	if err := repo.Create(context.Background(), s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.ID != id {
		t.Errorf("expected ID %v, got %v", id, s.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestSessionRepo_FindByTokenHash_Found(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	id, userID := uuid.New(), uuid.New()
	now := time.Now()
	mock.ExpectQuery(`SELECT .* FROM sessions WHERE token_hash`).
		WithArgs("tok").
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "user_id", "token_hash", "ip_address", "user_agent",
			"last_active_at", "expires_at", "created_at", "revoked_at",
		}).AddRow(id, userID, "tok", "127.0.0.1", "ua", now, now, now, nil))

	repo := postgres.NewSessionRepo(mock)
	s, err := repo.FindByTokenHash(context.Background(), "tok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.ID != id {
		t.Errorf("expected ID %v, got %v", id, s.ID)
	}
}

func TestSessionRepo_FindByTokenHash_NotFound(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	mock.ExpectQuery(`SELECT .* FROM sessions WHERE token_hash`).
		WithArgs("missing").
		WillReturnError(pgx.ErrNoRows)

	repo := postgres.NewSessionRepo(mock)
	_, err := repo.FindByTokenHash(context.Background(), "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSessionRepo_FindByUserID(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	userID := uuid.New()
	now := time.Now()
	mock.ExpectQuery(`SELECT .* FROM sessions WHERE user_id`).
		WithArgs(userID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "user_id", "token_hash", "ip_address", "user_agent",
			"last_active_at", "expires_at", "created_at", "revoked_at",
		}).AddRow(uuid.New(), userID, "tok", "127.0.0.1", "ua", now, now, now, nil))

	repo := postgres.NewSessionRepo(mock)
	sessions, err := repo.FindByUserID(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}
}

func TestSessionRepo_UpdateLastActive(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	id := uuid.New()
	at := time.Now()
	mock.ExpectExec(`UPDATE sessions SET last_active_at`).
		WithArgs(at, id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := postgres.NewSessionRepo(mock)
	if err := repo.UpdateLastActive(context.Background(), id, at); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestSessionRepo_Revoke(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	id := uuid.New()
	mock.ExpectExec(`UPDATE sessions SET revoked_at`).
		WithArgs(id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := postgres.NewSessionRepo(mock)
	if err := repo.Revoke(context.Background(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSessionRepo_RevokeAllForUser(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	userID := uuid.New()
	mock.ExpectExec(`UPDATE sessions SET revoked_at=NOW\(\) WHERE user_id`).
		WithArgs(userID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 2))

	repo := postgres.NewSessionRepo(mock)
	if err := repo.RevokeAllForUser(context.Background(), userID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSessionRepo_DeleteExpired(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	before := time.Now()
	mock.ExpectExec(`DELETE FROM sessions WHERE expires_at`).
		WithArgs(before).
		WillReturnResult(pgxmock.NewResult("DELETE", 3))

	repo := postgres.NewSessionRepo(mock)
	n, err := repo.DeleteExpired(context.Background(), before)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 deleted, got %d", n)
	}
}
