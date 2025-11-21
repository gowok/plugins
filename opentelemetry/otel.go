package opentelemetry

import (
	"context"
	"log/slog"
	"time"

	"github.com/gowok/fp/maps"
	"github.com/gowok/gowok"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

var traces = map[string]trace.TracerProviderOption{}

func Configure() {
	var config Config
	err := maps.ToStruct(maps.Get[map[string]any](gowok.Config.Map(), "opentelemetry"), &config)
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		return
	}
	opts := []trace.TracerProviderOption{
		trace.WithSampler(trace.ParentBased(trace.AlwaysSample())),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.Name),
			semconv.ServiceVersionKey.String("1.0.0"),
		)),
	}

	for _, configTrace := range config.Exporters {
		if !configTrace.Enabled {
			continue
		}

		switch configTrace.Driver {
		case "local":
			exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
			if err != nil {
				slog.Error("failed to initialize", "exporter", configTrace.Driver, "error", err)
				return
			}
			opts = append(opts, trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(exporter)))
			otel.SetTracerProvider(trace.NewTracerProvider(opts...))

		case "jaeger", "otlp":
			_opts := []otlptracehttp.Option{
				otlptracehttp.WithEndpoint(configTrace.Endpoint),
			}
			if configTrace.Insecure {
				_opts = append(_opts, otlptracehttp.WithInsecure())
			}
			exporter, err := otlptracehttp.New(context.Background(), _opts...)
			if err != nil {
				slog.Error("failed to initialize", "exporter", configTrace.Driver, "error", err)
				return
			}
			opts = append(opts, trace.WithBatcher(exporter, trace.WithBatchTimeout(3*time.Second)))
			otel.SetTracerProvider(trace.NewTracerProvider(opts...))

		case "prometheus":
			if configTrace.Endpoint == "" {
				slog.Error("failed to start", "metric", configTrace.Driver, "required", "endpoint")
				return
			}
			if err := runtimemetrics.Start(); err != nil {
				slog.Error("failed to start", "metric", configTrace.Driver, "error", err)
				return
			}

			exporter, err := prometheus.New()
			if err != nil {
				slog.Error("failed to start", "metric", configTrace.Driver, "error", err)
				return
			}
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
			otel.SetMeterProvider(provider)

			gowok.Web.Get(configTrace.Endpoint, promhttp.Handler().ServeHTTP)
		}
	}

}
