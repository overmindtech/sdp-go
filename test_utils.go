package sdp

import (
	sync "sync"

	"github.com/nats-io/nats.go"
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

// SendMessage Emulates a message being sent on a particular subject. The object
// passed should be the decoded content of the message i.e. the Item not the
// binary representation
func (t *TestConnection) SendMessage(subject string, object interface{}) {
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
			default:
				panic("unknown handler type")
			}
		}
	}
}
