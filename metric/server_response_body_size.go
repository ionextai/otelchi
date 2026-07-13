package metric

import (
	"fmt"
	"net/http"

	"github.com/ionextai/otelchi/internal/respwriter"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

const (
	metricNameServerResponseBodySize = "http.server.response.body.size"
	metricUnitServerResponseBodySize = "By"
	metricDescServerResponseBodySize = "Size of HTTP server response bodies."
)

// NewServerResponseBodySize records the size of HTTP server response bodies in bytes.
func NewServerResponseBodySize(cfg BaseConfig) func(next http.Handler) http.Handler {
	histogram, err := cfg.Meter.Int64Histogram(
		metricNameServerResponseBodySize,
		otelmetric.WithDescription(metricDescServerResponseBodySize),
		otelmetric.WithUnit(metricUnitServerResponseBodySize),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to create %s histogram: %v", metricNameServerResponseBodySize, err))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.ShouldRecord(r) {
				next.ServeHTTP(w, r)
				return
			}

			rw := respwriter.Get(w)
			defer respwriter.Put(rw)

			next.ServeHTTP(rw.ResponseWriter, r)

			outcome := cfg.OutcomeFunc(rw.StatusCode)
			attributes := append(
				cfg.AttributesFunc(r),
				semconv.HTTPStatusCode(rw.StatusCode),
				attribute.String("outcome", outcome),
			)

			histogram.Record(
				r.Context(),
				rw.BytesWritten,
				otelmetric.WithAttributes(attributes...),
			)
		})
	}
}
