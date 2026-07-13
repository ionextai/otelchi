package metric_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/ionextai/otelchi/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestServerResponseBodySize(t *testing.T) {
	const responseBody = "hello response body"

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	baseCfg := metric.NewBaseConfig(
		"test-server",
		metric.WithMeterProvider(provider),
		metric.WithAttributesFunc(testAttributes),
	)

	router := chi.NewRouter()
	router.Use(metric.NewServerResponseBodySize(baseCfg))
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(responseBody))
		require.NoError(t, err)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/test", nil))

	m := findMetric(t, collectMetrics(t, reader), "http.server.response.body.size")
	assert.Equal(t, "By", m.Unit)
	assert.Equal(t, "Size of HTTP server response bodies.", m.Description)

	histogram, ok := m.Data.(metricdata.Histogram[int64])
	require.True(t, ok)
	require.Len(t, histogram.DataPoints, 1)

	dp := histogram.DataPoints[0]
	assert.Equal(t, int64(len(responseBody)), dp.Sum)
	assert.Equal(t, uint64(1), dp.Count)
	assertHasAttribute(t, dp.Attributes, attribute.String("test.attr", "value"))
	assertHasAttribute(t, dp.Attributes, attribute.String("outcome", metric.Success))
	assertHasIntAttribute(t, dp.Attributes, semconv.HTTPStatusCodeKey, http.StatusOK)
}

func TestServerResponseBodySize_OutcomeFailure(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	baseCfg := metric.NewBaseConfig("test-server", metric.WithMeterProvider(provider))

	router := chi.NewRouter()
	router.Use(metric.NewServerResponseBodySize(baseCfg))
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/test", nil))

	m := findMetric(t, collectMetrics(t, reader), "http.server.response.body.size")
	histogram, ok := m.Data.(metricdata.Histogram[int64])
	require.True(t, ok)
	require.Len(t, histogram.DataPoints, 1)
	assertHasAttribute(t, histogram.DataPoints[0].Attributes, attribute.String("outcome", metric.Failure))
	assertHasIntAttribute(t, histogram.DataPoints[0].Attributes, semconv.HTTPStatusCodeKey, http.StatusInternalServerError)
}

func TestServerResponseBodySize_WithFilter(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	baseCfg := metric.NewBaseConfig(
		"test-server",
		metric.WithMeterProvider(provider),
		metric.WithFilter(func(r *http.Request) bool { return r.URL.Path != "/healthz" }),
	)

	handlerCalled := false
	router := chi.NewRouter()
	router.Use(metric.NewServerResponseBodySize(baseCfg))
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	assert.True(t, handlerCalled)

	rm := collectMetrics(t, reader)
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "http.server.response.body.size" {
				t.Fatalf("expected no http.server.response.body.size datapoint for filtered request")
			}
		}
	}
}
