# Plugin Opentelemetry

this package used for apply opentelemetry with gowok

## What is Opentelemetry?

OpenTelemetry is an open-source observability framework designed to provide a standardized way to collect, process, and export telemetry data from applications. It is a project under the Cloud Native Computing Foundation (CNCF) and aims to help developers and organizations monitor their applications and services effectively.

## Currently Supports
- [x] Jaeger Exporter
- [x] Metric Exporter

## How to use

on config.yaml or gowok.yaml set
``` yaml
opentelemetry:
  enabled: true
  serviceName: coba-coba
  # if true, this will be print on terminal
  localExporter: true
  # set jaeger exporter if you use jaeger
  jaegerExporter:
    enabled: true
    endpoint: https://jager.url/blablabla
  # set metric if you want to see metric with grafana or prometheus
  metricExporter:
    enabled: true
    path: /metrics
```

on code:
import package
``` go
    "github.com/gowok/plugins/opentelemetry"
```

add opentelemetry on configure function
``` go
    project.Configures(
		opentelemetry.Configure,
	)
```

## Full Example Code

``` yaml
app:
  web:
    enabled: true
    host: :8080

opentelemetry:
  enabled: true
  serviceName: coba-coba
  localExporter: true
  metricExporter:
    enabled: true
    path: /metrics
```

``` go
package main

import (
	"net/http"

	"github.com/gowok/gowok"
	"github.com/gowok/gowok/router"
	"github.com/gowok/plugins/opentelemetry"
	"github.com/gowok/plugins/opentelemetry/tracer"
)

func setupRoute() {
	router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {

		_, span := tracer.Start(r.Context(), "check")
		defer span.End()

		w.Write([]byte("pong"))
	})
}

func main() {

	project := gowok.Get()

	project.Configures(
		opentelemetry.Configure,
	)

	setupRoute()

	project.Run()

}
```