package metric_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ionextai/otelchi/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestServerRequestDuration(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	baseCfg := metric.NewBaseConfig(
		"test-server",
		metric.WithMeterProvider(provider),
		metric.WithAttributesFunc(testAttributes),
	)

	router := chi.NewRouter()
	router.Use(metric.NewServerRequestDuration(baseCfg))
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/test", nil))

	m := findMetric(t, collectMetrics(t, reader), "http.server.request.duration")
	assert.Equal(t, "s", m.Unit)
	assert.Equal(t, "Duration of HTTP server requests.", m.Description)

	histogram, ok := m.Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Len(t, histogram.DataPoints, 1)

	dp := histogram.DataPoints[0]
	assert.Greater(t, dp.Sum, 0.0)
	assert.Equal(t, uint64(1), dp.Count)
	assertHasAttribute(t, dp.Attributes, attribute.String("test.attr", "value"))
	assertHasAttribute(t, dp.Attributes, attribute.String("outcome", metric.Success))
	assertHasIntAttribute(t, dp.Attributes, semconv.HTTPStatusCodeKey, http.StatusOK)
}

func TestServerRequestDuration_OutcomeFailure(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	baseCfg := metric.NewBaseConfig(
		"test-server",
		metric.WithMeterProvider(provider),
	)

	router := chi.NewRouter()
	router.Use(metric.NewServerRequestDuration(baseCfg))
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/test", nil))

	m := findMetric(t, collectMetrics(t, reader), "http.server.request.duration")
	histogram, ok := m.Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Len(t, histogram.DataPoints, 1)

	assertHasAttribute(t, histogram.DataPoints[0].Attributes, attribute.String("outcome", metric.Failure))
	assertHasIntAttribute(t, histogram.DataPoints[0].Attributes, semconv.HTTPStatusCodeKey, http.StatusInternalServerError)
}

func TestServerRequestDuration_WithExplicitBucketBoundaries(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	customBounds := []float64{0.1, 0.5, 1}
	baseCfg := metric.NewBaseConfig(
		"test-server",
		metric.WithMeterProvider(provider),
		metric.WithExplicitBucketBoundaries(customBounds...),
	)

	router := chi.NewRouter()
	router.Use(metric.NewServerRequestDuration(baseCfg))
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/test", nil))

	m := findMetric(t, collectMetrics(t, reader), "http.server.request.duration")
	histogram, ok := m.Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Len(t, histogram.DataPoints, 1)
	assert.Equal(t, customBounds, histogram.DataPoints[0].Bounds)
}

func TestServerRequestDuration_WithFilter(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	baseCfg := metric.NewBaseConfig(
		"test-server",
		metric.WithMeterProvider(provider),
		metric.WithFilter(func(r *http.Request) bool { return r.URL.Path != "/healthz" }),
	)

	handlerCalled := false
	router := chi.NewRouter()
	router.Use(metric.NewServerRequestDuration(baseCfg))
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rec.Code)

	rm := collectMetrics(t, reader)
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "http.server.request.duration" {
				t.Fatalf("expected no http.server.request.duration datapoint for filtered request")
			}
		}
	}
}
