package startup_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/popatkaran/postulate/api/internal/startup"
)

var nopLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

// fakePinger is a test double for startup.Pinger.
type fakePinger struct{ err error }

func (f *fakePinger) Ping(_ context.Context) error { return f.err }

func TestCheckDatabase_ReturnsNilWhenPingSucceeds(t *testing.T) {
	err := startup.CheckDatabase(context.Background(), &fakePinger{}, "production", nopLogger)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckDatabase_ReturnsErrorWhenPingFails(t *testing.T) {
	pingErr := errors.New("connection refused")
	err := startup.CheckDatabase(context.Background(), &fakePinger{err: pingErr}, "production", nopLogger)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCheckDatabase_HintIncludedInDevelopment(t *testing.T) {
	var buf []byte
	capLogger := slog.New(slog.NewTextHandler(
		&logWriter{write: func(b []byte) { buf = append(buf, b...) }},
		nil,
	))

	pingErr := errors.New("connection refused")
	_ = startup.CheckDatabase(context.Background(), &fakePinger{err: pingErr}, "development", capLogger)

	if !contains(buf, "hint") {
		t.Error("expected 'hint' in log output for development env")
	}
}

func TestCheckDatabase_HintOmittedInProduction(t *testing.T) {
	var buf []byte
	capLogger := slog.New(slog.NewTextHandler(
		&logWriter{write: func(b []byte) { buf = append(buf, b...) }},
		nil,
	))

	pingErr := errors.New("connection refused")
	_ = startup.CheckDatabase(context.Background(), &fakePinger{err: pingErr}, "production", capLogger)

	if contains(buf, "hint") {
		t.Error("expected no 'hint' in log output for production env")
	}
}

// logWriter adapts a write func to io.Writer.
type logWriter struct{ write func([]byte) }

func (w *logWriter) Write(p []byte) (int, error) {
	w.write(p)
	return len(p), nil
}

func contains(buf []byte, s string) bool {
	return len(buf) > 0 && string(buf) != "" && indexBytes(buf, s) >= 0
}

func indexBytes(buf []byte, s string) int {
	for i := 0; i <= len(buf)-len(s); i++ {
		if string(buf[i:i+len(s)]) == s {
			return i
		}
	}
	return -1
}
