package domain

import (
	"time"

	"github.com/google/uuid"
)

// OAuthAccount links a third-party OAuth provider identity to a Postulate user.
type OAuthAccount struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Provider    string
	ProviderUID string
	Email       string
	AccessToken  *string
	RefreshToken *string
	TokenExpiry  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
