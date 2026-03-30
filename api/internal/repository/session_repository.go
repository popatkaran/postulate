package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/popatkaran/postulate/api/internal/domain"
)

// SessionRepository defines all persistence operations for the Session entity.
type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Session, error)
	UpdateLastActive(ctx context.Context, id uuid.UUID, at time.Time) error
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}
