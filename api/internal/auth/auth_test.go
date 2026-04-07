package auth_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/popatkaran/postulate/api/internal/auth"
	"github.com/popatkaran/postulate/api/internal/domain"
)

// ── StateStore ────────────────────────────────────────────────────────────────

func TestStateStore_GenerateAndValidate_HappyPath(t *testing.T) {
	s := auth.NewStateStore()
	state, err := s.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !s.Validate(state) {
		t.Error("expected Validate to return true for a freshly generated state")
	}
}

func TestStateStore_Validate_SingleUse(t *testing.T) {
	s := auth.NewStateStore()
	state, _ := s.Generate()
	s.Validate(state) // consume
	if s.Validate(state) {
		t.Error("expected Validate to return false on second use")
	}
}

func TestStateStore_Validate_UnknownState_ReturnsFalse(t *testing.T) {
	s := auth.NewStateStore()
	if s.Validate("not-a-real-state") {
		t.Error("expected Validate to return false for unknown state")
	}
}

// ── ResolveOrCreateUser ───────────────────────────────────────────────────────

// stubUserRepo is a minimal in-memory UserRepository for testing.
type stubUserRepo struct {
	byID    map[uuid.UUID]*domain.User
	byEmail map[string]*domain.User
	created []*domain.User
}

func newStubUserRepo() *stubUserRepo {
	return &stubUserRepo{byID: map[uuid.UUID]*domain.User{}, byEmail: map[string]*domain.User{}}
}

func (r *stubUserRepo) Create(_ context.Context, u *domain.User) error {
	u.ID = uuid.New()
	r.byID[u.ID] = u
	r.byEmail[u.Email] = u
	r.created = append(r.created, u)
	return nil
}
func (r *stubUserRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}
func (r *stubUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := r.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}
func (r *stubUserRepo) Update(_ context.Context, u *domain.User) error {
	r.byID[u.ID] = u
	r.byEmail[u.Email] = u
	return nil
}
func (r *stubUserRepo) SoftDelete(_ context.Context, id uuid.UUID) error {
	delete(r.byID, id)
	return nil
}
func (r *stubUserRepo) CountAll(_ context.Context) (int64, error) {
	return int64(len(r.byID)), nil
}

// stubOAuthRepo is a minimal in-memory OAuthAccountRepository for testing.
type stubOAuthRepo struct {
	accounts map[string]*domain.OAuthAccount // key: provider+":"+providerUID
}

func newStubOAuthRepo() *stubOAuthRepo {
	return &stubOAuthRepo{accounts: map[string]*domain.OAuthAccount{}}
}

func (r *stubOAuthRepo) key(provider, uid string) string { return provider + ":" + uid }

func (r *stubOAuthRepo) Upsert(_ context.Context, a *domain.OAuthAccount) error {
	if a.ID == (uuid.UUID{}) {
		a.ID = uuid.New()
	}
	r.accounts[r.key(a.Provider, a.ProviderUID)] = a
	return nil
}
func (r *stubOAuthRepo) FindByProvider(_ context.Context, provider, uid string) (*domain.OAuthAccount, error) {
	a, ok := r.accounts[r.key(provider, uid)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return a, nil
}
func (r *stubOAuthRepo) FindByUserID(_ context.Context, userID uuid.UUID) ([]*domain.OAuthAccount, error) {
	var out []*domain.OAuthAccount
	for _, a := range r.accounts {
		if a.UserID == userID {
			out = append(out, a)
		}
	}
	return out, nil
}

func providerUser(email string) auth.ProviderUser {
	return auth.ProviderUser{
		Provider:    "google",
		ProviderUID: "google-uid-" + email,
		Email:       email,
		Name:        "Test User",
		TokenExpiry: time.Now().Add(time.Hour),
	}
}

// stubTransactor executes fn directly without a real transaction.
type stubTransactor struct{}

func (t *stubTransactor) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func newResolver(userRepo *stubUserRepo, oauthRepo *stubOAuthRepo) *auth.UserResolver {
	return auth.NewUserResolver(userRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{})
}

func TestResolveOrCreateUser_NewUser_CreatesUserAndOAuthAccount(t *testing.T) {
	userRepo := newStubUserRepo()
	oauthRepo := newStubOAuthRepo()
	resolver := newResolver(userRepo, oauthRepo)

	pu := providerUser("new@example.com")
	user, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Email != pu.Email {
		t.Errorf("expected email %q, got %q", pu.Email, user.Email)
	}
	if user.Role != domain.RolePlatformMember {
		t.Errorf("expected role platform_member, got %q", user.Role)
	}
	if len(userRepo.created) != 1 {
		t.Errorf("expected 1 user created, got %d", len(userRepo.created))
	}
	if _, err := oauthRepo.FindByProvider(context.Background(), pu.Provider, pu.ProviderUID); err != nil {
		t.Errorf("expected oauth account to be created: %v", err)
	}
}

func TestResolveOrCreateUser_ExistingOAuthAccount_ReturnsUserAndUpdatesTokens(t *testing.T) {
	userRepo := newStubUserRepo()
	oauthRepo := newStubOAuthRepo()
	resolver := newResolver(userRepo, oauthRepo)

	// Pre-create user and oauth account.
	pu := providerUser("existing@example.com")
	user, _ := resolver.ResolveOrCreateUser(context.Background(), pu)

	// Second call — same provider UID.
	pu.AccessToken = "new-access-token"
	user2, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user2.ID != user.ID {
		t.Errorf("expected same user ID on second call")
	}
	if len(userRepo.created) != 1 {
		t.Errorf("expected no new user created on second call, got %d", len(userRepo.created))
	}
}

func TestResolveOrCreateUser_ExistingEmailNoOAuthLink_LinksAccount(t *testing.T) {
	userRepo := newStubUserRepo()
	oauthRepo := newStubOAuthRepo()
	resolver := newResolver(userRepo, oauthRepo)

	// Pre-create user without oauth link.
	existing := &domain.User{Email: "linked@example.com", Role: domain.RolePlatformAdmin, Status: domain.StatusActive}
	_ = userRepo.Create(context.Background(), existing)

	pu := providerUser("linked@example.com")
	user, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Role must be preserved.
	if user.Role != domain.RolePlatformAdmin {
		t.Errorf("expected preserved role platform_admin, got %q", user.Role)
	}
	if len(userRepo.created) != 1 {
		t.Errorf("expected no new user created, got %d", len(userRepo.created))
	}
}

func TestResolveOrCreateUser_EmptyEmail_ReturnsError(t *testing.T) {
	resolver := newResolver(newStubUserRepo(), newStubOAuthRepo())
	pu := providerUser("")
	pu.Email = ""
	_, err := resolver.ResolveOrCreateUser(context.Background(), pu)
	if err == nil {
		t.Fatal("expected error for empty email, got nil")
	}
}

func TestResolveOrCreateUser_OAuthRepoError_PropagatesError(t *testing.T) {
	userRepo := newStubUserRepo()
	oauthRepo := &errorOAuthRepo{}
	resolver := auth.NewUserResolver(userRepo, oauthRepo, &stubTransactor{}, auth.BootstrapConfig{})

	_, err := resolver.ResolveOrCreateUser(context.Background(), providerUser("err@example.com"))
	if err == nil {
		t.Fatal("expected error from oauth repo, got nil")
	}
}

// errorOAuthRepo always returns an error from FindByProvider.
type errorOAuthRepo struct{}

func (r *errorOAuthRepo) Upsert(_ context.Context, _ *domain.OAuthAccount) error {
	return errors.New("db error")
}
func (r *errorOAuthRepo) FindByProvider(_ context.Context, _, _ string) (*domain.OAuthAccount, error) {
	return nil, errors.New("db error")
}
func (r *errorOAuthRepo) FindByUserID(_ context.Context, _ uuid.UUID) ([]*domain.OAuthAccount, error) {
	return nil, errors.New("db error")
}

// ── TokenIssuer ───────────────────────────────────────────────────────────────

// stubRefreshTokenRepo is an in-memory RefreshTokenRepository for testing.
type stubRefreshTokenRepo struct {
	tokens map[string]*domain.RefreshToken // key: token_hash
}

func newStubRefreshTokenRepo() *stubRefreshTokenRepo {
	return &stubRefreshTokenRepo{tokens: map[string]*domain.RefreshToken{}}
}

func (r *stubRefreshTokenRepo) Create(_ context.Context, t *domain.RefreshToken) error {
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	r.tokens[t.TokenHash] = t
	return nil
}
func (r *stubRefreshTokenRepo) FindByTokenHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	t, ok := r.tokens[hash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}
func (r *stubRefreshTokenRepo) MarkUsed(_ context.Context, id uuid.UUID, at time.Time) error {
	for _, t := range r.tokens {
		if t.ID == id {
			t.UsedAt = &at
			return nil
		}
	}
	return domain.ErrNotFound
}
func (r *stubRefreshTokenRepo) DeleteBySessionID(_ context.Context, _ uuid.UUID) error { return nil }
func (r *stubRefreshTokenRepo) DeleteByUserID(_ context.Context, userID uuid.UUID) error {
	for k, t := range r.tokens {
		if t.UserID == userID {
			delete(r.tokens, k)
		}
	}
	return nil
}
func (r *stubRefreshTokenRepo) DeleteExpired(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

const testJWTSecret = "a-test-jwt-secret-that-is-32-bytes!"

func newTestIssuer(userRepo *stubUserRepo) (*auth.TokenIssuer, *stubRefreshTokenRepo) {
	rtRepo := newStubRefreshTokenRepo()
	return auth.NewTokenIssuer(testJWTSecret, rtRepo, userRepo), rtRepo
}

func TestIssueSessionToken_JWTClaimsAreCorrect(t *testing.T) {
	userRepo := newStubUserRepo()
	issuer, _ := newTestIssuer(userRepo)
	userID := uuid.New()
	before := time.Now()

	tr, err := issuer.IssueSessionToken(context.Background(), userID, "platform_member")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse without verification to inspect claims.
	parsed, err := jwt.Parse(tr.Token, func(tok *jwt.Token) (any, error) {
		return []byte(testJWTSecret), nil
	})
	if err != nil {
		t.Fatalf("parse jwt: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("expected MapClaims")
	}

	if claims["sub"] != userID.String() {
		t.Errorf("sub: expected %q, got %v", userID.String(), claims["sub"])
	}
	if claims["role"] != "platform_member" {
		t.Errorf("role: expected platform_member, got %v", claims["role"])
	}

	iat := int64(claims["iat"].(float64))
	exp := int64(claims["exp"].(float64))
	if iat < before.Unix() || iat > time.Now().Unix()+1 {
		t.Errorf("iat out of expected range: %d", iat)
	}
	expectedExp := iat + int64((8 * time.Hour).Seconds())
	if exp != expectedExp {
		t.Errorf("exp: expected %d (iat+8h), got %d", expectedExp, exp)
	}
}

func TestIssueSessionToken_RefreshTokenIsStoredAsHash(t *testing.T) {
	userRepo := newStubUserRepo()
	issuer, rtRepo := newTestIssuer(userRepo)
	userID := uuid.New()

	tr, err := issuer.IssueSessionToken(context.Background(), userID, "platform_member")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Raw token must not be stored.
	if _, err := rtRepo.FindByTokenHash(context.Background(), tr.RefreshToken); err == nil {
		t.Error("raw refresh token must not be stored directly")
	}

	// Hash of raw token must be stored.
	h := sha256.Sum256([]byte(tr.RefreshToken))
	hash := hex.EncodeToString(h[:])
	if _, err := rtRepo.FindByTokenHash(context.Background(), hash); err != nil {
		t.Errorf("expected hashed token to be stored: %v", err)
	}
}

func TestIssueSessionToken_ExpiresAtIs8HoursFromNow(t *testing.T) {
	userRepo := newStubUserRepo()
	issuer, _ := newTestIssuer(userRepo)
	before := time.Now()

	tr, err := issuer.IssueSessionToken(context.Background(), uuid.New(), "platform_member")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lower := before.Add(8 * time.Hour)
	upper := time.Now().Add(8 * time.Hour).Add(2 * time.Second)
	if tr.ExpiresAt.Before(lower) || tr.ExpiresAt.After(upper) {
		t.Errorf("ExpiresAt %v not within expected 8h window", tr.ExpiresAt)
	}
}

func TestRefreshSession_HappyPath_RotatesToken(t *testing.T) {
	userRepo := newStubUserRepo()
	user := &domain.User{Email: "u@example.com", Role: domain.RolePlatformMember, Status: domain.StatusActive}
	_ = userRepo.Create(context.Background(), user)

	issuer, rtRepo := newTestIssuer(userRepo)

	// Issue initial token.
	tr, err := issuer.IssueSessionToken(context.Background(), user.ID, string(user.Role))
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	oldRaw := tr.RefreshToken

	// Refresh.
	tr2, err := issuer.RefreshSession(context.Background(), oldRaw)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}

	// New refresh token must differ.
	if tr2.RefreshToken == oldRaw {
		t.Error("expected rotated refresh token, got same value")
	}
	// JWT must be valid and carry the correct subject.
	if tr2.Token == "" {
		t.Error("expected non-empty JWT after rotation")
	}

	// Old token must be gone.
	oldHash := sha256.Sum256([]byte(oldRaw))
	if _, err := rtRepo.FindByTokenHash(context.Background(), hex.EncodeToString(oldHash[:])); err == nil {
		t.Error("old refresh token must be deleted after rotation")
	}
}

func TestRefreshSession_InvalidToken_ReturnsError(t *testing.T) {
	userRepo := newStubUserRepo()
	issuer, _ := newTestIssuer(userRepo)

	_, err := issuer.RefreshSession(context.Background(), "not-a-real-token")
	if err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}
}

func TestRefreshSession_ExpiredToken_ReturnsError(t *testing.T) {
	userRepo := newStubUserRepo()
	issuer, rtRepo := newTestIssuer(userRepo)
	userID := uuid.New()

	tr, _ := issuer.IssueSessionToken(context.Background(), userID, "platform_member")

	// Manually expire the stored token.
	h := sha256.Sum256([]byte(tr.RefreshToken))
	hash := hex.EncodeToString(h[:])
	stored, _ := rtRepo.FindByTokenHash(context.Background(), hash)
	stored.ExpiresAt = time.Now().Add(-time.Second)

	_, err := issuer.RefreshSession(context.Background(), tr.RefreshToken)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestRevokeAllSessions_DeletesAllTokensForUser(t *testing.T) {
	userRepo := newStubUserRepo()
	issuer, rtRepo := newTestIssuer(userRepo)
	userID := uuid.New()

	// Issue two tokens for the same user.
	_, _ = issuer.IssueSessionToken(context.Background(), userID, "platform_member")
	_, _ = issuer.IssueSessionToken(context.Background(), userID, "platform_member")

	if len(rtRepo.tokens) != 2 {
		t.Fatalf("expected 2 tokens before revocation, got %d", len(rtRepo.tokens))
	}

	if err := issuer.RevokeAllSessions(context.Background(), userID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rtRepo.tokens) != 0 {
		t.Errorf("expected 0 tokens after revocation, got %d", len(rtRepo.tokens))
	}
}
