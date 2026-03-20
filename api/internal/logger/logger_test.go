package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/popatkaran/postulate/api/internal/config"
	applogger "github.com/popatkaran/postulate/api/internal/logger"
)

func defaultCfg() config.ObservabilityConfig {
	return config.ObservabilityConfig{
		ServiceID:  "test-api",
		InstanceID: "test-host",
		LogLevel:   "debug",
	}
}

// captureLog builds a buffer-backed logger, calls fn, and decodes the first log entry.
func captureLog(t *testing.T, cfg config.ObservabilityConfig, fn func(*slog.Logger)) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := applogger.NewWithHandler(inner, cfg)
	fn(logger)

	var entry map[string]any
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v\nraw: %s", err, buf.String())
	}
	return entry
}

func TestLogger_OutputIsValidJSON(t *testing.T) {
	entry := captureLog(t, defaultCfg(), func(l *slog.Logger) {
		l.Info("hello")
	})
	if entry == nil {
		t.Fatal("expected non-nil JSON entry")
	}
}
func TestLogger_RequiredFieldsPresent(t *testing.T) {
	entry := captureLog(t, defaultCfg(), func(l *slog.Logger) {
		l.Info("test message")
	})

	// "timestamp" is the Postulate standard field name (not slog's default "time").
	required := []string{"timestamp", "level", "msg", "traceId", "spanId", "serviceId", "instanceId"}
	for _, field := range required {
		if _, ok := entry[field]; !ok {
			t.Errorf("expected required field %q in log entry", field)
		}
	}
}

func TestLogger_TraceIDAndSpanIDEmptyWhenNoSpan(t *testing.T) {
	entry := captureLog(t, defaultCfg(), func(l *slog.Logger) {
		l.InfoContext(context.Background(), "no span")
	})

	if entry["traceId"] != "" {
		t.Errorf("expected empty traceId, got %v", entry["traceId"])
	}
	if entry["spanId"] != "" {
		t.Errorf("expected empty spanId, got %v", entry["spanId"])
	}
}

func TestLogger_TraceIDAndSpanIDPopulatedFromOTelContext(t *testing.T) {
	traceID, _ := trace.TraceIDFromHex("0af7651916cd43dd8448eb211c80319c")
	spanID, _ := trace.SpanIDFromHex("b7ad6b7169203331")
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

	entry := captureLog(t, defaultCfg(), func(l *slog.Logger) {
		l.InfoContext(ctx, "with span")
	})

	if entry["traceId"] != traceID.String() {
		t.Errorf("expected traceId %s, got %v", traceID.String(), entry["traceId"])
	}
	if entry["spanId"] != spanID.String() {
		t.Errorf("expected spanId %s, got %v", spanID.String(), entry["spanId"])
	}
}

func TestNew_ProductionEnvironmentEmitsJSON(t *testing.T) {
	var buf bytes.Buffer
	cfg := defaultCfg()
	logger := applogger.NewWithWriter(cfg, "production", &buf)
	logger.Info("prod log")

	var entry map[string]any
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("production logger must emit JSON: %v\nraw: %s", err, buf.String())
	}
	if entry["msg"] != "prod log" {
		t.Errorf("unexpected msg: %v", entry["msg"])
	}
}

func TestNew_StagingEnvironmentEmitsJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := applogger.NewWithWriter(defaultCfg(), "staging", &buf)
	logger.Info("staging log")

	var entry map[string]any
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("staging logger must emit JSON: %v\nraw: %s", err, buf.String())
	}
	if entry["msg"] != "staging log" {
		t.Errorf("unexpected msg: %v", entry["msg"])
	}
}

func TestNew_DevelopmentEnvironmentEmitsText(t *testing.T) {
	var buf bytes.Buffer
	logger := applogger.NewWithWriter(defaultCfg(), "development", &buf)
	logger.Info("dev log")

	if buf.Len() == 0 {
		t.Fatal("expected text output for development environment")
	}
	// Text handler output is not JSON — decoding should fail.
	var entry map[string]any
	if err := json.NewDecoder(&buf).Decode(&entry); err == nil {
		t.Error("expected non-JSON (text) output for development environment")
	}
}

func TestNew_InstanceIDFallsBackToHostname(t *testing.T) {
	var buf bytes.Buffer
	cfg := config.ObservabilityConfig{ServiceID: "svc", LogLevel: "debug"} // no InstanceID
	logger := applogger.NewWithWriter(cfg, "production", &buf)
	logger.Info("hostname test")

	var entry map[string]any
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if entry["instanceId"] == "" {
		t.Error("expected instanceId to be populated from hostname when not configured")
	}
}

func TestParseLevel_AllBranches(t *testing.T) {
	cases := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"info", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"verbose", slog.LevelInfo},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			var buf bytes.Buffer
			cfg := config.ObservabilityConfig{ServiceID: "svc", LogLevel: tc.input}
			logger := applogger.NewWithWriter(cfg, "production", &buf)

			// Log at the expected level — if the level is filtered the buffer stays empty.
			switch tc.expected {
			case slog.LevelDebug:
				logger.Debug("msg")
			case slog.LevelWarn:
				logger.Warn("msg")
			case slog.LevelError:
				logger.Error("msg")
			default:
				logger.Info("msg")
			}

			if buf.Len() == 0 {
				t.Errorf("parseLevel(%q): expected log output at level %v", tc.input, tc.expected)
			}
		})
	}
}
