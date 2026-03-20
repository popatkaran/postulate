package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	envConfigFile = "POSTULATE_CONFIG_FILE"
	defaultConfig = "./config.yaml"
	envPrefix     = "POSTULATE_"
)

// Load reads configuration from a YAML file and applies environment variable overrides.
// The config file path is taken from POSTULATE_CONFIG_FILE; defaults to ./config.yaml.
// If the default path does not exist the file is skipped; an explicit path that does
// not exist is an error.
func Load() (*Config, error) {
	path, explicit := configFilePath()

	cfg := &Config{}

	if err := loadYAML(path, explicit, cfg); err != nil {
		return nil, err
	}

	applyEnvOverrides(cfg)

	return cfg, nil
}

// configFilePath returns the config file path and whether it was explicitly set.
func configFilePath() (string, bool) {
	if v := os.Getenv(envConfigFile); v != "" {
		return v, true
	}
	return defaultConfig, false
}

// loadYAML reads and parses the YAML file into cfg.
// If the file is the default path and does not exist, it is silently skipped.
func loadYAML(path string, explicit bool, cfg *Config) error {
	// G304: path is sourced from a trusted environment variable or the hardcoded
	// default constant — not from user-supplied request input.
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !explicit {
			return nil // default path missing is acceptable
		}
		return fmt.Errorf("reading config file %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parsing config file %q: %w", path, err)
	}
	return nil
}

// applyEnvOverrides maps POSTULATE_* environment variables onto cfg fields.
// Convention: POSTULATE_SERVER_PORT → cfg.Server.Port
func applyEnvOverrides(cfg *Config) {
	setInt(&cfg.Server.Port, "POSTULATE_SERVER_PORT")
	setString(&cfg.Server.Environment, "POSTULATE_SERVER_ENVIRONMENT")
	setInt(&cfg.Server.ShutdownTimeoutSeconds, "POSTULATE_SERVER_SHUTDOWN_TIMEOUT_SECONDS")
	setString(&cfg.Observability.ServiceID, "POSTULATE_OBSERVABILITY_SERVICE_ID")
	setString(&cfg.Observability.InstanceID, "POSTULATE_OBSERVABILITY_INSTANCE_ID")
	setString(&cfg.Observability.OTLPEndpoint, "POSTULATE_OBSERVABILITY_OTLP_ENDPOINT")
	setString(&cfg.Observability.LogLevel, "POSTULATE_OBSERVABILITY_LOG_LEVEL")
}

func setString(field *string, key string) {
	if v := os.Getenv(key); v != "" {
		*field = v
	}
}

func setInt(field *int, key string) {
	v := os.Getenv(key)
	if v == "" {
		return
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err == nil {
		*field = n
	}
}
