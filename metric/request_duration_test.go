package metric_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"

	"github.com/ionextai/otelchi/metric"
)

func TestRequestDurationMillis(t *testing.T) {
	// setup environment
	expLatencyInMillis := 100

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	baseCfg := metric.NewBaseConfig("test-server", metric.WithMeterProvider(provider))
	middleware := metric.NewRequestDurationMillis(baseCfg)

	router := chi.NewRouter()
	router.Use(middleware)
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Duration(expLatencyInMillis) * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// read the recorded metrics
	var rm metricdata.ResourceMetrics
	err := reader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)

	metrics := rm.ScopeMetrics[0].Metrics
	require.Len(t, metrics, 1)

	hist, ok := metrics[0].Data.(metricdata.Histogram[int64])
	require.True(t, ok)
	require.Len(t, hist.DataPoints, 1)

	dp := hist.DataPoints[0]
	assert.GreaterOrEqual(t, dp.Sum, int64(expLatencyInMillis))
	assert.Equal(t, uint64(1), dp.Count)
	assertHasAttribute(t, dp.Attributes, attribute.String("outcome", metric.Success))
	assertHasIntAttribute(t, dp.Attributes, semconv.HTTPStatusCodeKey, http.StatusOK)
}

func TestRequestDurationMillis_OutcomeFailure(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	baseCfg := metric.NewBaseConfig("test-server", metric.WithMeterProvider(provider))
	middleware := metric.NewRequestDurationMillis(baseCfg)

	router := chi.NewRouter()
	router.Use(middleware)
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/test", nil))

	m := findMetric(t, collectMetrics(t, reader), "request_duration_millis")
	hist, ok := m.Data.(metricdata.Histogram[int64])
	require.True(t, ok)
	require.Len(t, hist.DataPoints, 1)
	assertHasAttribute(t, hist.DataPoints[0].Attributes, attribute.String("outcome", metric.Failure))
	assertHasIntAttribute(t, hist.DataPoints[0].Attributes, semconv.HTTPStatusCodeKey, http.StatusInternalServerError)
}

func TestRequestDurationMillis_WithFilter(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	baseCfg := metric.NewBaseConfig(
		"test-server",
		metric.WithMeterProvider(provider),
		metric.WithFilter(func(r *http.Request) bool { return r.URL.Path != "/healthz" }),
	)
	middleware := metric.NewRequestDurationMillis(baseCfg)

	handlerCalled := false
	router := chi.NewRouter()
	router.Use(middleware)
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

	assert.True(t, handlerCalled)

	rm := collectMetrics(t, reader)
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "request_duration_millis" {
				t.Fatalf("expected no request_duration_millis datapoint for filtered request")
			}
		}
	}
}
