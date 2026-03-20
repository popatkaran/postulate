# US-01-06 — RFC 7807 Error Response Format

**Epic:** Epic 01 — API Skeleton, Routing, and Health
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have

---

## 1. Story

As an **API consumer**, I need all error responses from the Postulate API to follow a consistent, documented structure so that the CLI and any other client can handle errors programmatically without parsing arbitrary error shapes.

---

## 2. Background

RFC 7807 defines a standard format for HTTP API error responses using the `application/problem+json` media type. Adopting this standard means error handling in the CLI is predictable — every error, from validation failures to authentication errors to internal server errors, has the same envelope.

This story establishes the error response infrastructure used by every handler across all subsequent Epics. It is foundational. Getting the error shape right here avoids a breaking change to the API contract later.

---

## 3. Acceptance Criteria

1. All error responses from the API use `Content-Type: application/problem+json`.
2. All error responses conform to the following JSON structure:
   ```json
   {
     "type": "https://postulate.dev/errors/validation-failed",
     "title": "Validation Failed",
     "status": 422,
     "detail": "The request body contained one or more invalid fields.",
     "instance": "/v1/generate/interview",
     "request_id": "01HXYZ..."
   }
   ```
3. The `type` field is a URI that uniquely identifies the error type. A catalogue of all defined error types is documented in `api/docs/error-types.md`.
4. The `status` field always matches the HTTP response status code.
5. The `instance` field is the request path from which the error originated.
6. The `request_id` field contains the unique request ID from the `X-Request-ID` header (introduced in US-01-07 — placeholder for now, populated once middleware is in place).
7. For validation errors, an additional `errors` field is present as an array of field-level error objects:
   ```json
   {
     "type": "https://postulate.dev/errors/validation-failed",
     "title": "Validation Failed",
     "status": 422,
     "detail": "The request body contained one or more invalid fields.",
     "instance": "/v1/generate/interview",
     "request_id": "01HXYZ...",
     "errors": [
       { "field": "service_name", "message": "must not be empty" },
       { "field": "language", "message": "must be one of: go, python, java, node" }
     ]
   }
   ```
8. Internal server errors (`500`) must not leak stack traces, internal error messages, or implementation details in the response body. The `detail` field for `500` responses contains a generic message. The full error is logged server-side.
9. A helper function `WriteError` is available to all handlers for writing a problem+json response in one call.
10. Unit tests cover all defined error types and the validation error extension.

---

## 4. Error Type Catalogue

The following error types are defined for this Epic. Further Epics will extend this catalogue.

| Type URI Suffix | HTTP Status | Title | When Used |
|---|---|---|---|
| `/errors/not-found` | 404 | Not Found | Resource does not exist |
| `/errors/method-not-allowed` | 405 | Method Not Allowed | HTTP method not supported on this route |
| `/errors/validation-failed` | 422 | Validation Failed | Request body or parameters failed validation |
| `/errors/internal-server-error` | 500 | Internal Server Error | Unhandled server-side error |

Base URI: `https://postulate.dev`

---

## 5. Tasks

### Task 1 — Define the problem struct
- Create `api/internal/problem/problem.go`
- Define `Problem` struct with fields: `Type string`, `Title string`, `Status int`, `Detail string`, `Instance string`, `RequestID string`
- Define `ValidationProblem` struct embedding `Problem` with an additional `Errors []FieldError` field
- Define `FieldError` struct: `Field string`, `Message string`
- Both structs must have `json` struct tags with camelCase field names matching Section 3

### Task 2 — Define the error type constants
- Create `api/internal/problem/types.go`
- Define constants for each error type URI in the catalogue defined in Section 4
- Define a `New` constructor function: `New(errorType, title string, status int, detail, instance string) *Problem`
- Define a `NewValidation` constructor: `NewValidation(detail, instance string, errors []FieldError) *ValidationProblem`

### Task 3 — Implement the WriteError helper
- Create `api/internal/problem/writer.go`
- Implement `Write(w http.ResponseWriter, r *http.Request, p *Problem)` function
- Set `Content-Type: application/problem+json` header
- Set the HTTP status code from `p.Status`
- Populate `p.Instance` from `r.URL.Path` if not already set
- Serialise and write the problem JSON body
- Implement `WriteValidation(w http.ResponseWriter, r *http.Request, p *ValidationProblem)` as an equivalent for validation errors

### Task 4 — Update 404 and 405 handlers in router
- Update `api/internal/router/router.go`
- Replace placeholder 404 and 405 handlers with handlers that use `problem.Write` with the correct error type constants

### Task 5 — Create error type documentation
- Create `api/docs/error-types.md`
- Document each defined error type: URI, HTTP status, title, description of when it is used
- Include example request and response for each error type

### Task 6 — Unit tests
- Create `api/internal/problem/problem_test.go`
- Test: `Write` sets `Content-Type: application/problem+json`
- Test: `Write` sets HTTP status code matching `p.Status`
- Test: `Write` populates `instance` from request path
- Test: `WriteValidation` includes `errors` array in response body
- Test: `500` response body does not include stack trace or raw error string
- Test: response body is valid JSON and all required fields are present

---

## 6. Definition of Done

- All tasks completed
- All error responses from the API use `application/problem+json`
- Error type documentation created and reviewed
- 404 and 405 handlers updated to use the problem writer
- All unit tests pass with `-race` flag
- `make lint` passes with zero issues
- PR reviewed and approved
- All acceptance criteria verified by reviewer
