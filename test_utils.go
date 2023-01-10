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
	Messages      []ResponseMessage
	messagesMutex sync.Mutex

	Subscriptions      map[string][]nats.MsgHandler
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

	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	msg := nats.Msg{
		Subject: subj,
		Data:    data,
	}
	t.runHandlers(&msg)

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

	t.runHandlers(msg)

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

// RequestMsg Simulates a request on the given subject, assigns a random
// response subject then calls the handler on the given subject, we are
// expecting the handler to be in the format: func(msg *nats.Msg)
func (t *TestConnection) RequestMsg(ctx context.Context, msg *nats.Msg) (*nats.Msg, error) {
	replySubject := randSeq(10)
	msg.Reply = replySubject
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
			if m, ok := reply.(*nats.Msg); ok {
				return &nats.Msg{
					Subject: replySubject,
					Data:    m.Data,
				}, nil
			} else {
				return nil, fmt.Errorf("reply was not a *nats.Msg, but a %T", reply)
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
	return &nats.Conn{}
}

// Drop Does nothing
func (n *TestConnection) Drop() {}

// runHandlers Runs the handlers for a given subject
func (t *TestConnection) runHandlers(msg *nats.Msg) {
	t.subscriptionsMutex.Lock()
	defer t.subscriptionsMutex.Unlock()

	handlers, ok := t.Subscriptions[msg.Subject]

	if ok {
		for _, handler := range handlers {
			handler(msg)
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
