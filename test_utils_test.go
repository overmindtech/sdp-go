package sdp

import (
	"context"
	"testing"
)

func TestRequestWithContext(t *testing.T) {
	tc := TestConnection{}

	// Create the responder
	tc.Subscribe("test", func(subject, reply string, req *ReverseLinksRequest) {
		tc.Publish(reply, &ReverseLinksResponse{
			LinkedItemRequests: []*ItemRequest{},
			Error:              "testing",
		})
	})

	request := ReverseLinksRequest{}

	response := ReverseLinksResponse{}

	err := tc.RequestWithContext(context.Background(), "test", &request, &response)

	if err != nil {
		t.Error(err)
	}

	if response.Error != "testing" {
		t.Errorf("expected error to be 'testing', got '%v'", response.Error)
	}
}
