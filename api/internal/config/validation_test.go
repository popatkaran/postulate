package config_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/popatkaran/postulate/api/internal/config"
)

func validConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:                   8080,
			Environment:            "development",
			ShutdownTimeoutSeconds: 30,
		},
		Observability: config.ObservabilityConfig{
			ServiceID: "postulate-api",
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Name:     "postulate_dev",
			User:     "postulate_dev",
			Password: "postulate_dev",
			SSLMode:  "disable",
		},
		Auth: config.AuthConfig{
			GoogleClientID:     "test-google-client-id",
			GoogleClientSecret: "test-google-client-secret",
			GitHubClientID:     "test-github-client-id",
			GitHubClientSecret: "test-github-client-secret",
			JWTSecret:          "a-test-jwt-secret-that-is-32-bytes!",
		},
	}
}

func TestValidate_ValidConfigPassesWithNoError(t *testing.T) {
	// Arrange
	cfg := validConfig()

	// Act
	err := config.Validate(cfg)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error for valid config, got %v", err)
	}
}

func TestValidate_MissingServiceIDProducesNamedError(t *testing.T) {
	// Arrange
	cfg := validConfig()
	cfg.Observability.ServiceID = ""

	// Act
	err := config.Validate(cfg)

	// Assert
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !containsField(err, "observability.service_id") {
		t.Errorf("expected error to name observability.service_id, got: %v", err)
	}
}

func TestValidate_InvalidPortProducesValidationError(t *testing.T) {
	// Arrange
	cfg := validConfig()
	cfg.Server.Port = 0

	// Act
	err := config.Validate(cfg)

	// Assert
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !containsField(err, "server.port") {
		t.Errorf("expected error to name server.port, got: %v", err)
	}
}

func TestValidate_InvalidEnvironmentProducesValidationError(t *testing.T) {
	// Arrange
	cfg := validConfig()
	cfg.Server.Environment = "unknown"

	// Act
	err := config.Validate(cfg)

	// Assert
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !containsField(err, "server.environment") {
		t.Errorf("expected error to name server.environment, got: %v", err)
	}
}

func TestValidate_AllErrorsReturnedTogether(t *testing.T) {
	// Arrange — multiple fields invalid simultaneously.
	cfg := &config.Config{}

	// Act
	err := config.Validate(cfg)

	// Assert
	if err == nil {
		t.Fatal("expected validation errors, got nil")
	}
	var ve config.ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors type, got %T", err)
	}
	if len(ve) < 4 {
		t.Errorf("expected at least 4 validation errors, got %d: %v", len(ve), ve)
	}
}

func TestValidate_ValidDatabaseConfigPassesValidation(t *testing.T) {
	// Arrange
	cfg := validConfig()

	// Act
	err := config.Validate(cfg)

	// Assert
	if err != nil {
		t.Fatalf("expected nil error for valid database config, got %v", err)
	}
}

func TestValidate_MissingDatabaseHostFailsWithNamedError(t *testing.T) {
	// Arrange
	cfg := validConfig()
	cfg.Database.Host = ""

	// Act
	err := config.Validate(cfg)

	// Assert
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !containsField(err, "database.host") {
		t.Errorf("expected error to name database.host, got: %v", err)
	}
}

func TestValidate_InvalidSSLModeFailsValidation(t *testing.T) {
	// Arrange
	cfg := validConfig()
	cfg.Database.SSLMode = "invalid-mode"

	// Act
	err := config.Validate(cfg)

	// Assert
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !containsField(err, "database.ssl_mode") {
		t.Errorf("expected error to name database.ssl_mode, got: %v", err)
	}
}

func TestValidate_MissingDatabasePasswordFailsValidation(t *testing.T) {
	// Arrange
	cfg := validConfig()
	cfg.Database.Password = ""

	// Act
	err := config.Validate(cfg)

	// Assert
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !containsField(err, "database.password") {
		t.Errorf("expected error to name database.password, got: %v", err)
	}
}

// containsField reports whether the error message references the given field name.
func containsField(err error, field string) bool {
	return err != nil && strings.Contains(err.Error(), field)
}

func TestValidate_MissingGoogleClientIDFailsValidation(t *testing.T) {
	cfg := validConfig()
	cfg.Auth.GoogleClientID = ""
	err := config.Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !containsField(err, "auth.google_client_id") {
		t.Errorf("expected error to name auth.google_client_id, got: %v", err)
	}
}

func TestValidate_MissingGoogleClientSecretFailsValidation(t *testing.T) {
	cfg := validConfig()
	cfg.Auth.GoogleClientSecret = ""
	err := config.Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !containsField(err, "auth.google_client_secret") {
		t.Errorf("expected error to name auth.google_client_secret, got: %v", err)
	}
}

func TestValidate_ShortJWTSecretFailsValidation(t *testing.T) {
	cfg := validConfig()
	cfg.Auth.JWTSecret = "tooshort"
	err := config.Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !containsField(err, "auth.jwt_secret") {
		t.Errorf("expected error to name auth.jwt_secret, got: %v", err)
	}
}

func TestValidate_MissingGitHubClientIDFailsValidation(t *testing.T) {
	cfg := validConfig()
	cfg.Auth.GitHubClientID = ""
	err := config.Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !containsField(err, "auth.github_client_id") {
		t.Errorf("expected error to name auth.github_client_id, got: %v", err)
	}
}

func TestValidate_MissingGitHubClientSecretFailsValidation(t *testing.T) {
	cfg := validConfig()
	cfg.Auth.GitHubClientSecret = ""
	err := config.Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !containsField(err, "auth.github_client_secret") {
		t.Errorf("expected error to name auth.github_client_secret, got: %v", err)
	}
}
