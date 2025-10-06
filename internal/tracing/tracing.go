package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// TracerProvider manages distributed tracing
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

// New creates a new tracer provider with OTLP HTTP exporter
func New(serviceName, endpoint string) (*TracerProvider, error) {
	// Create OTLP HTTP exporter
	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(), // Use WithTLSClientConfig for secure connections
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global tracer provider
	otel.SetTracerProvider(provider)

	tracer := provider.Tracer(serviceName)

	return &TracerProvider{
		provider: provider,
		tracer:   tracer,
	}, nil
}

// Tracer returns the tracer
func (tp *TracerProvider) Tracer() trace.Tracer {
	return tp.tracer
}

// Shutdown shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	return tp.provider.Shutdown(ctx)
}

// StartSpan starts a new span
func (tp *TracerProvider) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := tp.tracer.Start(ctx, name)
	span.SetAttributes(attrs...)
	return ctx, span
}
