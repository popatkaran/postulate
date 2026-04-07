package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/popatkaran/postulate/api/internal/domain"
)

// RefreshTokenRepository defines all persistence operations for the RefreshToken entity.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error
	DeleteBySessionID(ctx context.Context, sessionID uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}
