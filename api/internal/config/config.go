// Package config defines configuration structures and loading logic for the Postulate API server.
package config

// Config is the root configuration structure for the Postulate API server.
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Observability ObservabilityConfig `yaml:"observability"`
	Database      DatabaseConfig      `yaml:"database"`
	Auth          AuthConfig          `yaml:"auth"`
}

// AuthConfig holds OAuth provider and JWT configuration.
type AuthConfig struct {
	GoogleClientID     string `yaml:"google_client_id"`
	GoogleClientSecret string `yaml:"google_client_secret"`
	GitHubClientID     string `yaml:"github_client_id"`
	GitHubClientSecret string `yaml:"github_client_secret"`
	// BaseURL is the public base URL of this API, used to construct OAuth callback URLs.
	// Defaults to http://localhost:8080 in development.
	BaseURL   string `yaml:"base_url"`
	JWTSecret string `yaml:"jwt_secret"`
	// BootstrapAdminEmail designates the email address that receives platform_admin on
	// first login. Optional — startup must not fail if absent.
	// In production, if absent, a WARN is emitted and the dev fallback is disabled.
	BootstrapAdminEmail string `yaml:"bootstrap_admin_email"`
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
