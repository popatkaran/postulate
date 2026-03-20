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

// containsField reports whether the error message references the given field name.
func containsField(err error, field string) bool {
	return err != nil && strings.Contains(err.Error(), field)
}
