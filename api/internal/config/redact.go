package config

// SECURITY: LogSafe returns a representation of the configuration safe for
// logging. Any field designated sensitive must be added to the sensitiveFields
// slice below and must appear in the unit tests in redact_test.go.
// Current sensitive fields: none (Epic 01).
// Epic 02 will add: database.password, database.dsn.
func LogSafe(cfg *Config) map[string]any {
	return map[string]any{
		"server.port":                     cfg.Server.Port,
		"server.environment":              cfg.Server.Environment,
		"server.shutdown_timeout_seconds": cfg.Server.ShutdownTimeoutSeconds,
		"observability.service_id":        cfg.Observability.ServiceID,
		"observability.instance_id":       cfg.Observability.InstanceID,
		"observability.otlp_endpoint":     cfg.Observability.OTLPEndpoint,
		"observability.log_level":         cfg.Observability.LogLevel,
	}
}
