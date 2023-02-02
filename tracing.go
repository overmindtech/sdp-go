package sdp

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName    = "github.com/overmindtech/sdp-go/sdp"
	instrumentationVersion = "0.0.1"
)

var (
	tracer = otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(instrumentationVersion),
		trace.WithSchemaURL(semconv.SchemaURL),
	)
)

type CtxMsgHandler func(ctx context.Context, msg *nats.Msg)

func NewOtelExtractingHandler(spanName string, h CtxMsgHandler, t trace.Tracer, spanOpts ...trace.SpanStartOption) nats.MsgHandler {
	if h == nil {
		return nil
	}

	return func(msg *nats.Msg) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(msg.Header))

		ctx, span := t.Start(ctx, spanName, spanOpts...)
		defer span.End()

		h(ctx, msg)
	}
}

func InjectOtelTraceContext(ctx context.Context, msg *nats.Msg) {
	if msg.Header == nil {
		msg.Header = make(nats.Header)
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(msg.Header))
}
