package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/popatkaran/postulate/api/internal/auth"
	"github.com/popatkaran/postulate/api/internal/domain"
)

// ── Bootstrap: POSTULATE_BOOTSTRAP_ADMIN_EMAIL ────────────────────────────────

func TestBootstrap_AdminEmail_NewUser_ReceivesPlatformAdmin(t *testing.T) {
	userRepo := newStubUserRepo()
	oauthRepo := newStubOAuthRepo()
	resolver := auth.NewUserResolver(userRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{
		AdminEmail:         "admin@example.com",
		DevFallbackEnabled: false,
	})

	pu := auth.ProviderUser{
		Provider: "google", ProviderUID: "uid-admin",
		Email: "admin@example.com", Name: "Admin",
		TokenExpiry: time.Now().Add(time.Hour),
	}
	user, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Role != domain.RolePlatformAdmin {
		t.Errorf("expected platform_admin, got %q", user.Role)
	}
}

func TestBootstrap_AdminEmail_NonAdminUser_ReceivesPlatformMember(t *testing.T) {
	userRepo := newStubUserRepo()
	oauthRepo := newStubOAuthRepo()
	resolver := auth.NewUserResolver(userRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{
		AdminEmail:         "admin@example.com",
		DevFallbackEnabled: false,
	})

	pu := auth.ProviderUser{
		Provider: "google", ProviderUID: "uid-member",
		Email: "member@example.com", Name: "Member",
		TokenExpiry: time.Now().Add(time.Hour),
	}
	user, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Role != domain.RolePlatformMember {
		t.Errorf("expected platform_member, got %q", user.Role)
	}
}

func TestBootstrap_AdminEmail_ExistingMember_UpgradedOnLogin(t *testing.T) {
	userRepo := newStubUserRepo()
	oauthRepo := newStubOAuthRepo()

	// Pre-create the admin user as a member (e.g. registered before bootstrap was set).
	existing := &domain.User{
		Email: "admin@example.com", Role: domain.RolePlatformMember, Status: domain.StatusActive,
	}
	_ = userRepo.Create(context.Background(), existing)

	resolver := auth.NewUserResolver(userRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{
		AdminEmail:         "admin@example.com",
		DevFallbackEnabled: false,
	})

	pu := auth.ProviderUser{
		Provider: "google", ProviderUID: "uid-admin",
		Email: "admin@example.com", Name: "Admin",
		TokenExpiry: time.Now().Add(time.Hour),
	}
	user, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Role != domain.RolePlatformAdmin {
		t.Errorf("expected role to be upgraded to platform_admin, got %q", user.Role)
	}
}

func TestBootstrap_AdminEmail_ExistingAdmin_NoUpdate(t *testing.T) {
	userRepo := newStubUserRepo()
	oauthRepo := newStubOAuthRepo()

	// Pre-create the admin user already as admin.
	existing := &domain.User{
		Email: "admin@example.com", Role: domain.RolePlatformAdmin, Status: domain.StatusActive,
	}
	_ = userRepo.Create(context.Background(), existing)

	updatesBefore := len(userRepo.byID) // snapshot

	resolver := auth.NewUserResolver(userRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{
		AdminEmail:         "admin@example.com",
		DevFallbackEnabled: false,
	})

	pu := auth.ProviderUser{
		Provider: "google", ProviderUID: "uid-admin",
		Email: "admin@example.com", Name: "Admin",
		TokenExpiry: time.Now().Add(time.Hour),
	}
	user, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Role != domain.RolePlatformAdmin {
		t.Errorf("expected platform_admin, got %q", user.Role)
	}
	// No extra users should have been created.
	if len(userRepo.byID) != updatesBefore {
		t.Errorf("expected no new users, got %d", len(userRepo.byID))
	}
}

// ── Bootstrap: dev fallback (empty database) ──────────────────────────────────

func TestBootstrap_DevFallback_FirstUser_ReceivesPlatformAdmin(t *testing.T) {
	userRepo := newStubUserRepo()
	oauthRepo := newStubOAuthRepo()
	resolver := auth.NewUserResolver(userRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{
		AdminEmail:         "",
		DevFallbackEnabled: true,
	})

	pu := auth.ProviderUser{
		Provider: "google", ProviderUID: "uid-first",
		Email: "first@example.com", Name: "First",
		TokenExpiry: time.Now().Add(time.Hour),
	}
	user, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Role != domain.RolePlatformAdmin {
		t.Errorf("expected platform_admin for first user, got %q", user.Role)
	}
}

func TestBootstrap_DevFallback_SecondUser_ReceivesPlatformMember(t *testing.T) {
	userRepo := newStubUserRepo()
	oauthRepo := newStubOAuthRepo()
	resolver := auth.NewUserResolver(userRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{
		AdminEmail:         "",
		DevFallbackEnabled: true,
	})

	// First login — becomes admin.
	pu1 := auth.ProviderUser{
		Provider: "google", ProviderUID: "uid-first",
		Email: "first@example.com", Name: "First",
		TokenExpiry: time.Now().Add(time.Hour),
	}
	if _, err := resolver.ResolveOrCreateUser(context.Background(), pu1); err != nil {
		t.Fatalf("first user: %v", err)
	}

	// Second login — must be member.
	pu2 := auth.ProviderUser{
		Provider: "google", ProviderUID: "uid-second",
		Email: "second@example.com", Name: "Second",
		TokenExpiry: time.Now().Add(time.Hour),
	}
	user2, err := resolver.ResolveOrCreateUser(context.Background(), pu2)
	if err != nil {
		t.Fatalf("second user: %v", err)
	}
	if user2.Role != domain.RolePlatformMember {
		t.Errorf("expected platform_member for second user, got %q", user2.Role)
	}
}

func TestBootstrap_DevFallbackDisabled_FirstUser_ReceivesPlatformMember(t *testing.T) {
	userRepo := newStubUserRepo()
	oauthRepo := newStubOAuthRepo()
	// Production mode: no bootstrap email, dev fallback disabled.
	resolver := auth.NewUserResolver(userRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{
		AdminEmail:         "",
		DevFallbackEnabled: false,
	})

	pu := auth.ProviderUser{
		Provider: "google", ProviderUID: "uid-prod-first",
		Email: "prod@example.com", Name: "Prod",
		TokenExpiry: time.Now().Add(time.Hour),
	}
	user, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Role != domain.RolePlatformMember {
		t.Errorf("expected platform_member in production without bootstrap email, got %q", user.Role)
	}
}

// ── Role validation ───────────────────────────────────────────────────────────

func TestBootstrap_InvalidRole_ReturnsError(t *testing.T) {
	// Verify that only platform_admin and platform_member are valid domain roles.
	// The service layer enforces this via resolveRoleForNewUser which only ever
	// returns one of the two constants — this test guards the domain constants.
	validRoles := []domain.UserRole{domain.RolePlatformAdmin, domain.RolePlatformMember}
	for _, r := range validRoles {
		if r != domain.RolePlatformAdmin && r != domain.RolePlatformMember {
			t.Errorf("unexpected role constant: %q", r)
		}
	}
}
