package sdp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/durationpb"
)

// DefaultResponseInterval is the default period of time within which responses
// are sent (5 seconds)
const DefaultResponseInterval = (5 * time.Second)

// DefaultDrainDelay How long to wait after all is complete before draining all
// NATS connections
const DefaultDrainDelay = (100 * time.Millisecond)

// ResponseSender is a struct responsible for sending responses out on behalf of
// agents that are working on that request. Think of it as the agent side
// component of Responder
type ResponseSender struct {
	// How often to send responses. The expected next update will be 230% of
	// this value, allowing for one-and-a-bit missed responses before it is
	// marked as stalled
	ResponseInterval time.Duration
	ResponseSubject  string
	monitorRunning   sync.WaitGroup
	monitorKill      chan *Response // Sending to this channel will kill the response sender goroutine and publish the sent message as last msg on the subject
	responderName    string
	connection       EncodedConnection
	responseCtx      context.Context
}

// Start sends the first response on the given subject and connection to say
// that the request is being worked on. It also starts a go routine to continue
// sending responses until it is cancelled
func (rs *ResponseSender) Start(ctx context.Context, ec EncodedConnection, responderName string) {
	rs.monitorKill = make(chan *Response, 1)
	rs.responseCtx = ctx

	// Set the default if it's not set
	if rs.ResponseInterval == 0 {
		rs.ResponseInterval = DefaultResponseInterval
	}

	// Tell it to expect the next update in 230% of the expected time. This
	// allows for a response getting lost, plus some delay
	nextUpdateIn := durationpb.New(time.Duration((float64(rs.ResponseInterval) * 2.3)))

	// Set struct values
	rs.responderName = responderName
	rs.connection = ec

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
			ctx,
			rs.ResponseSubject,
			&resp,
		)
	}

	rs.monitorRunning.Add(1)

	// Start a goroutine to send further responses
	go func(ctx context.Context, respInterval time.Duration, ec EncodedConnection, r *Response, kill chan *Response) {
		defer sentry.Recover()
		// confirm closure on exit
		defer rs.monitorRunning.Done()

		if ec == nil {
			return
		}
		tick := time.NewTicker(respInterval)

		for {
			select {
			case r := <-kill:
				// If the context is cancelled then we don't want to do anything
				// other than exit
				tick.Stop()

				if r != nil {
					ec.Publish(
						ctx,
						rs.ResponseSubject,
						r,
					)
				}
				return
			case <-ctx.Done():
				// If the context is cancelled then we don't want to do anything
				// other than exit
				tick.Stop()

				return
			case <-tick.C:
				ec.Publish(
					ctx,
					rs.ResponseSubject,
					r,
				)
			}
		}
	}(ctx, rs.ResponseInterval, rs.connection, &resp, rs.monitorKill)
}

// Kill Kills the response sender immediately. This should be used if something
// has failed and you don't want to send a completed response
func (rs *ResponseSender) Kill() {
	rs.killWithResponse(nil)
}

func (rs *ResponseSender) killWithResponse(r *Response) {
	// send the stop signal to the goroutine from Start()
	rs.monitorKill <- r

	// wait for the sender to be actually done
	rs.monitorRunning.Wait()
}

// Done kills the responder but sends a final completion message
func (rs *ResponseSender) Done() {
	resp := Response{
		Responder: rs.responderName,
		State:     ResponderState_COMPLETE,
	}
	rs.killWithResponse(&resp)
}

// Error marks the request and completed with error, and sends the final error
// response
func (rs *ResponseSender) Error() {
	resp := Response{
		Responder: rs.responderName,
		State:     ResponderState_ERROR,
	}
	rs.killWithResponse(&resp)
}

// Cancel Marks the request as CANCELLED and sends the final response
func (rs *ResponseSender) Cancel() {
	resp := Response{
		Responder: rs.responderName,
		State:     ResponderState_CANCELLED,
	}
	rs.killWithResponse(&resp)
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

// QueryProgress represents the status of a request
type QueryProgress struct {
	// How long to wait after `MarkStarted()` has been called to get at least
	// one responder, if there are no responders in this time, the request will
	// be marked as completed
	StartTimeout time.Duration
	Query        *Query
	requestCtx   context.Context

	// How long to wait before draining NATS connections after all have
	// completed
	DrainDelay time.Duration

	responders      map[string]*Responder
	respondersMutex sync.RWMutex

	// Channel storage for sending back to the user
	itemChan       chan<- *Item
	errorChan      chan<- *QueryError
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

// NewQueryProgress returns a pointer to a QueryProgress object with the
// responders map initialized
func NewQueryProgress(q *Query) *QueryProgress {
	return &QueryProgress{
		Query:           q,
		DrainDelay:      DefaultDrainDelay,
		responders:      make(map[string]*Responder),
		doneChan:        make(chan struct{}),
		itemsProcessed:  new(int64),
		errorsProcessed: new(int64),
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
func (qp *QueryProgress) Start(ctx context.Context, ec EncodedConnection, itemChannel chan<- *Item, errorChannel chan<- *QueryError) error {
	if qp.started {
		return errors.New("already started")
	}

	if ec.Underlying() == nil {
		return errors.New("nil NATS connection")
	}

	if itemChannel == nil {
		return errors.New("nil item channel")
	}

	qp.requestCtx = ctx

	// Populate inboxes if they aren't already
	if qp.Query.ItemSubject == "" {
		qp.Query.ItemSubject = fmt.Sprintf("return.item.%v", nats.NewInbox())
	}

	if qp.Query.ResponseSubject == "" {
		qp.Query.ResponseSubject = fmt.Sprintf("return.response.%v", nats.NewInbox())
	}

	if qp.Query.ErrorSubject == "" {
		qp.Query.ErrorSubject = fmt.Sprintf("return.error.%v", nats.NewInbox())
	}

	if len(qp.Query.UUID) == 0 {
		u := uuid.New()
		qp.Query.UUID = u[:]
	}

	var requestSubject string

	if qp.Query.Scope == "" {
		return errors.New("cannot execute request with blank scope")
	}

	if qp.Query.Scope == WILDCARD {
		requestSubject = "request.all"
	} else {
		requestSubject = fmt.Sprintf("request.scope.%v", qp.Query.Scope)
	}

	// Store the channels
	qp.chanMutex.Lock()
	defer qp.chanMutex.Unlock()
	qp.itemChan = itemChannel
	qp.errorChan = errorChannel

	qp.subMutex.Lock()
	defer qp.subMutex.Unlock()

	var err error

	qp.itemSub, err = ec.Subscribe(qp.Query.ItemSubject, NewItemHandler("Request.ItemSubject", func(ctx context.Context, item *Item) {
		defer atomic.AddInt64(qp.itemsProcessed, 1)

		if item != nil {
			qp.chanMutex.RLock()
			defer qp.chanMutex.RUnlock()
			if qp.channelsClosed {
				var itemTime time.Time

				if item.GetMetadata() != nil {
					itemTime = item.GetMetadata().Timestamp.AsTime()
				}

				// This *should* never happen but I am seeing it happen
				// occasionally. In order to avoid a panic I'm instead going to
				// log it here
				log.WithFields(log.Fields{
					"Type":                 item.Type,
					"Scope":                item.Scope,
					"UniqueAttributeValue": item.UniqueAttributeValue(),
					"Item Timestamp":       itemTime.String(),
					"Current Time":         time.Now().String(),
				}).Error("SDP-GO ERROR: An Item was processed after Drain() was called. Please add these details to: https://github.com/overmindtech/sdp-go/issues/15.")

				return
			}

			qp.itemChan <- item
		}
	}))

	if err != nil {
		return err
	}

	qp.errorSub, err = ec.Subscribe(qp.Query.ErrorSubject, NewQueryErrorHandler("Request.ErrorSubject", func(ctx context.Context, err *QueryError) {
		defer atomic.AddInt64(qp.errorsProcessed, 1)

		if err != nil {
			span := trace.SpanFromContext(ctx)
			span.SetStatus(codes.Error, err.Error())
			span.SetAttributes(
				attribute.Int64("om.sdp.errorsProcessed", *qp.errorsProcessed),
				attribute.String("om.sdp.errorString", err.ErrorString),
				attribute.String("om.sdp.ErrorType", err.ErrorType.String()),
				attribute.String("om.scope", err.Scope),
				attribute.String("om.type", err.ItemType),
				attribute.String("om.sdp.SourceName", err.SourceName),
				attribute.String("om.sdp.ResponderName", err.ResponderName),
			)

			qp.chanMutex.RLock()
			defer qp.chanMutex.RUnlock()
			if qp.channelsClosed {
				// This *should* never happen but I am seeing it happen
				// occasionally. In order to avoid a panic I'm instead going to
				// log it here
				log.WithFields(log.Fields{
					"QueryUUID":     err.QueryUUID,
					"ErrorType":     err.ErrorType,
					"ErrorString":   err.ErrorString,
					"Scope":         err.Scope,
					"SourceName":    err.SourceName,
					"ItemType":      err.ItemType,
					"ResponderName": err.ResponderName,
				}).Error("SDP-GO ERROR: A QueryError was processed after Drain() was called. Please add these details to: https://github.com/overmindtech/sdp-go/issues/15.")
				return
			}

			qp.errorChan <- err
		}
	}))

	if err != nil {
		return err
	}

	qp.responseSub, err = ec.Subscribe(qp.Query.ResponseSubject, NewResponseHandler("ProcessResponse", qp.ProcessResponse))

	if err != nil {
		qp.itemSub.Unsubscribe()
		return err
	}

	err = ec.Publish(ctx, requestSubject, qp.Query)

	qp.markStarted()

	if err != nil {
		return err
	}

	return nil
}

// markStarted Marks the request as started and will cause it to be marked as
// done if there are no responders after StartTimeout duration
func (qp *QueryProgress) markStarted() {
	// We're using this mutex to also lock access to the context and cancel
	qp.respondersMutex.Lock()
	defer qp.respondersMutex.Unlock()

	qp.started = true
	qp.noResponderContext, qp.noRespondersCancel = context.WithCancel(context.Background())

	if qp.StartTimeout != 0 {
		go func(ctx context.Context) {
			defer sentry.Recover()
			startTimeout := time.NewTimer(qp.StartTimeout)
			select {
			case <-startTimeout.C:
				if qp.NumResponders() == 0 {
					qp.Drain()
				}
			case <-ctx.Done():
				startTimeout.Stop()
			}
		}(qp.noResponderContext)
	}
}

// Drain Tries to drain connections gracefully. If not though, connections are
// forcibly closed and the item and error channels closed
func (qp *QueryProgress) Drain() {
	// Use sync.Once to ensure that if this is called in parallel goroutines it
	// isn't run twice
	qp.drain.Do(func() {
		qp.subMutex.Lock()
		defer qp.subMutex.Unlock()

		if qp.noRespondersCancel != nil {
			// Cancel the no responders watcher to release the resources
			qp.noRespondersCancel()
		}

		// Close the item and error subscriptions
		unsubscribeGracefully(qp.itemSub)
		unsubscribeGracefully(qp.errorSub)

		if qp.responseSub != nil {
			// Drain the response connection to, but don't wait for callbacks to finish.
			// this is because this code here is likely called as part of a callback and
			// therefore would cause deadlock as it essentially waits for itself to
			// finish
			qp.responseSub.Unsubscribe()
		}

		// This double-checks that all callbacks are *definitely* complete to avoid
		// a situation where we close the channel with a goroutine still pending a
		// send. This is rare due to the use of RWMutex on the channel, but still
		// possible
		var itemsDelivered int64
		var errorsDelivered int64
		var err error

		for {
			itemsDelivered, err = qp.itemSub.Delivered()

			if err != nil {
				break
			}

			errorsDelivered, err = qp.errorSub.Delivered()

			if err != nil {
				break
			}

			if (itemsDelivered == *qp.itemsProcessed) && (errorsDelivered == *qp.errorsProcessed) {
				break
			}

			time.Sleep(50 * time.Millisecond)
		}

		qp.chanMutex.Lock()
		defer qp.chanMutex.Unlock()

		if qp.itemChan != nil {
			close(qp.itemChan)
		}

		if qp.errorChan != nil {
			close(qp.errorChan)
		}

		// Only if the drain is fully complete should we close the doneChan
		close(qp.doneChan)

		qp.channelsClosed = true
	})
}

// Done Returns a channel when the request is fully complete and all channels
// closed
func (qp *QueryProgress) Done() <-chan struct{} {
	return qp.doneChan
}

// Cancel Cancels a request and waits for all responders to report that they
// were finished, cancelled or to be marked as stalled. If the context expires
// before this happens, the request is cancelled forcibly, with subscriptions
// being removed and channels closed. This method will only return when
// cancellation is complete
//
// Returns a boolean indicating whether the cancellation needed to be forced
func (qp *QueryProgress) Cancel(ctx context.Context, ec EncodedConnection) bool {
	qp.AsyncCancel(ec)

	select {
	case <-qp.Done():
		// If the request finishes gracefully, that's good
		return false
	case <-ctx.Done():
		// If the context is cancelled first, then force the draining
		qp.Drain()
		return true
	}
}

// Cancel Sends a cancellation request for a given request
func (qp *QueryProgress) AsyncCancel(ec EncodedConnection) error {
	if ec == nil {
		return errors.New("nil NATS connection")
	}

	cancelRequest := CancelQuery{
		UUID: qp.Query.UUID,
	}

	var cancelSubject string

	if qp.Query.Scope == WILDCARD {
		cancelSubject = "cancel.all"
	} else {
		cancelSubject = fmt.Sprintf("cancel.scope.%v", qp.Query.Scope)
	}

	qp.cancelled = true

	err := ec.Publish(qp.requestCtx, cancelSubject, &cancelRequest)

	if err != nil {
		return err
	}

	// Check this immediately in case nothing had started yet
	if qp.allDone() {
		qp.Drain()
	}

	return nil
}

// Execute Executes a given request and waits for it to finish, returns the
// items that were found and any errors. The third return error value  will only
// be returned only if there is a problem making the request. Details of which
// responders have failed etc. should be determined using the typical methods
// like `NumError()`.
func (qp *QueryProgress) Execute(ctx context.Context, ec EncodedConnection) ([]*Item, []*QueryError, error) {
	items := make([]*Item, 0)
	errs := make([]*QueryError, 0)
	i := make(chan *Item)
	e := make(chan *QueryError)

	if ec == nil {
		return items, errs, errors.New("nil NATS connection")
	}

	err := qp.Start(ctx, ec, i, e)

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
func (qp *QueryProgress) ProcessResponse(ctx context.Context, response *Response) {
	func() {
		// Update the stored data
		qp.respondersMutex.Lock()
		defer qp.respondersMutex.Unlock()

		// As soon as we get a response, we can cancel the "no responders" goroutine
		if qp.noRespondersCancel != nil {
			qp.noRespondersCancel()
		}

		responder, exists := qp.responders[response.Responder]

		if exists {
			responder.CancelMonitor()
		} else {
			// If the responder is new, add it to the list
			responder = &Responder{
				Name: response.GetResponder(),
			}
			qp.responders[response.Responder] = responder
		}

		responder.SetState(response.State)
	}()

	// Check if we should expect another response
	expectFollowUp := (response.GetNextUpdateIn() != nil && response.State != ResponderState_COMPLETE)

	// If we are told to expect a new response, set up context for it
	if expectFollowUp {
		timeout := response.GetNextUpdateIn().AsDuration()

		monitorContext, monitorCancel := context.WithCancel(context.Background())

		responder := func() *Responder {
			qp.respondersMutex.RLock()
			defer qp.respondersMutex.RUnlock()
			return qp.responders[response.Responder]
		}()

		responder.SetMonitorContext(monitorContext, monitorCancel)

		// Create a goroutine to watch for a stalled connection
		go stallMonitor(monitorContext, timeout, responder, qp)
	}

	// Finally check to see if this was the final request and if so update the
	// chan
	if qp.allDone() {
		// at this point I need to add some slack in case the we have received
		// the completion response before the final item. The sources are
		// supposed to wait until all items have been sent in order to send
		// this, but NATS doesn't guarantee ordering so there's still a
		// reasonable chance that things will arrive in a weird order. This is a
		// pretty bad solution and realistically this should be addressed in the
		// protocol itself, but for now this will do. Especially since it
		// doesn't actually block anything that the client sees, it's just
		// delaying cleanup for a little longer than we need
		time.Sleep(qp.DrainDelay)

		qp.Drain()
	}
}

// NumWorking returns the number of responders that are in the Working state
func (qp *QueryProgress) NumWorking() int {
	qp.respondersMutex.RLock()
	defer qp.respondersMutex.RUnlock()

	var numWorking int

	for _, responder := range qp.responders {
		if responder.LastState() == ResponderState_WORKING {
			numWorking++
		}
	}

	return numWorking
}

// NumStalled returns the number of responders that are in the STALLED state
func (qp *QueryProgress) NumStalled() int {
	qp.respondersMutex.RLock()
	defer qp.respondersMutex.RUnlock()

	var numStalled int

	for _, responder := range qp.responders {
		if responder.LastState() == ResponderState_STALLED {
			numStalled++
		}
	}

	return numStalled
}

// NumComplete returns the number of responders that are in the COMPLETE state
func (qp *QueryProgress) NumComplete() int {
	qp.respondersMutex.RLock()
	defer qp.respondersMutex.RUnlock()

	var numComplete int

	for _, responder := range qp.responders {
		if responder.LastState() == ResponderState_COMPLETE {
			numComplete++
		}
	}

	return numComplete
}

// NumError returns the number of responders that are in the FAILED state
func (qp *QueryProgress) NumError() int {
	qp.respondersMutex.RLock()
	defer qp.respondersMutex.RUnlock()

	var numError int

	for _, responder := range qp.responders {
		if responder.LastState() == ResponderState_ERROR {
			numError++
		}
	}

	return numError
}

// NumCancelled returns the number of responders that are in the CANCELLED state
func (qp *QueryProgress) NumCancelled() int {
	qp.respondersMutex.RLock()
	defer qp.respondersMutex.RUnlock()

	var numCancelled int

	for _, responder := range qp.responders {
		if responder.LastState() == ResponderState_CANCELLED {
			numCancelled++
		}
	}

	return numCancelled
}

// NumResponders returns the total number of unique responders
func (qp *QueryProgress) NumResponders() int {
	qp.respondersMutex.RLock()
	defer qp.respondersMutex.RUnlock()
	return len(qp.responders)
}

// ResponderStates Returns the status details for all responders as a map.
// Where the key is the name of the responder and the value is its status
func (qp *QueryProgress) ResponderStates() map[string]ResponderState {
	statuses := make(map[string]ResponderState)
	qp.respondersMutex.RLock()
	defer qp.respondersMutex.RUnlock()
	for _, responder := range qp.responders {
		statuses[responder.Name] = responder.LastState()
	}

	return statuses
}

func (qp *QueryProgress) String() string {
	return fmt.Sprintf(
		"Working: %v\nStalled: %v\nComplete: %v\nFailed: %v\nCancelled: %v\nResponders: %v\n",
		qp.NumWorking(),
		qp.NumStalled(),
		qp.NumComplete(),
		qp.NumError(),
		qp.NumCancelled(),
		qp.NumResponders(),
	)
}

// Complete will return true if there are no remaining responders working
func (qp *QueryProgress) allDone() bool {
	if qp.NumResponders() > 0 || qp.cancelled {
		// If we have had at least one response, and there aren't any waiting
		// then we are going to assume that everything is done. It is of course
		// possible that there has just been a very fast responder and so a
		// minimum execution time might be a good idea
		return (qp.NumWorking() == 0)
	}
	// If there have been no responders at all we can't say that we're "done"
	return false
}

// stallMonitor watches for stalled connections. It should be passed the
// responder to monitor, the time to wait before marking the connection as
// stalled, and a context. The context is used to allow cancellation of the
// stall monitor from another thread in the case that another message is
// received.
func stallMonitor(context context.Context, timeout time.Duration, responder *Responder, qp *QueryProgress) {
	defer sentry.Recover()
	select {
	case <-context.Done():
		// If the context is cancelled then we don't want to do anything
		return
	case <-time.After(timeout):
		// If the timeout elapses before the context is cancelled it
		// means that we haven't received a response in the expected
		// time, we now need to mark that responder as STALLED
		responder.SetState(ResponderState_STALLED)

		if qp.allDone() {
			qp.Drain()
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
