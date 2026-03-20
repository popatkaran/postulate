package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/popatkaran/postulate/api/internal/handler"
)

func TestVersionHandler_Returns200WithBuildInfoFields(t *testing.T) {
	// Arrange
	info := handler.BuildInfo{
		Version:   "1.2.3",
		Commit:    "abc1234",
		BuildTime: "2026-03-19T09:00:00Z",
	}
	h := handler.NewVersionHandler(info, "production")
	req := httptest.NewRequest(http.MethodGet, "/v1/version", nil)
	rec := httptest.NewRecorder()

	// Act
	h.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	for _, field := range []string{"version", "commit", "build_time", "go_version", "environment"} {
		if body[field] == nil {
			t.Errorf("expected field %q in response body", field)
		}
	}
	if body["version"] != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %v", body["version"])
	}
	if body["environment"] != "production" {
		t.Errorf("expected environment production, got %v", body["environment"])
	}
}
