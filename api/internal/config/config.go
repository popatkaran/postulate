// Package config defines configuration structures and loading logic for the Postulate API server.
package config

// Config is the root configuration structure for the Postulate API server.
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Observability ObservabilityConfig `yaml:"observability"`
	Database      DatabaseConfig      `yaml:"database"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port                   int    `yaml:"port"`
	Environment            string `yaml:"environment"`
	ShutdownTimeoutSeconds int    `yaml:"shutdown_timeout_seconds"`
}

// ObservabilityConfig holds observability and telemetry configuration.
type ObservabilityConfig struct {
	ServiceID    string `yaml:"service_id"`
	InstanceID   string `yaml:"instance_id"`
	OTLPEndpoint string `yaml:"otlp_endpoint"`
	LogLevel     string `yaml:"log_level"`
}

// DatabaseConfig holds PostgreSQL connection configuration.
type DatabaseConfig struct {
	Host                   string `yaml:"host"`
	Port                   int    `yaml:"port"`
	Name                   string `yaml:"name"`
	User                   string `yaml:"user"`
	Password               string `yaml:"password"`
	SSLMode                string `yaml:"ssl_mode"`
	MaxOpenConns           int    `yaml:"max_open_conns"`
	MaxIdleConns           int    `yaml:"max_idle_conns"`
	ConnMaxLifetimeSeconds int    `yaml:"conn_max_lifetime_seconds"`
}
