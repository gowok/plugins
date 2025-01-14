package opentelemetry

import (
	"log"
	"log/slog"

	"github.com/gowok/gowok"
	"github.com/gowok/gowok/router"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

func Configure(project *gowok.Project) {

	configAny, ok := project.ConfigMap["opentelemetry"]
	if !ok {
		slog.Warn("no configuration", "plugin", "mongo")
		return
	}

	configMap, ok := configAny.(map[string]any)
	if !ok {
		slog.Warn("no configuration", "plugin", "mongo")
		return
	}
	config := ConfigFromMap(configMap)

	var opts []trace.TracerProviderOption

	if config.LocalExporter {
		// Inisialisasi console exporter
		exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			slog.Error("failed to initialize stdout exporter", "error", err)
			return
		}

		opts = append(opts, trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(exporter)))

	}

	if config.JaegerEnabled {
		jaegerExporter, err := jaeger.New(
			jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerExporter.Endpoint)),
		)
		if err != nil {
			slog.Error("failed to initialize jaeger exporter", "error", err)
			return
		}

		opts = append(opts, trace.WithBatcher(jaegerExporter))

		opts = append(opts, trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.ServiceName),
			// attribute.String("environment", "env"),
		)))
	}

	if config.MatricEnabled {
		if err := runtimemetrics.Start(); err != nil {
			slog.Error("failed to start runtime metrics", "error", err)
			return
		}

		exporter, err := prometheus.New()
		if err != nil {
			log.Fatal(err)
		}
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
		otel.SetMeterProvider(provider)

		router.Get(config.MatricExporter.Path, promhttp.Handler().ServeHTTP)
	}

	tracerProvider := trace.NewTracerProvider(
		opts...,
	)
	otel.SetTracerProvider(tracerProvider)
}
