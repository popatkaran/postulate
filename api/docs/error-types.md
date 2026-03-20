# Postulate API — Error Type Catalogue

All error responses from the Postulate API use `Content-Type: application/problem+json` per [RFC 7807](https://www.rfc-editor.org/rfc/rfc7807).

Base URI: `https://postulate.dev`

---

## Error Types

### `https://postulate.dev/errors/not-found`

| Field | Value |
|-------|-------|
| HTTP Status | `404 Not Found` |
| Title | `Not Found` |
| When Used | The requested resource does not exist. |

**Example response:**
```json
{
  "type": "https://postulate.dev/errors/not-found",
  "title": "Not Found",
  "status": 404,
  "detail": "",
  "instance": "/v1/projects/unknown-id",
  "request_id": "01HXYZ..."
}
```

---

### `https://postulate.dev/errors/method-not-allowed`

| Field | Value |
|-------|-------|
| HTTP Status | `405 Method Not Allowed` |
| Title | `Method Not Allowed` |
| When Used | The HTTP method used is not supported on this route. |

**Example response:**
```json
{
  "type": "https://postulate.dev/errors/method-not-allowed",
  "title": "Method Not Allowed",
  "status": 405,
  "detail": "",
  "instance": "/v1/projects",
  "request_id": "01HXYZ..."
}
```

---

### `https://postulate.dev/errors/validation-failed`

| Field | Value |
|-------|-------|
| HTTP Status | `422 Unprocessable Entity` |
| Title | `Validation Failed` |
| When Used | The request body or query parameters failed validation. |

This error type includes an additional `errors` array with field-level details.

**Example response:**
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

---

### `https://postulate.dev/errors/internal-server-error`

| Field | Value |
|-------|-------|
| HTTP Status | `500 Internal Server Error` |
| Title | `Internal Server Error` |
| When Used | An unhandled server-side error occurred. |

The `detail` field contains a generic message only. Internal error details are logged server-side and never exposed in the response body.

**Example response:**
```json
{
  "type": "https://postulate.dev/errors/internal-server-error",
  "title": "Internal Server Error",
  "status": 500,
  "detail": "An unexpected error occurred. Please try again later.",
  "instance": "/v1/generate/interview",
  "request_id": "01HXYZ..."
}
```

---

## Adding a New Error Type

1. Add a constant to `api/internal/problem/types.go`.
2. Add an entry to this catalogue with URI, status, title, and usage description.
3. Include an example request and response.
4. Open a PR tagged `postulate-api-contract` — error type additions are a contract change.
