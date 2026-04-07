package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/popatkaran/postulate/api/internal/auth"
	"github.com/popatkaran/postulate/api/internal/middleware"
)

const testSecret = "a-test-jwt-secret-that-is-32-bytes!"

// okHandler is a sentinel handler that records whether it was called.
func okHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

func signedToken(t *testing.T, claims jwt.MapClaims, secret string) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tok
}

func validClaims() jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"sub":  "user-id-123",
		"role": "platform_member",
		"iat":  now.Unix(),
		"exp":  now.Add(8 * time.Hour).Unix(),
	}
}

// ── AuthRequired ──────────────────────────────────────────────────────────────

func TestAuthRequired_NoHeader_Returns401(t *testing.T) {
	called := false
	h := middleware.AuthRequired([]byte(testSecret))(okHandler(&called))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	if called {
		t.Error("next handler must not be called on missing token")
	}
}

func TestAuthRequired_MalformedToken_Returns401(t *testing.T) {
	called := false
	h := middleware.AuthRequired([]byte(testSecret))(okHandler(&called))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer not.a.jwt")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	if called {
		t.Error("next handler must not be called on malformed token")
	}
}

func TestAuthRequired_ExpiredToken_Returns401(t *testing.T) {
	called := false
	h := middleware.AuthRequired([]byte(testSecret))(okHandler(&called))

	claims := jwt.MapClaims{
		"sub":  "user-id-123",
		"role": "platform_member",
		"iat":  time.Now().Add(-10 * time.Hour).Unix(),
		"exp":  time.Now().Add(-2 * time.Hour).Unix(), // expired, outside 60s skew
	}
	tok := signedToken(t, claims, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	if called {
		t.Error("next handler must not be called on expired token")
	}
}

func TestAuthRequired_WrongSecret_Returns401(t *testing.T) {
	called := false
	h := middleware.AuthRequired([]byte(testSecret))(okHandler(&called))

	tok := signedToken(t, validClaims(), "a-different-secret-that-is-32-bytes!")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthRequired_ValidToken_CallsNextAndInjectsIdentity(t *testing.T) {
	var capturedIdentity auth.Identity
	var identityOK bool

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedIdentity, identityOK = auth.IdentityFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	h := middleware.AuthRequired([]byte(testSecret))(next)
	tok := signedToken(t, validClaims(), testSecret)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !identityOK {
		t.Fatal("expected identity in context, got none")
	}
	if capturedIdentity.UserID != "user-id-123" {
		t.Errorf("UserID: expected user-id-123, got %q", capturedIdentity.UserID)
	}
	if capturedIdentity.Role != "platform_member" {
		t.Errorf("Role: expected platform_member, got %q", capturedIdentity.Role)
	}
}

func TestAuthRequired_TokenWithinClockSkew_Passes(t *testing.T) {
	// Token expired 30 seconds ago — within the 60s clock skew tolerance.
	called := false
	h := middleware.AuthRequired([]byte(testSecret))(okHandler(&called))

	claims := jwt.MapClaims{
		"sub":  "user-id-123",
		"role": "platform_member",
		"iat":  time.Now().Add(-8 * time.Hour).Unix(),
		"exp":  time.Now().Add(-30 * time.Second).Unix(),
	}
	tok := signedToken(t, claims, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 within clock skew, got %d", rec.Code)
	}
	if !called {
		t.Error("expected next handler to be called within clock skew")
	}
}

func TestAuthRequired_NonBearerScheme_Returns401(t *testing.T) {
	called := false
	h := middleware.AuthRequired([]byte(testSecret))(okHandler(&called))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// ── RequireRole ───────────────────────────────────────────────────────────────

func TestRequireRole_MatchingRole_CallsNext(t *testing.T) {
	called := false
	h := withIdentity("platform_admin", middleware.RequireRole("platform_admin")(okHandler(&called)))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !called {
		t.Error("expected next handler to be called for matching role")
	}
}

func TestRequireRole_WrongRole_Returns403(t *testing.T) {
	called := false
	h := withIdentity("platform_member", middleware.RequireRole("platform_admin")(okHandler(&called)))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
	if called {
		t.Error("next handler must not be called for wrong role")
	}
}

func TestRequireRole_NoIdentityInContext_Returns403(t *testing.T) {
	called := false
	h := middleware.RequireRole("platform_admin")(okHandler(&called))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

// withIdentity injects an Identity into the request context before calling h.
func withIdentity(role string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := auth.ContextWithIdentity(r.Context(), auth.Identity{UserID: "uid", Role: role})
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TestAuthRequired_NonHMACSigningMethod_Returns401(t *testing.T) {
	// A token signed with RS256 (non-HMAC) must be rejected.
	called := false
	h := middleware.AuthRequired([]byte(testSecret))(okHandler(&called))

	// Craft a token with a different algorithm header by using none algorithm.
	// jwt library won't sign with "none", so we manually build a fake token.
	// Easiest: sign with HMAC but then tamper the header to claim RS256.
	// Instead, use an unsupported method token that the library will reject.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyIn0.invalidsig")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for non-HMAC token, got %d", rec.Code)
	}
	if called {
		t.Error("next handler must not be called for non-HMAC token")
	}
}

func TestAuthRequired_MissingSubClaim_Returns401(t *testing.T) {
	called := false
	h := middleware.AuthRequired([]byte(testSecret))(okHandler(&called))

	claims := jwt.MapClaims{
		"role": "platform_member",
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(8 * time.Hour).Unix(),
	}
	tok := signedToken(t, claims, testSecret)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing sub, got %d", rec.Code)
	}
	if called {
		t.Error("next handler must not be called when sub is missing")
	}
}

func TestAuthRequired_MissingRoleClaim_Returns401(t *testing.T) {
	called := false
	h := middleware.AuthRequired([]byte(testSecret))(okHandler(&called))

	claims := jwt.MapClaims{
		"sub": "user-id-123",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(8 * time.Hour).Unix(),
	}
	tok := signedToken(t, claims, testSecret)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing role, got %d", rec.Code)
	}
	if called {
		t.Error("next handler must not be called when role is missing")
	}
}
