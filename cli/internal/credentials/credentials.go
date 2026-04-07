// Package credentials manages the ~/.postulate/auth.json file.
// All file paths and permission constants are centralised here.
package credentials

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	dirPerm  = 0o700
	filePerm = 0o600
)

// AuthFile is the structure persisted to ~/.postulate/auth.json.
type AuthFile struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"` // ISO 8601
	Role         string `json:"role"`
	APIURL       string `json:"api_url"`
}

// authFilePath returns the absolute path to auth.json.
func authFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".postulate", "auth.json"), nil
}

// Load reads and parses auth.json. Returns ErrNotFound if the file is absent.
func Load() (*AuthFile, error) {
	path, err := authFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read auth file: %w", err)
	}
	var af AuthFile
	if err := json.Unmarshal(data, &af); err != nil {
		return nil, fmt.Errorf("parse auth file: %w", err)
	}
	return &af, nil
}

// Save writes af to ~/.postulate/auth.json with permissions 600.
// Creates ~/.postulate/ (700) if it does not exist.
func Save(af *AuthFile) error {
	path, err := authFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("create credentials dir: %w", err)
	}
	data, err := json.MarshalIndent(af, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal auth file: %w", err)
	}
	if err := os.WriteFile(path, data, filePerm); err != nil {
		return fmt.Errorf("write auth file: %w", err)
	}
	return nil
}

// Delete removes auth.json. No-op if the file does not exist.
func Delete() error {
	path, err := authFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete auth file: %w", err)
	}
	return nil
}

// NeedsRefresh reports whether the token in af expires within 30 minutes.
func NeedsRefresh(af *AuthFile) bool {
	t, err := time.Parse(time.RFC3339, af.ExpiresAt)
	if err != nil {
		return true // treat unparseable expiry as expired
	}
	return time.Until(t) < 30*time.Minute
}

// ErrNotFound is returned by Load when auth.json does not exist.
var ErrNotFound = errors.New("auth file not found")
