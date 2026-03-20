// Package handler contains HTTP handlers for the Postulate API.
package handler

import (
	"encoding/json"
	"net/http"
	"sync/atomic"

	"github.com/popatkaran/postulate/api/internal/health"
)

// HealthHandler serves GET /health using the registered health aggregator.
type HealthHandler struct {
	aggregator *health.Aggregator
}

// NewHealthHandler constructs a HealthHandler.
func NewHealthHandler(aggregator *health.Aggregator) *HealthHandler {
	return &HealthHandler{aggregator: aggregator}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	result := h.aggregator.Check(r.Context())

	status := http.StatusOK
	if result.Status != health.StatusHealthy {
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

// ReadyHandler serves GET /ready.
// It returns 200 when the server is ready and 503 during startup or shutdown.
type ReadyHandler struct {
	ready atomic.Bool
}

// SetReady marks the server as ready (true) or not ready (false).
func (h *ReadyHandler) SetReady(v bool) { h.ready.Store(v) }

// SetNotReady marks the server as not ready. Called at the start of graceful shutdown.
func (h *ReadyHandler) SetNotReady() { h.ready.Store(false) }

func (h *ReadyHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.ready.Load() {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}` + "\n"))
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`{"status":"not_ready"}` + "\n"))
}

// LiveHandler serves GET /live.
// It returns 200 unconditionally — if the process can serve this request, it is alive.
type LiveHandler struct{}

func (h *LiveHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"alive"}` + "\n"))
}
