package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/popatkaran/postulate/api/internal/auth"
	"github.com/popatkaran/postulate/api/internal/domain"
	"github.com/popatkaran/postulate/api/internal/handler"
	"log/slog"
	"os"
)

// ── in-memory stubs (minimal, only what TokenIssuer needs) ───────────────────

type memRefreshRepo struct {
	tokens map[string]*domain.RefreshToken
}

func newMemRefreshRepo() *memRefreshRepo {
	return &memRefreshRepo{tokens: map[string]*domain.RefreshToken{}}
}

func (r *memRefreshRepo) Create(_ context.Context, t *domain.RefreshToken) error {
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	r.tokens[t.TokenHash] = t
	return nil
}
func (r *memRefreshRepo) FindByTokenHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	t, ok := r.tokens[hash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}
func (r *memRefreshRepo) MarkUsed(_ context.Context, id uuid.UUID, at time.Time) error {
	for _, t := range r.tokens {
		if t.ID == id {
			t.UsedAt = &at
		}
	}
	return nil
}
func (r *memRefreshRepo) DeleteBySessionID(_ context.Context, _ uuid.UUID) error { return nil }
func (r *memRefreshRepo) DeleteByUserID(_ context.Context, userID uuid.UUID) error {
	for k, t := range r.tokens {
		if t.UserID == userID {
			delete(r.tokens, k)
		}
	}
	return nil
}
func (r *memRefreshRepo) DeleteExpired(_ context.Context, _ time.Time) (int64, error) { return 0, nil }

type memUserRepo struct {
	users map[uuid.UUID]*domain.User
}

func newMemUserRepo() *memUserRepo { return &memUserRepo{users: map[uuid.UUID]*domain.User{}} }

func (r *memUserRepo) Create(_ context.Context, u *domain.User) error {
	u.ID = uuid.New()
	r.users[u.ID] = u
	return nil
}
func (r *memUserRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}
func (r *memUserRepo) FindByEmail(_ context.Context, _ string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (r *memUserRepo) Update(_ context.Context, u *domain.User) error {
	r.users[u.ID] = u
	return nil
}
func (r *memUserRepo) SoftDelete(_ context.Context, id uuid.UUID) error {
	delete(r.users, id)
	return nil
}
func (r *memUserRepo) CountAll(_ context.Context) (int64, error) {
	return int64(len(r.users)), nil
}

const tokenHandlerJWTSecret = "a-test-jwt-secret-that-is-32-bytes!"

func newTokenHandlerWithUser() (*handler.TokenHandler, *domain.User, string) {
	rtRepo := newMemRefreshRepo()
	userRepo := newMemUserRepo()
	user := &domain.User{Role: domain.RolePlatformMember, Status: domain.StatusActive}
	_ = userRepo.Create(context.Background(), user)

	issuer := auth.NewTokenIssuer(tokenHandlerJWTSecret, rtRepo, userRepo)
	tr, _ := issuer.IssueSessionToken(context.Background(), user.ID, string(user.Role))

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return handler.NewTokenHandler(issuer, logger), user, tr.RefreshToken
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestTokenRefresh_HappyPath_Returns200WithNewTokens(t *testing.T) {
	h, _, rawToken := newTokenHandlerWithUser()

	body, _ := json.Marshal(map[string]string{"refresh_token": rawToken})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/token/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for _, key := range []string{"token", "refresh_token", "expires_at", "role"} {
		if resp[key] == "" || resp[key] == nil {
			t.Errorf("expected non-empty %q in response", key)
		}
	}
	// New refresh token must differ from the original.
	if resp["refresh_token"] == rawToken {
		t.Error("expected rotated refresh token, got same value")
	}
}

func TestTokenRefresh_InvalidToken_Returns401(t *testing.T) {
	h, _, _ := newTokenHandlerWithUser()

	body, _ := json.Marshal(map[string]string{"refresh_token": "not-a-real-token"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/token/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestTokenRefresh_MissingBody_Returns400(t *testing.T) {
	h, _, _ := newTokenHandlerWithUser()

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/token/refresh", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestTokenRefresh_MalformedJSON_Returns400(t *testing.T) {
	h, _, _ := newTokenHandlerWithUser()

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/token/refresh", bytes.NewReader([]byte(`not-json`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// ── Revoke (DELETE /v1/auth/token) ────────────────────────────────────────────

func newRevokeRequest(userID string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, "/v1/auth/token", nil)
	if userID != "" {
		ctx := auth.ContextWithIdentity(req.Context(), auth.Identity{UserID: userID, Role: "platform_member"})
		req = req.WithContext(ctx)
	}
	return req
}

func TestTokenRevoke_HappyPath_Returns204(t *testing.T) {
	h, user, _ := newTokenHandlerWithUser()
	rec := httptest.NewRecorder()

	h.Revoke(rec, newRevokeRequest(user.ID.String()))

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTokenRevoke_NoTokens_Returns204_Idempotent(t *testing.T) {
	// User with no active refresh tokens — should still return 204.
	h, _, _ := newTokenHandlerWithUser()
	rec := httptest.NewRecorder()

	h.Revoke(rec, newRevokeRequest(uuid.New().String()))

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 (idempotent), got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTokenRevoke_NoIdentityInContext_Returns401(t *testing.T) {
	h, _, _ := newTokenHandlerWithUser()
	req := httptest.NewRequest(http.MethodDelete, "/v1/auth/token", nil)
	rec := httptest.NewRecorder()

	h.Revoke(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestTokenRevoke_InvalidUserIDInClaims_Returns401(t *testing.T) {
	h, _, _ := newTokenHandlerWithUser()
	rec := httptest.NewRecorder()

	h.Revoke(rec, newRevokeRequest("not-a-uuid"))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestTokenRevoke_DBError_Returns500(t *testing.T) {
	rtRepo := &errRefreshRepo{}
	userRepo := newMemUserRepo()
	issuer := auth.NewTokenIssuer(tokenHandlerJWTSecret, rtRepo, userRepo)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	h := handler.NewTokenHandler(issuer, logger)

	rec := httptest.NewRecorder()
	h.Revoke(rec, newRevokeRequest(uuid.New().String()))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// errRefreshRepo always returns an error from DeleteByUserID.
type errRefreshRepo struct{}

func (r *errRefreshRepo) Create(_ context.Context, _ *domain.RefreshToken) error {
	return nil
}
func (r *errRefreshRepo) FindByTokenHash(_ context.Context, _ string) (*domain.RefreshToken, error) {
	return nil, domain.ErrNotFound
}
func (r *errRefreshRepo) MarkUsed(_ context.Context, _ uuid.UUID, _ time.Time) error { return nil }
func (r *errRefreshRepo) DeleteBySessionID(_ context.Context, _ uuid.UUID) error     { return nil }
func (r *errRefreshRepo) DeleteByUserID(_ context.Context, _ uuid.UUID) error {
	return errors.New("db error")
}
func (r *errRefreshRepo) DeleteExpired(_ context.Context, _ time.Time) (int64, error) { return 0, nil }
