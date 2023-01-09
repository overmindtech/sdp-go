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
	Subscriptions      map[string][]nats.MsgHandler
	messagesMutex      sync.Mutex
	subscriptionsMutex sync.Mutex
}

// assert interface implementation
var _ EncodedConnection = (*TestConnection)(nil)

// Publish Test publish method, notes down the subject and the message
func (t *TestConnection) Publish(ctx context.Context, subj string, m proto.Message) error {
	t.messagesMutex.Lock()
	t.Messages = append(t.Messages, ResponseMessage{
		Subject: subj,
		V:       m,
	})
	t.messagesMutex.Unlock()

	t.runHandlers(subj, m)

	return nil
}

// PublishMsg Test publish method, notes down the subject and the message
func (t *TestConnection) PublishMsg(ctx context.Context, msg *nats.Msg) error {
	t.messagesMutex.Lock()
	t.Messages = append(t.Messages, ResponseMessage{
		Subject: msg.Subject,
		V:       msg.Data,
	})
	t.messagesMutex.Unlock()

	t.runHandlers(msg.Subject, msg.Data)

	return nil
}

func (t *TestConnection) Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
	t.subscriptionsMutex.Lock()
	defer t.subscriptionsMutex.Unlock()

	if t.Subscriptions == nil {
		t.Subscriptions = make(map[string][]nats.MsgHandler)
	}

	t.Subscriptions[subj] = append(t.Subscriptions[subj], cb)

	return nil, nil
}

func (t *TestConnection) QueueSubscribe(subj, queue string, cb nats.MsgHandler) (*nats.Subscription, error) {
	panic("TODO")
}

// RequestWithContext Simulates a request on the given subject, assigns a random
// response subject then calls the handler on the given subject, we are
// expecting the handler to be in the format: func(subject, reply string, o *obj)
func (t *TestConnection) Request(ctx context.Context, subj string, in proto.Message, out proto.Message) error {
	panic("TODO")
	// reply := randSeq(10)
	// replies := make(chan *nats.Msg, 128)

	// handlers, ok := func() ([]nats.MsgHandler, bool) {
	// 	t.subscriptionsMutex.Lock()
	// 	defer t.subscriptionsMutex.Unlock()
	// 	handlers, ok := t.Subscriptions[subj]
	// 	return handlers, ok
	// }()

	// if ok {
	// 	// Subscribe to the reply subject
	// 	t.Subscribe(reply, func(msg *nats.Msg) {
	// 		replies <- msg
	// 	})

	// 	// Run the handlers

	// 	data, err := proto.Marshal(in)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	for _, handler := range handlers {
	// 		// create a new Msg for each handler
	// 		inMsg := nats.Msg{
	// 			Subject: subj,
	// 			Data:    data,
	// 		}
	// 		handler(&inMsg)
	// 	}
	// } else {
	// 	return fmt.Errorf("no responders on subject %v", subj)
	// }

	// // Assign the first result to vPtr
	// select {
	// case reply, ok := <-replies:
	// 	if ok {
	// 		// Encode and decode again into the pointer given
	// 		if m, ok := proto.Unmarshal(reply.Data); ok {
	// 			b, err := proto.Marshal(m)

	// 			if err != nil {
	// 				return err
	// 			}

	// 			if vMsg, ok := vPtr.(proto.Message); ok {
	// 				err = proto.Unmarshal(b, vMsg)

	// 				if err != nil {
	// 					return err
	// 				}
	// 			} else {
	// 				return errors.New("vPtr was not a protobuf message type")
	// 			}
	// 		} else {
	// 			return errors.New("response was not a protobuf type")
	// 		}
	// 	}
	// case <-ctx.Done():
	// 	return ctx.Err()
	// }

	// return nil
}

// RequestWithContext Simulates a request on the given subject, assigns a random
// response subject then calls the handler on the given subject, we are
// expecting the handler to be in the format: func(msg *nats.Msg)
func (t *TestConnection) RequestMsg(ctx context.Context, msg *nats.Msg) (*nats.Msg, error) {
	replySubject := randSeq(10)
	replies := make(chan interface{}, 128)

	t.subscriptionsMutex.Lock()
	handlers, ok := t.Subscriptions[msg.Subject]
	t.subscriptionsMutex.Unlock()

	if ok {
		// Subscribe to the reply subject
		t.Subscribe(replySubject, func(msg *nats.Msg) {
			replies <- msg
		})

		// Run the handlers
		for _, handler := range handlers {
			handler(msg)
		}
	} else {
		return nil, fmt.Errorf("no responders on subject %v", msg.Subject)
	}

	// Return the first result
	select {
	case reply, ok := <-replies:
		if ok {
			// Encode and decode again into the pointer given
			if m, ok := reply.([]byte); ok {
				return &nats.Msg{
					Subject: replySubject,
					Data:    m,
				}, nil
			} else {
				return nil, errors.New("response was not a []byte")
			}
		} else {
			return nil, errors.New("no replies")
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Status Always returns nats.CONNECTED
func (n *TestConnection) Status() nats.Status {
	return nats.CONNECTED
}

// Stats Always returns empty/zero nats.Statistics
func (n *TestConnection) Stats() nats.Statistics {
	return nats.Statistics{}
}

// LastError Always returns nil
func (n *TestConnection) LastError() error {
	return nil
}

// Drain Always returns nil
func (n *TestConnection) Drain() error {
	return nil
}

// Close Does nothing
func (n *TestConnection) Close() {}

// Underlying Always returns nil
func (n *TestConnection) Underlying() *nats.Conn {
	return nil
}

// Drop Does nothing
func (n *TestConnection) Drop() {}

// runHandlers Runs the handlers for a given subject
func (t *TestConnection) runHandlers(subject string, object interface{}) {
	panic("TODO")
	// t.subscriptionsMutex.Lock()
	// defer t.subscriptionsMutex.Unlock()

	// handlers, ok := t.Subscriptions[subject]

	// if ok {
	// 	for _, handler := range handlers {
	// 		switch v := handler.(type) {
	// 		case func(*Item):
	// 			i := object.(*Item)
	// 			go v(i)
	// 		case func(*Response):
	// 			r := object.(*Response)
	// 			go v(r)
	// 		case func(*ItemRequest):
	// 			r := object.(*ItemRequest)
	// 			go v(r)
	// 		case func(*CancelItemRequest):
	// 			r := object.(*CancelItemRequest)
	// 			go v(r)
	// 		case func(*Reference):
	// 			r := object.(*Reference)
	// 			go v(r)
	// 		case func(*ReverseLinksRequest):
	// 			r := object.(*ReverseLinksRequest)
	// 			go v(r)
	// 		case func(*ReverseLinksResponse):
	// 			r := object.(*ReverseLinksResponse)
	// 			go v(r)
	// 		case func(*GatewayRequest):
	// 			r := object.(*GatewayRequest)
	// 			go v(r)
	// 		case func(*GatewayResponse):
	// 			r := object.(*GatewayResponse)
	// 			go v(r)
	// 		case func(interface{}):
	// 			go v(object)
	// 		default:
	// 			panic(fmt.Sprintf("unknown handler type: %v", v))
	// 		}
	// 	}
	// }
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
