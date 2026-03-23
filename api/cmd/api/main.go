// Package main is the entrypoint for the Postulate API server.
// It wires all dependencies and manages the server lifecycle.
// No business logic lives here — only construction and orchestration.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/popatkaran/postulate/api/internal/config"
	"github.com/popatkaran/postulate/api/internal/handler"
	"github.com/popatkaran/postulate/api/internal/health"
	applogger "github.com/popatkaran/postulate/api/internal/logger"
	"github.com/popatkaran/postulate/api/internal/router"
	"github.com/popatkaran/postulate/api/internal/server"
	"github.com/popatkaran/postulate/api/internal/startup"
	"github.com/popatkaran/postulate/api/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
)

func main() {
	// healthcheck mode: used by Docker HEALTHCHECK instruction.
	// Performs a single GET /health and exits 0 on 200, 1 otherwise.
	if len(os.Args) == 2 && os.Args[1] == "-healthcheck" {
		port := os.Getenv("POSTULATE_SERVER_PORT")
		if port == "" {
			port = "8080"
		}
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", port)) //nolint:noctx
		if err != nil || resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Bootstrap logger before config so pre-config errors are still visible.
	bootstrap := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		bootstrap.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	if err := config.Validate(cfg); err != nil {
		bootstrap.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	logger := applogger.New(cfg.Observability, cfg.Server.Environment)
	logger.Info("configuration loaded", "config", config.LogSafe(cfg))

	// Database reachability check — must pass before server accepts requests.
	readyHandler := &handler.ReadyHandler{}
	if err := startup.CheckDatabase(context.Background(), cfg.Database, cfg.Server.Environment, logger); err != nil {
		os.Exit(1)
	}
	readyHandler.SetReady(true)

	// OTel SDK — errors are non-fatal; server continues with no-op providers.
	otelShutdown, err := telemetry.Setup(context.Background(), cfg.Observability)
	if err != nil {
		logger.Warn("OTel setup failed, continuing with no-op providers", "error", err)
		otelShutdown = func(_ context.Context) error { return nil }
	}

	metrics, err := telemetry.NewMetrics(otel.GetMeterProvider())
	if err != nil {
		logger.Warn("metrics setup failed, continuing with no-op metrics", "error", err)
		metrics, _ = telemetry.NewMetrics(noop.NewMeterProvider())
	}

	// Health aggregator.
	aggregator := &health.Aggregator{}
	aggregator.Register(&health.ServerContributor{})

	// Handlers.
	handlers := router.Handlers{
		Health:  handler.NewHealthHandler(aggregator),
		Ready:   readyHandler,
		Live:    &handler.LiveHandler{},
		Version: handler.NewVersionHandler(handler.DefaultBuildInfo(), cfg.Server.Environment),
	}

	r := router.New(logger, metrics, handlers)
	srv := server.New(cfg.Server, r, logger)

	// Signal context — cancelled on SIGTERM or SIGINT.
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)

	// Start server in background.
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Start()
	}()

	// Block until signal or server error.
	select {
	case err := <-serverErr:
		stop()
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server exited unexpectedly", "error", err)
			os.Exit(1)
		}
	case <-sigCtx.Done():
		stop()

		logger.Info("shutdown signal received", "signal", sigCtx.Err())
		readyHandler.SetNotReady()

		timeout := time.Duration(cfg.Server.ShutdownTimeoutSeconds) * time.Second
		logger.Info("drain period started", "timeout_seconds", cfg.Server.ShutdownTimeoutSeconds)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Warn("shutdown timed out — some requests may have been dropped", "error", err)
		} else {
			logger.Info("shutdown complete")
		}

		// Flush and close OTel providers after HTTP server drains.
		otelCtx, otelCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer otelCancel()
		if err := otelShutdown(otelCtx); err != nil {
			logger.Warn("OTel shutdown error", "error", err)
		}
	}
}
