package domain

import (
	"time"

	"github.com/google/uuid"
)

// Session represents an active login session for a user.
type Session struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	TokenHash    string
	IPAddress    string
	UserAgent    string
	LastActiveAt time.Time
	ExpiresAt    time.Time
	CreatedAt    time.Time
	RevokedAt    *time.Time
}
