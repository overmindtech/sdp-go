// Code generated by "genhandler ReverseLinksRequest"; DO NOT EDIT

package sdp

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/trace"
)

func NewReverseLinksRequestHandler(spanName string, h func(ctx context.Context, i *ReverseLinksRequest), spanOpts ...trace.SpanStartOption) nats.MsgHandler {
	return NewOtelExtractingHandler(spanName, func(ctx context.Context, m *nats.Msg) {
		var i ReverseLinksRequest
		err := Unmarshal(ctx, m.Data, &i)
		if err != nil {
			return
		}
		h(ctx, &i)
	})
}

func NewRawReverseLinksRequestHandler(spanName string, h func(ctx context.Context, m *nats.Msg, i *ReverseLinksRequest), spanOpts ...trace.SpanStartOption) nats.MsgHandler {
	return NewOtelExtractingHandler(spanName, func(ctx context.Context, m *nats.Msg) {
		var i ReverseLinksRequest
		err := Unmarshal(ctx, m.Data, &i)
		if err != nil {
			return
		}
		h(ctx, m, &i)
	})
}