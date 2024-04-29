package auth

import (
	"github.com/overmindtech/sdp-go"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/overmindtech/sdp-go/auth"

var tracer = otel.GetTracerProvider().Tracer(
	instrumentationName,
	trace.WithInstrumentationVersion(sdp.LibraryVersion),
	trace.WithSchemaURL(semconv.SchemaURL),
)
