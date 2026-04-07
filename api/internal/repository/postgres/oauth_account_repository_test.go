//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/popatkaran/postulate/api/internal/domain"
	"github.com/popatkaran/postulate/api/internal/repository/postgres"
)

func TestOAuthAccountRepo_Upsert_And_FindByProvider(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	ctx := context.Background()

	userRepo := postgres.NewUserRepo(pool)
	oauthRepo := postgres.NewOAuthAccountRepo(pool)

	u := newUser("oauth-upsert@example.com")
	if err := userRepo.Create(ctx, u); err != nil {
		t.Fatalf("Create user: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, "DELETE FROM users WHERE id=$1", u.ID) }) //nolint:errcheck

	acc := &domain.OAuthAccount{
		UserID:      u.ID,
		Provider:    "google",
		ProviderUID: "google-uid-001",
		Email:       "oauth-upsert@example.com",
	}
	if err := oauthRepo.Upsert(ctx, acc); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if acc.ID == (uuid.UUID{}) {
		t.Error("expected ID to be populated after Upsert")
	}

	found, err := oauthRepo.FindByProvider(ctx, "google", "google-uid-001")
	if err != nil {
		t.Fatalf("FindByProvider: %v", err)
	}
	if found.UserID != u.ID {
		t.Errorf("UserID mismatch: got %v want %v", found.UserID, u.ID)
	}
}

func TestOAuthAccountRepo_Upsert_UpdatesExisting(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	ctx := context.Background()

	userRepo := postgres.NewUserRepo(pool)
	oauthRepo := postgres.NewOAuthAccountRepo(pool)

	u := newUser("oauth-update@example.com")
	if err := userRepo.Create(ctx, u); err != nil {
		t.Fatalf("Create user: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, "DELETE FROM users WHERE id=$1", u.ID) }) //nolint:errcheck

	token := "token-v1"
	acc := &domain.OAuthAccount{
		UserID: u.ID, Provider: "github", ProviderUID: "gh-002",
		Email: "oauth-update@example.com", AccessToken: &token,
	}
	if err := oauthRepo.Upsert(ctx, acc); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	updated := "token-v2"
	acc.AccessToken = &updated
	if err := oauthRepo.Upsert(ctx, acc); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	found, err := oauthRepo.FindByProvider(ctx, "github", "gh-002")
	if err != nil {
		t.Fatalf("FindByProvider: %v", err)
	}
	if found.AccessToken == nil || *found.AccessToken != "token-v2" {
		t.Errorf("expected updated access_token, got %v", found.AccessToken)
	}
}

func TestOAuthAccountRepo_FindByProvider_NotFound(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()

	oauthRepo := postgres.NewOAuthAccountRepo(pool)
	_, err := oauthRepo.FindByProvider(context.Background(), "google", "nonexistent-uid")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestOAuthAccountRepo_FindByUserID(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()
	ctx := context.Background()

	userRepo := postgres.NewUserRepo(pool)
	oauthRepo := postgres.NewOAuthAccountRepo(pool)

	u := newUser("oauth-list@example.com")
	if err := userRepo.Create(ctx, u); err != nil {
		t.Fatalf("Create user: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, "DELETE FROM users WHERE id=$1", u.ID) }) //nolint:errcheck

	for _, provider := range []string{"google", "github"} {
		acc := &domain.OAuthAccount{
			UserID: u.ID, Provider: provider,
			ProviderUID: provider + "-uid-list",
			Email:       "oauth-list@example.com",
		}
		if err := oauthRepo.Upsert(ctx, acc); err != nil {
			t.Fatalf("Upsert %s: %v", provider, err)
		}
	}

	accounts, err := oauthRepo.FindByUserID(ctx, u.ID)
	if err != nil {
		t.Fatalf("FindByUserID: %v", err)
	}
	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(accounts))
	}
}

func TestOAuthAccountRepo_FindByUserID_Empty(t *testing.T) {
	pool := integrationPool(t)
	defer pool.Close()

	oauthRepo := postgres.NewOAuthAccountRepo(pool)
	accounts, err := oauthRepo.FindByUserID(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("FindByUserID: %v", err)
	}
	if len(accounts) != 0 {
		t.Errorf("expected 0 accounts, got %d", len(accounts))
	}
}
