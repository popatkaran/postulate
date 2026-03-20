package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Tracing wraps the handler with OTel HTTP instrumentation.
// It extracts the W3C traceparent from inbound headers, creates a span per request,
// and records http.method, http.route, http.status_code, and http.url attributes.
// The Chi route pattern is used as http.route so spans are grouped by route, not raw path.
func Tracing(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, "",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			// Use the matched Chi route pattern as the span name.
			if pattern := chi.RouteContext(r.Context()).RoutePattern(); pattern != "" {
				return r.Method + " " + pattern
			}
			return r.Method + " " + r.URL.Path
		}),
	)
}
