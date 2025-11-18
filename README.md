This fork adds status code tracking in recording request writer and "outcome" attributes for mterics.
the "outcome" attribute allows us to easily filter by "success" or "failure".

# otelchi

[![compatibility-test](https://github.com/riandyrn/otelchi/actions/workflows/compatibility-test.yaml/badge.svg)](https://github.com/riandyrn/otelchi/actions/workflows/compatibility-test.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/riandyrn/otelchi)](https://goreportcard.com/report/github.com/riandyrn/otelchi)
[![Documentation](https://godoc.org/github.com/riandyrn/otelchi?status.svg)](https://pkg.go.dev/mod/github.com/riandyrn/otelchi)

OpenTelemetry instrumentation for [go-chi/chi](https://github.com/go-chi/chi).

Essentially this is an adaptation from [otelmux](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/instrumentation/github.com/gorilla/mux/otelmux) but instead of using `gorilla/mux`, we use `go-chi/chi`.

Currently, this library can only instrument traces and metrics.

Contributions are welcomed!

## Install

```bash
$ go get github.com/ionext/otelchi
```

## Examples

See [examples](./examples) for details.
