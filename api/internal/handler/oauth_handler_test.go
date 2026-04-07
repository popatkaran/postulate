package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/popatkaran/postulate/api/internal/auth"
	"github.com/popatkaran/postulate/api/internal/handler"
	"log/slog"
	"os"
)

// newTestOAuthHandler builds an OAuthHandler with real in-memory collaborators.
// The TokenIssuer is nil because token issuance requires a DB; tests that reach
// that path are covered by auth_test.go and integration tests.
func newTestOAuthHandler() *handler.OAuthHandler {
	states := auth.NewStateStore()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return handler.NewOAuthHandler(states, nil, nil, logger)
}

// ── BeginGitHub ───────────────────────────────────────────────────────────────

func TestBeginGitHub_NoSessionSecret_Returns400OrRedirect(t *testing.T) {
	// Without SESSION_SECRET, gothic cannot store the state in a cookie session
	// and returns an error. The handler converts this to a 4xx or 5xx response.
	// We assert that the response is not 200 (i.e. the flow did not succeed).
	h := newTestOAuthHandler()
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/oauth/github", nil)
	q := req.URL.Query()
	q.Set("provider", "github")
	req.URL.RawQuery = q.Encode()
	rec := httptest.NewRecorder()

	h.BeginGitHub(rec, req)

	if rec.Code == http.StatusOK {
		t.Errorf("expected non-200 when gothic cannot initialise session, got 200")
	}
}

// ── CallbackGitHub ────────────────────────────────────────────────────────────

func TestCallbackGitHub_MissingState_Returns400(t *testing.T) {
	h := newTestOAuthHandler()
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/oauth/github/callback", nil)
	rec := httptest.NewRecorder()

	h.CallbackGitHub(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCallbackGitHub_InvalidState_Returns400(t *testing.T) {
	h := newTestOAuthHandler()
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/oauth/github/callback?state=invalid-state", nil)
	rec := httptest.NewRecorder()

	h.CallbackGitHub(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCallbackGitHub_ProviderError_Returns401(t *testing.T) {
	states := auth.NewStateStore()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	h := handler.NewOAuthHandler(states, nil, nil, logger)

	state, _ := states.Generate()
	req := httptest.NewRequest(http.MethodGet,
		"/v1/auth/oauth/github/callback?state="+state+"&error=access_denied", nil)
	rec := httptest.NewRecorder()

	h.CallbackGitHub(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestCallbackGitHub_ValidStateButNoGothProvider_Returns500(t *testing.T) {
	// State is valid; gothic.CompleteUserAuth will fail because no provider is
	// registered — handler must return 500.
	states := auth.NewStateStore()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	h := handler.NewOAuthHandler(states, nil, nil, logger)

	state, _ := states.Generate()
	req := httptest.NewRequest(http.MethodGet,
		"/v1/auth/oauth/github/callback?state="+state+"&provider=github&code=somecode", nil)
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()

	h.CallbackGitHub(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// ── BeginGoogle ───────────────────────────────────────────────────────────────

func TestBeginGoogle_NoSessionSecret_ReturnsNon200(t *testing.T) {
	h := newTestOAuthHandler()
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/oauth/google?provider=google", nil)
	rec := httptest.NewRecorder()

	h.BeginGoogle(rec, req)

	if rec.Code == http.StatusOK {
		t.Errorf("expected non-200 when gothic cannot initialise session, got 200")
	}
}

// ── CallbackGoogle ────────────────────────────────────────────────────────────

func TestCallbackGoogle_MissingState_Returns400(t *testing.T) {
	h := newTestOAuthHandler()
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/oauth/google/callback", nil)
	rec := httptest.NewRecorder()

	h.CallbackGoogle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCallbackGoogle_InvalidState_Returns400(t *testing.T) {
	h := newTestOAuthHandler()
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/oauth/google/callback?state=bad-state", nil)
	rec := httptest.NewRecorder()

	h.CallbackGoogle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCallbackGoogle_ProviderError_Returns401(t *testing.T) {
	states := auth.NewStateStore()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	h := handler.NewOAuthHandler(states, nil, nil, logger)

	state, _ := states.Generate()
	req := httptest.NewRequest(http.MethodGet,
		"/v1/auth/oauth/google/callback?state="+state+"&error=access_denied", nil)
	rec := httptest.NewRecorder()

	h.CallbackGoogle(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestCallbackGoogle_ValidStateButNoGothProvider_Returns500(t *testing.T) {
	states := auth.NewStateStore()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	h := handler.NewOAuthHandler(states, nil, nil, logger)

	state, _ := states.Generate()
	req := httptest.NewRequest(http.MethodGet,
		"/v1/auth/oauth/google/callback?state="+state+"&provider=google&code=somecode", nil)
	rec := httptest.NewRecorder()

	h.CallbackGoogle(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestCallbackGoogle_UsedState_Returns400(t *testing.T) {
	states := auth.NewStateStore()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	h := handler.NewOAuthHandler(states, nil, nil, logger)

	state, _ := states.Generate()
	// First use — consumes the state (will fail at CompleteUserAuth, but state is consumed).
	req1 := httptest.NewRequest(http.MethodGet,
		"/v1/auth/oauth/google/callback?state="+state+"&code=code1", nil)
	rec1 := httptest.NewRecorder()
	h.CallbackGoogle(rec1, req1)

	// Second use of the same state — must return 400.
	req2 := httptest.NewRequest(http.MethodGet,
		"/v1/auth/oauth/google/callback?state="+state+"&code=code2", nil)
	rec2 := httptest.NewRecorder()
	h.CallbackGoogle(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on reused state, got %d", rec2.Code)
	}
}
