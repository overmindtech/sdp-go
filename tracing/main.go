package tracing

import (
	_ "embed"

	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/overmindtech/sdp-go"

//go:generate sh -c "echo -n $(git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD) > commit.txt"
//go:embed commit.txt
var ServiceVersion string

var (
	tracer = otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(ServiceVersion),
		trace.WithSchemaURL(semconv.SchemaURL),
	)
)

func Tracer() trace.Tracer {
	return tracer
}
