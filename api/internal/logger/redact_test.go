package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/popatkaran/postulate/api/internal/config"
	applogger "github.com/popatkaran/postulate/api/internal/logger"
)

func captureField(t *testing.T, key string, value any) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	inner := slog.NewJSONHandler(&buf, opts)
	logger := applogger.NewWithHandler(inner, config.ObservabilityConfig{
		ServiceID:  "test",
		InstanceID: "host",
	})
	logger.Info("test", key, value)

	var entry map[string]any
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}
	return entry
}

func TestRedact_PasswordFieldIsRedacted(t *testing.T) {
	entry := captureField(t, "password", "s3cr3t")
	if entry["password"] != "[redacted]" {
		t.Errorf("expected [redacted], got %v", entry["password"])
	}
}

func TestRedact_APIKeyFieldIsRedacted(t *testing.T) {
	entry := captureField(t, "api_key", "abc123")
	if entry["api_key"] != "[redacted]" {
		t.Errorf("expected [redacted], got %v", entry["api_key"])
	}
}

func TestRedact_AccessTokenFieldIsRedacted(t *testing.T) {
	entry := captureField(t, "access_token", "tok_xyz")
	if entry["access_token"] != "[redacted]" {
		t.Errorf("expected [redacted], got %v", entry["access_token"])
	}
}

func TestRedact_UsernameFieldIsNotRedacted(t *testing.T) {
	entry := captureField(t, "username", "alice")
	if entry["username"] != "alice" {
		t.Errorf("expected alice, got %v", entry["username"])
	}
}

func TestRedact_MessageFieldIsNotRedacted(t *testing.T) {
	entry := captureField(t, "message", "hello world")
	if entry["message"] != "hello world" {
		t.Errorf("expected 'hello world', got %v", entry["message"])
	}
}
