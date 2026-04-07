package router_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/popatkaran/postulate/api/internal/handler"
	"github.com/popatkaran/postulate/api/internal/health"
	"github.com/popatkaran/postulate/api/internal/router"
)

func newTestRouter() http.Handler {
	aggregator := &health.Aggregator{}
	aggregator.Register(&health.ServerContributor{})

	readyH := &handler.ReadyHandler{}
	readyH.SetReady(true)

	h := router.Handlers{
		Health:  handler.NewHealthHandler(aggregator),
		Ready:   readyH,
		Live:    &handler.LiveHandler{},
		Version: handler.NewVersionHandler(handler.DefaultBuildInfo(), "test"),
	}
	return router.New(slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, []byte("test-jwt-secret-that-is-32-bytes!"), h)
}

func TestUnregisteredRoute_Returns404ProblemJSON(t *testing.T) {
	// Arrange
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rec := httptest.NewRecorder()

	// Act
	r.ServeHTTP(rec, req)

	// Assert
	assertProblemResponse(t, rec, http.StatusNotFound)
}

func TestUnsupportedMethod_Returns405ProblemJSON(t *testing.T) {
	// Arrange — minimal chi router with the same error handlers; no subrouter conflict.
	r := chi.NewRouter()
	router.ApplyErrorHandlers(r)
	r.Get("/probe", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodDelete, "/probe", nil)
	rec := httptest.NewRecorder()

	// Act
	r.ServeHTTP(rec, req)

	// Assert
	assertProblemResponse(t, rec, http.StatusMethodNotAllowed)
}

func assertProblemResponse(t *testing.T, rec *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()
	if rec.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("expected Content-Type application/problem+json, got %s", ct)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["status"] != float64(expectedStatus) {
		t.Errorf("expected status %d in body, got %v", expectedStatus, body["status"])
	}
}
