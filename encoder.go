package sdp

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

// UnknownFieldsError This error is returned when a message is decoded with
// remaining unknown fields. This happens when a type was provided that is
// somewhat compatible with the actual message but not quite the tight one
type UnknownFieldsError struct {
	Message proto.Message
}

func (e UnknownFieldsError) Error() string {
	return fmt.Sprintf(
		"unmarshal to %T had unknown fields, likely a type mismatch. Unknowns: %v",
		e.Message,
		e.Message.ProtoReflect().GetUnknown(),
	)
}

// Unmarshal A version of proto.Unmarshal with additional error checking
func Unmarshal(b []byte, m proto.Message) error {
	err := proto.Unmarshal(b, m)

	if err != nil {
		return err
	}

	// Check for possible type mismatch. If the wong type was provided it
	// may have been able to partially decode the message, but there will be
	// some remaining unknown fields. If there are some, fail.
	if m.ProtoReflect().GetUnknown() != nil {
		return UnknownFieldsError{
			Message: m,
		}
	}

	return nil
}
