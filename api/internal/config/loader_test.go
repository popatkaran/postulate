package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/popatkaran/postulate/api/internal/config"
)

// writeYAML writes content to a temp file and returns its path.
func writeYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("closing temp file: %v", err)
	}
	return f.Name()
}

const validYAML = `
server:
  port: 9090
  environment: staging
  shutdown_timeout_seconds: 15
observability:
  service_id: test-api
  log_level: debug
`

func TestLoad_SuccessFromFile(t *testing.T) {
	// Arrange
	path := writeYAML(t, validYAML)
	t.Setenv("POSTULATE_CONFIG_FILE", path)

	// Act
	cfg, err := config.Load()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Server.Environment != "staging" {
		t.Errorf("expected environment staging, got %s", cfg.Server.Environment)
	}
	if cfg.Observability.ServiceID != "test-api" {
		t.Errorf("expected service_id test-api, got %s", cfg.Observability.ServiceID)
	}
}

func TestLoad_EnvVarOverridesTakesPrecedence(t *testing.T) {
	// Arrange
	path := writeYAML(t, validYAML)
	t.Setenv("POSTULATE_CONFIG_FILE", path)
	t.Setenv("POSTULATE_SERVER_PORT", "7777")
	t.Setenv("POSTULATE_OBSERVABILITY_SERVICE_ID", "overridden-api")

	// Act
	cfg, err := config.Load()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Server.Port != 7777 {
		t.Errorf("expected port 7777 from env override, got %d", cfg.Server.Port)
	}
	if cfg.Observability.ServiceID != "overridden-api" {
		t.Errorf("expected service_id overridden-api, got %s", cfg.Observability.ServiceID)
	}
}

func TestLoad_MissingDefaultConfigFileProceeds(t *testing.T) {
	// Arrange — ensure POSTULATE_CONFIG_FILE is unset and default path does not exist.
	t.Setenv("POSTULATE_CONFIG_FILE", "")
	// Change working dir to a temp dir that has no config.yaml.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	// Act
	cfg, err := config.Load()

	// Assert
	if err != nil {
		t.Fatalf("expected no error when default config absent, got %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestLoad_ExplicitMissingFileReturnsError(t *testing.T) {
	// Arrange
	t.Setenv("POSTULATE_CONFIG_FILE", filepath.Join(t.TempDir(), "nonexistent.yaml"))

	// Act
	_, err := config.Load()

	// Assert
	if err == nil {
		t.Fatal("expected error for missing explicit config file, got nil")
	}
}
