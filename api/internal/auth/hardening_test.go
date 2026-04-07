package auth_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/markbates/goth"
	"github.com/popatkaran/postulate/api/internal/auth"
	"github.com/popatkaran/postulate/api/internal/domain"
)

// ── Identity ──────────────────────────────────────────────────────────────────

func TestContextWithIdentity_RoundTrip(t *testing.T) {
	id := auth.Identity{UserID: "uid-123", Role: "platform_admin"}
	ctx := auth.ContextWithIdentity(context.Background(), id)
	got, ok := auth.IdentityFromContext(ctx)
	if !ok {
		t.Fatal("expected identity in context")
	}
	if got.UserID != id.UserID || got.Role != id.Role {
		t.Errorf("identity mismatch: got %+v", got)
	}
}

func TestIdentityFromContext_Empty_ReturnsFalse(t *testing.T) {
	_, ok := auth.IdentityFromContext(context.Background())
	if ok {
		t.Error("expected false for empty context")
	}
}

// ── GothUserToProviderUser ────────────────────────────────────────────────────

func TestGothUserToProviderUser_MapsAllFields(t *testing.T) {
	expiry := time.Now().Add(time.Hour)
	gu := goth.User{
		Provider:     "google",
		UserID:       "gid-123",
		Email:        "user@example.com",
		Name:         "Test User",
		AccessToken:  "at",
		RefreshToken: "rt",
		ExpiresAt:    expiry,
	}
	pu := auth.GothUserToProviderUser(gu)
	if pu.Provider != "google" || pu.ProviderUID != "gid-123" ||
		pu.Email != "user@example.com" || pu.Name != "Test User" ||
		pu.AccessToken != "at" || pu.RefreshToken != "rt" ||
		!pu.TokenExpiry.Equal(expiry) {
		t.Errorf("field mapping incorrect: %+v", pu)
	}
}

// ── RegisterProviders ─────────────────────────────────────────────────────────

func TestRegisterProviders_DoesNotPanic(t *testing.T) {
	// RegisterProviders must not panic with valid (even fake) credentials.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RegisterProviders panicked: %v", r)
		}
	}()
	auth.RegisterProviders("gid", "gsecret", "ghid", "ghsecret", "http://localhost:8080")
}

// ── RevokeAllSessions error path ──────────────────────────────────────────────

func TestRevokeAllSessions_RepoError_ReturnsError(t *testing.T) {
	issuer := auth.NewTokenIssuer("a-test-jwt-secret-that-is-32-bytes!", &errDeleteRepo{}, newStubUserRepo())
	err := issuer.RevokeAllSessions(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}

// errDeleteRepo returns an error from DeleteByUserID.
type errDeleteRepo struct{}

func (r *errDeleteRepo) Create(_ context.Context, _ *domain.RefreshToken) error { return nil }
func (r *errDeleteRepo) FindByTokenHash(_ context.Context, _ string) (*domain.RefreshToken, error) {
	return nil, domain.ErrNotFound
}
func (r *errDeleteRepo) MarkUsed(_ context.Context, _ uuid.UUID, _ time.Time) error { return nil }
func (r *errDeleteRepo) DeleteBySessionID(_ context.Context, _ uuid.UUID) error     { return nil }
func (r *errDeleteRepo) DeleteByUserID(_ context.Context, _ uuid.UUID) error {
	return errors.New("db error")
}
func (r *errDeleteRepo) DeleteExpired(_ context.Context, _ time.Time) (int64, error) { return 0, nil }

// ── Concurrent first-login: exactly one platform_admin ───────────────────────

func TestConcurrentFirstLogin_ExactlyOnePlatformAdmin(t *testing.T) {
	// Two goroutines call ResolveOrCreateUser simultaneously against an empty
	// in-memory user store. The transactor stub executes fn directly (no real DB),
	// but the stubUserRepo.CountAll is based on len(byID) which is updated by
	// Create. Because both goroutines share the same stubUserRepo and the stub
	// is not thread-safe, we serialise via a mutex in a custom transactor that
	// holds the lock across the count+create pair — simulating the serialisable
	// transaction guarantee.
	userRepo := newStubUserRepo()
	oauthRepo := newStubOAuthRepo()

	tx := &serialisedTransactor{mu: &sync.Mutex{}}
	resolver := auth.NewUserResolver(userRepo, oauthRepo, tx, auth.BootstrapConfig{
		DevFallbackEnabled: true,
	})

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		i := i
		go func() {
			defer wg.Done()
			pu := auth.ProviderUser{
				Provider:    "google",
				ProviderUID: "uid-concurrent-" + string(rune('A'+i)),
				Email:       "concurrent" + string(rune('A'+i)) + "@example.com",
				Name:        "User",
				TokenExpiry: time.Now().Add(time.Hour),
			}
			_, _ = resolver.ResolveOrCreateUser(context.Background(), pu)
		}()
	}
	wg.Wait()

	adminCount := 0
	for _, u := range userRepo.byID {
		if u.Role == domain.RolePlatformAdmin {
			adminCount++
		}
	}
	if adminCount != 1 {
		t.Errorf("expected exactly 1 platform_admin, got %d", adminCount)
	}
}

// serialisedTransactor holds a mutex across the entire fn call, simulating
// a serialisable transaction that prevents concurrent zero-row races.
type serialisedTransactor struct{ mu *sync.Mutex }

func (t *serialisedTransactor) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return fn(ctx)
}

// ── OAuth state single-use ────────────────────────────────────────────────────

func TestStateStore_SingleUse_SecondValidateReturnsFalse(t *testing.T) {
	s := auth.NewStateStore()
	state, _ := s.Generate()
	if !s.Validate(state) {
		t.Fatal("first Validate should return true")
	}
	if s.Validate(state) {
		t.Error("second Validate must return false (single-use)")
	}
}

// ── Used refresh token rejected ───────────────────────────────────────────────

func TestRefreshSession_UsedToken_ReturnsError(t *testing.T) {
	userRepo := newStubUserRepo()
	user := &domain.User{Email: "u@example.com", Role: domain.RolePlatformMember, Status: domain.StatusActive}
	_ = userRepo.Create(context.Background(), user)

	issuer, _ := newTestIssuer(userRepo)
	tr, _ := issuer.IssueSessionToken(context.Background(), user.ID, string(user.Role))

	// First refresh — succeeds and rotates.
	_, err := issuer.RefreshSession(context.Background(), tr.RefreshToken)
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}

	// Second use of the original token — must be rejected.
	_, err = issuer.RefreshSession(context.Background(), tr.RefreshToken)
	if err == nil {
		t.Fatal("expected error for used refresh token, got nil")
	}
}

// ── Production bootstrap guard ────────────────────────────────────────────────

func TestBootstrap_Production_NoEmail_FirstUserGetsMember(t *testing.T) {
	userRepo := newStubUserRepo()
	oauthRepo := newStubOAuthRepo()
	resolver := auth.NewUserResolver(userRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{
		AdminEmail:         "",
		DevFallbackEnabled: false, // production: fallback disabled
	})

	pu := auth.ProviderUser{
		Provider: "google", ProviderUID: "uid-prod",
		Email: "prod@example.com", Name: "Prod",
		TokenExpiry: time.Now().Add(time.Hour),
	}
	user, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Role != domain.RolePlatformMember {
		t.Errorf("production without bootstrap email: expected platform_member, got %q", user.Role)
	}
}

// ── ResolveOrCreateUser transactor error ──────────────────────────────────────

func TestResolveOrCreateUser_TransactorError_ReturnsError(t *testing.T) {
	userRepo := newStubUserRepo()
	oauthRepo := newStubOAuthRepo()
	resolver := auth.NewUserResolver(userRepo, oauthRepo, &errTransactor{}, auth.BootstrapConfig{})

	pu := auth.ProviderUser{
		Provider: "google", ProviderUID: "uid-tx-err",
		Email: "txerr@example.com", Name: "Err",
		TokenExpiry: time.Now().Add(time.Hour),
	}
	_, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err == nil {
		t.Fatal("expected error from transactor, got nil")
	}
}

// errTransactor always returns an error.
type errTransactor struct{}

func (t *errTransactor) WithTransaction(_ context.Context, _ func(context.Context) error) error {
	return errors.New("tx error")
}

// ── resolveRoleForNewUser count error ─────────────────────────────────────────

func TestResolveOrCreateUser_CountError_ReturnsError(t *testing.T) {
	userRepo := &errCountRepo{stubUserRepo: newStubUserRepo()}
	oauthRepo := newStubOAuthRepo()
	resolver := auth.NewUserResolver(userRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{
		DevFallbackEnabled: true,
	})

	pu := auth.ProviderUser{
		Provider: "google", ProviderUID: "uid-count-err",
		Email: "counterr@example.com", Name: "Err",
		TokenExpiry: time.Now().Add(time.Hour),
	}
	_, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err == nil {
		t.Fatal("expected error from CountAll, got nil")
	}
}

// errCountRepo wraps stubUserRepo but returns an error from CountAll.
type errCountRepo struct{ *stubUserRepo }

func (r *errCountRepo) CountAll(_ context.Context) (int64, error) {
	return 0, errors.New("count error")
}

// ── StateStore.Generate rand error (not injectable — skip; covered by happy path) ──

// ── RefreshSession: already-used token path ───────────────────────────────────

func TestRefreshSession_AlreadyUsedToken_ReturnsError(t *testing.T) {
	userRepo := newStubUserRepo()
	user := &domain.User{Email: "u@example.com", Role: domain.RolePlatformMember, Status: domain.StatusActive}
	_ = userRepo.Create(context.Background(), user)
	issuer, rtRepo := newTestIssuer(userRepo)

	tr, _ := issuer.IssueSessionToken(context.Background(), user.ID, string(user.Role))

	// Manually mark the token as used.
	h := sha256.Sum256([]byte(tr.RefreshToken))
	hash := hex.EncodeToString(h[:])
	stored, _ := rtRepo.FindByTokenHash(context.Background(), hash)
	usedAt := time.Now()
	stored.UsedAt = &usedAt

	_, err := issuer.RefreshSession(context.Background(), tr.RefreshToken)
	if err == nil {
		t.Fatal("expected error for already-used token, got nil")
	}
}

// ── ResolveOrCreateUser: upsert error on existing oauth account ───────────────

func TestResolveOrCreateUser_ExistingOAuthAccount_UpsertError_ReturnsError(t *testing.T) {
	userRepo := newStubUserRepo()
	oauthRepo := newStubOAuthRepo()
	resolver := auth.NewUserResolver(userRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{})

	// First call creates the user and oauth account.
	pu := providerUser("upsert-err@example.com")
	_, _ = resolver.ResolveOrCreateUser(context.Background(), pu)

	// Now replace the oauth repo with one that errors on Upsert.
	errOAuth := &upsertErrOAuthRepo{inner: oauthRepo}
	resolver2 := auth.NewUserResolver(userRepo, errOAuth, &stubTransactor{}, auth.BootstrapConfig{})

	_, err := resolver2.ResolveOrCreateUser(context.Background(), pu)
	if err == nil {
		t.Fatal("expected error from Upsert on existing account, got nil")
	}
}

// upsertErrOAuthRepo delegates FindByProvider to inner but errors on Upsert.
type upsertErrOAuthRepo struct{ inner *stubOAuthRepo }

func (r *upsertErrOAuthRepo) Upsert(_ context.Context, _ *domain.OAuthAccount) error {
	return errors.New("upsert error")
}
func (r *upsertErrOAuthRepo) FindByProvider(ctx context.Context, provider, uid string) (*domain.OAuthAccount, error) {
	return r.inner.FindByProvider(ctx, provider, uid)
}
func (r *upsertErrOAuthRepo) FindByUserID(ctx context.Context, id uuid.UUID) ([]*domain.OAuthAccount, error) {
	return r.inner.FindByUserID(ctx, id)
}

// ── ResolveOrCreateUser: applyBootstrapUpgrade update error ──────────────────

func TestResolveOrCreateUser_BootstrapUpgrade_UpdateError_ReturnsError(t *testing.T) {
	userRepo := &errUpdateUserRepo{stubUserRepo: newStubUserRepo()}
	oauthRepo := newStubOAuthRepo()

	// Pre-create the admin user as a member.
	existing := &domain.User{
		Email: "admin@example.com", Role: domain.RolePlatformMember, Status: domain.StatusActive,
	}
	_ = userRepo.Create(context.Background(), existing)

	resolver := auth.NewUserResolver(userRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{
		AdminEmail: "admin@example.com",
	})

	pu := auth.ProviderUser{
		Provider: "google", ProviderUID: "uid-admin-upd",
		Email: "admin@example.com", Name: "Admin",
		TokenExpiry: time.Now().Add(time.Hour),
	}
	_, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err == nil {
		t.Fatal("expected error from Update during bootstrap upgrade, got nil")
	}
}

// errUpdateUserRepo wraps stubUserRepo but errors on Update.
type errUpdateUserRepo struct{ *stubUserRepo }

func (r *errUpdateUserRepo) Update(_ context.Context, _ *domain.User) error {
	return errors.New("update error")
}

// ── ResolveOrCreateUser: existing email, upsert error ────────────────────────

func TestResolveOrCreateUser_ExistingEmail_UpsertError_ReturnsError(t *testing.T) {
	userRepo := newStubUserRepo()
	existing := &domain.User{Email: "existing@example.com", Role: domain.RolePlatformMember, Status: domain.StatusActive}
	_ = userRepo.Create(context.Background(), existing)

	errOAuth := &upsertErrOAuthRepo{inner: newStubOAuthRepo()}
	resolver := auth.NewUserResolver(userRepo, errOAuth, &stubTransactor{}, auth.BootstrapConfig{})

	pu := providerUser("existing@example.com")
	_, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err == nil {
		t.Fatal("expected error from Upsert for existing email, got nil")
	}
}

// ── IssueSessionToken: store error ───────────────────────────────────────────

func TestIssueSessionToken_StoreError_ReturnsError(t *testing.T) {
	issuer := auth.NewTokenIssuer("a-test-jwt-secret-that-is-32-bytes!", &errCreateRepo{}, newStubUserRepo())
	_, err := issuer.IssueSessionToken(context.Background(), uuid.New(), "platform_member")
	if err == nil {
		t.Fatal("expected error when refresh token store fails, got nil")
	}
}

// errCreateRepo errors on Create.
type errCreateRepo struct{}

func (r *errCreateRepo) Create(_ context.Context, _ *domain.RefreshToken) error {
	return errors.New("store error")
}
func (r *errCreateRepo) FindByTokenHash(_ context.Context, _ string) (*domain.RefreshToken, error) {
	return nil, domain.ErrNotFound
}
func (r *errCreateRepo) MarkUsed(_ context.Context, _ uuid.UUID, _ time.Time) error { return nil }
func (r *errCreateRepo) DeleteBySessionID(_ context.Context, _ uuid.UUID) error     { return nil }
func (r *errCreateRepo) DeleteByUserID(_ context.Context, _ uuid.UUID) error        { return nil }
func (r *errCreateRepo) DeleteExpired(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

// ── RefreshSession: DeleteByUserID error ─────────────────────────────────────

func TestRefreshSession_DeleteError_ReturnsError(t *testing.T) {
	userRepo := newStubUserRepo()
	user := &domain.User{Email: "u@example.com", Role: domain.RolePlatformMember, Status: domain.StatusActive}
	_ = userRepo.Create(context.Background(), user)

	rtRepo := newStubRefreshTokenRepo()
	issuer := auth.NewTokenIssuer("a-test-jwt-secret-that-is-32-bytes!", rtRepo, userRepo)
	tr, _ := issuer.IssueSessionToken(context.Background(), user.ID, string(user.Role))

	// Replace repo with one that errors on DeleteByUserID.
	errRepo := &errDeleteOnRotateRepo{inner: rtRepo}
	issuer2 := auth.NewTokenIssuer("a-test-jwt-secret-that-is-32-bytes!", errRepo, userRepo)

	_, err := issuer2.RefreshSession(context.Background(), tr.RefreshToken)
	if err == nil {
		t.Fatal("expected error when DeleteByUserID fails during rotation, got nil")
	}
}

// errDeleteOnRotateRepo delegates all methods to inner but errors on DeleteByUserID.
type errDeleteOnRotateRepo struct{ inner *stubRefreshTokenRepo }

func (r *errDeleteOnRotateRepo) Create(ctx context.Context, t *domain.RefreshToken) error {
	return r.inner.Create(ctx, t)
}
func (r *errDeleteOnRotateRepo) FindByTokenHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	return r.inner.FindByTokenHash(ctx, hash)
}
func (r *errDeleteOnRotateRepo) MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.inner.MarkUsed(ctx, id, at)
}
func (r *errDeleteOnRotateRepo) DeleteBySessionID(ctx context.Context, id uuid.UUID) error {
	return r.inner.DeleteBySessionID(ctx, id)
}
func (r *errDeleteOnRotateRepo) DeleteByUserID(_ context.Context, _ uuid.UUID) error {
	return errors.New("delete error")
}
func (r *errDeleteOnRotateRepo) DeleteExpired(ctx context.Context, t time.Time) (int64, error) {
	return r.inner.DeleteExpired(ctx, t)
}

// ── ResolveOrCreateUser: FindByID error on existing oauth account ─────────────

func TestResolveOrCreateUser_ExistingOAuthAccount_FindByIDError_ReturnsError(t *testing.T) {
	userRepo := newStubUserRepo()
	oauthRepo := newStubOAuthRepo()
	resolver := auth.NewUserResolver(userRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{})

	// Create user and oauth account.
	pu := providerUser("findbyid-err@example.com")
	user, _ := resolver.ResolveOrCreateUser(context.Background(), pu)

	// Now replace userRepo with one that errors on FindByID.
	errUserRepo := &errFindByIDRepo{stubUserRepo: userRepo}
	resolver2 := auth.NewUserResolver(errUserRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{})

	// Second call — oauth account exists, FindByID will error.
	_, err := resolver2.ResolveOrCreateUser(context.Background(), pu)
	if err == nil {
		t.Fatalf("expected error from FindByID, got nil (user: %v)", user.ID)
	}
}

// errFindByIDRepo wraps stubUserRepo but errors on FindByID.
type errFindByIDRepo struct{ *stubUserRepo }

func (r *errFindByIDRepo) FindByID(_ context.Context, _ uuid.UUID) (*domain.User, error) {
	return nil, errors.New("findbyid error")
}

// ── ResolveOrCreateUser: Create error inside transaction ─────────────────────

func TestResolveOrCreateUser_CreateError_ReturnsError(t *testing.T) {
	errUserRepo := &errCreateUserRepo{stubUserRepo: newStubUserRepo()}
	oauthRepo := newStubOAuthRepo()
	resolver := auth.NewUserResolver(errUserRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{})

	pu := providerUser("create-err@example.com")
	_, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err == nil {
		t.Fatal("expected error from Create, got nil")
	}
}

// errCreateUserRepo wraps stubUserRepo but errors on Create.
type errCreateUserRepo struct{ *stubUserRepo }

func (r *errCreateUserRepo) Create(_ context.Context, _ *domain.User) error {
	return errors.New("create error")
}

// ── ResolveOrCreateUser: Upsert error inside transaction ─────────────────────

func TestResolveOrCreateUser_NewUser_UpsertError_ReturnsError(t *testing.T) {
	userRepo := newStubUserRepo()
	errOAuth := &upsertErrOAuthRepo{inner: newStubOAuthRepo()}
	resolver := auth.NewUserResolver(userRepo, errOAuth, &stubTransactor{}, auth.BootstrapConfig{})

	pu := providerUser("new-upsert-err@example.com")
	_, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err == nil {
		t.Fatal("expected error from Upsert inside transaction, got nil")
	}
}

// ── RefreshSession: user not found ───────────────────────────────────────────

func TestRefreshSession_UserNotFound_ReturnsError(t *testing.T) {
	userRepo := newStubUserRepo()
	rtRepo := newStubRefreshTokenRepo()
	issuer := auth.NewTokenIssuer("a-test-jwt-secret-that-is-32-bytes!", rtRepo, userRepo)

	// Issue a token for a user ID that doesn't exist in userRepo.
	orphanID := uuid.New()
	tr, _ := issuer.IssueSessionToken(context.Background(), orphanID, "platform_member")

	_, err := issuer.RefreshSession(context.Background(), tr.RefreshToken)
	if err == nil {
		t.Fatal("expected error when user not found during refresh, got nil")
	}
}
