package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/popatkaran/postulate/api/internal/middleware"
	"github.com/popatkaran/postulate/api/internal/telemetry"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// newBufLogger captures slog JSON output into a bytes.Buffer.
func newBufLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// newTestMetrics builds a real *telemetry.Metrics backed by a ManualReader.
func newTestMetrics(t *testing.T) (*telemetry.Metrics, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m, err := telemetry.NewMetrics(mp)
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}
	return m, reader
}

func TestAccessLog_ProducesLogEntryWithRequiredFields(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	logger := newBufLogger(&buf)
	m, _ := newTestMetrics(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	chain := middleware.RequestID(middleware.AccessLog(logger, m)(handler))

	req := httptest.NewRequest(http.MethodGet, "/v1/version", nil)
	rec := httptest.NewRecorder()

	// Act
	chain.ServeHTTP(rec, req)

	// Assert
	var entry map[string]any
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("expected a log entry, got: %v\nraw: %s", err, buf.String())
	}
	for _, field := range []string{"requestId", "method", "path", "status", "duration_ms", "response_bytes"} {
		if _, ok := entry[field]; !ok {
			t.Errorf("expected field %q in access log entry", field)
		}
	}
	if entry["msg"] != "request completed" {
		t.Errorf("expected message 'request completed', got %v", entry["msg"])
	}
}

func TestAccessLog_HealthPathProducesNoLogEntry(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	logger := newBufLogger(&buf)
	m, _ := newTestMetrics(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	chain := middleware.AccessLog(logger, m)(handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Act
	chain.ServeHTTP(rec, req)

	// Assert
	if buf.Len() > 0 {
		t.Errorf("expected no log entry for /health, got: %s", buf.String())
	}
}

func TestAccessLog_LogEntryContainsCorrectMethodPathStatus(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	logger := newBufLogger(&buf)
	m, _ := newTestMetrics(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	chain := middleware.RequestID(middleware.AccessLog(logger, m)(handler))

	req := httptest.NewRequest(http.MethodPost, "/v1/projects", nil)
	rec := httptest.NewRecorder()

	// Act
	chain.ServeHTTP(rec, req)

	// Assert
	var entry map[string]any
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("invalid log JSON: %v", err)
	}
	if entry["method"] != http.MethodPost {
		t.Errorf("expected method POST, got %v", entry["method"])
	}
	if entry["path"] != "/v1/projects" {
		t.Errorf("expected path /v1/projects, got %v", entry["path"])
	}
	if entry["status"] != float64(http.StatusCreated) {
		t.Errorf("expected status 201, got %v", entry["status"])
	}
}

func TestAccessLog_CustomExcludedPathProducesNoLogEntry(t *testing.T) {
	var buf bytes.Buffer
	logger := newBufLogger(&buf)
	m, _ := newTestMetrics(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	chain := middleware.AccessLog(logger, m, "/custom/excluded")(handler)

	req := httptest.NewRequest(http.MethodGet, "/custom/excluded", nil)
	chain.ServeHTTP(httptest.NewRecorder(), req)

	if buf.Len() > 0 {
		t.Errorf("expected no log entry for custom excluded path, got: %s", buf.String())
	}
}

func TestAccessLog_NonExcludedPathProducesLogEntry(t *testing.T) {
	var buf bytes.Buffer
	logger := newBufLogger(&buf)
	m, _ := newTestMetrics(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	chain := middleware.AccessLog(logger, m, "/custom/excluded")(handler)

	req := httptest.NewRequest(http.MethodGet, "/v1/other", nil)
	chain.ServeHTTP(httptest.NewRecorder(), req)

	if buf.Len() == 0 {
		t.Error("expected a log entry for non-excluded path")
	}
}

func TestAccessLog_DefaultExclusionsAppliedWhenNilExcludedPaths(t *testing.T) {
	for _, path := range []string{"/health", "/health/ready", "/health/live"} {
		t.Run(path, func(t *testing.T) {
			var buf bytes.Buffer
			logger := newBufLogger(&buf)
			m, _ := newTestMetrics(t)
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			chain := middleware.AccessLog(logger, m)(handler)
			chain.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
			if buf.Len() > 0 {
				t.Errorf("expected no log entry for default excluded path %s", path)
			}
		})
	}
}

func TestAccessLog_MetricsRecordRequestCalled(t *testing.T) {
	var buf bytes.Buffer
	logger := newBufLogger(&buf)
	m, reader := newTestMetrics(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	chain := middleware.AccessLog(logger, m)(handler)
	chain.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/test", nil))

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, met := range sm.Metrics {
			if met.Name == "http.server.request.duration" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected http.server.request.duration metric to be recorded")
	}
}

func TestAccessLog_ResponseBytesMatchBodySize(t *testing.T) {
	var buf bytes.Buffer
	logger := newBufLogger(&buf)
	m, _ := newTestMetrics(t)
	body := []byte("hello world") // 11 bytes
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
	chain := middleware.RequestID(middleware.AccessLog(logger, m)(handler))
	chain.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/test", nil))

	var entry map[string]any
	if err := json.NewDecoder(&buf).Decode(&entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if entry["response_bytes"] != float64(len(body)) {
		t.Errorf("expected response_bytes=%d, got %v", len(body), entry["response_bytes"])
	}
}

func TestRequestIDFromContext_EmptyWhenNotSet(t *testing.T) {
	id := middleware.RequestIDFromContext(context.Background())
	if id != "" {
		t.Errorf("expected empty string, got %q", id)
	}
}
