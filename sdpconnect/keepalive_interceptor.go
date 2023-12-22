package sdpconnect

import (
	"context"

	"connectrpc.com/connect"
	"github.com/overmindtech/sdp-go"
)

// Create a new interceptor that will ensure that sources are alive on all
// requests. This interceptor will call `KeepaliveSources` on the management
// service to ensure that the sources are alive. This will be done in a
// goroutine so that the request is not blocked. If the management service is
// not set, then this interceptor will do nothing.
//
// For services that actually require the sources to be alive, they can use the
// WaitForSources function to wait for the sources to be ready. This function
// will block until the sources are ready.
func NewKeepaliveSourcesInterceptor(managementClient ManagementServiceClient) connect.Interceptor {
	return &KeepaliveSourcesInterceptor{
		management: managementClient,
	}
}

// WaitForSources will wait for the sources to be ready after they have been
// woken up by the `KeepaliveSourcesInterceptor`. If this context was create
// without the interceptor, then this function will return immediately. If the
// waking of the sources returns an error it will be returned via this function
func WaitForSources(ctx context.Context) error {
	// Check the context key
	if readyFunc := ctx.Value(keepaliveSourcesReadyContextKey{}); readyFunc != nil {
		// Call the function
		return readyFunc.(waitForSourcesFunc)()
	} else {
		// Return immediately
		return nil
	}
}

type KeepaliveSourcesInterceptor struct {
	management ManagementServiceClient
}

// keepaliveSourcesReadyContextKey is the context key used to determine if the
// keepalive sources interceptor has run and the sources are ready
type keepaliveSourcesReadyContextKey struct{}

// A func that waits for the sources to be ready
type waitForSourcesFunc func() error

func (i *KeepaliveSourcesInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return connect.UnaryFunc(func(ctx context.Context, ar connect.AnyRequest) (connect.AnyResponse, error) {
		// Wake the sources
		ctx = i.wakeSources(ctx)

		return next(ctx, ar)
	})
}

func (i *KeepaliveSourcesInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return connect.StreamingClientFunc(func(ctx context.Context, s connect.Spec) connect.StreamingClientConn {
		// Wake the sources
		ctx = i.wakeSources(ctx)

		return next(ctx, s)
	})
}

func (i *KeepaliveSourcesInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return connect.StreamingHandlerFunc(func(ctx context.Context, shc connect.StreamingHandlerConn) error {
		// Wake the sources
		ctx = i.wakeSources(ctx)

		return next(ctx, shc)
	})
}

// Actually does the work of waking the sources and attaching the channel to the
// context. Returns a new context that has the channel attached to it
func (i *KeepaliveSourcesInterceptor) wakeSources(ctx context.Context) context.Context {
	if i.management == nil {
		return ctx
	}

	// If the function has already been set, then we don't need to do
	// anything since the middleware has already run
	if readyFunc := ctx.Value(keepaliveSourcesReadyContextKey{}); readyFunc != nil {
		return ctx
	}

	// Create a buffered channel so that if the value is never used, the
	// goroutine that keeps the sources awake can close. This will be
	// garbage collected when there are no longer any references to it,
	// which will happen once the context is garbage collected after the
	// request is fully completed
	sourcesReady := make(chan error, 1)

	// Attach a function to the context that will wait for the sources to be
	// ready
	ctx = context.WithValue(ctx, keepaliveSourcesReadyContextKey{}, waitForSourcesFunc(func() error {
		return <-sourcesReady
	}))

	// Make the request in another goroutine so that we don't block the
	// request
	go func() {
		defer close(sourcesReady)

		// Make the request to keep the source awake
		_, err := i.management.KeepaliveSources(ctx, &connect.Request[sdp.KeepaliveSourcesRequest]{
			Msg: &sdp.KeepaliveSourcesRequest{
				WaitForHealthy: true,
			},
		})

		// Send the error to the channel
		sourcesReady <- err
	}()

	return ctx
}
