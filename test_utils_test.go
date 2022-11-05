package sdp

import (
	"context"
	"testing"
)

func TestRequestWithContext(t *testing.T) {
	type Request struct {
		RequestString string
	}

	type Response struct {
		ResponseString string
	}

	tc := TestConnection{
		RequestHandler: func(subject string, v, vPtr interface{}) error {
			if req, ok := v.(Request); ok {
				if resp, ok := vPtr.(*Response); ok {
					resp.ResponseString = req.RequestString
				}
			}

			return nil
		},
	}

	request := Request{
		RequestString: "foo",
	}

	response := Response{}

	err := tc.RequestWithContext(context.Background(), "test", request, &response)

	if err != nil {
		t.Error(err)
	}

	if response.ResponseString != request.RequestString {
		t.Errorf("expected response string to be %v, got %v", request.RequestString, response.ResponseString)
	}
}
