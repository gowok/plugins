package tracer

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func Start(ctx context.Context, name string) (context.Context, trace.Span) {
	return otel.Tracer("").Start(ctx, name)
}

func TraceID(ctx context.Context, fallback ...string) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return firstOrEmpty(fallback)
	}
	return sc.TraceID().String()
}

func SpanID(ctx context.Context, fallback ...string) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return firstOrEmpty(fallback)
	}
	return sc.SpanID().String()
}

func firstOrEmpty(fallback []string) string {
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}
