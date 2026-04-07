package auth

import (
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
)

// RegisterProviders initialises Goth with the configured OAuth providers.
// Must be called once at startup before any OAuth handler is invoked.
func RegisterProviders(googleClientID, googleClientSecret, githubClientID, githubClientSecret, baseURL string) {
	goth.UseProviders(
		google.New(googleClientID, googleClientSecret,
			baseURL+"/v1/auth/oauth/google/callback",
			"openid", "email", "profile"),
		github.New(githubClientID, githubClientSecret,
			baseURL+"/v1/auth/oauth/github/callback",
			"read:user", "user:email"),
	)
}
