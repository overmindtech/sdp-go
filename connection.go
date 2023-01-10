package sdp

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

// EncodedConnection is an interface that allows messages to be published to it.
// In production this would always be filled by a *nats.EncodedConn, however in
// testing we will mock this with something that does nothing
type EncodedConnection interface {
	Publish(ctx context.Context, subj string, m proto.Message) error
	PublishMsg(ctx context.Context, msg *nats.Msg) error
	Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error)
	QueueSubscribe(subj, queue string, cb nats.MsgHandler) (*nats.Subscription, error)
	RequestMsg(ctx context.Context, msg *nats.Msg) (*nats.Msg, error)

	Status() nats.Status
	Stats() nats.Statistics
	LastError() error

	Drain() error
	Close()

	Underlying() *nats.Conn
	Drop()
}

type EncodedConnectionImpl struct {
	*nats.Conn
}

// assert interface implementation
var _ EncodedConnection = (*EncodedConnectionImpl)(nil)

func (ec *EncodedConnectionImpl) Publish(ctx context.Context, subj string, m proto.Message) error {
	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}

	msg := &nats.Msg{
		Subject: subj,
		Data:    data,
	}
	return ec.PublishMsg(ctx, msg)
}

func (ec *EncodedConnectionImpl) PublishMsg(ctx context.Context, msg *nats.Msg) error {
	InjectOtelTraceContext(ctx, msg)
	return ec.Conn.PublishMsg(msg)
}

// Subscribe Use NewMsgHandler to get a nats.MsgHandler with otel propagation and protobuf marshaling
func (ec *EncodedConnectionImpl) Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
	return ec.Conn.Subscribe(subj, cb)
}

// QueueSubscribe Use NewMsgHandler to get a nats.MsgHandler with otel propagation and protobuf marshaling
func (ec *EncodedConnectionImpl) QueueSubscribe(subj, queue string, cb nats.MsgHandler) (*nats.Subscription, error) {
	return ec.Conn.QueueSubscribe(subj, queue, cb)
}

func (ec *EncodedConnectionImpl) RequestMsg(ctx context.Context, msg *nats.Msg) (*nats.Msg, error) {
	InjectOtelTraceContext(ctx, msg)
	return ec.Conn.RequestMsgWithContext(ctx, msg)
}

func (ec *EncodedConnectionImpl) Drain() error {
	return ec.Conn.Drain()
}
func (ec *EncodedConnectionImpl) Close() {
	ec.Conn.Close()
}

func (ec *EncodedConnectionImpl) Status() nats.Status {
	return ec.Conn.Status()
}

func (ec *EncodedConnectionImpl) Stats() nats.Statistics {
	return ec.Conn.Stats()
}

func (ec *EncodedConnectionImpl) LastError() error {
	return ec.Conn.LastError()
}

func (ec *EncodedConnectionImpl) Underlying() *nats.Conn {
	return ec.Conn
}

// Drop Drops the underlying connection completely
func (ec *EncodedConnectionImpl) Drop() {
	ec.Conn = nil
}

// Unmarshal Does a proto.Unmarshal and logs errors in a consistent way
func Unmarshal(ctx context.Context, b []byte, m proto.Message) error {
	err := proto.Unmarshal(b, m)
	if err != nil {
		log.WithContext(ctx).Errorf("Error parsing message: %v", err)
		trace.SpanFromContext(ctx).SetStatus(codes.Error, fmt.Sprintf("Error parsing message: %v", err))
		return err
	}

	// Check for possible type mismatch. If the wong type was provided it
	// may have been able to partially decode the message, but there will be
	// some remaining unknown fields. If there are some, fail.
	if unk := m.ProtoReflect().GetUnknown(); unk != nil {
		err = fmt.Errorf("unmarshal to %T had unknown fields, likely a type mismatch. Unknowns: %v", m, unk)
		log.WithContext(ctx).Errorf("Error parsing message: %v", err)
		trace.SpanFromContext(ctx).SetStatus(codes.Error, fmt.Sprintf("Error parsing message: %v", err))
		return err
	}
	return nil
}

//go:generate go run genhandler.go Item
//go:generate go run genhandler.go ItemRequest
//go:generate go run genhandler.go ItemRequestError
//go:generate go run genhandler.go Response
//go:generate go run genhandler.go GatewayResponse
//go:generate go run genhandler.go ReverseLinksRequest
//go:generate go run genhandler.go CancelItemRequest
