package xtrace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type contextKey string

const traceIDKey contextKey = "traceID"

// Config holds tracing configuration.
type Config struct {
	Enabled    bool    `yaml:"enabled"`
	Endpoint   string  `yaml:"endpoint"`    // e.g. "http://jaeger:4318/v1/traces"
	Service    string  `yaml:"service"`     // service name
	SampleRate float64 `yaml:"sample_rate"` // 0.0 - 1.0
}

// InitTracer initializes OpenTelemetry tracing with OTLP HTTP exporter.
// Returns a shutdown function that must be called on application exit.
func InitTracer(cfg Config) (func(), error) {
	if !cfg.Enabled {
		return func() {}, nil
	}

	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 1.0
	}

	ctx := context.Background()

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.Endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.Service),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdown := func() {
		_ = tp.Shutdown(ctx)
	}

	return shutdown, nil
}

// GenerateTraceID creates a new random 16-byte hex trace ID.
func GenerateTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// WithTraceID stores a trace ID in the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// ExtractTraceID retrieves the trace ID from context. Returns empty string if not present.
func ExtractTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(traceIDKey).(string); ok {
		return v
	}
	return ""
}

// TraceIDHeader is the HTTP header used to propagate trace IDs.
const TraceIDHeader = "X-Trace-ID"

// Middleware returns an HTTP middleware that extracts or generates a trace ID.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(TraceIDHeader)
		if traceID == "" {
			traceID = GenerateTraceID()
		}
		ctx := WithTraceID(r.Context(), traceID)
		w.Header().Set(TraceIDHeader, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
