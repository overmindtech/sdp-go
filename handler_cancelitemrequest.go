// Code generated by "genhandler CancelItemRequest"; DO NOT EDIT

package sdp

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/trace"
)

func NewCancelItemRequestHandler(spanName string, h func(ctx context.Context, i *CancelItemRequest), spanOpts ...trace.SpanStartOption) nats.MsgHandler {
	return NewOtelExtractingHandler(spanName, func(ctx context.Context, m *nats.Msg) {
		var i CancelItemRequest
		err := Unmarshal(ctx, m.Data, &i)
		if err != nil {
			return
		}
		h(ctx, &i)
	})
}

func NewRawCancelItemRequestHandler(spanName string, h func(ctx context.Context, m *nats.Msg, i *CancelItemRequest), spanOpts ...trace.SpanStartOption) nats.MsgHandler {
	return NewOtelExtractingHandler(spanName, func(ctx context.Context, m *nats.Msg) {
		var i CancelItemRequest
		err := Unmarshal(ctx, m.Data, &i)
		if err != nil {
			return
		}
		h(ctx, m, &i)
	})
}