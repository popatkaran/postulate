package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/popatkaran/postulate/api/internal/config"
	applogger "github.com/popatkaran/postulate/api/internal/logger"
)

func TestHandler_TimestampFieldEmittedNotTime(t *testing.T) {
	entry := captureLog(t, defaultCfg(), func(l *slog.Logger) {
		l.Info("ts check")
	})
	if _, ok := entry["timestamp"]; !ok {
		t.Error("expected 'timestamp' field in log output")
	}
	if _, ok := entry["time"]; ok {
		t.Error("unexpected 'time' field in log output — should be 'timestamp'")
	}
}

func TestHandler_WithGroup_NestsFieldsUnderGroupName(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := applogger.NewWithHandler(inner, config.ObservabilityConfig{
		ServiceID:  "test-api",
		InstanceID: "test-host",
	})

	logger.WithGroup("request").Info("grouped", "method", "GET")

	var entry map[string]any
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}

	group, ok := entry["request"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'request' group in log entry, got: %v", entry)
	}
	if group["method"] != "GET" {
		t.Errorf("expected method=GET inside group, got %v", group["method"])
	}
}

func TestHandler_WithGroup_ReturnsNewHandler(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := applogger.NewWithHandler(inner, config.ObservabilityConfig{
		ServiceID: "test-api",
	})

	// WithGroup must not panic and must return a usable logger.
	grouped := logger.WithGroup("grp")
	grouped.Info("ok")

	if buf.Len() == 0 {
		t.Error("expected log output after WithGroup, got none")
	}
}
