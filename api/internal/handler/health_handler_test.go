package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/popatkaran/postulate/api/internal/handler"
	"github.com/popatkaran/postulate/api/internal/health"
)

// unhealthyContributor is a test double that always reports unhealthy.
type unhealthyContributor struct{}

func (u *unhealthyContributor) Name() string { return "failing-dep" }
func (u *unhealthyContributor) Check(_ context.Context) health.CheckResult {
	return health.CheckResult{Status: health.StatusUnhealthy, Message: "connection refused"}
}

func newHealthyAggregator() *health.Aggregator {
	a := &health.Aggregator{}
	a.Register(&health.ServerContributor{})
	return a
}

func newUnhealthyAggregator() *health.Aggregator {
	a := &health.Aggregator{}
	a.Register(&health.ServerContributor{})
	a.Register(&unhealthyContributor{})
	return a
}

func TestHealthHandler_AllHealthy_Returns200(t *testing.T) {
	// Arrange
	h := handler.NewHealthHandler(newHealthyAggregator())
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Act
	h.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHealthHandler_AnyUnhealthy_Returns503(t *testing.T) {
	// Arrange
	h := handler.NewHealthHandler(newUnhealthyAggregator())
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Act
	h.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHealthHandler_ResponseBodyMatchesSchema(t *testing.T) {
	// Arrange
	h := handler.NewHealthHandler(newUnhealthyAggregator())
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Act
	h.ServeHTTP(rec, req)

	// Assert
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["status"] == nil {
		t.Error("expected 'status' field in response body")
	}
	if body["timestamp"] == nil {
		t.Error("expected 'timestamp' field in response body")
	}
	checks, ok := body["checks"].(map[string]any)
	if !ok {
		t.Fatal("expected 'checks' to be an object")
	}
	if _, ok := checks["server"]; !ok {
		t.Error("expected 'server' key in checks")
	}
	if _, ok := checks["failing-dep"]; !ok {
		t.Error("expected 'failing-dep' key in checks")
	}
}

func TestReadyHandler_WhenReady_Returns200(t *testing.T) {
	// Arrange
	h := &handler.ReadyHandler{}
	h.SetReady(true)
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	// Act
	h.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestReadyHandler_WhenNotReady_Returns503(t *testing.T) {
	// Arrange
	h := &handler.ReadyHandler{}
	h.SetReady(false)
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	// Act
	h.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestLiveHandler_AlwaysReturns200(t *testing.T) {
	// Arrange
	h := &handler.LiveHandler{}
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rec := httptest.NewRecorder()

	// Act
	h.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
