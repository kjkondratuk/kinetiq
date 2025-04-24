package middleware

import (
	"github.com/go-chi/chi/v5"
	"github.com/kjkondratuk/kinetiq/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"net/http"
	"time"
)

// OtelMiddleware adds OpenTelemetry instrumentation to HTTP requests
func OtelMiddleware(instr *otel.Instrumentation) func(next http.Handler) http.Handler {
	// Create metrics
	requestCounter, _ := instr.CreateCounter(
		"kinetiq.http.requests",
		"Number of HTTP requests",
	)
	
	errorCounter, _ := instr.CreateCounter(
		"kinetiq.http.errors",
		"Number of HTTP errors",
	)
	
	requestDuration, _ := instr.CreateHistogram(
		"kinetiq.http.duration",
		"Duration of HTTP requests in milliseconds",
	)
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a new context with a span
			ctx, span := instr.StartSpan(r.Context(), "HTTP "+r.Method+" "+r.URL.Path)
			defer span.End()
			
			// Add attributes to the span
			span.SetAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.host", r.Host),
				attribute.String("http.user_agent", r.UserAgent()),
				attribute.String("http.remote_addr", r.RemoteAddr),
			)
			
			// Get the route pattern
			routePattern := chi.RouteContext(r.Context()).RoutePattern()
			if routePattern != "" {
				span.SetAttributes(attribute.String("http.route", routePattern))
			}
			
			// Create a wrapped response writer to capture the status code
			ww := &statusRecorder{ResponseWriter: w, Status: http.StatusOK}
			
			// Measure request duration
			start := time.Now()
			
			// Call the next handler
			next.ServeHTTP(ww, r.WithContext(ctx))
			
			// Record request duration
			duration := float64(time.Since(start).Milliseconds())
			requestDuration.Record(ctx, duration, metric.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.route", routePattern),
				attribute.Int("http.status_code", ww.Status),
			))
			
			// Record request count
			requestCounter.Add(ctx, 1, metric.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.route", routePattern),
				attribute.Int("http.status_code", ww.Status),
			))
			
			// Record error count if status code is 4xx or 5xx
			if ww.Status >= 400 {
				errorCounter.Add(ctx, 1, metric.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.route", routePattern),
					attribute.Int("http.status_code", ww.Status),
				))
			}
			
			// Add status code to span
			span.SetAttributes(attribute.Int("http.status_code", ww.Status))
		})
	}
}

// statusRecorder is a wrapper for http.ResponseWriter that records the status code
type statusRecorder struct {
	http.ResponseWriter
	Status int
}

// WriteHeader captures the status code before calling the underlying ResponseWriter
func (r *statusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}