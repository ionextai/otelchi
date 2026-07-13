package metric_test

import (
	"net/http"
	"testing"

	"github.com/ionextai/otelchi/metric"
	"github.com/stretchr/testify/assert"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestDefaultOutcomeFunc(t *testing.T) {
	cfg := metric.NewBaseConfig("test-server")

	tests := []struct {
		statusCode int
		want       string
	}{
		{http.StatusOK, metric.Success},
		{http.StatusMovedPermanently, metric.Success},
		{http.StatusNotFound, metric.Success},
		{http.StatusTooManyRequests, metric.Success},
		{http.StatusInternalServerError, metric.Failure},
		{http.StatusServiceUnavailable, metric.Failure},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, cfg.OutcomeFunc(tt.statusCode))
	}
}

func TestWithOutcomeFunc(t *testing.T) {
	cfg := metric.NewBaseConfig(
		"test-server",
		metric.WithOutcomeFunc(func(statusCode int) string {
			if statusCode >= 400 {
				return metric.Failure
			}
			return metric.Success
		}),
	)

	assert.Equal(t, metric.Failure, cfg.OutcomeFunc(http.StatusTooManyRequests))
	assert.Equal(t, metric.Success, cfg.OutcomeFunc(http.StatusOK))
}

func TestShouldRecord_NoFilters(t *testing.T) {
	cfg := metric.NewBaseConfig("test-server")
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)

	assert.True(t, cfg.ShouldRecord(req))
}

func TestShouldRecord_SingleFilter(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	cfg := metric.NewBaseConfig(
		"test-server",
		metric.WithMeterProvider(provider),
		metric.WithFilter(func(r *http.Request) bool {
			return r.URL.Path != "/healthz"
		}),
	)

	healthzReq, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	testReq, _ := http.NewRequest(http.MethodGet, "/test", nil)

	assert.False(t, cfg.ShouldRecord(healthzReq))
	assert.True(t, cfg.ShouldRecord(testReq))
}

func TestShouldRecord_MultipleFiltersANDed(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	cfg := metric.NewBaseConfig(
		"test-server",
		metric.WithMeterProvider(provider),
		metric.WithFilter(func(r *http.Request) bool { return r.URL.Path != "/healthz" }),
		metric.WithFilter(func(r *http.Request) bool { return r.Method != http.MethodOptions }),
	)

	allowed, _ := http.NewRequest(http.MethodGet, "/test", nil)
	excludedByFirst, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	excludedBySecond, _ := http.NewRequest(http.MethodOptions, "/test", nil)

	assert.True(t, cfg.ShouldRecord(allowed))
	assert.False(t, cfg.ShouldRecord(excludedByFirst))
	assert.False(t, cfg.ShouldRecord(excludedBySecond))
}
