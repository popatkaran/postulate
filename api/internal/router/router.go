// Package router constructs the Postulate API HTTP router.
package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/popatkaran/postulate/api/internal/handler"
	"github.com/popatkaran/postulate/api/internal/middleware"
	"github.com/popatkaran/postulate/api/internal/problem"
	"github.com/popatkaran/postulate/api/internal/telemetry"
)

// Handlers groups the HTTP handlers required by the router.
type Handlers struct {
	Health  *handler.HealthHandler
	Ready   *handler.ReadyHandler
	Live    *handler.LiveHandler
	Version *handler.VersionHandler
}

// New constructs and returns a configured chi.Router.
// Middleware order: Tracing → RequestID → AccessLog (with metrics).
func New(logger *slog.Logger, metrics *telemetry.Metrics, h Handlers) chi.Router {
	r := chi.NewRouter()

	// Tracing must be first so the span context is available to all downstream middleware.
	r.Use(middleware.Tracing)
	r.Use(middleware.RequestID)
	r.Use(middleware.AccessLog(logger, metrics))

	ApplyErrorHandlers(r)

	methodNotAllowed := problemHandler(http.StatusMethodNotAllowed, problem.TypeMethodNotAllowed, "Method Not Allowed")

	// Operational endpoints — unauthenticated.
	r.Route("/health", func(r chi.Router) {
		r.MethodNotAllowed(methodNotAllowed)
		r.Get("/", h.Health.ServeHTTP)
		r.Get("/ready", h.Ready.ServeHTTP)
		r.Get("/live", h.Live.ServeHTTP)
	})

	// Authentication endpoints — unauthenticated.
	r.Route("/v1/auth", func(r chi.Router) {
		r.MethodNotAllowed(methodNotAllowed)
	})

	// Versioned API endpoints — authenticated (middleware added in later stories).
	r.Route("/v1", func(r chi.Router) {
		r.MethodNotAllowed(methodNotAllowed)
		r.Get("/version", h.Version.ServeHTTP)
	})

	logger.Info("router initialised")

	return r
}

// ApplyErrorHandlers registers RFC 7807-compliant 404 and 405 handlers on r.
// Exposed so tests can wire the same handlers onto a minimal router.
func ApplyErrorHandlers(r chi.Router) {
	r.NotFound(problemHandler(http.StatusNotFound, problem.TypeNotFound, "Not Found"))
	r.MethodNotAllowed(problemHandler(http.StatusMethodNotAllowed, problem.TypeMethodNotAllowed, "Method Not Allowed"))
}

// problemHandler returns an http.HandlerFunc that writes an RFC 7807 problem response.
func problemHandler(status int, problemType, title string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := problem.New(problemType, title, status, "", r.URL.Path)
		problem.Write(w, r, p)
	}
}
