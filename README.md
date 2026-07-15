This fork adds status code tracking in recording request writer and, on top of that, an "outcome"
attribute for metrics. The `http.status_code` attribute (per OpenTelemetry HTTP metrics semantic
conventions) and the fork-specific "outcome" attribute — which lets us easily filter by "success" or
"failure" — are both available fork-wide, including on the semantic-convention metrics
(`http.server.request.duration`, `http.server.response.body.size`), not just the legacy ones.

# otelchi

[![compatibility-test](https://github.com/ionextai/otelchi/actions/workflows/compatibility-test.yaml/badge.svg)](https://github.com/ionextai/otelchi/actions/workflows/compatibility-test.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ionextai/otelchi)](https://goreportcard.com/report/github.com/ionextai/otelchi)
[![Documentation](https://godoc.org/github.com/ionextai/otelchi?status.svg)](https://pkg.go.dev/mod/github.com/ionextai/otelchi)

OpenTelemetry instrumentation for [go-chi/chi](https://github.com/go-chi/chi).

Essentially this is an adaptation from [otelmux](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/github.com/gorilla/mux/otelmux) but instead of using `gorilla/mux`, we use `go-chi/chi`.

Currently, this library can only instrument traces and metrics.

Contributions are welcomed!

## Install

```bash
$ go get github.com/ionextai/otelchi
```

## Examples

See [examples](./examples) for details.

## Metrics

The `metric` package provides OpenTelemetry semantic-convention compliant HTTP server metric middleware:

- `http.server.request.duration`
- `http.server.active_requests`
- `http.server.request.body.size`
- `http.server.response.body.size`

Legacy metric middleware for `request_duration_millis`, `requests_inflight`, and `response_size_bytes` is still available but deprecated.

### Configuration options

- `WithOutcomeFunc(fn func(statusCode int) string)` — overrides how the `outcome` attribute is derived from the response status code. Defaults to classifying `>= 500` as `"failure"` and everything else (including 4xx) as `"success"`. Use this to redefine the threshold per service, e.g. to also treat `429` as a failure.
- `WithFilter(filter metric.Filter)` — excludes matching requests from all metric recordings (e.g. health checks, readiness probes), so noisy endpoints don't skew histograms and alerting thresholds. Mirrors the tracing package's `WithFilter`. The next handler still runs; only the metric recording is skipped.
- `WithExplicitBucketBoundaries(bounds ...float64)` — overrides the histogram bucket boundaries used by `http.server.request.duration`, so bucket resolution can be tuned to service-specific SLO/alerting thresholds.

See [`metric/config.go`](./metric/config.go) godoc for details.
