package config

// SECURITY: LogSafe returns a representation of the configuration safe for
// logging. Any field designated sensitive must be added here and must appear
// in the unit tests in redact_test.go.
// Sensitive fields:
//   - database.password — never logged; appears as [redacted]
func LogSafe(cfg *Config) map[string]any {
	return map[string]any{
		"server.port":                     cfg.Server.Port,
		"server.environment":              cfg.Server.Environment,
		"server.shutdown_timeout_seconds": cfg.Server.ShutdownTimeoutSeconds,
		"observability.service_id":        cfg.Observability.ServiceID,
		"observability.instance_id":       cfg.Observability.InstanceID,
		"observability.otlp_endpoint":     cfg.Observability.OTLPEndpoint,
		"observability.log_level":         cfg.Observability.LogLevel,
		"database.host":                   cfg.Database.Host,
		"database.port":                   cfg.Database.Port,
		"database.name":                   cfg.Database.Name,
		"database.user":                   cfg.Database.User,
		"database.password":               "[redacted]",
		"database.ssl_mode":               cfg.Database.SSLMode,
	}
}
