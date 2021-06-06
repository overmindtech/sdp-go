package sdp

import (
	"sync"
	"testing"
	"time"
)

type ResponseMessage struct {
	Subject string
	V       interface{}
}

// TestPublisher Used to mock a NATS connection for testing
type TestPublisher struct {
	Messages      []ResponseMessage
	messagesMutex *sync.Mutex
}

// Publish Test publish method, notes down the subject and the message
func (t *TestPublisher) Publish(subject string, v interface{}) error {
	if t.messagesMutex == nil {
		t.messagesMutex = &sync.Mutex{}
	}

	t.messagesMutex.Lock()
	t.Messages = append(t.Messages, ResponseMessage{
		Subject: subject,
		V:       v,
	})
	t.messagesMutex.Unlock()
	return nil
}

func TestResponseStart(t *testing.T) {
	rs := ResponseSender{
		ResponseInterval: (10 * time.Millisecond),
		ResponseSubject:  "responses",
	}

	tp := TestPublisher{
		Messages: make([]ResponseMessage, 0),
	}

	// Start sending responses
	rs.Start(&tp, "test")

	// Give it enough time for ~10 responses
	time.Sleep(100 * time.Millisecond)

	// Stop
	rs.Done()

	// Inspect what was sent
	if len(tp.Messages) <= 10 {
		t.Errorf("Expected <= 10 responses to be sent, found %v", len(tp.Messages))
	}

	// Make sure that the final message was a complation one
	finalMessage := tp.Messages[len(tp.Messages)-1]

	if finalResponse, ok := finalMessage.V.(*Response); ok {
		if finalResponse.State != Response_COMPLETE {
			t.Errorf("Expected final message state to be COMPLETE (1), found: %v", finalResponse.State)
		}
	} else {
		t.Errorf("Final message did not contain a valid response object. Message content type %T", finalMessage.V)
	}
}

func TestDefaultResponseInterval(t *testing.T) {
	rs := ResponseSender{}

	rs.Start(&TestPublisher{}, "")
	rs.Kill()

	if rs.ResponseInterval != DEFAULTRESPONSEINTERVAL {
		t.Fatal("Response sender interval failed to default")
	}
}
