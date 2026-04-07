package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/popatkaran/postulate/api/internal/database"
	"github.com/popatkaran/postulate/api/internal/domain"
)

// OAuthAccountRepo is the pgx-backed implementation of repository.OAuthAccountRepository.
type OAuthAccountRepo struct{ pool database.Pool }

// NewOAuthAccountRepo constructs an OAuthAccountRepo.
func NewOAuthAccountRepo(pool database.Pool) *OAuthAccountRepo {
	return &OAuthAccountRepo{pool: pool}
}

func (r *OAuthAccountRepo) Upsert(ctx context.Context, a *domain.OAuthAccount) error {
	q := querier(ctx, r.pool)
	return mapErr(q.QueryRow(ctx,
		`INSERT INTO oauth_accounts (user_id, provider, provider_uid, email, access_token, refresh_token, token_expiry)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (provider, provider_uid) DO UPDATE
		   SET email=$4, access_token=$5, refresh_token=$6, token_expiry=$7, updated_at=NOW()
		 RETURNING id, created_at, updated_at`,
		a.UserID, a.Provider, a.ProviderUID, a.Email,
		a.AccessToken, a.RefreshToken, a.TokenExpiry,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt))
}

func (r *OAuthAccountRepo) FindByProvider(ctx context.Context, provider, providerUID string) (*domain.OAuthAccount, error) {
	a := &domain.OAuthAccount{}
	err := querier(ctx, r.pool).QueryRow(ctx,
		`SELECT id, user_id, provider, provider_uid, email, access_token, refresh_token, token_expiry, created_at, updated_at
		 FROM oauth_accounts WHERE provider=$1 AND provider_uid=$2`,
		provider, providerUID,
	).Scan(&a.ID, &a.UserID, &a.Provider, &a.ProviderUID, &a.Email,
		&a.AccessToken, &a.RefreshToken, &a.TokenExpiry, &a.CreatedAt, &a.UpdatedAt)
	return a, mapErr(err)
}

func (r *OAuthAccountRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.OAuthAccount, error) {
	rows, err := querier(ctx, r.pool).Query(ctx,
		`SELECT id, user_id, provider, provider_uid, email, access_token, refresh_token, token_expiry, created_at, updated_at
		 FROM oauth_accounts WHERE user_id=$1`,
		userID,
	)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	var accounts []*domain.OAuthAccount
	for rows.Next() {
		a := &domain.OAuthAccount{}
		if err := rows.Scan(&a.ID, &a.UserID, &a.Provider, &a.ProviderUID, &a.Email,
			&a.AccessToken, &a.RefreshToken, &a.TokenExpiry, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, mapErr(err)
		}
		accounts = append(accounts, a)
	}
	return accounts, mapErr(rows.Err())
}
