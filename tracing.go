package sdp

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/bufbuild/connect-go"
	"github.com/getsentry/sentry-go"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
		ctx := context.Background()

		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(msg.Header))

		ctx, span := t.Start(ctx, spanName, spanOpts...)
		defer span.End()

		h(ctx, msg)
	}
}

func NewAsyncOtelExtractingHandler(spanName string, h CtxMsgHandler, t trace.Tracer, spanOpts ...trace.SpanStartOption) nats.MsgHandler {
	if h == nil {
		return nil
	}

	return func(msg *nats.Msg) {
		go func() {
			defer sentry.Recover()

			ctx := context.Background()
			ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(msg.Header))

			ctx, span := t.Start(ctx, spanName, spanOpts...)
			defer span.End()

			h(ctx, msg)
		}()
	}
}

func InjectOtelTraceContext(ctx context.Context, msg *nats.Msg) {
	if msg.Header == nil {
		msg.Header = make(nats.Header)
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(msg.Header))
}

type sentryInterceptor struct{}

// NewSentryInterceptor pass this to connect handlers as `connect.WithInterceptors(NewSentryInterceptor())` to recover from panics in the handler and report them to sentry. Otherwise panics get recovered by connect-go itself and do not get reported to sentry.
func NewSentryInterceptor() connect.Interceptor {
	return &sentryInterceptor{}
}

func (i *sentryInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	// Same as previous UnaryInterceptorFunc.
	return connect.UnaryFunc(func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
		defer sentry.Recover()
		return next(ctx, req)
	})
}

func (*sentryInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return connect.StreamingClientFunc(func(
		ctx context.Context,
		spec connect.Spec,
	) connect.StreamingClientConn {
		defer sentry.Recover()
		conn := next(ctx, spec)
		return conn
	})
}

func (i *sentryInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return connect.StreamingHandlerFunc(func(
		ctx context.Context,
		conn connect.StreamingHandlerConn,
	) error {
		defer sentry.Recover()
		return next(ctx, conn)
	})
}

// LogRecoverToReturn Recovers from a panic, logs and forwards it sentry and otel, then returns
// Does nothing when there is no panic.
func LogRecoverToReturn(ctx *context.Context, loc string) {
	err := recover()
	if err == nil {
		return
	}

	stack := string(debug.Stack())

	sentry.CurrentHub().Recover(err)

	msg := fmt.Sprintf("unhandled panic in %v: %v", loc, err)
	log.WithFields(log.Fields{"loc": loc, "stack": stack}).Error(msg)

	if ctx != nil {
		span := trace.SpanFromContext(*ctx)
		span.SetAttributes(attribute.String("om.panic.loc", loc))
		span.SetAttributes(attribute.String("om.panic.stack", stack))
	}
}
