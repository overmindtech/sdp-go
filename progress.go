package sdp

import (
	"context"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
)

// ResponderStatus represents the state of the responder using the WORKING,
// STALLED, COMPLETE and FAILED constants
type ResponderStatus int

const (
	// WORKING means that the responder is still actively working on the request
	WORKING = 0
	// STALLED means that we have not received an update within the expected
	// time
	STALLED = 1
	// COMPLETE means that the responder has completed the query
	COMPLETE = 2
	// FAILED means that the responder encountered an error
	FAILED = 3
)

// DEFAULTRESPONSEINTERVAL is the default period of time within which responses
// are sent (5 seconds)
const DEFAULTRESPONSEINTERVAL = (5 * time.Second)

// EncodedPublisher is an interface that allows messages to be published to it.
// In production this would always be filled by a *nats.EncodedConn, however in
// testing we will mock this with something that does nothing
type EncodedPublisher interface {
	Publish(subject string, v interface{}) error
}

// RequestProgress represents the status of a request
type RequestProgress struct {
	Responders      map[string]*Responder
	Request         ItemRequest
	respondersMutex sync.RWMutex
	doneChan        chan bool
}

// Responder represents the status of a responder
type Responder struct {
	SDPContext     string
	MonitorContext context.Context
	MonitorCancel  context.CancelFunc
	LastStatus     ResponderStatus
	LastStatusTime time.Time
	Error          error
}

// ResponseSender is a struct responsible for sending responses out on behalf of
// agents that are wortking on that request. Think of it as the agent side
// component of Responder
type ResponseSender struct {
	// How often to send responses. The expected next update will be 230% of
	// this value, allowing for one-and-a-bit missed responses before it is
	// marked as stalled
	ResponseInterval time.Duration
	ResponseSubject  string
	monitorContext   context.Context
	monitorCancel    context.CancelFunc
	requestContext   string
	connection       EncodedPublisher
}

// Start sends the first response on the given subject and connection to say
// that the request is being worked on. It also starts a go routine to continue
// sending responses until it is cancelled
//
// Note that the NATS connection must be an encoded connection that is able to
// encode and decode SDP messages. This can be done using
// `nats.RegisterEncoder("sdp", &sdp.ENCODER)`
func (rs *ResponseSender) Start(natsConnection EncodedPublisher, requestContext string) {
	rs.monitorContext, rs.monitorCancel = context.WithCancel(context.Background())

	// Set the default if it's not set
	if rs.ResponseInterval == 0 {
		rs.ResponseInterval = DEFAULTRESPONSEINTERVAL
	}

	// Tell it to expect the next update in 230% of the expected time. This
	// allows for a response getting lost, plus some delay
	nextUpdateIn := durationpb.New(time.Duration((float64(rs.ResponseInterval) * 2.3)))

	// Set struct values
	rs.requestContext = requestContext
	rs.connection = natsConnection

	// Create the response before starting the goroutine since it only needs to
	// be done once
	resp := Response{
		Context:      rs.requestContext,
		State:        Response_WORKING,
		NextUpdateIn: nextUpdateIn,
	}

	// Send the initial response
	rs.connection.Publish(
		rs.ResponseSubject,
		&resp,
	)

	// Start a goroutine to send further responses
	go func() {
		for {
			select {
			case <-rs.monitorContext.Done():
				// If the context is cancelled then we don't want to do anything
				// other than exit
				return
			case <-time.After(rs.ResponseInterval):
				rs.connection.Publish(
					rs.ResponseSubject,
					&resp,
				)
			}
		}
	}()
}

// Kill Kills the response sender immediately. This should be used if something
// has failed and you don't want to send a completed respnose
func (rs *ResponseSender) Kill() {
	rs.monitorCancel()
}

// Done kills the responder but sends a final complation message
func (rs *ResponseSender) Done() {
	rs.Kill()

	// Create the response before starting the goroutine since it only needs to
	// be done once
	resp := Response{
		Context: rs.requestContext,
		State:   Response_COMPLETE,
	}

	// Send the initial response
	rs.connection.Publish(
		rs.ResponseSubject,
		&resp,
	)
}

// Error marks the request and completed with error, and send the final error
func (rs *ResponseSender) Error(err *ItemRequestError) {
	rs.Kill()

	// Create the response before starting the goroutine since it only needs to
	// be done once
	resp := Response{
		Context: rs.requestContext,
		State:   Response_ERROR,
		Error:   err,
	}

	// Send the initial response
	rs.connection.Publish(
		rs.ResponseSubject,
		&resp,
	)
}

// SetStatus updates the status and last status time of the responder
func (re *Responder) SetStatus(s ResponderStatus) {
	re.LastStatus = s
	re.LastStatusTime = time.Now()
}

// NewRequestProgress returns a pointer to a RequestProgress object with the
// responders map initialised
func NewRequestProgress() *RequestProgress {
	return &RequestProgress{
		Responders: make(map[string]*Responder),
		// Buffered chan allows for handling function to finish even if nobody
		// is listening on the channel. It is possible that this could overflow
		// if we got into a done condition >1024 times but I would think this is
		// extremely unlikely. Maybe famous last words?
		doneChan: make(chan bool, 1024),
	}
}

// ProcessResponse processes an SDP Response and updates the database
// accordingly
func (rp *RequestProgress) ProcessResponse(response *Response) {
	// Convert to a local status representation
	var status ResponderStatus

	switch s := response.GetState(); s {
	case Response_WORKING:
		status = WORKING
	case Response_COMPLETE:
		status = COMPLETE
	case Response_ERROR:
		status = FAILED
	}

	// Update the stored data
	rp.respondersMutex.Lock()

	responder, exists := rp.Responders[response.Context]

	if exists {
		if responder.MonitorCancel != nil {
			// Cancel the previous stall monitor
			responder.MonitorCancel()
		}

		// Update the status of the responder
		responder.SetStatus(status)
	} else {
		// If the responder is new, add it to the list
		rp.Responders[response.Context] = &Responder{
			SDPContext:     response.GetContext(),
			LastStatus:     status,
			LastStatusTime: time.Now(),
			Error:          response.Error,
		}
	}

	rp.respondersMutex.Unlock()

	// Check if we should expect another response
	expectFollowUp := (response.GetNextUpdateIn() != nil)

	// If we are told to expect a new response, set up context for it
	if expectFollowUp {
		timeout := response.GetNextUpdateIn().AsDuration()

		montorContext, monitorCancel := context.WithCancel(context.Background())

		rp.respondersMutex.RLock()
		responder = rp.Responders[response.Context]
		rp.respondersMutex.RUnlock()

		responder.MonitorContext = montorContext
		responder.MonitorCancel = monitorCancel

		// Create a goroutine to watch for a stalled connection
		go StallMonitor(montorContext, timeout, responder, rp)
	}

	// Finally check to see if this was the final request and if so update the
	// chan
	rp.checkDoneChan()
}

// StallMonitor watches for stalled connections. It should be passed the
// responder to monitor, the time to wait before marking the connection as
// stalled, and a context. The context is used to allow cancellation of the
// stall monitor from another thread in the case that another message is
// received.
func StallMonitor(context context.Context, timeout time.Duration, responder *Responder, rp *RequestProgress) {
	select {
	case <-context.Done():
		// If the context is cancelled then we don't want to do anything
		return
	case <-time.After(timeout):
		// If the timeout elapses before the context is cancelled it
		// means that we haven't received a response in the expected
		// time, we now need to mark that responder as STALLED
		responder.SetStatus(STALLED)
		rp.checkDoneChan()
		return
	}
}

// NumWorking returns the number of responders that are in the Working state
func (rp *RequestProgress) NumWorking() int {
	rp.respondersMutex.RLock()
	defer rp.respondersMutex.RUnlock()

	var numWorking int

	for _, responder := range rp.Responders {
		if responder.LastStatus == WORKING {
			numWorking++
		}
	}

	return numWorking
}

// NumStalled returns the number of responders that are in the STALLED state
func (rp *RequestProgress) NumStalled() int {
	rp.respondersMutex.RLock()
	defer rp.respondersMutex.RUnlock()

	var numStalled int

	for _, responder := range rp.Responders {
		if responder.LastStatus == STALLED {
			numStalled++
		}
	}

	return numStalled
}

// NumComplete returns the number of responders that are in the COMPLETE state
func (rp *RequestProgress) NumComplete() int {
	rp.respondersMutex.RLock()
	defer rp.respondersMutex.RUnlock()

	var numComplete int

	for _, responder := range rp.Responders {
		if responder.LastStatus == COMPLETE {
			numComplete++
		}
	}

	return numComplete
}

// NumFailed returns the number of responders that are in the FAILED state
func (rp *RequestProgress) NumFailed() int {
	rp.respondersMutex.RLock()
	defer rp.respondersMutex.RUnlock()

	var numFailed int

	for _, responder := range rp.Responders {
		if responder.LastStatus == FAILED {
			numFailed++
		}
	}

	return numFailed
}

// NumResponders returns the total number of unique responders
func (rp *RequestProgress) NumResponders() int {
	rp.respondersMutex.RLock()
	defer rp.respondersMutex.RUnlock()
	return len(rp.Responders)
}

// Done returns a channel which will have a value pushed to it of all requests
// are complete. It is designed to be used similarly to time.After()
func (rp *RequestProgress) Done() chan bool {
	return rp.doneChan
}

// checkDoneChan checks everything is complete and if so pushes to the chan
func (rp *RequestProgress) checkDoneChan() {
	if rp.allDone() {
		// If everything is done then send `true` to the chan so that whatever
		// is waiting on it can continue
		rp.Done() <- true
	}
}

// Complete will return true if there are no remaining responders working
func (rp *RequestProgress) allDone() bool {
	if rp.NumResponders() > 0 {
		// If we have had at least one response, and there aren't any waiting
		// then we are going to assume that everything is done. It is of  course
		// possible that there has just been a very fast responder and so a
		// minimum execution time might be a good idea
		return (rp.NumWorking() == 0)
	}
	// If there have been no responders at all we can't say that we're "done"
	return false
}

func (rs ResponderStatus) String() string {
	switch rs {
	case WORKING:
		return "working"
	case STALLED:
		return "stalled"
	case COMPLETE:
		return "complete"
	case FAILED:
		return "failed"
	default:
		return "unknown"
	}
}
