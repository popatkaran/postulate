package domain

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a single-use token used to obtain a new session token.
type RefreshToken struct {
	ID        uuid.UUID
	SessionID uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
