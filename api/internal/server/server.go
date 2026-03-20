// Package server manages the HTTP server lifecycle for the Postulate API.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/popatkaran/postulate/api/internal/config"
)

// Server wraps an http.Server and provides lifecycle management.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

// New constructs a Server from the provided configuration, router, and logger.
// No global state is referenced.
func New(cfg config.ServerConfig, router http.Handler, logger *slog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// Start begins listening for incoming connections and logs the bound address.
// It returns http.ErrServerClosed when the server is shut down gracefully.
func (s *Server) Start() error {
	s.logger.Info("server starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown stops accepting new connections and waits for in-flight requests
// to complete, respecting the provided context deadline.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("server shutting down")
	return s.httpServer.Shutdown(ctx)
}
