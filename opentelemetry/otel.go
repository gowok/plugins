package opentelemetry

import (
	"context"
	"log/slog"

	"github.com/gowok/gowok"
	"github.com/gowok/gowok/maps"
	"github.com/gowok/gowok/router"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

func Configure(project *gowok.Project) {
	var config Config
	err := maps.ToStruct(maps.Get[map[string]any](project.ConfigMap, "opentelemetry"), &config)
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		return
	}
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

	if config.JaegerExporter.Enabled {
		exporter, err := otlptracehttp.New(context.Background(), otlptracehttp.WithEndpoint(config.JaegerExporter.Endpoint))
		if err != nil {
			slog.Error("failed to initialize jaeger exporter", "error", err)
			return
		}

		opts = append(opts,
			trace.WithBatcher(exporter),
			trace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(config.ServiceName),
				// attribute.String("environment", "env"),
			)),
		)
	}

	if config.MetricExporter.Enabled {
		if err := runtimemetrics.Start(); err != nil {
			slog.Error("failed to start runtime metrics", "error", err)
			return
		}

		exporter, err := prometheus.New()
		if err != nil {
			slog.Error("failed to initialize prometheus exporter", "error", err)
			return
		}
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
		otel.SetMeterProvider(provider)

		router.Get(config.MetricExporter.Path, promhttp.Handler().ServeHTTP)
	}

	tracerProvider := trace.NewTracerProvider(
		opts...,
	)
	otel.SetTracerProvider(tracerProvider)
}
