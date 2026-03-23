package database_test

import (
	"strings"
	"testing"

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
	// DSN should contain "password=" but with no value following it before the next field.
	if strings.Contains(dsn, "password=secret") {
		t.Error("BuildDSN: non-empty password leaked when password is empty")
	}
	if !strings.Contains(dsn, "password=") {
		t.Error("BuildDSN: password field missing entirely")
	}
}
