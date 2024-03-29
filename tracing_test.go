package sdp

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestTraceContextPropagation(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	tc := TestConnection{
		Messages: make([]ResponseMessage, 0),
	}

	outerCtx := context.Background()
	outerCtx, outerSpan := tp.Tracer("outerTracer").Start(outerCtx, "outer span")
	// outerJson, err := outerSpan.SpanContext().MarshalJSON()
	// if err != nil {
	// 	t.Errorf("error marshalling outerSpan: %v", err)
	// } else {
	// 	if !bytes.Equal(outerJson, []byte("{\"TraceID\":\"00000000000000000000000000000000\",\"SpanID\":\"0000000000000000\",\"TraceFlags\":\"00\",\"TraceState\":\"\",\"Remote\":false}")) {
	// 		t.Errorf("outer span has unexpected context: %v", string(outerJson))
	// 	}
	// }
	handlerCalled := make(chan struct{})
	tc.Subscribe("test.subject", NewOtelExtractingHandler("inner span", func(innerCtx context.Context, msg *nats.Msg) {
		handlerCalled <- struct{}{}

		_, innerSpan := tp.Tracer("innerTracer").Start(innerCtx, "innerSpan")
		// innerJson, err := innerSpan.SpanContext().MarshalJSON()
		// if err != nil {
		// 	t.Errorf("error marshalling innerSpan: %v", err)
		// } else {
		// 	if !bytes.Equal(innerJson, []byte("{\"TraceID\":\"00000000000000000000000000000000\",\"SpanID\":\"0000000000000000\",\"TraceFlags\":\"00\",\"TraceState\":\"\",\"Remote\":false}")) {
		// 		t.Errorf("inner span has unexpected context: %v", string(innerJson))
		// 	}
		// }
		if innerSpan.SpanContext().TraceID() != outerSpan.SpanContext().TraceID() {
			t.Error("inner span did not link up to outer span")
		}
	}, tp.Tracer("providedTracer")))

	m := &nats.Msg{
		Subject: "test.subject",
		Data:    make([]byte, 0),
	}

	InjectOtelTraceContext(outerCtx, m)
	tc.PublishMsg(outerCtx, m)

	<-handlerCalled
}
