package credentials_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/popatkaran/postulate/cli/internal/credentials"
)

func withTempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	return dir
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	withTempHome(t)
	af := &credentials.AuthFile{
		Token:        "tok",
		RefreshToken: "rt",
		ExpiresAt:    "2099-01-01T00:00:00Z",
		Role:         "platform_member",
		APIURL:       "https://api.example.com",
	}
	if err := credentials.Save(af); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := credentials.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Token != af.Token || got.Role != af.Role || got.APIURL != af.APIURL {
		t.Errorf("round-trip mismatch: got %+v", got)
	}
}

func TestSave_CreatesDir700AndFile600(t *testing.T) {
	home := withTempHome(t)
	af := &credentials.AuthFile{Token: "t", RefreshToken: "r", ExpiresAt: "2099-01-01T00:00:00Z", Role: "platform_member", APIURL: "u"}
	if err := credentials.Save(af); err != nil {
		t.Fatalf("Save: %v", err)
	}
	dirInfo, err := os.Stat(filepath.Join(home, ".postulate"))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Errorf("dir perm: expected 0700, got %o", dirInfo.Mode().Perm())
	}
	fileInfo, err := os.Stat(filepath.Join(home, ".postulate", "auth.json"))
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if fileInfo.Mode().Perm() != 0o600 {
		t.Errorf("file perm: expected 0600, got %o", fileInfo.Mode().Perm())
	}
}

func TestLoad_MissingFile_ReturnsErrNotFound(t *testing.T) {
	withTempHome(t)
	_, err := credentials.Load()
	if err != credentials.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete_RemovesFile(t *testing.T) {
	withTempHome(t)
	af := &credentials.AuthFile{Token: "t", RefreshToken: "r", ExpiresAt: "2099-01-01T00:00:00Z", Role: "platform_member", APIURL: "u"}
	_ = credentials.Save(af)
	if err := credentials.Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := credentials.Load()
	if err != credentials.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDelete_NoFile_NoError(t *testing.T) {
	withTempHome(t)
	if err := credentials.Delete(); err != nil {
		t.Errorf("expected no error deleting absent file, got %v", err)
	}
}

func TestNeedsRefresh_ExpiresInOneHour_ReturnsFalse(t *testing.T) {
	af := &credentials.AuthFile{ExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}
	if credentials.NeedsRefresh(af) {
		t.Error("expected NeedsRefresh=false for token expiring in 1h")
	}
}

func TestNeedsRefresh_ExpiresIn10Minutes_ReturnsTrue(t *testing.T) {
	af := &credentials.AuthFile{ExpiresAt: time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339)}
	if !credentials.NeedsRefresh(af) {
		t.Error("expected NeedsRefresh=true for token expiring in 10m")
	}
}

func TestNeedsRefresh_AlreadyExpired_ReturnsTrue(t *testing.T) {
	af := &credentials.AuthFile{ExpiresAt: time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)}
	if !credentials.NeedsRefresh(af) {
		t.Error("expected NeedsRefresh=true for already-expired token")
	}
}

func TestNeedsRefresh_UnparsableExpiry_ReturnsTrue(t *testing.T) {
	af := &credentials.AuthFile{ExpiresAt: "not-a-date"}
	if !credentials.NeedsRefresh(af) {
		t.Error("expected NeedsRefresh=true for unparseable expiry")
	}
}
