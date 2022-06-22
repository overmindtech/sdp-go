package sdp

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestResponseNilPublisher(t *testing.T) {
	rs := ResponseSender{
		ResponseInterval: (10 * time.Millisecond),
		ResponseSubject:  "responses",
	}

	// Start sending responses with a nil connection, should not panic
	rs.Start(nil, "test")

	// Give it enough time for ~10 responses
	time.Sleep(100 * time.Millisecond)

	// Stop
	rs.Done()
}

func TestResponseSenderDone(t *testing.T) {
	rs := ResponseSender{
		ResponseInterval: (10 * time.Millisecond),
		ResponseSubject:  "responses",
	}

	tp := TestConnection{
		Messages: make([]ResponseMessage, 0),
	}

	// Start sending responses
	rs.Start(&tp, "test")

	// Give it enough time for ~10 responses
	time.Sleep(100 * time.Millisecond)

	// Stop
	rs.Done()

	// Inspect what was sent
	tp.messagesMutex.Lock()
	if len(tp.Messages) <= 10 {
		t.Errorf("Expected <= 10 responses to be sent, found %v", len(tp.Messages))
	}

	// Make sure that the final message was a completion one
	finalMessage := tp.Messages[len(tp.Messages)-1]
	tp.messagesMutex.Unlock()

	if finalResponse, ok := finalMessage.V.(*Response); ok {
		if finalResponse.State != ResponderState_COMPLETE {
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

	tp := TestConnection{
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
	tp.messagesMutex.Lock()
	if len(tp.Messages) <= 10 {
		t.Errorf("Expected <= 10 responses to be sent, found %v", len(tp.Messages))
	}

	// Make sure that the final message was a complation one
	finalMessage := tp.Messages[len(tp.Messages)-1]
	tp.messagesMutex.Unlock()

	if finalResponse, ok := finalMessage.V.(*Response); ok {
		if finalResponse.State != ResponderState_ERROR {
			t.Errorf("Expected final message state to be ERROR, found: %v", finalResponse.State)
		}

		if finalResponse.Error.Error() != "Unknown" {
			t.Errorf("Expected error string to be 'Unknown', got '%v'", finalResponse.Error.Error())
		}
	} else {
		t.Errorf("Final message did not contain a valid response object. Message content type %T", finalMessage.V)
	}
}

func TestResponseSenderCancel(t *testing.T) {
	rs := ResponseSender{
		ResponseInterval: (10 * time.Millisecond),
		ResponseSubject:  "responses",
	}

	tp := TestConnection{
		Messages: make([]ResponseMessage, 0),
	}

	// Start sending responses
	rs.Start(&tp, "test")

	// Give it enough time for >10 responses
	time.Sleep(120 * time.Millisecond)

	// Stop
	rs.Cancel()

	// Inspect what was sent
	tp.messagesMutex.Lock()
	if len(tp.Messages) <= 10 {
		t.Errorf("Expected <= 10 responses to be sent, found %v", len(tp.Messages))
	}

	// Make sure that the final message was a completion one
	finalMessage := tp.Messages[len(tp.Messages)-1]
	tp.messagesMutex.Unlock()

	if finalResponse, ok := finalMessage.V.(*Response); ok {
		if finalResponse.State != ResponderState_CANCELLED {
			t.Errorf("Expected final message state to be CANCELLED, found: %v", finalResponse.State)
		}
	} else {
		t.Errorf("Final message did not contain a valid response object. Message content type %T", finalMessage.V)
	}
}

func TestDefaultResponseInterval(t *testing.T) {
	rs := ResponseSender{}

	rs.Start(&TestConnection{}, "")
	rs.Kill()

	if rs.ResponseInterval != DEFAULTRESPONSEINTERVAL {
		t.Fatal("Response sender interval failed to default")
	}
}

// Test object used for validation that metrics are coming out properly
type ExpectedMetrics struct {
	Working    int
	Stalled    int
	Cancelled  int
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
	if x := rp.NumCancelled(); x != em.Cancelled {
		return fmt.Errorf("Expected NumCancelled to be %v, got %v", em.Cancelled, x)
	}

	rStatus := rp.ResponderStatuses()
	rErrors := rp.ResponderErrors()

	if len(rStatus) != em.Responders {
		return fmt.Errorf("Expected ResponderStatuses to have %v responders, got %v", em.Responders, len(rStatus))
	}

	if len(rErrors) != em.Failed {
		return fmt.Errorf("Expected ResponderErrors to have %v responders with errors, got %v", em.Failed, len(rErrors))
	}

	return nil
}

func TestRequestProgressNormal(t *testing.T) {
	rp := NewRequestProgress(&itemRequest)

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
			State:        ResponderState_WORKING,
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

	t.Run("Processing come other contexts also responding", func(t *testing.T) {
		// Then another context starts working
		rp.ProcessResponse(&Response{
			Responder:    "test2",
			State:        ResponderState_WORKING,
			NextUpdateIn: durationpb.New(10 * time.Millisecond),
		})

		rp.ProcessResponse(&Response{
			Responder:    "test3",
			State:        ResponderState_WORKING,
			NextUpdateIn: durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    3,
			Stalled:    0,
			Complete:   0,
			Failed:     0,
			Responders: 3,
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
			State:        ResponderState_WORKING,
			NextUpdateIn: durationpb.New(10 * time.Millisecond),
		})

		// Test 2 finishes
		rp.ProcessResponse(&Response{
			Responder: "test2",
			State:     ResponderState_COMPLETE,
		})

		// Test 3 still working
		rp.ProcessResponse(&Response{
			Responder:    "test3",
			State:        ResponderState_WORKING,
			NextUpdateIn: durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    2,
			Stalled:    0,
			Complete:   1,
			Failed:     0,
			Responders: 3,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	t.Run("When one is cancelled", func(t *testing.T) {
		time.Sleep(5 * time.Millisecond)

		// test 1 still working
		rp.ProcessResponse(&Response{
			Responder:    "test1",
			State:        ResponderState_WORKING,
			NextUpdateIn: durationpb.New(10 * time.Millisecond),
		})

		// Test 3 cancelled
		rp.ProcessResponse(&Response{
			Responder: "test3",
			State:     ResponderState_CANCELLED,
		})

		expected = ExpectedMetrics{
			Working:    1,
			Stalled:    0,
			Complete:   1,
			Failed:     0,
			Cancelled:  1,
			Responders: 3,
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
			State:        ResponderState_COMPLETE,
			NextUpdateIn: durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    0,
			Stalled:    0,
			Complete:   2,
			Failed:     0,
			Cancelled:  1,
			Responders: 3,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	if rp.allDone() == false {
		t.Error("expected allDone() to be true")
	}
}

func TestRequestProgressParallel(t *testing.T) {
	rp := NewRequestProgress(&itemRequest)

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

	t.Run("Processing many bunched responses", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := 0; i != 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Test the initial response
				rp.ProcessResponse(&Response{
					Responder:    "test1",
					State:        ResponderState_WORKING,
					NextUpdateIn: durationpb.New(10 * time.Millisecond),
				})
			}()
		}

		wg.Wait()

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
}

func TestRequestProgressStalled(t *testing.T) {
	rp := NewRequestProgress(&itemRequest)

	// Make sure that the details are correct initially
	var expected ExpectedMetrics

	t.Run("Processing the initial response", func(t *testing.T) {
		// Test the initial response
		rp.ProcessResponse(&Response{
			Responder:    "test1",
			State:        ResponderState_WORKING,
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

		if _, ok := rp.responders["test1"]; !ok {
			t.Error("Could not get responder for context test1")
		}
	})

	t.Run("After a responder recovers from a stall", func(t *testing.T) {
		// See if it will un-stall itself
		rp.ProcessResponse(&Response{
			Responder:    "test1",
			State:        ResponderState_COMPLETE,
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
	rp := NewRequestProgress(&itemRequest)

	// Make sure that the details are correct initially
	var expected ExpectedMetrics

	t.Run("Processing the initial response", func(t *testing.T) {
		// Test the initial response
		rp.ProcessResponse(&Response{
			Responder:    "test1",
			State:        ResponderState_WORKING,
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
			State:     ResponderState_ERROR,
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

func TestStart(t *testing.T) {
	rp := NewRequestProgress(&itemRequest)

	conn := TestConnection{}
	items := make(chan *Item)

	err := rp.Start(&conn, items)

	if err != nil {
		t.Fatal(err)
	}

	if _, ok := conn.Subscriptions[itemRequest.ItemSubject]; !ok {
		t.Errorf("subscription %v not created", itemRequest.ItemSubject)
	}

	if _, ok := conn.Subscriptions[itemRequest.ResponseSubject]; !ok {
		t.Errorf("subscription %v not created", itemRequest.ResponseSubject)
	}

	if len(conn.Messages) != 1 {
		t.Errorf("expected 1 message to be sent, got %v", len(conn.Messages))
	}

	// Test that the handlers work
	conn.Publish(itemRequest.ItemSubject, &item)

	receivedItem := <-items

	if receivedItem.Hash() != item.Hash() {
		t.Error("item hash mismatch")
	}
}

func TestCancel(t *testing.T) {
	t.Run("With no responders", func(t *testing.T) {
		conn := TestConnection{}

		rp := NewRequestProgress(&itemRequest)

		itemChan := make(chan *Item)

		err := rp.Start(&conn, itemChan)

		if err != nil {
			t.Fatal(err)
		}

		err = rp.Cancel(&conn)

		if err != nil {
			t.Fatal(err)
		}

		t.Run("ensuring the cancel is sent", func(t *testing.T) {
			time.Sleep(100 * time.Millisecond)

			if len(conn.Messages) != 2 {
				t.Fatal("did not receive cancellation message")
			}
		})

		t.Run("ensure it is marked as done", func(t *testing.T) {
			expected := ExpectedMetrics{
				Working:    0,
				Stalled:    0,
				Complete:   0,
				Failed:     0,
				Responders: 0,
			}

			if err := expected.Validate(rp); err != nil {
				t.Error(err)
			}
		})

		t.Run("making sure channels closed", func(t *testing.T) {
			// If the chan is still open this will block forever
			<-itemChan
		})
	})

}

func TestExecute(t *testing.T) {
	conn := TestConnection{}
	u := uuid.New()

	t.Run("with no responders", func(t *testing.T) {
		req := ItemRequest{
			Type:            "user",
			Method:          RequestMethod_GET,
			Query:           "Dylan",
			LinkDepth:       0,
			Context:         "global",
			IgnoreCache:     false,
			UUID:            u[:],
			Timeout:         durationpb.New(10 * time.Second),
			ItemSubject:     "items",
			ResponseSubject: "responses",
		}

		rp := NewRequestProgress(&req)
		rp.StartTimeout = 100 * time.Millisecond

		_, err := rp.Execute(&conn)

		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("with a full response set", func(t *testing.T) {
		req := ItemRequest{
			Type:            "user",
			Method:          RequestMethod_GET,
			Query:           "Dylan",
			LinkDepth:       0,
			Context:         "global",
			IgnoreCache:     false,
			UUID:            u[:],
			Timeout:         durationpb.New(10 * time.Second),
			ItemSubject:     "items1",
			ResponseSubject: "responses1",
		}

		rp := NewRequestProgress(&req)

		go func() {
			delay := 100 * time.Millisecond

			time.Sleep(delay)

			conn.Publish(req.ResponseSubject, &Response{
				Responder:       "test",
				State:           ResponderState_WORKING,
				ItemRequestUUID: req.UUID,
				NextUpdateIn: &durationpb.Duration{
					Seconds: 10,
					Nanos:   0,
				},
			})

			time.Sleep(delay)

			conn.Publish(req.ItemSubject, &item)

			time.Sleep(delay)

			conn.Publish(req.ItemSubject, &item)

			time.Sleep(delay)

			conn.Publish(req.ResponseSubject, &Response{
				Responder:       "test",
				State:           ResponderState_COMPLETE,
				ItemRequestUUID: req.UUID,
			})
		}()

		// TODO: Get these final tests working

		items, err := rp.Execute(&conn)

		if err != nil {
			t.Fatal(err)
		}

		if rp.NumComplete() != 1 {
			t.Errorf("expected num complete to be 1, got %v", rp.NumComplete())
		}

		if len(items) != 2 {
			t.Errorf("expected 2 items got %v", len(items))
		}
	})
}

func TestRealNats(t *testing.T) {
	nc, err := nats.Connect("nats://localhost,nats://nats")

	if err != nil {
		t.Skip("No NATS connection")
	}

	nats.RegisterEncoder("sdp", &ENCODER)
	enc, _ := nats.NewEncodedConn(nc, "sdp")

	u := uuid.New()

	req := ItemRequest{
		Type:    "person",
		Method:  RequestMethod_GET,
		Query:   "dylan",
		Context: "global",
		UUID:    u[:],
	}

	rp := NewRequestProgress(&req)
	ready := make(chan bool)

	go func() {
		enc.Subscribe("request.context.global", func(r *ItemRequest) {
			delay := 100 * time.Millisecond

			time.Sleep(delay)

			enc.Publish(req.ResponseSubject, &Response{
				Responder:       "test",
				State:           ResponderState_WORKING,
				ItemRequestUUID: req.UUID,
				NextUpdateIn: &durationpb.Duration{
					Seconds: 10,
					Nanos:   0,
				},
			})

			time.Sleep(delay)

			enc.Publish(req.ItemSubject, &item)

			enc.Publish(req.ItemSubject, &item)

			enc.Publish(req.ResponseSubject, &Response{
				Responder:       "test",
				State:           ResponderState_COMPLETE,
				ItemRequestUUID: req.UUID,
			})
		})
		ready <- true
	}()

	slowChan := make(chan *Item)

	<-ready

	err = rp.Start(enc, slowChan)

	if err != nil {
		t.Fatal(err)
	}

	for i := range slowChan {
		time.Sleep(100 * time.Millisecond)

		t.Log(i)
	}
}
