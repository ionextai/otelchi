package metric

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ionextai/otelchi/internal/respwriter"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

const (
	metricNameServerRequestDuration = "http.server.request.duration"
	metricUnitServerRequestDuration = "s"
	metricDescServerRequestDuration = "Duration of HTTP server requests."
)

// defaultServerRequestDurationBucketBoundaries are the same explicit bucket
// boundaries used by otelhttp, so request duration histograms are useful for
// common HTTP latency ranges and stay consistent with other OpenTelemetry
// HTTP server instrumentation. Override via WithExplicitBucketBoundaries.
var defaultServerRequestDurationBucketBoundaries = []float64{
	0.005, 0.01, 0.025, 0.05, 0.075, 0.1,
	0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10,
}

// NewServerRequestDuration records the duration of HTTP server requests in seconds.
func NewServerRequestDuration(cfg BaseConfig) func(next http.Handler) http.Handler {
	bounds := cfg.RequestDurationBucketBoundaries
	if len(bounds) == 0 {
		bounds = defaultServerRequestDurationBucketBoundaries
	}

	histogram, err := cfg.Meter.Float64Histogram(
		metricNameServerRequestDuration,
		otelmetric.WithDescription(metricDescServerRequestDuration),
		otelmetric.WithUnit(metricUnitServerRequestDuration),
		otelmetric.WithExplicitBucketBoundaries(bounds...),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to create %s histogram: %v", metricNameServerRequestDuration, err))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.ShouldRecord(r) {
				next.ServeHTTP(w, r)
				return
			}

			rw := respwriter.Get(w)
			defer respwriter.Put(rw)

			startTime := time.Now()

			next.ServeHTTP(rw.ResponseWriter, r)

			duration := time.Since(startTime)
			outcome := cfg.OutcomeFunc(rw.StatusCode)
			attributes := append(
				cfg.AttributesFunc(r),
				semconv.HTTPStatusCode(rw.StatusCode),
				attribute.String("outcome", outcome),
			)

			histogram.Record(
				r.Context(),
				float64(duration)/float64(time.Second),
				otelmetric.WithAttributes(attributes...),
			)
		})
	}
}
