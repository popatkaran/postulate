package handler

import (
	"encoding/json"
	"net/http"
	"runtime"
)

// These variables are injected at build time via -ldflags.
// Defaults are used when building without the Makefile (e.g. go test).
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

// BuildInfo holds build-time metadata injected via Go linker flags.
type BuildInfo struct {
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	BuildTime   string `json:"build_time"`
	GoVersion   string `json:"go_version"`
	Environment string `json:"environment"`
}

// DefaultBuildInfo returns a BuildInfo populated from the linker-injected variables.
func DefaultBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   version,
		Commit:    commit,
		BuildTime: buildTime,
	}
}

// VersionHandler serves GET /v1/version.
type VersionHandler struct {
	info BuildInfo
}

// NewVersionHandler constructs a VersionHandler with the given build info and environment.
// GoVersion is always populated from the running runtime.
func NewVersionHandler(info BuildInfo, environment string) *VersionHandler {
	info.GoVersion = runtime.Version()
	info.Environment = environment
	return &VersionHandler{info: info}
}

func (h *VersionHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(h.info)
}
