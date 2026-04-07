package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/popatkaran/postulate/api/internal/auth"
)

const (
	clockSkew = 60 * time.Second

	// RFC 7807 type URIs — duplicated here to avoid an import cycle with the
	// problem package (problem imports middleware for RequestIDFromContext).
	typeUnauthorized = "https://postulate.dev/errors/unauthorized"
	typeForbidden    = "https://postulate.dev/errors/forbidden"
)

// AuthRequired validates the Bearer JWT on every request.
// On success it injects auth.Identity into the request context.
// On any failure it returns 401 Unauthorized — the response does not distinguish
// missing header, malformed token, or expired token.
//
// Known limitation: a valid JWT for a deactivated user still passes until natural
// expiry. This is the accepted trade-off for a stateless JWT design; no database
// call is made in this hot path.
func AuthRequired(jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := bearerToken(r)
			if raw == "" {
				writeAuthProblem(w, r, http.StatusUnauthorized, typeUnauthorized, "Unauthorized")
				return
			}

			tok, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return jwtSecret, nil
			}, jwt.WithLeeway(clockSkew))

			if err != nil || !tok.Valid {
				writeAuthProblem(w, r, http.StatusUnauthorized, typeUnauthorized, "Unauthorized")
				return
			}

			claims, ok := tok.Claims.(jwt.MapClaims)
			if !ok {
				writeAuthProblem(w, r, http.StatusUnauthorized, typeUnauthorized, "Unauthorized")
				return
			}

			sub, _ := claims["sub"].(string)
			role, _ := claims["role"].(string)
			if sub == "" || role == "" {
				writeAuthProblem(w, r, http.StatusUnauthorized, typeUnauthorized, "Unauthorized")
				return
			}

			ctx := auth.ContextWithIdentity(r.Context(), auth.Identity{UserID: sub, Role: role})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole gates access by role. Must be chained after AuthRequired.
// Returns 403 Forbidden if the caller's role does not match the required role.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := auth.IdentityFromContext(r.Context())
			if !ok || id.Role != role {
				writeAuthProblem(w, r, http.StatusForbidden, typeForbidden, "Forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// bearerToken extracts the raw token from the Authorization header.
// Returns empty string if the header is absent or not a Bearer scheme.
func bearerToken(r *http.Request) string {
	v := r.Header.Get("Authorization")
	if !strings.HasPrefix(v, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(v, "Bearer ")
}

// writeAuthProblem writes an RFC 7807 response without importing the problem
// package, avoiding an import cycle (problem → middleware → problem).
func writeAuthProblem(w http.ResponseWriter, r *http.Request, status int, errType, title string) {
	p := struct {
		Type      string `json:"type"`
		Title     string `json:"title"`
		Status    int    `json:"status"`
		Detail    string `json:"detail"`
		Instance  string `json:"instance"`
		RequestID string `json:"request_id,omitempty"`
	}{
		Type:      errType,
		Title:     title,
		Status:    status,
		Detail:    "authentication required",
		Instance:  r.URL.Path,
		RequestID: RequestIDFromContext(r.Context()),
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}
