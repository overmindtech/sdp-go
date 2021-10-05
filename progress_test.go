package sdp

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
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

func TestResponseSenderDone(t *testing.T) {
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

func TestResponseSenderError(t *testing.T) {
	rs := ResponseSender{
		ResponseInterval: (10 * time.Millisecond),
		ResponseSubject:  "responses",
	}

	tp := TestPublisher{
		Messages: make([]ResponseMessage, 0),
	}

	// Start sending responses
	rs.Start(&tp, "test")

	// Give it enough time for >10 responses
	time.Sleep(120 * time.Millisecond)

	// Stop
	rs.Error(&ItemRequestError{
		ErrorType:   ItemRequestError_OTHER,
		ErrorString: "Unknown",
		Context:     "test",
	})

	// Inspect what was sent
	if len(tp.Messages) <= 10 {
		t.Errorf("Expected <= 10 responses to be sent, found %v", len(tp.Messages))
	}

	// Make sure that the final message was a complation one
	finalMessage := tp.Messages[len(tp.Messages)-1]

	if finalResponse, ok := finalMessage.V.(*Response); ok {
		if finalResponse.State != Response_ERROR {
			t.Errorf("Expected final message state to be ERROR, found: %v", finalResponse.State)
		}

		if finalResponse.Error.Error() != "Unknown" {
			t.Errorf("Expected error string to be 'Unknown', got '%v'", finalResponse.Error.Error())
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

// Test object used for validation that metrics are coming out properly
type ExpectedMetrics struct {
	Working    int
	Stalled    int
	Complete   int
	Failed     int
	Responders int
}

// Validate Checks that metrics are as expected and returns an error if not
func (em ExpectedMetrics) Validate(rp *RequestProgress) error {
	if x := rp.NumWorking(); x != em.Working {
		return fmt.Errorf("Expected NumWorking to be %v, got %v", em.Working, x)
	}
	if x := rp.NumStalled(); x != em.Stalled {
		return fmt.Errorf("Expected NumStalled to be %v, got %v", em.Stalled, x)
	}
	if x := rp.NumComplete(); x != em.Complete {
		return fmt.Errorf("Expected NumComplete to be %v, got %v", em.Complete, x)
	}
	if x := rp.NumFailed(); x != em.Failed {
		return fmt.Errorf("Expected NumFailed to be %v, got %v", em.Failed, x)
	}
	if x := rp.NumResponders(); x != em.Responders {
		return fmt.Errorf("Expected NumResponders to be %v, got %v", em.Responders, x)
	}

	return nil
}

func TestRequestProgressNormal(t *testing.T) {
	rp := NewRequestProgress()

	// Make sure that the details are correct initially
	var expected ExpectedMetrics

	expected = ExpectedMetrics{
		Working:    0,
		Stalled:    0,
		Complete:   0,
		Failed:     0,
		Responders: 0,
	}

	if err := expected.Validate(rp); err != nil {
		t.Error(err)
	}

	t.Run("Processing initial response", func(t *testing.T) {
		// Test the initial response
		rp.ProcessResponse(&Response{
			Responder:    "test1",
			State:        Response_WORKING,
			NextUpdateIn: durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    1,
			Stalled:    0,
			Complete:   0,
			Failed:     0,
			Responders: 1,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	t.Run("Processing another context also responding", func(t *testing.T) {
		// Then another context starts working
		rp.ProcessResponse(&Response{
			Responder:    "test2",
			State:        Response_WORKING,
			NextUpdateIn: durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    2,
			Stalled:    0,
			Complete:   0,
			Failed:     0,
			Responders: 2,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	t.Run("When some are complete and some are not", func(t *testing.T) {
		time.Sleep(5 * time.Millisecond)

		// test 1 still working
		rp.ProcessResponse(&Response{
			Responder:    "test1",
			State:        Response_WORKING,
			NextUpdateIn: durationpb.New(10 * time.Millisecond),
		})

		// Test 2 finishes
		rp.ProcessResponse(&Response{
			Responder:    "test2",
			State:        Response_COMPLETE,
			NextUpdateIn: durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    1,
			Stalled:    0,
			Complete:   1,
			Failed:     0,
			Responders: 2,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	t.Run("When the final responder finishes", func(t *testing.T) {
		time.Sleep(5 * time.Millisecond)

		// Test 1 finishes
		rp.ProcessResponse(&Response{
			Responder:    "test1",
			State:        Response_COMPLETE,
			NextUpdateIn: durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    0,
			Stalled:    0,
			Complete:   2,
			Failed:     0,
			Responders: 2,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	if rp.allDone() == false {
		t.Error("expected allDone() to be true")
	}
}

func TestRequestProgressStalled(t *testing.T) {
	rp := NewRequestProgress()

	// Make sure that the details are correct initially
	var expected ExpectedMetrics

	t.Run("Processing the initial response", func(t *testing.T) {
		// Test the initial response
		rp.ProcessResponse(&Response{
			Responder:    "test1",
			State:        Response_WORKING,
			NextUpdateIn: durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    1,
			Stalled:    0,
			Complete:   0,
			Failed:     0,
			Responders: 1,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	t.Run("After a responder has stalled", func(t *testing.T) {
		// Wait long enough for the thing to be marked as stalled
		time.Sleep(12 * time.Millisecond)

		expected = ExpectedMetrics{
			Working:    0,
			Stalled:    1,
			Complete:   0,
			Failed:     0,
			Responders: 1,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}

		if _, ok := rp.Responders["test1"]; !ok {
			t.Error("Could not get responder for context test1")
		}
	})

	t.Run("After a responder recovers from a stall", func(t *testing.T) {
		// See if it will un-stall itself
		rp.ProcessResponse(&Response{
			Responder:    "test1",
			State:        Response_COMPLETE,
			NextUpdateIn: durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    0,
			Stalled:    0,
			Complete:   1,
			Failed:     0,
			Responders: 1,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	if rp.allDone() == false {
		t.Error("expected allDone() to be true")
	}
}

func TestRequestProgressError(t *testing.T) {
	rp := NewRequestProgress()

	// Make sure that the details are correct initially
	var expected ExpectedMetrics

	t.Run("Processing the initial response", func(t *testing.T) {
		// Test the initial response
		rp.ProcessResponse(&Response{
			Responder:    "test1",
			State:        Response_WORKING,
			NextUpdateIn: durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    1,
			Stalled:    0,
			Complete:   0,
			Failed:     0,
			Responders: 1,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	t.Run("After a responder has failed", func(t *testing.T) {
		rp.ProcessResponse(&Response{
			Responder: "test1",
			State:     Response_ERROR,
			Error: &ItemRequestError{
				ErrorType:   ItemRequestError_NOCONTEXT,
				ErrorString: "Context X not found",
				Context:     "X",
			},
		})

		expected = ExpectedMetrics{
			Working:    0,
			Stalled:    0,
			Complete:   0,
			Failed:     1,
			Responders: 1,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	t.Run("Ensuring that a failed responder does not get marked as stalled", func(t *testing.T) {
		time.Sleep(12 * time.Millisecond)

		expected = ExpectedMetrics{
			Working:    0,
			Stalled:    0,
			Complete:   0,
			Failed:     1,
			Responders: 1,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	if rp.allDone() == false {
		t.Error("expected allDone() to be true")
	}
}
