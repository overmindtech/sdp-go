package sdp

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/MrAlias/otel-schema-utils/schema"
	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

var testCtx context.Context

func TestMain(m *testing.M) {
	key, _ := os.LookupEnv("HONEYCOMB_API_KEY")
	opts := make([]otlptracehttp.Option, 0)
	if key != "" {
		opts = []otlptracehttp.Option{
			otlptracehttp.WithEndpoint("api.honeycomb.io"),
			otlptracehttp.WithHeaders(map[string]string{"x-honeycomb-team": key}),
		}
	}

	if err := InitTracer(opts...); err != nil {
		log.Fatal(err)
	}

	var span trace.Span
	testCtx, span = tracer.Start(context.Background(), "sdp.TestMain")

	exitCode := m.Run()

	span.End()

	ShutdownTracer()

	os.Exit(exitCode)
}

func tracingResource() *resource.Resource {
	// Identify your application using resource detection
	resources := []*resource.Resource{}

	hostRes, err := resource.New(context.Background(),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		log.WithError(err).Error("error initialising host resource")
		return nil
	}
	resources = append(resources, hostRes)

	localRes, err := resource.New(context.Background(),
		resource.WithSchemaURL(semconv.SchemaURL),
		// Add your own custom attributes to identify your application
		resource.WithAttributes(
			semconv.ServiceNameKey.String("sdp-go"),
			semconv.ServiceVersionKey.String(LibraryVersion),
		),
	)
	if err != nil {
		log.WithError(err).Error("error initialising local resource")
		return nil
	}
	resources = append(resources, localRes)

	conv := schema.NewConverter(schema.NewLocalClient())
	res, err := conv.MergeResources(context.Background(), semconv.SchemaURL, resources...)

	if err != nil {
		log.WithError(err).Error("error merging resource")
		return nil
	}
	return res
}

var tp *sdktrace.TracerProvider

func InitTracerWithHoneycomb(key string, opts ...otlptracehttp.Option) error {
	if key != "" {
		opts = append(opts,
			otlptracehttp.WithEndpoint("api.honeycomb.io"),
			otlptracehttp.WithHeaders(map[string]string{"x-honeycomb-team": key}),
		)
	}
	return InitTracer(opts...)
}

func InitTracer(opts ...otlptracehttp.Option) error {
	if sentry_dsn := viper.GetString("sentry-dsn"); sentry_dsn != "" {
		var environment string
		if viper.GetString("run-mode") == "release" {
			environment = "prod"
		} else {
			environment = "dev"
		}
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              sentry_dsn,
			AttachStacktrace: true,
			EnableTracing:    false,
			Environment:      environment,
			// Set TracesSampleRate to 1.0 to capture 100%
			// of transactions for performance monitoring.
			// We recommend adjusting this value in production,
			TracesSampleRate: 1.0,
		})
		if err != nil {
			log.Errorf("sentry.Init: %s", err)
		}
		// setup recovery for an unexpected panic in this function
		defer sentry.Flush(2 * time.Second)
		defer sentry.Recover()
		log.Info("sentry configured")
	}

	client := otlptracehttp.NewClient(opts...)
	otlpExp, err := otlptrace.New(context.Background(), client)
	if err != nil {
		return fmt.Errorf("creating OTLP trace exporter: %w", err)
	}
	log.Infof("otlptracehttp client configured itself: %v", client)

	tracerOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithBatcher(otlpExp),
		sdktrace.WithResource(tracingResource()),
	}
	if viper.GetBool("stdout-trace-dump") {
		stdoutExp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return err
		}
		tracerOpts = append(tracerOpts, sdktrace.WithBatcher(stdoutExp))
	}
	tp = sdktrace.NewTracerProvider(tracerOpts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return nil
}

// nolint: contextcheck // deliberate use of local context to avoid getting tangled up in any existing timeouts or cancels
func ShutdownTracer() {
	// Flush buffered events before the program terminates.
	defer sentry.Flush(5 * time.Second)

	// ensure that we do not wait indefinitely on the trace provider shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := tp.ForceFlush(ctx); err != nil {
		log.WithContext(ctx).Printf("Error flushing tracer provider: %v", err)
	}
	if err := tp.Shutdown(ctx); err != nil {
		log.WithContext(ctx).Printf("Error shutting down tracer provider: %v", err)
	}
}
