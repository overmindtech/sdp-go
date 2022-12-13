package sdp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/types/known/durationpb"
)

// DEFAULTRESPONSEINTERVAL is the default period of time within which responses
// are sent (5 seconds)
const DEFAULTRESPONSEINTERVAL = (5 * time.Second)

const ClosedChannelItemError = `SDP-GO ERROR: An Item was processed after Drain() was called. Item details:
	Type: %v
	Scope: %v
	Unique Attribute Value: %v
	Timestamp: %v
	Current Time: %v

	Please add these details to: https://github.com/overmindtech/sdp-go/issues/15`

const ClosedChannelError = `SDP-GO ERROR: An ItemRequestError was processed after Drain() was called. Error details:
	ItemRequestUUID: %v
	ErrorType: %v
	ErrorString: %v
	Scope: %v
	SourceName: %v
	ItemType: %v
	ResponderName: %v


	Please add these details to: https://github.com/overmindtech/sdp-go/issues/15`

// EncodedConnection is an interface that allows messages to be published to it.
// In production this would always be filled by a *nats.EncodedConn, however in
// testing we will mock this with something that does nothing
type EncodedConnection interface {
	Publish(subject string, v interface{}) error
	Subscribe(subject string, cb nats.Handler) (*nats.Subscription, error)
	RequestWithContext(ctx context.Context, subject string, v interface{}, vPtr interface{}) error
}

// ResponseSender is a struct responsible for sending responses out on behalf of
// agents that are working on that request. Think of it as the agent side
// component of Responder
type ResponseSender struct {
	// How often to send responses. The expected next update will be 230% of
	// this value, allowing for one-and-a-bit missed responses before it is
	// marked as stalled
	ResponseInterval time.Duration
	ResponseSubject  string
	monitorKill      chan struct{} // Sending to this channel will kill the response sender goroutine
	responderName    string
	connection       EncodedConnection
}

// Start sends the first response on the given subject and connection to say
// that the request is being worked on. It also starts a go routine to continue
// sending responses until it is cancelled
//
// Note that the NATS connection must be an encoded connection that is able to
// encode and decode SDP messages. This can be done using
// `nats.RegisterEncoder("sdp", &sdp.ENCODER)`
func (rs *ResponseSender) Start(natsConnection EncodedConnection, responderName string) {
	rs.monitorKill = make(chan struct{})

	// Set the default if it's not set
	if rs.ResponseInterval == 0 {
		rs.ResponseInterval = DEFAULTRESPONSEINTERVAL
	}

	// Tell it to expect the next update in 230% of the expected time. This
	// allows for a response getting lost, plus some delay
	nextUpdateIn := durationpb.New(time.Duration((float64(rs.ResponseInterval) * 2.3)))

	// Set struct values
	rs.responderName = responderName
	rs.connection = natsConnection

	// Create the response before starting the goroutine since it only needs to
	// be done once
	resp := Response{
		Responder:    rs.responderName,
		State:        ResponderState_WORKING,
		NextUpdateIn: nextUpdateIn,
	}

	if rs.connection != nil {
		// Send the initial response
		rs.connection.Publish(
			rs.ResponseSubject,
			&resp,
		)
	}

	// Start a goroutine to send further responses
	go func(respInterval time.Duration, conn EncodedConnection, r *Response, kill chan struct{}) {
		tick := time.NewTicker(respInterval)

		for {
			select {
			case <-kill:
				// If the context is cancelled then we don't want to do anything
				// other than exit
				tick.Stop()

				return
			case <-tick.C:
				if conn != nil {
					conn.Publish(
						rs.ResponseSubject,
						r,
					)
				}
			}
		}
	}(rs.ResponseInterval, rs.connection, &resp, rs.monitorKill)
}

// Kill Kills the response sender immediately. This should be used if something
// has failed and you don't want to send a completed response
func (rs *ResponseSender) Kill() {
	// This will block until the channel has been read from, meaning that we can
	// be sure that the goroutine is actually closing down and won't send any
	// more messages
	rs.monitorKill <- struct{}{}
}

// Done kills the responder but sends a final completion message
func (rs *ResponseSender) Done() {
	rs.Kill()

	// Create the response before starting the goroutine since it only needs to
	// be done once
	resp := Response{
		Responder: rs.responderName,
		State:     ResponderState_COMPLETE,
	}

	if rs.connection != nil {
		// Send the initial response
		rs.connection.Publish(
			rs.ResponseSubject,
			&resp,
		)
	}
}

// Error marks the request and completed with error, and sends the final error
// response
func (rs *ResponseSender) Error() {
	rs.Kill()

	// Create the response before starting the goroutine since it only needs to
	// be done once
	resp := Response{
		Responder: rs.responderName,
		State:     ResponderState_ERROR,
	}

	if rs.connection != nil {
		// Send the initial response
		rs.connection.Publish(
			rs.ResponseSubject,
			&resp,
		)
	}
}

// Cancel Marks the request as CANCELLED and sends the final response
func (rs *ResponseSender) Cancel() {
	rs.Kill()

	resp := Response{
		Responder: rs.responderName,
		State:     ResponderState_CANCELLED,
	}

	if rs.connection != nil {
		rs.connection.Publish(
			rs.ResponseSubject,
			&resp,
		)
	}
}

// Responder represents the status of a responder
type Responder struct {
	Name           string
	monitorContext context.Context
	monitorCancel  context.CancelFunc
	lastState      ResponderState
	lastStateTime  time.Time
	mutex          sync.RWMutex
}

// CancelMonitor Cancels the running stall monitor goroutine if there is one
func (re *Responder) CancelMonitor() {
	re.mutex.Lock()
	defer re.mutex.Unlock()

	if re.monitorCancel != nil {
		re.monitorCancel()
	}
}

// SetMonitorContext Saves the context details for the monitor goroutine so that
// it can be cancelled later, freeing up resources
func (re *Responder) SetMonitorContext(ctx context.Context, cancel context.CancelFunc) {
	re.mutex.Lock()
	defer re.mutex.Unlock()

	re.monitorContext = ctx
	re.monitorCancel = cancel
}

// SetState updates the state and last state time of the responder
func (re *Responder) SetState(s ResponderState) {
	re.mutex.Lock()
	defer re.mutex.Unlock()

	re.lastState = s
	re.lastStateTime = time.Now()
}

// LastState Returns the last state response for a given responder
func (re *Responder) LastState() ResponderState {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	return re.lastState
}

// LastStateTime Returns the last state response for a given responder
func (re *Responder) LastStateTime() time.Time {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	return re.lastStateTime
}

// RequestProgress represents the status of a request
type RequestProgress struct {
	// How long to wait after `MarkStarted()` has been called to get at least
	// one responder, if there are no responders in this time, the request will
	// be marked as completed
	StartTimeout time.Duration
	Request      *ItemRequest

	responders      map[string]*Responder
	respondersMutex sync.RWMutex

	// Channel storage for sending back to the user
	itemChan       chan<- *Item
	errorChan      chan<- *ItemRequestError
	doneChan       chan struct{} // Closed when request is fully complete
	chanMutex      sync.RWMutex
	channelsClosed bool // Additional protection against send on closed chan. This isn't brilliant but I can't think of a better way at the moment
	drain          sync.Once

	started   bool
	cancelled bool
	subMutex  sync.Mutex

	// NATS subscriptions
	itemSub     *nats.Subscription
	responseSub *nats.Subscription
	errorSub    *nats.Subscription

	// Counters for how many things we have sent over the channels. This is
	// required to make sure that we aren't closing channels that have pending
	// things to be sent on them
	itemsProcessed  *int64
	errorsProcessed *int64

	noResponderContext context.Context
	noRespondersCancel context.CancelFunc
}

// NewRequestProgress returns a pointer to a RequestProgress object with the
// responders map initialised
func NewRequestProgress(request *ItemRequest) *RequestProgress {
	return &RequestProgress{
		Request:         request,
		responders:      make(map[string]*Responder),
		doneChan:        make(chan struct{}),
		itemsProcessed:  new(int64),
		errorsProcessed: new(int64),
	}
}

// MarkStarted Marks the request as started and will cause it to be marked as
// done if there are no responders after StartTimeout duration
func (rp *RequestProgress) MarkStarted() {
	// We're using this mutex to also lock access to the context and cancel
	rp.respondersMutex.Lock()
	defer rp.respondersMutex.Unlock()

	rp.started = true
	rp.noResponderContext, rp.noRespondersCancel = context.WithCancel(context.Background())

	if rp.StartTimeout != 0 {
		go func(ctx context.Context) {
			startTimeout := time.NewTimer(rp.StartTimeout)
			select {
			case <-startTimeout.C:
				if rp.NumResponders() == 0 {
					rp.Drain()
				}
			case <-ctx.Done():
				startTimeout.Stop()
			}
		}(rp.noResponderContext)
	}
}

// Start Starts a given request, sending items to the supplied itemChannel. It
// is up to the user to watch for completion. When the request does complete,
// the NATS subscriptions will automatically drain and the itemChannel will be
// closed.
//
// The fact that the items chan is closed when all items have been received
// means that the only thing a user needs to do in order to process all items
// and then continue is range over the channel e.g.
//
//	for item := range itemChannel {
//		// Do something with the item
//		fmt.Println(item)
//
//		// This loop  will exit once the request is finished
//	}
func (rp *RequestProgress) Start(natsConnection EncodedConnection, itemChannel chan<- *Item, errorChannel chan<- *ItemRequestError) error {
	if rp.started {
		return errors.New("already started")
	}

	if natsConnection == nil {
		return errors.New("nil NATS connection")
	}

	if itemChannel == nil {
		return errors.New("nil item channel")
	}

	// Populate inboxes if they aren't already
	if rp.Request.ItemSubject == "" {
		rp.Request.ItemSubject = fmt.Sprintf("return.item.%v", nats.NewInbox())
	}

	if rp.Request.ResponseSubject == "" {
		rp.Request.ResponseSubject = fmt.Sprintf("return.response.%v", nats.NewInbox())
	}

	if rp.Request.ErrorSubject == "" {
		rp.Request.ErrorSubject = fmt.Sprintf("return.error.%v", nats.NewInbox())
	}

	if len(rp.Request.UUID) == 0 {
		u := uuid.New()
		rp.Request.UUID = u[:]
	}

	var requestSubject string

	if rp.Request.Scope == "" {
		return errors.New("cannot execute request with blank scope")
	}

	if rp.Request.Scope == WILDCARD {
		requestSubject = "request.all"
	} else {
		requestSubject = fmt.Sprintf("request.scope.%v", rp.Request.Scope)
	}

	// Store the channels
	rp.chanMutex.Lock()
	defer rp.chanMutex.Unlock()
	rp.itemChan = itemChannel
	rp.errorChan = errorChannel

	rp.subMutex.Lock()
	defer rp.subMutex.Unlock()

	var err error

	rp.itemSub, err = natsConnection.Subscribe(rp.Request.ItemSubject, func(item *Item) {
		defer atomic.AddInt64(rp.itemsProcessed, 1)

		if item != nil {
			rp.chanMutex.RLock()
			defer rp.chanMutex.RUnlock()
			if rp.channelsClosed {
				var itemTime time.Time

				if item.GetMetadata() != nil {
					itemTime = item.GetMetadata().Timestamp.AsTime()
				}

				// This *should* never happen but I am seeing it happen
				// occasionally. In order to avoid a panic I'm instead going to
				// log it here
				fmt.Printf(
					ClosedChannelItemError,
					item.Type,
					item.Scope,
					item.UniqueAttributeValue(),
					itemTime.String(),
					time.Now().String(),
				)

				return
			}

			rp.itemChan <- item
		}
	})

	if err != nil {
		return err
	}

	rp.errorSub, err = natsConnection.Subscribe(rp.Request.ErrorSubject, func(err *ItemRequestError) {
		defer atomic.AddInt64(rp.errorsProcessed, 1)

		if err != nil {
			rp.chanMutex.RLock()
			defer rp.chanMutex.RUnlock()
			if rp.channelsClosed {
				// This *should* never happen but I am seeing it happen
				// occasionally. In order to avoid a panic I'm instead going to
				// log it here
				fmt.Printf(
					ClosedChannelError,
					err.ItemRequestUUID,
					err.ErrorType,
					err.ErrorString,
					err.Scope,
					err.SourceName,
					err.ItemType,
					err.ResponderName,
				)

				return
			}

			rp.errorChan <- err
		}
	})

	if err != nil {
		return err
	}

	rp.responseSub, err = natsConnection.Subscribe(rp.Request.ResponseSubject, func(response *Response) {
		rp.ProcessResponse(response)
	})

	if err != nil {
		rp.itemSub.Unsubscribe()
		return err
	}

	err = natsConnection.Publish(requestSubject, rp.Request)

	rp.MarkStarted()

	if err != nil {
		return err
	}

	return nil
}

// Drain Tries to drain connections gracefully. If not though, connections are
// forcibly closed and the item and error channels closed
func (rp *RequestProgress) Drain() {
	// Use sync.Once to ensure that if this is called in parallel goroutines it
	// isn't run twice
	rp.drain.Do(func() {
		rp.subMutex.Lock()
		defer rp.subMutex.Unlock()

		if rp.noRespondersCancel != nil {
			// Cancel the no responders watcher to release the resources
			rp.noRespondersCancel()
		}

		// Close the item and error subscriptions
		unsubscribeGracefully(rp.itemSub)
		unsubscribeGracefully(rp.errorSub)

		if rp.responseSub != nil {
			// Drain the response connection to, but don't wait for callbacks to finish.
			// this is because this code here is likely called as part of a callback and
			// therefore would cause deadlock as it essentially waits for itself to
			// finish
			rp.responseSub.Unsubscribe()
		}

		// This double-checks that all callbacks are *definitely* complete to avoid
		// a situation where we close the channel with a goroutine still pending a
		// send. This is rare due to the use of RWMutex on the channel, but still
		// possible
		var itemsDelivered int64
		var errorsDelivered int64
		var err error

		for {
			itemsDelivered, err = rp.itemSub.Delivered()

			if err != nil {
				break
			}

			errorsDelivered, err = rp.errorSub.Delivered()

			if err != nil {
				break
			}

			if (itemsDelivered == *rp.itemsProcessed) && (errorsDelivered == *rp.errorsProcessed) {
				break
			}

			time.Sleep(50 * time.Millisecond)
		}

		rp.chanMutex.Lock()
		defer rp.chanMutex.Unlock()

		if rp.itemChan != nil {
			close(rp.itemChan)
		}

		if rp.errorChan != nil {
			close(rp.errorChan)
		}

		// Only if the drain is fully complete should we close the doneChan
		close(rp.doneChan)
	})
}

// Done Returns a channel when the request is fully complete and all channels
// closed
func (rp *RequestProgress) Done() <-chan struct{} {
	return rp.doneChan
}

// Cancel Cancels a request and waits for all responders to report that they
// were finished, cancelled or to be marked as stalled. If the context expires
// before this happens, the request is cancelled forcibly, with subscriptions
// being removed and channels closed. This method will only return when
// cancellation is complete
//
// Returns a boolean indicating whether the cancellation needed to be forced
func (rp *RequestProgress) Cancel(ctx context.Context, natsConnection EncodedConnection) bool {
	rp.AsyncCancel(natsConnection)

	select {
	case <-rp.Done():
		// If the request finishes gracefully, that's good
		return false
	case <-ctx.Done():
		// If the context is cancelled first, then force the draining
		rp.Drain()
		return true
	}
}

// Cancel Sends a cancellation request for a given request
func (rp *RequestProgress) AsyncCancel(natsConnection EncodedConnection) error {
	if natsConnection == nil {
		return errors.New("nil NATS connection")
	}

	cancelRequest := CancelItemRequest{
		UUID: rp.Request.UUID,
	}

	var cancelSubject string

	if rp.Request.Scope == WILDCARD {
		cancelSubject = "cancel.all"
	} else {
		cancelSubject = fmt.Sprintf("cancel.scope.%v", rp.Request.Scope)
	}

	rp.cancelled = true

	err := natsConnection.Publish(cancelSubject, &cancelRequest)

	if err != nil {
		return err
	}

	// Check this immediately in case nothing had started yet
	if rp.allDone() {
		rp.Drain()
	}

	return nil
}

// Execute Executes a given request and waits for it to finish, returns the
// items that were found and any errors. The third return error value  will only
// be returned only if there is a problem making the request. Details of which
// responders have failed etc. should be determined using the typical methods
// like `NumError()`.
func (rp *RequestProgress) Execute(natsConnection EncodedConnection) ([]*Item, []*ItemRequestError, error) {
	items := make([]*Item, 0)
	errs := make([]*ItemRequestError, 0)
	i := make(chan *Item)
	e := make(chan *ItemRequestError)

	if natsConnection == nil {
		return items, errs, errors.New("nil NATS connection")
	}

	err := rp.Start(natsConnection, i, e)

	if err != nil {
		return items, errs, err
	}

	for {
		// Read items and errors
		select {
		case item, ok := <-i:
			if ok {
				items = append(items, item)
			} else {
				// If the channel is closed, set it to nil so we don't receive
				// from it any more
				i = nil
			}
		case err, ok := <-e:
			if ok {
				errs = append(errs, err)
			} else {
				e = nil
			}
		}

		if i == nil && e == nil {
			// If both channels are closed then we're done
			break
		}
	}

	return items, errs, nil
}

// ProcessResponse processes an SDP Response and updates the database
// accordingly
func (rp *RequestProgress) ProcessResponse(response *Response) {
	// Update the stored data
	rp.respondersMutex.Lock()

	// As soon as we get a response, we can cancel the "no responders" goroutine
	if rp.noRespondersCancel != nil {
		rp.noRespondersCancel()
	}

	responder, exists := rp.responders[response.Responder]

	if exists {
		responder.CancelMonitor()
	} else {
		// If the responder is new, add it to the list
		responder = &Responder{
			Name: response.GetResponder(),
		}
		rp.responders[response.Responder] = responder
	}

	responder.SetState(response.State)

	rp.respondersMutex.Unlock()

	// Check if we should expect another response
	expectFollowUp := (response.GetNextUpdateIn() != nil)

	// If we are told to expect a new response, set up context for it
	if expectFollowUp {
		timeout := response.GetNextUpdateIn().AsDuration()

		monitorContext, monitorCancel := context.WithCancel(context.Background())

		rp.respondersMutex.RLock()
		responder = rp.responders[response.Responder]
		rp.respondersMutex.RUnlock()

		responder.SetMonitorContext(monitorContext, monitorCancel)

		// Create a goroutine to watch for a stalled connection
		go StallMonitor(monitorContext, timeout, responder, rp)
	}

	// Finally check to see if this was the final request and if so update the
	// chan
	if rp.allDone() {
		rp.Drain()
	}
}

// NumWorking returns the number of responders that are in the Working state
func (rp *RequestProgress) NumWorking() int {
	rp.respondersMutex.RLock()
	defer rp.respondersMutex.RUnlock()

	var numWorking int

	for _, responder := range rp.responders {
		if responder.LastState() == ResponderState_WORKING {
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

	for _, responder := range rp.responders {
		if responder.LastState() == ResponderState_STALLED {
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

	for _, responder := range rp.responders {
		if responder.LastState() == ResponderState_COMPLETE {
			numComplete++
		}
	}

	return numComplete
}

// NumError returns the number of responders that are in the FAILED state
func (rp *RequestProgress) NumError() int {
	rp.respondersMutex.RLock()
	defer rp.respondersMutex.RUnlock()

	var numError int

	for _, responder := range rp.responders {
		if responder.LastState() == ResponderState_ERROR {
			numError++
		}
	}

	return numError
}

// NumCancelled returns the number of responders that are in the CANCELLED state
func (rp *RequestProgress) NumCancelled() int {
	rp.respondersMutex.RLock()
	defer rp.respondersMutex.RUnlock()

	var numCancelled int

	for _, responder := range rp.responders {
		if responder.LastState() == ResponderState_CANCELLED {
			numCancelled++
		}
	}

	return numCancelled
}

// NumResponders returns the total number of unique responders
func (rp *RequestProgress) NumResponders() int {
	rp.respondersMutex.RLock()
	defer rp.respondersMutex.RUnlock()
	return len(rp.responders)
}

// ResponderStates Returns the status details for all responders as a map.
// Where the key is the name of the responder and the value is its status
func (rp *RequestProgress) ResponderStates() map[string]ResponderState {
	statuses := make(map[string]ResponderState)
	rp.respondersMutex.RLock()
	defer rp.respondersMutex.RUnlock()
	for _, responder := range rp.responders {
		statuses[responder.Name] = responder.LastState()
	}

	return statuses
}

func (rp *RequestProgress) String() string {
	return fmt.Sprintf(
		"Working: %v\nStalled: %v\nComplete: %v\nFailed: %v\nCancelled: %v\nResponders: %v\n",
		rp.NumWorking(),
		rp.NumStalled(),
		rp.NumComplete(),
		rp.NumError(),
		rp.NumCancelled(),
		rp.NumResponders(),
	)
}

// Complete will return true if there are no remaining responders working
func (rp *RequestProgress) allDone() bool {
	if rp.NumResponders() > 0 || rp.cancelled {
		// If we have had at least one response, and there aren't any waiting
		// then we are going to assume that everything is done. It is of course
		// possible that there has just been a very fast responder and so a
		// minimum execution time might be a good idea
		return (rp.NumWorking() == 0)
	}
	// If there have been no responders at all we can't say that we're "done"
	return false
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
		responder.SetState(ResponderState_STALLED)

		if rp.allDone() {
			rp.Drain()
		}

		return
	}
}

// unsubscribeGracefully Closes a NATS subscription gracefully, this includes
// draining, unsubscribing and ensuring that all callbacks are complete
func unsubscribeGracefully(c *nats.Subscription) error {
	if c != nil {
		// Drain NATS connections
		err := c.Drain()

		if err != nil {
			// If that fails, fall back to an unsubscribe
			err = c.Unsubscribe()

			if err != nil {
				return err
			}
		}

		// Wait for all items to finish processing, including all callbacks
		for {
			messages, _, _ := c.Pending()

			if messages > 0 {
				time.Sleep(50 * time.Millisecond)
			} else {
				break
			}
		}
	}

	return nil
}
