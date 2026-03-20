package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/popatkaran/postulate/api/internal/telemetry"
)

// defaultExcludedPaths lists paths that produce no access log entry.
var defaultExcludedPaths = []string{"/health", "/health/ready", "/health/live"}

// AccessLog returns middleware that logs one structured entry per completed request
// and records HTTP server metrics via m (may be nil — metrics are skipped when nil).
// Paths in excludedPaths (defaults applied when nil) produce no log entry.
func AccessLog(logger *slog.Logger, m *telemetry.Metrics, excludedPaths ...string) func(http.Handler) http.Handler {
	excluded := defaultExcludedPaths
	if len(excludedPaths) > 0 {
		excluded = excludedPaths
	}

	isExcluded := func(path string) bool {
		for _, p := range excluded {
			if strings.EqualFold(path, p) {
				return true
			}
		}
		return false
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isExcluded(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			route := r.URL.Path
			if rctx := chi.RouteContext(r.Context()); rctx != nil {
				if p := rctx.RoutePattern(); p != "" {
					route = p
				}
			}

			if m != nil {
				m.IncActive(r.Context(), r.Method, route)
			}

			start := time.Now()
			rec := newStatusRecorder(w)
			next.ServeHTTP(rec, r)

			durationMs := float64(time.Since(start).Milliseconds())

			if m != nil {
				m.RecordRequest(r.Context(), r.Method, route, rec.status, durationMs)
			}

			logger.InfoContext(r.Context(), "request completed",
				"requestId", RequestIDFromContext(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", durationMs,
				"response_bytes", rec.bytes,
			)
		})
	}
}
