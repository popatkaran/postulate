package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/popatkaran/postulate/api/internal/middleware"
)

func TestRequestID_GeneratesULIDWhenHeaderAbsent(t *testing.T) {
	// Arrange
	var capturedID string
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedID = middleware.RequestIDFromContext(r.Context())
	})

	// Act
	middleware.RequestID(handler).ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/", nil),
	)

	// Assert
	if capturedID == "" {
		t.Error("expected a generated request ID, got empty string")
	}
}

func TestRequestID_PropagatesValidHeaderValue(t *testing.T) {
	// Arrange — provide a valid ULID (Crockford base32, 26 chars) in the incoming header.
	const existingID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	var capturedID string
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedID = middleware.RequestIDFromContext(r.Context())
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", existingID)

	// Act
	middleware.RequestID(handler).ServeHTTP(httptest.NewRecorder(), req)

	// Assert
	if capturedID != existingID {
		t.Errorf("expected %s, got %s", existingID, capturedID)
	}
}

func TestRequestID_GeneratesNewIDForInvalidHeader(t *testing.T) {
	// Arrange — provide a non-ULID/UUID value.
	var capturedID string
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedID = middleware.RequestIDFromContext(r.Context())
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "not-a-valid-id!!")

	// Act
	middleware.RequestID(handler).ServeHTTP(httptest.NewRecorder(), req)

	// Assert
	if capturedID == "not-a-valid-id!!" {
		t.Error("expected invalid header to be replaced with a generated ID")
	}
	if capturedID == "" {
		t.Error("expected a generated ID, got empty string")
	}
}

func TestRequestID_SetsResponseHeader(t *testing.T) {
	// Arrange
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	// Act
	middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	// Assert
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID response header to be set")
	}
}

func TestRequestID_IDRetrievableFromContext(t *testing.T) {
	// Arrange
	var contextID, headerID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextID = middleware.RequestIDFromContext(r.Context())
		headerID = w.Header().Get("X-Request-ID")
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	// Act
	middleware.RequestID(handler).ServeHTTP(rec, req)

	// Assert
	if contextID == "" {
		t.Error("expected request ID in context")
	}
	if contextID != headerID {
		t.Errorf("context ID %q does not match response header %q", contextID, headerID)
	}
}
