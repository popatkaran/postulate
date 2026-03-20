package problem

import (
	"encoding/json"
	"net/http"

	"github.com/popatkaran/postulate/api/internal/middleware"
)

// Write serialises p as application/problem+json and writes it to w.
// p.Instance is populated from r.URL.Path when not already set.
// p.RequestID is populated from the request context when not already set.
func Write(w http.ResponseWriter, r *http.Request, p *Problem) {
	if p.Instance == "" {
		p.Instance = r.URL.Path
	}
	if p.RequestID == "" {
		p.RequestID = middleware.RequestIDFromContext(r.Context())
	}
	writeJSON(w, p.Status, p)
}

// WriteValidation serialises p as application/problem+json and writes it to w.
func WriteValidation(w http.ResponseWriter, r *http.Request, p *ValidationProblem) {
	if p.Instance == "" {
		p.Instance = r.URL.Path
	}
	if p.RequestID == "" {
		p.RequestID = middleware.RequestIDFromContext(r.Context())
	}
	writeJSON(w, p.Status, p)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
