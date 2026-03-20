package problem_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/popatkaran/postulate/api/internal/problem"
)

func TestWrite_SetsContentTypeProblemJSON(t *testing.T) {
	// Arrange
	p := problem.New(problem.TypeNotFound, "Not Found", http.StatusNotFound, "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/things/1", nil)

	// Act
	problem.Write(rec, req, p)

	// Assert
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("expected application/problem+json, got %s", ct)
	}
}

func TestWrite_SetsHTTPStatusFromProblemStatus(t *testing.T) {
	// Arrange
	p := problem.New(problem.TypeNotFound, "Not Found", http.StatusNotFound, "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/things/1", nil)

	// Act
	problem.Write(rec, req, p)

	// Assert
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestWrite_PopulatesInstanceFromRequestPath(t *testing.T) {
	// Arrange — instance left empty; should be filled from request path.
	p := problem.New(problem.TypeNotFound, "Not Found", http.StatusNotFound, "", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/things/42", nil)

	// Act
	problem.Write(rec, req, p)

	// Assert
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["instance"] != "/v1/things/42" {
		t.Errorf("expected instance /v1/things/42, got %v", body["instance"])
	}
}

func TestWriteValidation_IncludesErrorsArray(t *testing.T) {
	// Arrange
	p := problem.NewValidation("Invalid fields.", "/v1/generate", []problem.FieldError{
		{Field: "service_name", Message: "must not be empty"},
		{Field: "language", Message: "must be one of: go, python"},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/generate", nil)

	// Act
	problem.WriteValidation(rec, req, p)

	// Assert
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	errs, ok := body["errors"].([]any)
	if !ok || len(errs) != 2 {
		t.Errorf("expected errors array with 2 items, got %v", body["errors"])
	}
}

func TestWrite_500ResponseBodyHasNoStackTrace(t *testing.T) {
	// Arrange — internal error must use a generic detail, not a raw error string.
	p := problem.New(
		problem.TypeInternalServerError,
		"Internal Server Error",
		http.StatusInternalServerError,
		"An unexpected error occurred. Please try again later.",
		"",
	)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/generate", nil)

	// Act
	problem.Write(rec, req, p)

	// Assert
	body := rec.Body.String()
	for _, leak := range []string{"goroutine", "runtime/debug", "panic", ".go:"} {
		if strings.Contains(body, leak) {
			t.Errorf("response body must not contain %q (stack trace leak)", leak)
		}
	}
}

func TestWrite_ResponseBodyContainsAllRequiredFields(t *testing.T) {
	// Arrange
	p := problem.New(problem.TypeValidationFailed, "Validation Failed", 422, "Bad input.", "/v1/x")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/x", nil)

	// Act
	problem.Write(rec, req, p)

	// Assert
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, field := range []string{"type", "title", "status", "detail", "instance"} {
		if _, ok := body[field]; !ok {
			t.Errorf("expected field %q in response body", field)
		}
	}
	if body["status"] != float64(422) {
		t.Errorf("expected status 422, got %v", body["status"])
	}
}
