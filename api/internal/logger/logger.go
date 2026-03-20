// Package logger constructs the structured slog.Logger for the Postulate API.
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/popatkaran/postulate/api/internal/config"
)

// NewWithHandler constructs a *slog.Logger using the provided inner handler wrapped
// with the otelHandler. Used in tests to redirect output to a buffer.
func NewWithHandler(inner slog.Handler, cfg config.ObservabilityConfig) *slog.Logger {
	instanceID := cfg.InstanceID
	if instanceID == "" {
		if h, err := os.Hostname(); err == nil {
			instanceID = h
		}
	}
	return slog.New(&otelHandler{inner: inner}).With(
		"serviceId", cfg.ServiceID,
		"instanceId", instanceID,
	)
}

// New constructs a *slog.Logger configured for the given environment.
// In production/staging the underlying handler emits JSON to stdout.
// In development human-readable text is used.
// serviceId and instanceId are pre-set as default attributes on every record.
func New(cfg config.ObservabilityConfig, environment string) *slog.Logger {
	return NewWithWriter(cfg, environment, os.Stdout)
}

// NewWithWriter is like New but writes to w instead of os.Stdout.
// Use this in tests to capture log output.
func NewWithWriter(cfg config.ObservabilityConfig, environment string, w io.Writer) *slog.Logger {
	level := parseLevel(cfg.LogLevel)
	opts := &slog.HandlerOptions{Level: level}

	var inner slog.Handler
	if environment == "production" || environment == "staging" {
		inner = slog.NewJSONHandler(w, opts)
	} else {
		inner = slog.NewTextHandler(w, opts)
	}

	instanceID := cfg.InstanceID
	if instanceID == "" {
		if h, err := os.Hostname(); err == nil {
			instanceID = h
		}
	}

	return slog.New(&otelHandler{inner: inner}).With(
		"serviceId", cfg.ServiceID,
		"instanceId", instanceID,
	)
}

// parseLevel converts a config log level string to slog.Level.
// Defaults to slog.LevelInfo for unrecognised values.
func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
