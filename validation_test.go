package sdp

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestValidateItem(t *testing.T) {
	t.Run("item is fine", func(t *testing.T) {
		err := newItem().Validate()

		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Item is nil", func(t *testing.T) {
		var i *Item
		err := i.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("item has empty Type", func(t *testing.T) {
		i := newItem()

		i.Type = ""

		err := i.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("item has empty UniqueAttribute", func(t *testing.T) {
		i := newItem()

		i.UniqueAttribute = ""

		err := i.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("item has nil Attributes", func(t *testing.T) {
		i := newItem()

		i.Attributes = nil

		err := i.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("item has empty Context", func(t *testing.T) {
		i := newItem()

		i.Context = ""

		err := i.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("item has empty UniqueAttributeValue", func(t *testing.T) {
		i := newItem()

		i.Attributes.Set(i.UniqueAttribute, "")

		err := i.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestValidateReference(t *testing.T) {
	t.Run("Reference is fine", func(t *testing.T) {
		r := newReference()

		err := r.Validate()

		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Reference is nil", func(t *testing.T) {
		var r *Reference

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("reference has empty Type", func(t *testing.T) {
		r := newReference()

		r.Type = ""

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("reference has empty UniqueAttributeValue", func(t *testing.T) {
		r := newReference()

		r.UniqueAttributeValue = ""

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("reference has empty Context", func(t *testing.T) {
		r := newReference()

		r.Context = ""

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestValidateEdge(t *testing.T) {
	t.Run("Edge is fine", func(t *testing.T) {
		e := newEdge()

		err := e.Validate()

		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Edge has nil From", func(t *testing.T) {
		e := newEdge()

		e.From = nil

		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("Edge has nil To", func(t *testing.T) {
		e := newEdge()

		e.To = nil

		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("Edge has invalid From", func(t *testing.T) {
		e := newEdge()

		e.From.Type = ""

		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("Edge has invalid To", func(t *testing.T) {
		e := newEdge()

		e.To.Context = ""

		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestValidateReverseLinksRequest(t *testing.T) {
	t.Run("ReverseLinksRequest is fine", func(t *testing.T) {
		r := newReverseLinksRequest()

		err := r.Validate()

		if err != nil {
			t.Error(err)
		}
	})

	t.Run("ReverseLinksRequest is nil", func(t *testing.T) {
		var r *ReverseLinksRequest

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("ReverseLinksRequest has nil Item", func(t *testing.T) {
		r := newReverseLinksRequest()

		r.Item = nil

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("ReverseLinksRequest has invalid Item", func(t *testing.T) {
		r := newReverseLinksRequest()

		r.Item.Type = ""

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestValidateResponse(t *testing.T) {
	t.Run("Response is fine", func(t *testing.T) {
		r := newResponse()

		err := r.Validate()

		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Response is nil", func(t *testing.T) {
		var r *Response

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("Response has empty Responder", func(t *testing.T) {
		r := newResponse()
		r.Responder = ""

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("Response has empty ItemRequestUUID", func(t *testing.T) {
		r := newResponse()
		r.ItemRequestUUID = nil

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestValidateItemRequestError(t *testing.T) {
	t.Run("ItemRequestError is fine", func(t *testing.T) {
		e := newItemRequestError()

		err := e.Validate()

		if err != nil {
			t.Error(err)
		}
	})

	t.Run("ItemRequestError is nil", func(t *testing.T) {

	})

	t.Run("ItemRequestError has empty ItemRequestUUID", func(t *testing.T) {
		e := newItemRequestError()
		e.ItemRequestUUID = nil
		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("ItemRequestError has empty ErrorString", func(t *testing.T) {
		e := newItemRequestError()
		e.ErrorString = ""
		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("ItemRequestError has empty Context", func(t *testing.T) {
		e := newItemRequestError()
		e.Context = ""
		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("ItemRequestError has empty SourceName", func(t *testing.T) {
		e := newItemRequestError()
		e.SourceName = ""
		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("ItemRequestError has empty ItemType", func(t *testing.T) {
		e := newItemRequestError()
		e.ItemType = ""
		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("ItemRequestError has empty ResponderName", func(t *testing.T) {
		e := newItemRequestError()
		e.ResponderName = ""
		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestValidateItemRequest(t *testing.T) {
	t.Run("ItemRequest is fine", func(t *testing.T) {
		r := newItemRequest()

		err := r.Validate()

		if err != nil {
			t.Error(err)
		}
	})

	t.Run("ItemRequest is nil", func(t *testing.T) {

	})

	t.Run("ItemRequest has empty Type", func(t *testing.T) {
		r := newItemRequest()
		r.Type = ""
		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}

	})

	t.Run("ItemRequest has empty Context", func(t *testing.T) {
		r := newItemRequest()
		r.Context = ""
		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}

	})

	t.Run("Response has empty UUID", func(t *testing.T) {
		r := newItemRequest()
		r.UUID = nil
		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}

	})

	t.Run("ItemRequest cannot have empty Query when method is Get", func(t *testing.T) {
		r := newItemRequest()
		r.Method = RequestMethod_GET
		r.Query = ""
		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

}

func newItemRequest() *ItemRequest {
	u := uuid.New()

	return &ItemRequest{
		Type:            "person",
		Method:          RequestMethod_GET,
		Query:           "Dylan",
		LinkDepth:       1,
		Context:         "global",
		UUID:            u[:],
		Timeout:         durationpb.New(time.Second),
		IgnoreCache:     false,
		ItemSubject:     "return.items.1",
		ResponseSubject: "return.responses.1",
		ErrorSubject:    "return.errors.1",
	}
}

func newItemRequestError() *ItemRequestError {
	u := uuid.New()

	return &ItemRequestError{
		ItemRequestUUID: u[:],
		ErrorType:       ItemRequestError_OTHER,
		ErrorString:     "bad",
		Context:         "global",
		SourceName:      "test-source",
		ItemType:        "test",
		ResponderName:   "test-responder",
	}
}

func newResponse() *Response {
	u := uuid.New()

	return &Response{
		Responder:       "foo",
		State:           ResponderState_WORKING,
		NextUpdateIn:    durationpb.New(time.Second),
		ItemRequestUUID: u[:],
	}
}

func newReverseLinksRequest() *ReverseLinksRequest {
	return &ReverseLinksRequest{
		Item:    newReference(),
		Timeout: durationpb.New(time.Second),
	}
}

func newEdge() *Edge {
	return &Edge{
		From: newReference(),
		To:   newReference(),
	}
}

func newReference() *Reference {
	return &Reference{
		Type:                 "person",
		UniqueAttributeValue: "Dylan",
		Context:              "global",
	}
}

func newItem() *Item {
	return &Item{
		Type:               "user",
		UniqueAttribute:    "name",
		Context:            "test",
		LinkedItemRequests: []*ItemRequest{},
		LinkedItems:        []*Reference{},
		Attributes: &ItemAttributes{
			AttrStruct: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name": {
						Kind: &structpb.Value_StringValue{
							StringValue: "bar",
						},
					},
				},
			},
		},
		Metadata: &Metadata{
			SourceName: "users",
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
			SourceDuration: &durationpb.Duration{
				Seconds: 1,
				Nanos:   1,
			},
			SourceDurationPerItem: &durationpb.Duration{
				Seconds: 0,
				Nanos:   500,
			},
		},
	}
}
