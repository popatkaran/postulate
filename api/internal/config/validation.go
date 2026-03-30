package config

import (
	"fmt"
	"strings"
)

// validEnvironments lists the accepted values for server.environment.
var validEnvironments = map[string]bool{
	"development": true,
	"staging":     true,
	"production":  true,
}

// validSSLModes lists the accepted values for database.ssl_mode.
var validSSLModes = map[string]bool{
	"disable":     true,
	"require":     true,
	"verify-full": true,
}

// ValidationErrors is a collection of field-level validation failures.
// It implements the error interface so all errors are returned together.
type ValidationErrors []string

func (ve ValidationErrors) Error() string {
	return "configuration validation failed:\n  " + strings.Join(ve, "\n  ")
}

// Validate checks all required fields and value constraints on cfg.
// All violations are collected and returned as a single ValidationErrors error.
// Returns nil when cfg is valid.
func Validate(cfg *Config) error {
	var errs ValidationErrors

	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		errs = append(errs, fmt.Sprintf("server.port: must be between 1 and 65535 (got %d)", cfg.Server.Port))
	}

	if !validEnvironments[cfg.Server.Environment] {
		errs = append(errs, fmt.Sprintf(
			"server.environment: must be one of development|staging|production (got %q)",
			cfg.Server.Environment,
		))
	}

	if cfg.Server.ShutdownTimeoutSeconds < 1 {
		errs = append(errs, fmt.Sprintf(
			"server.shutdown_timeout_seconds: must be a positive integer (got %d)",
			cfg.Server.ShutdownTimeoutSeconds,
		))
	}

	if cfg.Observability.ServiceID == "" {
		errs = append(errs, "observability.service_id: must be a non-empty string")
	}

	if cfg.Database.Host == "" {
		errs = append(errs, "database.host: must be a non-empty string")
	}

	if cfg.Database.Port < 1 || cfg.Database.Port > 65535 {
		errs = append(errs, fmt.Sprintf("database.port: must be between 1 and 65535 (got %d)", cfg.Database.Port))
	}

	if cfg.Database.Name == "" {
		errs = append(errs, "database.name: must be a non-empty string")
	}

	if cfg.Database.User == "" {
		errs = append(errs, "database.user: must be a non-empty string")
	}

	if cfg.Database.Password == "" {
		errs = append(errs, "database.password: must be a non-empty string")
	}

	if !validSSLModes[cfg.Database.SSLMode] {
		errs = append(errs, fmt.Sprintf(
			"database.ssl_mode: must be one of disable|require|verify-full (got %q)",
			cfg.Database.SSLMode,
		))
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}
