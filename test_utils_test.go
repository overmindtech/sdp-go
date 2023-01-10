package sdp

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

func TestRequest(t *testing.T) {
	tc := TestConnection{}

	// Create the responder
	tc.Subscribe("test", func(msg *nats.Msg) {
		tc.Publish(context.Background(), msg.Reply, &ReverseLinksResponse{
			LinkedItemRequests: []*ItemRequest{},
			Error:              "testing",
		})
	})

	request := ReverseLinksRequest{}

	data, err := proto.Marshal(&request)
	if err != nil {
		t.Fatal(err)
	}
	msg := nats.Msg{
		Subject: "test",
		Data:    data,
	}
	replyMsg, err := tc.RequestMsg(context.Background(), &msg)
	if err != nil {
		t.Fatal(err)
	}

	response := ReverseLinksResponse{}
	err = proto.Unmarshal(replyMsg.Data, &response)

	if err != nil {
		t.Error(err)
	}

	if response.Error != "testing" {
		t.Errorf("expected error to be 'testing', got '%v'", response.Error)
	}
}
