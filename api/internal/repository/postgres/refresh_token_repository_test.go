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

// --- RefreshTokenRepo ---

func TestRefreshTokenRepo_Create_HappyPath(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	id := uuid.New()
	now := time.Now()
	tok := &domain.RefreshToken{
		SessionID: uuid.New(), UserID: uuid.New(),
		TokenHash: "rthash", ExpiresAt: now.Add(time.Hour),
	}

	mock.ExpectQuery(`INSERT INTO refresh_tokens`).
		WithArgs(tok.SessionID, tok.UserID, tok.TokenHash, tok.ExpiresAt).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at"}).AddRow(id, now))

	repo := postgres.NewRefreshTokenRepo(mock)
	if err := repo.Create(context.Background(), tok); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.ID != id {
		t.Errorf("expected ID %v, got %v", id, tok.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRefreshTokenRepo_FindByTokenHash_Found(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	id, sessID, userID := uuid.New(), uuid.New(), uuid.New()
	now := time.Now()
	mock.ExpectQuery(`SELECT .* FROM refresh_tokens WHERE token_hash`).
		WithArgs("rthash").
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "session_id", "user_id", "token_hash", "expires_at", "used_at", "created_at",
		}).AddRow(id, sessID, userID, "rthash", now, nil, now))

	repo := postgres.NewRefreshTokenRepo(mock)
	tok, err := repo.FindByTokenHash(context.Background(), "rthash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.ID != id {
		t.Errorf("expected ID %v, got %v", id, tok.ID)
	}
}

func TestRefreshTokenRepo_FindByTokenHash_NotFound(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	mock.ExpectQuery(`SELECT .* FROM refresh_tokens WHERE token_hash`).
		WithArgs("missing").
		WillReturnError(pgx.ErrNoRows)

	repo := postgres.NewRefreshTokenRepo(mock)
	_, err := repo.FindByTokenHash(context.Background(), "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRefreshTokenRepo_MarkUsed(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	id := uuid.New()
	at := time.Now()
	mock.ExpectExec(`UPDATE refresh_tokens SET used_at`).
		WithArgs(at, id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := postgres.NewRefreshTokenRepo(mock)
	if err := repo.MarkUsed(context.Background(), id, at); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRefreshTokenRepo_DeleteBySessionID(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	sessID := uuid.New()
	mock.ExpectExec(`DELETE FROM refresh_tokens WHERE session_id`).
		WithArgs(sessID).
		WillReturnResult(pgxmock.NewResult("DELETE", 2))

	repo := postgres.NewRefreshTokenRepo(mock)
	if err := repo.DeleteBySessionID(context.Background(), sessID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRefreshTokenRepo_DeleteExpired(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	before := time.Now()
	mock.ExpectExec(`DELETE FROM refresh_tokens WHERE expires_at`).
		WithArgs(before).
		WillReturnResult(pgxmock.NewResult("DELETE", 5))

	repo := postgres.NewRefreshTokenRepo(mock)
	n, err := repo.DeleteExpired(context.Background(), before)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 deleted, got %d", n)
	}
}

func TestRefreshTokenRepo_DeleteByUserID(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()

	userID := uuid.New()
	mock.ExpectExec(`DELETE FROM refresh_tokens WHERE user_id`).
		WithArgs(userID).
		WillReturnResult(pgxmock.NewResult("DELETE", 3))

	repo := postgres.NewRefreshTokenRepo(mock)
	if err := repo.DeleteByUserID(context.Background(), userID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
