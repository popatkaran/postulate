package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/popatkaran/postulate/api/internal/database"
	"github.com/popatkaran/postulate/api/internal/domain"
)

// SessionRepo is the pgx-backed implementation of repository.SessionRepository.
type SessionRepo struct{ pool database.Pool }

// NewSessionRepo constructs a SessionRepo.
func NewSessionRepo(pool database.Pool) *SessionRepo { return &SessionRepo{pool: pool} }

func (r *SessionRepo) Create(ctx context.Context, s *domain.Session) error {
	return mapErr(querier(ctx, r.pool).QueryRow(ctx,
		`INSERT INTO sessions (user_id, token_hash, ip_address, user_agent, last_active_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		s.UserID, s.TokenHash, s.IPAddress, s.UserAgent, s.LastActiveAt, s.ExpiresAt,
	).Scan(&s.ID, &s.CreatedAt))
}

func (r *SessionRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	s := &domain.Session{}
	err := querier(ctx, r.pool).QueryRow(ctx,
		`SELECT id, user_id, token_hash, ip_address, user_agent, last_active_at,
		        expires_at, created_at, revoked_at
		 FROM sessions WHERE token_hash = $1`, tokenHash,
	).Scan(&s.ID, &s.UserID, &s.TokenHash, &s.IPAddress, &s.UserAgent,
		&s.LastActiveAt, &s.ExpiresAt, &s.CreatedAt, &s.RevokedAt)
	return s, mapErr(err)
}

func (r *SessionRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Session, error) {
	rows, err := querier(ctx, r.pool).Query(ctx,
		`SELECT id, user_id, token_hash, ip_address, user_agent, last_active_at,
		        expires_at, created_at, revoked_at
		 FROM sessions WHERE user_id = $1`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.Session
	for rows.Next() {
		s := &domain.Session{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.TokenHash, &s.IPAddress, &s.UserAgent,
			&s.LastActiveAt, &s.ExpiresAt, &s.CreatedAt, &s.RevokedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *SessionRepo) UpdateLastActive(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := querier(ctx, r.pool).Exec(ctx,
		`UPDATE sessions SET last_active_at=$1 WHERE id=$2`, at, id)
	return mapErr(err)
}

func (r *SessionRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := querier(ctx, r.pool).Exec(ctx,
		`UPDATE sessions SET revoked_at=NOW() WHERE id=$1 AND revoked_at IS NULL`, id)
	return mapErr(err)
}

func (r *SessionRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := querier(ctx, r.pool).Exec(ctx,
		`UPDATE sessions SET revoked_at=NOW() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	return mapErr(err)
}

func (r *SessionRepo) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	tag, err := querier(ctx, r.pool).Exec(ctx,
		`DELETE FROM sessions WHERE expires_at < $1`, before)
	return tag.RowsAffected(), mapErr(err)
}
