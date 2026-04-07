// Package auth implements OAuth provider setup, user resolution, and session
// token issuance for the Postulate API.
package auth

import (
	"time"

	"github.com/markbates/goth"
)

// ProviderUser is the internal representation of a user returned by an OAuth
// provider. goth.User must be mapped to this type immediately — it must not
// propagate beyond this package.
type ProviderUser struct {
	Provider     string
	ProviderUID  string // "sub" claim (Google) or numeric user ID as string (GitHub)
	Email        string
	Name         string
	AccessToken  string
	RefreshToken string
	TokenExpiry  time.Time
}

// GothUserToProviderUser maps a goth.User to the internal ProviderUser type.
// Handlers must call this immediately and not retain the goth.User value.
func GothUserToProviderUser(u goth.User) ProviderUser {
	return ProviderUser{
		Provider:     u.Provider,
		ProviderUID:  u.UserID,
		Email:        u.Email,
		Name:         u.Name,
		AccessToken:  u.AccessToken,
		RefreshToken: u.RefreshToken,
		TokenExpiry:  u.ExpiresAt,
	}
}
