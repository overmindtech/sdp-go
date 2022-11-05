package sdp

import (
	"context"
	sync "sync"

	"github.com/nats-io/nats.go"
)

type ResponseMessage struct {
	Subject string
	V       interface{}
}

// TestConnection Used to mock a NATS connection for testing
type TestConnection struct {
	Messages      []ResponseMessage
	Subscriptions map[string][]nats.Handler
	// RequestHandler Executed when a user runs RequestWithContext
	RequestHandler     func(subject string, v interface{}, vPtr interface{}) error
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

// RequestWithContext Simulates a request on the given subject, the input v is
// the sent data, and received data will be added to vPtr. This ha handled by
func (t *TestConnection) RequestWithContext(ctx context.Context, subject string, v interface{}, vPtr interface{}) error {
	return t.RequestHandler(subject, v, vPtr)
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
			default:
				panic("unknown handler type")
			}
		}
	}
}
