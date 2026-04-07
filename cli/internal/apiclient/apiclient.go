// Package apiclient provides a minimal HTTP client for Postulate API calls
// needed by the CLI (token refresh and logout).
package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// RefreshResponse holds the fields returned by POST /v1/auth/token/refresh.
type RefreshResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
	Role         string `json:"role"`
}

// Refresh calls POST /v1/auth/token/refresh and returns the new token set.
func Refresh(apiURL, refreshToken string) (*RefreshResponse, error) {
	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	resp, err := http.Post(apiURL+"/v1/auth/token/refresh", "application/json", bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("token refresh request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: status %d", resp.StatusCode)
	}
	var r RefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decode refresh response: %w", err)
	}
	return &r, nil
}

// Logout calls DELETE /v1/auth/token with the given Bearer token.
// Returns the HTTP status code and any transport error.
func Logout(apiURL, token string) (int, error) {
	req, err := http.NewRequest(http.MethodDelete, apiURL+"/v1/auth/token", nil)
	if err != nil {
		return 0, fmt.Errorf("build logout request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("logout request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	return resp.StatusCode, nil
}
