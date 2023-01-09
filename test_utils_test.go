package sdp

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
)

func TestRequestWithContext(t *testing.T) {
	tc := TestConnection{}

	// Create the responder
	tc.Subscribe("test", NewRawMsgHandler("test", func(ctx context.Context, msg *nats.Msg, req *ReverseLinksRequest) {
		tc.Publish(ctx, msg.Reply, &ReverseLinksResponse{
			LinkedItemRequests: []*ItemRequest{},
			Error:              "testing",
		})
	}))

	request := ReverseLinksRequest{}

	response := ReverseLinksResponse{}

	err := tc.Request(context.Background(), "test", &request, &response)

	if err != nil {
		t.Error(err)
	}

	if response.Error != "testing" {
		t.Errorf("expected error to be 'testing', got '%v'", response.Error)
	}
}
