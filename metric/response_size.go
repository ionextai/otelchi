package metric

import (
	"fmt"
	"net/http"

	"github.com/ionextai/otelchi/internal/respwriter"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	metricNameResponseSizeBytes = "response_size_bytes"
	metricUnitResponseSizeBytes = "By"
	metricDescResponseSizeBytes = "Measures the size of the response in bytes."
)

// NewResponseSizeBytes records the size of the HTTP response body in bytes.
//
// Deprecated: use NewServerResponseBodySize instead.
func NewResponseSizeBytes(cfg BaseConfig) func(next http.Handler) http.Handler {
	// init metric, here we are using histogram for capturing response size
	histogram, err := cfg.Meter.Int64Histogram(
		metricNameResponseSizeBytes,
		otelmetric.WithDescription(metricDescResponseSizeBytes),
		otelmetric.WithUnit(metricUnitResponseSizeBytes),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to create %s histogram: %v", metricNameResponseSizeBytes, err))
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

			// execute next http handler
			next.ServeHTTP(rw.ResponseWriter, r)

			// determine success/failure
			outcome := cfg.OutcomeFunc(rw.StatusCode)

			attributes := append(cfg.AttributesFunc(r), attribute.String("outcome", outcome))

			// record the response size
			histogram.Record(
				r.Context(),
				rw.BytesWritten,
				otelmetric.WithAttributes(
					attributes...,
				),
			)
		})
	}
}
