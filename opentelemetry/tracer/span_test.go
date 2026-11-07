package tracer

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestTraceIDAndSpanIDWithoutSpan(t *testing.T) {
	if got := TraceID(context.Background()); got != "" {
		t.Fatalf("TraceID without span = %q, want empty", got)
	}
	if got := SpanID(context.Background()); got != "" {
		t.Fatalf("SpanID without span = %q, want empty", got)
	}

	ctx := trace.ContextWithSpanContext(context.Background(), trace.SpanContext{})
	if got := TraceID(ctx); got != "" {
		t.Fatalf("TraceID with invalid span = %q, want empty", got)
	}
}

func TestTraceIDAndSpanIDFallback(t *testing.T) {
	ctx := context.Background()

	if got := TraceID(ctx, "unknown"); got != "unknown" {
		t.Fatalf("TraceID fallback = %q, want %q", got, "unknown")
	}
	if got := SpanID(ctx, "unknown"); got != "unknown" {
		t.Fatalf("SpanID fallback = %q, want %q", got, "unknown")
	}

	if got := TraceID(ctx, "first", "second"); got != "first" {
		t.Fatalf("TraceID fallback = %q, want %q", got, "first")
	}

	invalid := trace.ContextWithSpanContext(ctx, trace.SpanContext{})
	if got := TraceID(invalid, "unknown"); got != "unknown" {
		t.Fatalf("TraceID fallback with invalid span = %q, want %q", got, "unknown")
	}
}

func TestTraceIDAndSpanIDFallbackIgnoredWithSpan(t *testing.T) {
	want := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01, 0x02},
		SpanID:     trace.SpanID{0x03, 0x04},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), want)

	if got, expected := TraceID(ctx, "unknown"), want.TraceID().String(); got != expected {
		t.Fatalf("TraceID = %q, want %q", got, expected)
	}
	if got, expected := SpanID(ctx, "unknown"), want.SpanID().String(); got != expected {
		t.Fatalf("SpanID = %q, want %q", got, expected)
	}
}

func TestTraceIDAndSpanIDWithSpanContext(t *testing.T) {
	want := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01, 0x02},
		SpanID:     trace.SpanID{0x03, 0x04},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), want)

	if got, expected := TraceID(ctx), want.TraceID().String(); got != expected {
		t.Fatalf("TraceID = %q, want %q", got, expected)
	}
	if got, expected := SpanID(ctx), want.SpanID().String(); got != expected {
		t.Fatalf("SpanID = %q, want %q", got, expected)
	}
}

func TestStartPopulatesTraceIDs(t *testing.T) {
	previous := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample())))

	ctx, span := Start(context.Background(), "test")
	defer span.End()

	if !span.SpanContext().IsValid() {
		t.Fatal("expected a valid span context")
	}
	if got, expected := TraceID(ctx), span.SpanContext().TraceID().String(); got != expected {
		t.Fatalf("TraceID = %q, want %q", got, expected)
	}
	if got, expected := SpanID(ctx), span.SpanContext().SpanID().String(); got != expected {
		t.Fatalf("SpanID = %q, want %q", got, expected)
	}
}
