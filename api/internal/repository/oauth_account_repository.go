package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/popatkaran/postulate/api/internal/domain"
)

// OAuthAccountRepository defines persistence operations for OAuth provider accounts.
type OAuthAccountRepository interface {
	// Upsert inserts or updates the OAuth account for the given provider + provider_uid.
	Upsert(ctx context.Context, account *domain.OAuthAccount) error
	// FindByProvider returns the OAuth account for the given provider and provider_uid.
	FindByProvider(ctx context.Context, provider, providerUID string) (*domain.OAuthAccount, error)
	// FindByUserID returns all OAuth accounts linked to the given user.
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.OAuthAccount, error)
}
