package config_test

import (
	"testing"

	"github.com/popatkaran/postulate/api/internal/config"
)

func fullConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:                   8080,
			Environment:            "production",
			ShutdownTimeoutSeconds: 30,
		},
		Observability: config.ObservabilityConfig{
			ServiceID:    "postulate-api",
			InstanceID:   "host-1",
			OTLPEndpoint: "otel:4317",
			LogLevel:     "info",
		},
	}
}

var expectedKeys = []string{
	"server.port",
	"server.environment",
	"server.shutdown_timeout_seconds",
	"observability.service_id",
	"observability.instance_id",
	"observability.otlp_endpoint",
	"observability.log_level",
}

func TestLogSafe_AllKeysPresent(t *testing.T) {
	m := config.LogSafe(fullConfig())
	for _, key := range expectedKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("expected key %q in LogSafe output", key)
		}
	}
}

func TestLogSafe_NoSensitiveFieldsInEpic01_AllValuesUnredacted(t *testing.T) {
	cfg := fullConfig()
	m := config.LogSafe(cfg)

	if m["server.port"] != cfg.Server.Port {
		t.Errorf("server.port: expected %v, got %v", cfg.Server.Port, m["server.port"])
	}
	if m["server.environment"] != cfg.Server.Environment {
		t.Errorf("server.environment: expected %v, got %v", cfg.Server.Environment, m["server.environment"])
	}
	if m["observability.service_id"] != cfg.Observability.ServiceID {
		t.Errorf("observability.service_id: expected %v, got %v", cfg.Observability.ServiceID, m["observability.service_id"])
	}
}

func TestLogSafe_DoesNotMutateOriginalConfig(t *testing.T) {
	cfg := fullConfig()
	original := *cfg

	_ = config.LogSafe(cfg)

	if cfg.Server.Port != original.Server.Port {
		t.Error("LogSafe mutated Server.Port")
	}
	if cfg.Observability.ServiceID != original.Observability.ServiceID {
		t.Error("LogSafe mutated Observability.ServiceID")
	}
}

func TestLogSafe_ServerAndObservabilityTopLevelKeysPresent(t *testing.T) {
	m := config.LogSafe(fullConfig())

	serverKeys := []string{"server.port", "server.environment", "server.shutdown_timeout_seconds"}
	for _, k := range serverKeys {
		if _, ok := m[k]; !ok {
			t.Errorf("expected server key %q in LogSafe output", k)
		}
	}

	obsKeys := []string{"observability.service_id", "observability.instance_id", "observability.otlp_endpoint", "observability.log_level"}
	for _, k := range obsKeys {
		if _, ok := m[k]; !ok {
			t.Errorf("expected observability key %q in LogSafe output", k)
		}
	}
}
