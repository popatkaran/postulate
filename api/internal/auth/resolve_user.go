package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/popatkaran/postulate/api/internal/domain"
	"github.com/popatkaran/postulate/api/internal/repository"
)

// BootstrapConfig carries the platform admin bootstrap settings derived from
// configuration and environment. It is constructed once at startup and injected
// into UserResolver.
type BootstrapConfig struct {
	// AdminEmail, when non-empty, designates the email that always receives
	// platform_admin on first login (POSTULATE_BOOTSTRAP_ADMIN_EMAIL).
	AdminEmail string
	// DevFallbackEnabled allows the first user on an empty database to receive
	// platform_admin. Must be false when POSTULATE_ENV=production and AdminEmail
	// is not set.
	DevFallbackEnabled bool
}

// UserResolver resolves or creates a Postulate user from an OAuth provider identity.
type UserResolver struct {
	userRepo         repository.UserRepository
	oauthAccountRepo repository.OAuthAccountRepository
	transactor       repository.Transactor
	bootstrap        BootstrapConfig
}

// NewUserResolver constructs a UserResolver.
func NewUserResolver(
	userRepo repository.UserRepository,
	oauthAccountRepo repository.OAuthAccountRepository,
	transactor repository.Transactor,
	bootstrap BootstrapConfig,
) *UserResolver {
	return &UserResolver{
		userRepo:         userRepo,
		oauthAccountRepo: oauthAccountRepo,
		transactor:       transactor,
		bootstrap:        bootstrap,
	}
}

// ResolveOrCreateUser looks up or creates a user from the given OAuth provider identity.
//
// Resolution order:
//  1. Look up oauth_accounts by (provider, provider_uid) — if found, load user,
//     update OAuth token fields, and apply any role upgrade from bootstrap config.
//  2. If not found, look up users by email — if found, insert the oauth_accounts link
//     and preserve the existing role (applying bootstrap upgrade if applicable).
//  3. If neither found, create a new users row inside a transaction and apply
//     bootstrap role assignment. The zero-row check and INSERT are atomic to prevent
//     two concurrent first-logins both receiving platform_admin.
//
// Bootstrap logic runs only during user creation (case 3) or on the first login of
// the designated bootstrap admin email (cases 1 and 2).
func (r *UserResolver) ResolveOrCreateUser(ctx context.Context, pu ProviderUser) (*domain.User, error) {
	if pu.Email == "" {
		return nil, fmt.Errorf("provider returned no email address")
	}

	// 1. Look up by OAuth account.
	account, err := r.oauthAccountRepo.FindByProvider(ctx, pu.Provider, pu.ProviderUID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("find oauth account: %w", err)
	}

	if account != nil {
		// Found — load user and refresh token fields.
		user, err := r.userRepo.FindByID(ctx, account.UserID)
		if err != nil {
			return nil, fmt.Errorf("find user by id: %w", err)
		}
		account.AccessToken = strPtr(pu.AccessToken)
		account.RefreshToken = strPtr(pu.RefreshToken)
		account.TokenExpiry = &pu.TokenExpiry
		if err := r.oauthAccountRepo.Upsert(ctx, account); err != nil {
			return nil, fmt.Errorf("update oauth account tokens: %w", err)
		}
		// Apply bootstrap upgrade if the designated admin email logs in and is
		// currently a member.
		if err := r.applyBootstrapUpgrade(ctx, user); err != nil {
			return nil, err
		}
		return user, nil
	}

	// 2. Look up by email — link may be missing.
	user, err := r.userRepo.FindByEmail(ctx, pu.Email)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	if user != nil {
		// Existing user without an OAuth link — insert the link and apply bootstrap
		// upgrade if applicable.
		if err := r.applyBootstrapUpgrade(ctx, user); err != nil {
			return nil, err
		}
		if err := r.oauthAccountRepo.Upsert(ctx, buildOAuthAccount(user.ID, pu)); err != nil {
			return nil, fmt.Errorf("upsert oauth account: %w", err)
		}
		return user, nil
	}

	// 3. New user — determine role and create atomically.
	//
	// The zero-row check and INSERT run inside a single serializable transaction.
	// This prevents two concurrent first-logins both receiving platform_admin:
	// the second transaction will see count=1 after the first commits and will
	// assign platform_member instead.
	var newUser *domain.User
	err = r.transactor.WithTransaction(ctx, func(txCtx context.Context) error {
		role, err := r.resolveRoleForNewUser(txCtx, pu.Email)
		if err != nil {
			return err
		}
		newUser = &domain.User{
			Email:    pu.Email,
			FullName: pu.Name,
			Role:     role,
			Status:   domain.StatusActive,
		}
		if err := r.userRepo.Create(txCtx, newUser); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		if err := r.oauthAccountRepo.Upsert(txCtx, buildOAuthAccount(newUser.ID, pu)); err != nil {
			return fmt.Errorf("upsert oauth account: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return newUser, nil
}

// applyBootstrapUpgrade upgrades user to platform_admin when the bootstrap admin
// email is configured and matches. No-op for all other users.
func (r *UserResolver) applyBootstrapUpgrade(ctx context.Context, user *domain.User) error {
	if r.bootstrap.AdminEmail == "" || user.Email != r.bootstrap.AdminEmail {
		return nil
	}
	if user.Role == domain.RolePlatformAdmin {
		return nil
	}
	user.Role = domain.RolePlatformAdmin
	if err := r.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("upgrade bootstrap admin role: %w", err)
	}
	return nil
}

// resolveRoleForNewUser determines the role for a brand-new user.
//
// Priority:
//  1. If BootstrapConfig.AdminEmail matches → platform_admin.
//  2. If DevFallbackEnabled and the users table is empty → platform_admin.
//  3. Otherwise → platform_member.
//
// Must be called inside the same transaction as the INSERT so the count is
// consistent and the zero-row check is atomic.
func (r *UserResolver) resolveRoleForNewUser(ctx context.Context, email string) (domain.UserRole, error) {
	// Explicit bootstrap email always wins.
	if r.bootstrap.AdminEmail != "" && email == r.bootstrap.AdminEmail {
		return domain.RolePlatformAdmin, nil
	}

	// Dev fallback: first user on an empty database.
	if r.bootstrap.DevFallbackEnabled {
		count, err := r.userRepo.CountAll(ctx)
		if err != nil {
			return "", fmt.Errorf("count users: %w", err)
		}
		if count == 0 {
			return domain.RolePlatformAdmin, nil
		}
	}

	return domain.RolePlatformMember, nil
}

// buildOAuthAccount constructs a new OAuthAccount from a provider identity.
func buildOAuthAccount(userID uuid.UUID, pu ProviderUser) *domain.OAuthAccount {
	return &domain.OAuthAccount{
		UserID:       userID,
		Provider:     pu.Provider,
		ProviderUID:  pu.ProviderUID,
		Email:        pu.Email,
		AccessToken:  strPtr(pu.AccessToken),
		RefreshToken: strPtr(pu.RefreshToken),
		TokenExpiry:  &pu.TokenExpiry,
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
