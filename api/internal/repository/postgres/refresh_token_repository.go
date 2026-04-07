package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/popatkaran/postulate/api/internal/database"
	"github.com/popatkaran/postulate/api/internal/domain"
)

// RefreshTokenRepo is the pgx-backed implementation of repository.RefreshTokenRepository.
type RefreshTokenRepo struct{ pool database.Pool }

// NewRefreshTokenRepo constructs a RefreshTokenRepo.
func NewRefreshTokenRepo(pool database.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{pool: pool}
}

func (r *RefreshTokenRepo) Create(ctx context.Context, t *domain.RefreshToken) error {
	return mapErr(querier(ctx, r.pool).QueryRow(ctx,
		`INSERT INTO refresh_tokens (session_id, user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		t.SessionID, t.UserID, t.TokenHash, t.ExpiresAt,
	).Scan(&t.ID, &t.CreatedAt))
}

func (r *RefreshTokenRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	t := &domain.RefreshToken{}
	err := querier(ctx, r.pool).QueryRow(ctx,
		`SELECT id, session_id, user_id, token_hash, expires_at, used_at, created_at
		 FROM refresh_tokens WHERE token_hash = $1`, tokenHash,
	).Scan(&t.ID, &t.SessionID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt)
	return t, mapErr(err)
}

func (r *RefreshTokenRepo) MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := querier(ctx, r.pool).Exec(ctx,
		`UPDATE refresh_tokens SET used_at=$1 WHERE id=$2`, at, id)
	return mapErr(err)
}

func (r *RefreshTokenRepo) DeleteBySessionID(ctx context.Context, sessionID uuid.UUID) error {
	_, err := querier(ctx, r.pool).Exec(ctx,
		`DELETE FROM refresh_tokens WHERE session_id=$1`, sessionID)
	return mapErr(err)
}

func (r *RefreshTokenRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := querier(ctx, r.pool).Exec(ctx,
		`DELETE FROM refresh_tokens WHERE user_id=$1`, userID)
	return mapErr(err)
}

func (r *RefreshTokenRepo) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	tag, err := querier(ctx, r.pool).Exec(ctx,
		`DELETE FROM refresh_tokens WHERE expires_at < $1`, before)
	return tag.RowsAffected(), mapErr(err)
}
