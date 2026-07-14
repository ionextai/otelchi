package metric

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"

	"github.com/ionextai/otelchi/version"
)

const (
	ScopeName = "github.com/ionextai/otelchi/metric"
	Success   = "success"
	Failure   = "failure"
)

// Filter is a predicate used to determine whether metrics should be recorded
// for a given request. A Filter must return true for the request to be
// recorded. Mirrors otelchi.Filter for the tracing middleware.
type Filter func(*http.Request) bool

// BaseConfig is used to configure the metrics middleware.
type BaseConfig struct {
	// for initialization
	meterProvider otelmetric.MeterProvider

	// actual config state
	Meter          otelmetric.Meter
	ServerName     string
	AttributesFunc func(req *http.Request) []attribute.KeyValue

	// OutcomeFunc derives the `outcome` attribute ("success"/"failure") from
	// the response status code. Defaults to getOutcome (5xx == failure).
	OutcomeFunc func(statusCode int) string

	// Filters are consulted before recording any metric for a request. If
	// any filter returns false, no metric is recorded for that request, but
	// the next handler still runs. See WithFilter.
	Filters []Filter

	// RequestDurationBucketBoundaries overrides the histogram bucket
	// boundaries used by NewServerRequestDuration's http.server.request.duration
	// histogram. Not consumed by any other metric constructor.
	RequestDurationBucketBoundaries []float64
}

// ShouldRecord reports whether metrics should be recorded for r, per the
// configured Filters. All filters must return true for recording to occur.
func (cfg BaseConfig) ShouldRecord(r *http.Request) bool {
	for _, filter := range cfg.Filters {
		if !filter(r) {
			return false
		}
	}
	return true
}

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*BaseConfig)
}

type optionFunc func(*BaseConfig)

func (o optionFunc) apply(c *BaseConfig) {
	o(c)
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global provider is used.
func WithMeterProvider(provider otelmetric.MeterProvider) Option {
	return optionFunc(func(cfg *BaseConfig) {
		cfg.meterProvider = provider
	})
}

// WithAttributesFunc specifies a function called to set attributes on a metric record for a given request.
// If none is specified, otel `http.method`, `http.scheme` and `http.route` is used.
func WithAttributesFunc(fn func(req *http.Request) []attribute.KeyValue) Option {
	return optionFunc(func(cfg *BaseConfig) {
		cfg.AttributesFunc = fn
	})
}

// WithOutcomeFunc overrides how the `outcome` attribute is derived from the
// response status code. By default, only status codes >= 500 are classified
// as "failure" (see Failure/Success constants); everything else, including
// 4xx client errors, is classified as "success". Use this to redefine the
// threshold, e.g. to also treat 429 as a failure.
func WithOutcomeFunc(fn func(statusCode int) string) Option {
	return optionFunc(func(cfg *BaseConfig) {
		cfg.OutcomeFunc = fn
	})
}

// WithFilter adds a filter used to decide whether a request's metrics should
// be recorded. If any filter returns false, no metric is recorded for that
// request (all filters must return true for recording to occur), but the
// next handler is still invoked. Use this to exclude noisy routes (e.g.
// health checks, readiness probes) from skewing histograms and alerts.
// Mirrors otelchi.WithFilter for the tracing middleware.
func WithFilter(filter Filter) Option {
	return optionFunc(func(cfg *BaseConfig) {
		cfg.Filters = append(cfg.Filters, filter)
	})
}

// WithExplicitBucketBoundaries overrides the histogram bucket boundaries
// used by NewServerRequestDuration's http.server.request.duration histogram,
// so bucket resolution can be tuned to service-specific SLO thresholds. It
// has no effect on any other metric constructor.
func WithExplicitBucketBoundaries(bounds ...float64) Option {
	return optionFunc(func(cfg *BaseConfig) {
		cfg.RequestDurationBucketBoundaries = bounds
	})
}

// NewBaseConfig builds a BaseConfig for serverName, applying opts on top of
// the defaults: the global meter provider, an OutcomeFunc that classifies
// only 5xx status codes as "failure" (see getOutcome), and an AttributesFunc
// that sets `http.method`, `http.scheme`, and (once known) `http.route`. Use
// the With* options to override any of these before passing the resulting
// BaseConfig to one of the metric constructors (e.g. NewServerRequestDuration).
func NewBaseConfig(serverName string, opts ...Option) BaseConfig {
	// init base config
	cfg := BaseConfig{
		ServerName:  serverName,
		OutcomeFunc: getOutcome,
		AttributesFunc: func(req *http.Request) []attribute.KeyValue {
			schema := semconv.HTTPSchemeHTTP
			if req.TLS != nil {
				schema = semconv.HTTPSchemeHTTPS
			}

			attrs := []attribute.KeyValue{
				semconv.HTTPMethod(req.Method),
				schema,
			}

			if route := chi.RouteContext(req.Context()).RoutePattern(); route != "" {
				attrs = append(attrs, semconv.HTTPRoute(route))
			}

			return attrs
		},
	}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	if cfg.meterProvider == nil {
		cfg.meterProvider = otel.GetMeterProvider()
	}
	cfg.Meter = cfg.meterProvider.Meter(
		ScopeName,
		otelmetric.WithSchemaURL(semconv.SchemaURL),
		otelmetric.WithInstrumentationVersion(version.Version()),
		otelmetric.WithInstrumentationAttributes(
			semconv.ServiceName(serverName),
		),
	)

	return cfg
}

func getOutcome(statusCode int) string {
	if statusCode >= 500 {
		return Failure
	}
	return Success
}
