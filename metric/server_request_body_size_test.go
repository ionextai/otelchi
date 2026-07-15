package metric_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/ionextai/otelchi/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestServerRequestBodySize(t *testing.T) {
	const requestBody = "hello request body"

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	baseCfg := metric.NewBaseConfig(
		"test-server",
		metric.WithMeterProvider(provider),
		metric.WithAttributesFunc(testAttributes),
	)

	router := chi.NewRouter()
	router.Use(metric.NewServerRequestBodySize(baseCfg))
	router.Post("/test", func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(requestBody))
	router.ServeHTTP(httptest.NewRecorder(), req)

	m := findMetric(t, collectMetrics(t, reader), "http.server.request.body.size")
	assert.Equal(t, "By", m.Unit)
	assert.Equal(t, "Size of HTTP server request bodies.", m.Description)

	histogram, ok := m.Data.(metricdata.Histogram[int64])
	require.True(t, ok)
	require.Len(t, histogram.DataPoints, 1)

	dp := histogram.DataPoints[0]
	assert.Equal(t, int64(len(requestBody)), dp.Sum)
	assert.Equal(t, uint64(1), dp.Count)
	assertHasAttribute(t, dp.Attributes, attribute.String("test.attr", "value"))
}

func TestServerRequestBodySize_WithFilter(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	baseCfg := metric.NewBaseConfig(
		"test-server",
		metric.WithMeterProvider(provider),
		metric.WithFilter(func(r *http.Request) bool { return r.URL.Path != "/healthz" }),
	)

	handlerCalled := false
	router := chi.NewRouter()
	router.Use(metric.NewServerRequestBodySize(baseCfg))
	router.Post("/healthz", func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		_, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/healthz", strings.NewReader("body"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.True(t, handlerCalled)

	rm := collectMetrics(t, reader)
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "http.server.request.body.size" {
				t.Fatalf("expected no http.server.request.body.size datapoint for filtered request")
			}
		}
	}
}
