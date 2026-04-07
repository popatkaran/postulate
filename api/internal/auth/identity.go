package auth

import "context"

// contextKey is an unexported type for context keys in this package,
// preventing collisions with keys from other packages.
type contextKey int

const identityKey contextKey = iota

// Identity holds the authenticated caller's identity, extracted from JWT claims.
// No database call is made to populate this — it is resolved entirely from the token.
type Identity struct {
	UserID string
	Role   string
}

// ContextWithIdentity returns a copy of ctx carrying the given Identity.
func ContextWithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey, id)
}

// IdentityFromContext retrieves the Identity stored in ctx.
// Returns the zero value and false if no identity is present.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey).(Identity)
	return id, ok
}
