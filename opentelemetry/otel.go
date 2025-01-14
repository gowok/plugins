package opentelemetry

import (
	"log/slog"

	"github.com/gowok/gowok"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
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

	opts = append(opts, trace.WithResource(resource.NewWithAttributes(
		"service.name",
	)))

	if config.LocalExporter {
		// Inisialisasi console exporter
		exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			slog.Error("failed to initialize stdout exporter", "error", err)
			return
		}

		opts = append(opts, trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(exporter)))

	}

	if config.JaegerExporter {
		jaegerExporter, err := jaeger.New(
			jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.Jaeger.Endpoint)), // Ganti dengan URL Jaeger Anda
		)
		if err != nil {
			slog.Error("failed to initialize jaeger exporter", "error", err)
			return
		}

		opts = append(opts, trace.WithBatcher(jaegerExporter))

		opts = append(opts, trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("service-name"),
			attribute.String("environment", "env"),
		)))
	}

	tracerProvider := trace.NewTracerProvider(
		opts...,
	)
	otel.SetTracerProvider(tracerProvider)
}
