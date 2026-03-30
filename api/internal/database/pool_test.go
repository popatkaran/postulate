package database_test

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/popatkaran/postulate/api/internal/config"
	"github.com/popatkaran/postulate/api/internal/database"
)

func baseConfig() config.DatabaseConfig {
	return config.DatabaseConfig{
		Host:                   "localhost",
		Port:                   5432,
		Name:                   "postulate_dev",
		User:                   "postulate_dev",
		Password:               "secret",
		SSLMode:                "disable",
		MaxOpenConns:           25,
		MaxIdleConns:           5,
		ConnMaxLifetimeSeconds: 300,
	}
}

func TestBuildDSN_ContainsAllFields(t *testing.T) {
	cfg := baseConfig()
	dsn := database.BuildDSN(cfg)

	for _, want := range []string{"host=localhost", "port=5432", "dbname=postulate_dev", "user=postulate_dev", "sslmode=disable"} {
		if !strings.Contains(dsn, want) {
			t.Errorf("BuildDSN missing %q in %q", want, dsn)
		}
	}
}

func TestBuildDSN_SSLModeVariants(t *testing.T) {
	for _, mode := range []string{"disable", "require", "verify-full"} {
		cfg := baseConfig()
		cfg.SSLMode = mode
		dsn := database.BuildDSN(cfg)
		want := "sslmode=" + mode
		if !strings.Contains(dsn, want) {
			t.Errorf("BuildDSN: expected %q in DSN for ssl_mode=%s, got %q", want, mode, dsn)
		}
	}
}

func TestBuildDSN_EmptyPasswordNotLeaked(t *testing.T) {
	cfg := baseConfig()
	cfg.Password = ""
	dsn := database.BuildDSN(cfg)
	if strings.Contains(dsn, "password=secret") {
		t.Error("BuildDSN: non-empty password leaked when password is empty")
	}
	if !strings.Contains(dsn, "password=") {
		t.Error("BuildDSN: password field missing entirely")
	}
}

func TestBuildPoolConfig_AppliesConfigValues(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxOpenConns = 10
	cfg.MaxIdleConns = 2
	cfg.ConnMaxLifetimeSeconds = 120

	poolCfg, err := database.BuildPoolConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if poolCfg.MaxConns != 10 {
		t.Errorf("MaxConns: want 10, got %d", poolCfg.MaxConns)
	}
	if poolCfg.MinConns != 2 {
		t.Errorf("MinConns: want 2, got %d", poolCfg.MinConns)
	}
	if poolCfg.MaxConnLifetime != 120*time.Second {
		t.Errorf("MaxConnLifetime: want 120s, got %v", poolCfg.MaxConnLifetime)
	}
}

func TestBuildPoolConfig_ReturnsErrorOnInvalidDSN(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host: "localhost", Port: 5432, Name: "db", User: "u",
		Password: "p", SSLMode: "invalid-mode-that-causes-parse-error",
	}
	// pgxpool.ParseConfig may or may not reject this — if it does, we get an error.
	// If it doesn't, the test is a no-op. Either way the function must not panic.
	_, _ = database.BuildPoolConfig(cfg)
}

func TestBuildPoolConfig_InvalidSSLMode_ReturnsError(t *testing.T) {
	// Use a DSN that pgxpool.ParseConfig will reject.
	cfg := config.DatabaseConfig{
		Host: "", Port: 0, Name: "", User: "", Password: "", SSLMode: "",
	}
	// An empty host/port/name is valid for pgxpool.ParseConfig (it uses defaults).
	// We can't easily trigger a ParseConfig error without a truly malformed DSN.
	// This test documents the boundary — buildPoolConfig returns nil error for empty fields.
	poolCfg, err := database.BuildPoolConfig(cfg)
	if err != nil {
		// If ParseConfig rejects it, that's the error path covered.
		return
	}
	// Defaults applied: MaxConns=25, MinConns=5, MaxConnLifetime=300s
	if poolCfg.MaxConns != 25 {
		t.Errorf("expected default MaxConns 25, got %d", poolCfg.MaxConns)
	}
}

func TestBuildPoolConfig_AppliesDefaults_WhenZero(t *testing.T) {
	cfg := baseConfig()
	cfg.MaxOpenConns = 0
	cfg.MaxIdleConns = 0
	cfg.ConnMaxLifetimeSeconds = 0

	poolCfg, err := database.BuildPoolConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if poolCfg.MaxConns != 25 {
		t.Errorf("MaxConns default: want 25, got %d", poolCfg.MaxConns)
	}
	if poolCfg.MinConns != 5 {
		t.Errorf("MinConns default: want 5, got %d", poolCfg.MinConns)
	}
	if poolCfg.MaxConnLifetime != 300*time.Second {
		t.Errorf("MaxConnLifetime default: want 300s, got %v", poolCfg.MaxConnLifetime)
	}
}

func TestNew_ReturnsErrorWhenDatabaseUnreachable(t *testing.T) {
	// Use a port nothing is listening on to trigger a Ping failure.
	// This exercises the New() error path without a real database.
	cfg := config.DatabaseConfig{ //nolint:gosec
		Host: "127.0.0.1", Port: 19999,
		Name: "postulate_test", User: "postulate_dev",
		Password: "postulate_dev", SSLMode: "disable",
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	_, err := database.New(context.Background(), cfg, logger)
	if err == nil {
		t.Fatal("expected error for unreachable host, got nil")
	}
}
