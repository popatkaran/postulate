// Package middleware provides HTTP middleware for the Postulate API.
package middleware

import (
	"context"
	"net/http"
	"regexp"

	"github.com/oklog/ulid/v2"
)

// contextKey is an unexported type for context keys in this package,
// preventing collisions with keys from other packages.
type contextKey int

const requestIDKey contextKey = iota

// validRequestID matches ULID (26 chars, Crockford base32) or UUID (8-4-4-4-12 hex).
var validRequestID = regexp.MustCompile(
	`^[0-9A-HJKMNP-TV-Z]{26}$|^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
)

// RequestIDFromContext returns the request ID stored in ctx, or empty string.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// ContextWithRequestID returns a copy of ctx with the given request ID stored.
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID is middleware that assigns a unique ULID to every request.
// It accepts an existing X-Request-ID header when it is a valid ULID or UUID;
// otherwise it generates a new ULID. The ID is stored in the request context
// and echoed back in the X-Request-ID response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !validRequestID.MatchString(id) {
			id = ulid.Make().String()
		}
		ctx := ContextWithRequestID(r.Context(), id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
