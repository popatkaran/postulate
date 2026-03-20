// Package problem implements RFC 7807 problem details for HTTP APIs.
package problem

// Problem is an RFC 7807 problem details response body.
type Problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
	Instance  string `json:"instance"`
	RequestID string `json:"request_id,omitempty"`
}

// ValidationProblem extends Problem with field-level validation errors.
type ValidationProblem struct {
	Problem
	Errors []FieldError `json:"errors"`
}

// FieldError describes a single field-level validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
