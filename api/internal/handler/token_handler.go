package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/popatkaran/postulate/api/internal/auth"
	"github.com/popatkaran/postulate/api/internal/middleware"
	"github.com/popatkaran/postulate/api/internal/problem"
)

// TokenHandler handles token lifecycle endpoints (refresh).
type TokenHandler struct {
	issuer *auth.TokenIssuer
	logger *slog.Logger
}

// NewTokenHandler constructs a TokenHandler.
func NewTokenHandler(issuer *auth.TokenIssuer, logger *slog.Logger) *TokenHandler {
	return &TokenHandler{issuer: issuer, logger: logger}
}

// Refresh rotates a refresh token and issues a new JWT.
// POST /v1/auth/token/refresh
func (h *TokenHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
		problem.Write(w, r, problem.New(
			problem.TypeBadRequest, "Bad Request", http.StatusBadRequest,
			"refresh_token is required", "",
		))
		return
	}

	tr, err := h.issuer.RefreshSession(r.Context(), body.RefreshToken)
	if err != nil {
		// Do not distinguish "not found" from "expired" — return generic 401.
		problem.Write(w, r, problem.New(
			problem.TypeUnauthorized, "Unauthorized", http.StatusUnauthorized,
			"invalid or expired refresh token", "",
		))
		return
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

// Revoke deletes all refresh tokens for the authenticated user (logout).
// DELETE /v1/auth/token
//
// Outstanding JWTs remain valid until their natural 8-hour expiry after this call.
// This is the accepted trade-off for a stateless JWT design — no token blocklist is
// maintained. The CLI deletes auth.json immediately on receiving 204.
func (h *TokenHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		problem.Write(w, r, problem.New(
			problem.TypeUnauthorized, "Unauthorized", http.StatusUnauthorized,
			"authentication required", "",
		))
		return
	}

	userID, err := uuid.Parse(id.UserID)
	if err != nil {
		problem.Write(w, r, problem.New(
			problem.TypeUnauthorized, "Unauthorized", http.StatusUnauthorized,
			"authentication required", "",
		))
		return
	}

	if err := h.issuer.RevokeAllSessions(r.Context(), userID); err != nil {
		h.logger.Error("revoke sessions failed", "error", err, "request_id", middleware.RequestIDFromContext(r.Context()))
		problem.Write(w, r, problem.New(
			problem.TypeInternalServerError, "Internal Server Error", http.StatusInternalServerError,
			"failed to revoke session", "",
		))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
