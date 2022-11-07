package sdp

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	sync "sync"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type ResponseMessage struct {
	Subject string
	V       interface{}
}

// TestConnection Used to mock a NATS connection for testing
type TestConnection struct {
	Messages           []ResponseMessage
	Subscriptions      map[string][]nats.Handler
	messagesMutex      sync.Mutex
	subscriptionsMutex sync.Mutex
}

// Publish Test publish method, notes down the subject and the message
func (t *TestConnection) Publish(subject string, v interface{}) error {
	t.messagesMutex.Lock()
	t.Messages = append(t.Messages, ResponseMessage{
		Subject: subject,
		V:       v,
	})
	t.messagesMutex.Unlock()

	t.runHandlers(subject, v)

	return nil
}

func (t *TestConnection) Subscribe(subject string, cb nats.Handler) (*nats.Subscription, error) {
	t.subscriptionsMutex.Lock()
	defer t.subscriptionsMutex.Unlock()

	if t.Subscriptions == nil {
		t.Subscriptions = make(map[string][]nats.Handler)
	}

	t.Subscriptions[subject] = append(t.Subscriptions[subject], cb)

	return nil, nil
}

// RequestWithContext Simulates a request on the given subject, assigns a random
// response subject then calls the handler on the given subject, we are
// expecting the handler to be in the format: func(subject, reply string, o *obj)
func (t *TestConnection) RequestWithContext(ctx context.Context, subject string, v interface{}, vPtr interface{}) error {
	reply := randSeq(10)
	replies := make(chan interface{}, 128)

	t.subscriptionsMutex.Lock()
	handlers, ok := t.Subscriptions[subject]
	t.subscriptionsMutex.Unlock()

	if ok {
		// Subscribe to the reply subject
		t.Subscribe(reply, func(i interface{}) {
			replies <- i
		})

		// Run the handlers
		for _, handler := range handlers {
			switch h := handler.(type) {
			// Currently these are the only services that implement true
			// request-response patterns
			case func(subject, reply string, o *ReverseLinksRequest):
				req := v.(*ReverseLinksRequest)
				h(subject, reply, req)
			case func(subject, reply string, o *GatewayRequest):
				req := v.(*GatewayRequest)
				h(subject, reply, req)
			}
		}
	} else {
		return fmt.Errorf("no responders on subject %v", subject)
	}

	// Assign the first result to vPtr
	select {
	case reply, ok := <-replies:
		if ok {
			// Encode and decode again into the pointer given
			if m, ok := reply.(proto.Message); ok {
				b, err := proto.Marshal(m)

				if err != nil {
					return err
				}

				if vMsg, ok := vPtr.(proto.Message); ok {
					err = proto.Unmarshal(b, vMsg)

					if err != nil {
						return err
					}
				} else {
					return errors.New("vPtr was not a protobuf message type")
				}
			} else {
				return errors.New("response was not a protobuf type")
			}
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// runHandlers Runs the handlers for a given subject
func (t *TestConnection) runHandlers(subject string, object interface{}) {
	t.subscriptionsMutex.Lock()
	defer t.subscriptionsMutex.Unlock()

	handlers, ok := t.Subscriptions[subject]

	if ok {
		for _, handler := range handlers {
			switch v := handler.(type) {
			case func(*Item):
				i := object.(*Item)
				go v(i)
			case func(*Response):
				r := object.(*Response)
				go v(r)
			case func(*ItemRequest):
				r := object.(*ItemRequest)
				go v(r)
			case func(*CancelItemRequest):
				r := object.(*CancelItemRequest)
				go v(r)
			case func(*Reference):
				r := object.(*Reference)
				go v(r)
			case func(*ReverseLinksRequest):
				r := object.(*ReverseLinksRequest)
				go v(r)
			case func(*ReverseLinksResponse):
				r := object.(*ReverseLinksResponse)
				go v(r)
			case func(interface{}):
				go v(object)
			default:
				panic("unknown handler type")
			}
		}
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
