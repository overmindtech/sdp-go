package sdp

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var Encoder = SDPEncoder{}

var itemRequest = ItemRequest{
	Type:      "user",
	Method:    RequestMethod_FIND,
	LinkDepth: 10,
	Context:   "test",
}

var itemAttributes = ItemAttributes{
	AttrStruct: &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"foo": {
				Kind: &structpb.Value_StringValue{
					StringValue: "bar",
				},
			},
		},
	},
}

var metadata = Metadata{
	BackendName: "users",
	SourceRequest: &ItemRequest{
		Type:            "user",
		Method:          RequestMethod_FIND,
		Query:           "*",
		LinkDepth:       12,
		Context:         "testContext",
		ItemSubject:     "items",
		ResponseSubject: "responses",
	},
	Timestamp: timestamppb.Now(),
	BackendDuration: &durationpb.Duration{
		Seconds: 1,
		Nanos:   1,
	},
	BackendDurationPerItem: &durationpb.Duration{
		Seconds: 0,
		Nanos:   500,
	},
	BackendPackage: "test",
}

var item = Item{
	Type:            "user",
	UniqueAttribute: "name",
	Attributes:      &itemAttributes,
	Metadata:        &metadata,
}

var items = Items{
	Items: []*Item{
		&item,
	},
}

var reference = Reference{
	Type:                 "user",
	UniqueAttributeValue: "dylan",
	Context:              "test",
}

var itemRequestError = ItemRequestError{
	ErrorType:   ItemRequestError_OTHER,
	ErrorString: "uh oh",
	Context:     "test",
}

var response = Response{
	Context: "test",
	State:   Response_WORKING,
	NextUpdateIn: &durationpb.Duration{
		Seconds: 10,
		Nanos:   0,
	},
}

var messages = []proto.Message{
	&itemRequest,
	&itemAttributes,
	&metadata,
	&item,
	&items,
	&reference,
	&itemRequestError,
	&response,
}

// TestEncode Make sure that we can incode all of the message types without
// raising any errors
func TestEncode(t *testing.T) {
	for _, message := range messages {
		_, err := Encoder.Encode("testSubject", message)

		if err != nil {
			t.Error(err)
		}
	}
}

func TestResponse(t *testing.T) {
	_, err := Encoder.Encode("testSubject", &response)

	if err != nil {
		t.Error(err)
	}
}

var decodeTests = []struct {
	Message proto.Message
	Target  interface{}
}{
	{
		Message: &itemRequest,
		Target:  &ItemRequest{},
	},
	{
		Message: &itemAttributes,
		Target:  &ItemAttributes{},
	},
	{
		Message: &metadata,
		Target:  &Metadata{},
	},
	{
		Message: &item,
		Target:  &Item{},
	},
	{
		Message: &items,
		Target:  &Items{},
	},
	{
		Message: &reference,
		Target:  &Reference{},
	},
	{
		Message: &itemRequestError,
		Target:  &ItemRequestError{},
	},
	{
		Message: &response,
		Target:  &Response{},
	},
}

// TestDecode Make sure that we can decode all of the message
func TestDecode(t *testing.T) {
	for _, decTest := range decodeTests {
		// Marshal to binary
		b, err := proto.Marshal(decTest.Message)

		if err != nil {
			t.Fatal(err)
		}

		err = Encoder.Decode("testSubject", b, decTest.Target)

		if err != nil {
			t.Error(err)
		}
	}
}
