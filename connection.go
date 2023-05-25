package sdp

import (
	"context"
	"fmt"
	reflect "reflect"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
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

func recordMessage(ctx context.Context, name, subj, typ, msg string) {
	log.WithContext(ctx).WithFields(log.Fields{
		"msg type": typ,
		"subj":     subj,
		"msg":      msg,
	}).Trace(name)
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(
		attribute.String("om.sdp.subject", subj),
		attribute.String("om.sdp.message", msg),
	))
}

func (ec *EncodedConnectionImpl) Publish(ctx context.Context, subj string, m proto.Message) error {
	// TODO: protojson.Format is pretty expensive, replace with summarized data
	recordMessage(ctx, "Publish", subj, fmt.Sprint(reflect.TypeOf(m)), protojson.Format(m))

	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}

	msg := &nats.Msg{
		Subject: subj,
		Data:    data,
	}
	InjectOtelTraceContext(ctx, msg)
	return ec.Conn.PublishMsg(msg)
}

func (ec *EncodedConnectionImpl) PublishMsg(ctx context.Context, msg *nats.Msg) error {
	recordMessage(ctx, "Publish", msg.Subject, "[]byte", "binary")

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
	recordMessage(ctx, "RequestMsg", msg.Subject, "[]byte", "binary")
	InjectOtelTraceContext(ctx, msg)
	reply, err := ec.Conn.RequestMsgWithContext(ctx, msg)

	if err != nil {
		recordMessage(ctx, "RequestMsg Error", msg.Subject, fmt.Sprint(reflect.TypeOf(err)), err.Error())
	} else {
		recordMessage(ctx, "RequestMsg Reply", msg.Subject, "[]byte", "binary")
	}
	return reply, err
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
		recordMessage(ctx, "Unmarshal err", "unknown", fmt.Sprint(reflect.TypeOf(err)), err.Error())
		log.WithContext(ctx).Errorf("Error parsing message: %v", err)
		trace.SpanFromContext(ctx).SetStatus(codes.Error, fmt.Sprintf("Error parsing message: %v", err))
		return err
	}

	// Check for possible type mismatch. If the wong type was provided it
	// may have been able to partially decode the message, but there will be
	// some remaining unknown fields. If there are some, fail.
	if unk := m.ProtoReflect().GetUnknown(); unk != nil {
		err = fmt.Errorf("unmarshal to %T had unknown fields, likely a type mismatch. Unknowns: %v", m, unk)
		recordMessage(ctx, "Unmarshal unknown", "unknown", fmt.Sprint(reflect.TypeOf(m)), protojson.Format(m))
		log.WithContext(ctx).Errorf("Error parsing message: %v", err)
		trace.SpanFromContext(ctx).SetStatus(codes.Error, fmt.Sprintf("Error parsing message: %v", err))
		return err
	}

	recordMessage(ctx, "Unmarshal", "unknown", fmt.Sprint(reflect.TypeOf(m)), protojson.Format(m))
	return nil
}

//go:generate go run genhandler.go Item

//go:generate go run genhandler.go Query
//go:generate go run genhandler.go QueryResponse
//go:generate go run genhandler.go QueryError
//go:generate go run genhandler.go CancelQuery
//go:generate go run genhandler.go UndoQuery

//go:generate go run genhandler.go GatewayResponse
//go:generate go run genhandler.go Response
//go:generate go run genhandler.go ReverseLinksRequest
