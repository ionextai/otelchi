package metric

import (
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"

	"github.com/ionextai/otelchi/internal/respwriter"
)

const (
	metricNameRequestDurationMs = "request_duration_millis"
	metricUnitRequestDurationMs = "ms"
	metricDescRequestDurationMs = "Measures the latency of HTTP requests processed by the server, in milliseconds."
)

// Deprecated: use NewServerRequestDuration instead.
func NewRequestDurationMillis(cfg BaseConfig) func(next http.Handler) http.Handler {
	// init metric, here we are using histogram for capturing request duration
	histogram, err := cfg.Meter.Int64Histogram(
		metricNameRequestDurationMs,
		otelmetric.WithDescription(metricDescRequestDurationMs),
		otelmetric.WithUnit(metricUnitRequestDurationMs),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to create %s histogram: %v", metricNameRequestDurationMs, err))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.ShouldRecord(r) {
				next.ServeHTTP(w, r)
				return
			}

			// get recording response writer
			rw := respwriter.Get(w)
			defer respwriter.Put(rw)

			// capture the start time of the request
			startTime := time.Now()

			// execute next http handler
			next.ServeHTTP(rw.ResponseWriter, r)

			// determine success/failure
			outcome := cfg.OutcomeFunc(rw.StatusCode)

			attributes := append(
				cfg.AttributesFunc(r),
				semconv.HTTPStatusCode(rw.StatusCode),
				attribute.String("outcome", outcome),
			)

			// record the request duration
			duration := time.Since(startTime)
			histogram.Record(
				r.Context(),
				int64(duration.Milliseconds()),
				otelmetric.WithAttributes(
					attributes...,
				),
			)
		})
	}
}
