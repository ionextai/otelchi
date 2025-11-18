package metric

import (
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	metricNameRequestDurationMs = "request_duration_millis"
	metricUnitRequestDurationMs = "ms"
	metricDescRequestDurationMs = "Measures the latency of HTTP requests processed by the server, in milliseconds."
)

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
			// get recording response writer
			rrw := getRRW(w)
			defer putRRW(rrw)

			// capture the start time of the request
			startTime := time.Now()

			// execute next http handler
			next.ServeHTTP(rrw.writer, r)

			// determine success/failure
			outcome := getOutcome(rrw.statusCode)

			attributes := append(cfg.AttributesFunc(r), attribute.String("outcome", outcome))

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
