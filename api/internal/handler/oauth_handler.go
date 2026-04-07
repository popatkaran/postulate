package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/markbates/goth/gothic"
	"github.com/popatkaran/postulate/api/internal/auth"
	"github.com/popatkaran/postulate/api/internal/problem"
)

// OAuthHandler handles the Google OAuth 2.0 authorisation code flow.
type OAuthHandler struct {
	states   *auth.StateStore
	resolver *auth.UserResolver
	issuer   *auth.TokenIssuer
	logger   *slog.Logger
}

// NewOAuthHandler constructs an OAuthHandler.
func NewOAuthHandler(
	states *auth.StateStore,
	resolver *auth.UserResolver,
	issuer *auth.TokenIssuer,
	logger *slog.Logger,
) *OAuthHandler {
	return &OAuthHandler{states: states, resolver: resolver, issuer: issuer, logger: logger}
}

// BeginGitHub initiates the GitHub OAuth flow.
// GET /v1/auth/oauth/github
func (h *OAuthHandler) BeginGitHub(w http.ResponseWriter, r *http.Request) {
	state, err := h.states.Generate()
	if err != nil {
		h.logger.Error("failed to generate oauth state", "error", err)
		problem.Write(w, r, problem.New(problem.TypeInternalServerError, "Internal Server Error", http.StatusInternalServerError, "", ""))
		return
	}
	q := r.URL.Query()
	q.Set("state", state)
	r.URL.RawQuery = q.Encode()

	gothic.BeginAuthHandler(w, r)
}

// CallbackGitHub handles the GitHub OAuth callback.
// GET /v1/auth/oauth/github/callback
func (h *OAuthHandler) CallbackGitHub(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if !h.states.Validate(state) {
		problem.Write(w, r, problem.New(
			problem.TypeBadRequest, "Bad Request", http.StatusBadRequest,
			"invalid or expired state parameter", "",
		))
		return
	}

	if errParam := r.URL.Query().Get("error"); errParam != "" {
		problem.Write(w, r, problem.New(
			problem.TypeUnauthorized, "Unauthorized", http.StatusUnauthorized,
			"OAuth provider returned an error", "",
		))
		return
	}

	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		h.logger.Error("oauth complete user auth failed", "error", err)
		problem.Write(w, r, problem.New(
			problem.TypeInternalServerError, "Internal Server Error", http.StatusInternalServerError, "", "",
		))
		return
	}

	pu := auth.GothUserToProviderUser(gothUser)

	if pu.Email == "" {
		problem.Write(w, r, problem.New(
			problem.TypeUnprocessableEntity, "Unprocessable Entity", http.StatusUnprocessableEntity,
			"OAuth provider returned no email address", "",
		))
		return
	}

	user, err := h.resolver.ResolveOrCreateUser(r.Context(), pu)
	if err != nil {
		h.logger.Error("resolve or create user failed", "error", err)
		problem.Write(w, r, problem.New(
			problem.TypeInternalServerError, "Internal Server Error", http.StatusInternalServerError, "", "",
		))
		return
	}

	tr, err := h.issuer.IssueSessionToken(r.Context(), user.ID, string(user.Role))
	if err != nil {
		h.logger.Error("issue session token failed", "error", err)
		problem.Write(w, r, problem.New(
			problem.TypeInternalServerError, "Internal Server Error", http.StatusInternalServerError, "", "",
		))
		return
	}

	if redirectURI := r.URL.Query().Get("redirect_uri"); redirectURI != "" {
		u, err := url.Parse(redirectURI)
		if err == nil {
			q := u.Query()
			q.Set("token", tr.Token)
			q.Set("refresh_token", tr.RefreshToken)
			q.Set("expires_at", tr.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"))
			q.Set("role", tr.Role)
			u.RawQuery = q.Encode()
			http.Redirect(w, r, u.String(), http.StatusFound)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"token":         tr.Token,
		"refresh_token": tr.RefreshToken,
		"expires_at":    tr.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		"role":          tr.Role,
	})
}
func (h *OAuthHandler) BeginGoogle(w http.ResponseWriter, r *http.Request) {
	state, err := h.states.Generate()
	if err != nil {
		h.logger.Error("failed to generate oauth state", "error", err)
		problem.Write(w, r, problem.New(problem.TypeInternalServerError, "Internal Server Error", http.StatusInternalServerError, "", ""))
		return
	}
	// Inject state into the request so gothic picks it up.
	q := r.URL.Query()
	q.Set("state", state)
	r.URL.RawQuery = q.Encode()

	gothic.BeginAuthHandler(w, r)
}

// CallbackGoogle handles the Google OAuth callback.
// GET /v1/auth/oauth/google/callback
func (h *OAuthHandler) CallbackGoogle(w http.ResponseWriter, r *http.Request) {
	// Validate state — single-use, 5-minute TTL.
	state := r.URL.Query().Get("state")
	if !h.states.Validate(state) {
		problem.Write(w, r, problem.New(
			problem.TypeBadRequest, "Bad Request", http.StatusBadRequest,
			"invalid or expired state parameter", "",
		))
		return
	}

	// Check if the provider returned an error (e.g. user denied access).
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		problem.Write(w, r, problem.New(
			problem.TypeUnauthorized, "Unauthorized", http.StatusUnauthorized,
			"OAuth provider returned an error", "",
		))
		return
	}

	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		h.logger.Error("oauth complete user auth failed", "error", err)
		problem.Write(w, r, problem.New(
			problem.TypeInternalServerError, "Internal Server Error", http.StatusInternalServerError, "", "",
		))
		return
	}

	pu := auth.GothUserToProviderUser(gothUser)

	if pu.Email == "" {
		problem.Write(w, r, problem.New(
			problem.TypeUnprocessableEntity, "Unprocessable Entity", http.StatusUnprocessableEntity,
			"OAuth provider returned no email address", "",
		))
		return
	}

	user, err := h.resolver.ResolveOrCreateUser(r.Context(), pu)
	if err != nil {
		h.logger.Error("resolve or create user failed", "error", err)
		problem.Write(w, r, problem.New(
			problem.TypeInternalServerError, "Internal Server Error", http.StatusInternalServerError, "", "",
		))
		return
	}

	tr, err := h.issuer.IssueSessionToken(r.Context(), user.ID, string(user.Role))
	if err != nil {
		h.logger.Error("issue session token failed", "error", err)
		problem.Write(w, r, problem.New(
			problem.TypeInternalServerError, "Internal Server Error", http.StatusInternalServerError, "", "",
		))
		return
	}

	// CLI clients supply a redirect_uri (loopback callback server).
	if redirectURI := r.URL.Query().Get("redirect_uri"); redirectURI != "" {
		u, err := url.Parse(redirectURI)
		if err == nil {
			q := u.Query()
			q.Set("token", tr.Token)
			q.Set("refresh_token", tr.RefreshToken)
			q.Set("expires_at", tr.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"))
			q.Set("role", tr.Role)
			u.RawQuery = q.Encode()
			http.Redirect(w, r, u.String(), http.StatusFound)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"token":         tr.Token,
		"refresh_token": tr.RefreshToken,
		"expires_at":    tr.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		"role":          tr.Role,
	})
}
