package sdp

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/proto"
)

var ENCODER = SDPEncoder{}

// UnknownFieldsError This error is returned when a message is decoded with
// remaining unknown fields. This happens when a type was provided that is
// somewhat compatible with the actual message but not quite the tight one
type UnknownFieldsError struct {
	TargetPtr interface{}
	Message   proto.Message
}

func (e UnknownFieldsError) Error() string {
	return fmt.Sprintf(
		"unmarshal to %T had unknown fields, likely a type mismatch. Unknowns: %v",
		e.TargetPtr,
		e.Message.ProtoReflect().GetUnknown(),
	)
}

// SDPEncoder Provides encoding and decoding functionality for SDP objects to
// convert to and from binary for transmission over the NATS network
type SDPEncoder struct{}

// Encode Encodes an SDP object to a binary protobuf format for sending on NATS
func (e *SDPEncoder) Encode(subject string, v interface{}) ([]byte, error) {
	// If the interface is a protocol buffer message then we are capable of
	// marshalling it
	if msg, ok := v.(proto.Message); ok {
		return proto.Marshal(msg)
	}

	return nil, fmt.Errorf("could not encode to protobuf: %v", v)
}

// Decode Decodes a NATS binary message, assuming that it's a valid SDP protobuf
// message
func (e *SDPEncoder) Decode(subject string, data []byte, vPtr interface{}) error {
	if msg, ok := vPtr.(proto.Message); ok {
		err := proto.Unmarshal(data, msg)

		if err != nil {
			return err
		}

		// Check for possible type mismatch. If the wong type was provided it
		// may have been able to partially decode the message, but there will be
		// some remaining unknown fields. If there are some, fail.
		if msg.ProtoReflect().GetUnknown() != nil {
			return UnknownFieldsError{
				TargetPtr: vPtr,
				Message:   msg,
			}
		}

		// This means it worked
		return err
	}

	return fmt.Errorf("cannot decode SDP message into variable of type %T, must be a proto.Message", vPtr)
}

func BuildMsg(ctx context.Context, subject string, in proto.Message) (nats.Msg, error) {
	data, err := proto.Marshal(in)
	if err != nil {
		return nats.Msg{}, err
	}
	headers := make(nats.Header)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(headers))
	return nats.Msg{
		Subject: subject,
		Header:  headers,
		Data:    data,
	}, nil
}

func ParseMsg(ctx context.Context, in *nats.Msg, out proto.Message) (context.Context, error) {
	err := proto.Unmarshal(in.Data, out)

	if err != nil {
		return ctx, err
	}

	// Check for possible type mismatch. If the wong type was provided it
	// may have been able to partially decode the message, but there will be
	// some remaining unknown fields. If there are some, fail.
	if out.ProtoReflect().GetUnknown() != nil {
		return ctx, UnknownFieldsError{
			Message: out,
		}
	}

	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(in.Header))

	// This means it worked
	return ctx, err
}
