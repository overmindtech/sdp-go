// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: apikey.proto

package sdpconnect

import (
	context "context"
	errors "errors"
	connect_go "github.com/bufbuild/connect-go"
	sdp_go "github.com/overmindtech/sdp-go"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect_go.IsAtLeastVersion0_1_0

const (
	// ApiKeyServiceName is the fully-qualified name of the ApiKeyService service.
	ApiKeyServiceName = "apikeys.ApiKeyService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// ApiKeyServiceCreateAPIKeyProcedure is the fully-qualified name of the ApiKeyService's
	// CreateAPIKey RPC.
	ApiKeyServiceCreateAPIKeyProcedure = "/apikeys.ApiKeyService/CreateAPIKey"
	// ApiKeyServiceGetAPIKeyProcedure is the fully-qualified name of the ApiKeyService's GetAPIKey RPC.
	ApiKeyServiceGetAPIKeyProcedure = "/apikeys.ApiKeyService/GetAPIKey"
	// ApiKeyServiceUpdateAPIKeyProcedure is the fully-qualified name of the ApiKeyService's
	// UpdateAPIKey RPC.
	ApiKeyServiceUpdateAPIKeyProcedure = "/apikeys.ApiKeyService/UpdateAPIKey"
	// ApiKeyServiceListAPIKeysProcedure is the fully-qualified name of the ApiKeyService's ListAPIKeys
	// RPC.
	ApiKeyServiceListAPIKeysProcedure = "/apikeys.ApiKeyService/ListAPIKeys"
	// ApiKeyServiceDeleteAPIKeyProcedure is the fully-qualified name of the ApiKeyService's
	// DeleteAPIKey RPC.
	ApiKeyServiceDeleteAPIKeyProcedure = "/apikeys.ApiKeyService/DeleteAPIKey"
	// ApiKeyServiceExchangeKeyForTokenProcedure is the fully-qualified name of the ApiKeyService's
	// ExchangeKeyForToken RPC.
	ApiKeyServiceExchangeKeyForTokenProcedure = "/apikeys.ApiKeyService/ExchangeKeyForToken"
)

// ApiKeyServiceClient is a client for the apikeys.ApiKeyService service.
type ApiKeyServiceClient interface {
	// Creates an API key, pending access token generation from Auth0. The key
	// cannot be used until the user has been redirected to the given URL which
	// allows Auth0 to actually generate an access token
	CreateAPIKey(context.Context, *connect_go.Request[sdp_go.CreateAPIKeyRequest]) (*connect_go.Response[sdp_go.CreateAPIKeyResponse], error)
	GetAPIKey(context.Context, *connect_go.Request[sdp_go.GetAPIKeyRequest]) (*connect_go.Response[sdp_go.GetAPIKeyResponse], error)
	UpdateAPIKey(context.Context, *connect_go.Request[sdp_go.UpdateAPIKeyRequest]) (*connect_go.Response[sdp_go.UpdateAPIKeyResponse], error)
	ListAPIKeys(context.Context, *connect_go.Request[sdp_go.ListAPIKeysRequest]) (*connect_go.Response[sdp_go.ListAPIKeysResponse], error)
	DeleteAPIKey(context.Context, *connect_go.Request[sdp_go.DeleteAPIKeyRequest]) (*connect_go.Response[sdp_go.DeleteAPIKeyResponse], error)
	// Exchanges an Overmind API key for an Oauth access token. That token can
	// then be used to access all other Overmind APIs
	ExchangeKeyForToken(context.Context, *connect_go.Request[sdp_go.ExchangeKeyForTokenRequest]) (*connect_go.Response[sdp_go.ExchangeKeyForTokenResponse], error)
}

// NewApiKeyServiceClient constructs a client for the apikeys.ApiKeyService service. By default, it
// uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewApiKeyServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) ApiKeyServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &apiKeyServiceClient{
		createAPIKey: connect_go.NewClient[sdp_go.CreateAPIKeyRequest, sdp_go.CreateAPIKeyResponse](
			httpClient,
			baseURL+ApiKeyServiceCreateAPIKeyProcedure,
			opts...,
		),
		getAPIKey: connect_go.NewClient[sdp_go.GetAPIKeyRequest, sdp_go.GetAPIKeyResponse](
			httpClient,
			baseURL+ApiKeyServiceGetAPIKeyProcedure,
			opts...,
		),
		updateAPIKey: connect_go.NewClient[sdp_go.UpdateAPIKeyRequest, sdp_go.UpdateAPIKeyResponse](
			httpClient,
			baseURL+ApiKeyServiceUpdateAPIKeyProcedure,
			opts...,
		),
		listAPIKeys: connect_go.NewClient[sdp_go.ListAPIKeysRequest, sdp_go.ListAPIKeysResponse](
			httpClient,
			baseURL+ApiKeyServiceListAPIKeysProcedure,
			opts...,
		),
		deleteAPIKey: connect_go.NewClient[sdp_go.DeleteAPIKeyRequest, sdp_go.DeleteAPIKeyResponse](
			httpClient,
			baseURL+ApiKeyServiceDeleteAPIKeyProcedure,
			opts...,
		),
		exchangeKeyForToken: connect_go.NewClient[sdp_go.ExchangeKeyForTokenRequest, sdp_go.ExchangeKeyForTokenResponse](
			httpClient,
			baseURL+ApiKeyServiceExchangeKeyForTokenProcedure,
			opts...,
		),
	}
}

// apiKeyServiceClient implements ApiKeyServiceClient.
type apiKeyServiceClient struct {
	createAPIKey        *connect_go.Client[sdp_go.CreateAPIKeyRequest, sdp_go.CreateAPIKeyResponse]
	getAPIKey           *connect_go.Client[sdp_go.GetAPIKeyRequest, sdp_go.GetAPIKeyResponse]
	updateAPIKey        *connect_go.Client[sdp_go.UpdateAPIKeyRequest, sdp_go.UpdateAPIKeyResponse]
	listAPIKeys         *connect_go.Client[sdp_go.ListAPIKeysRequest, sdp_go.ListAPIKeysResponse]
	deleteAPIKey        *connect_go.Client[sdp_go.DeleteAPIKeyRequest, sdp_go.DeleteAPIKeyResponse]
	exchangeKeyForToken *connect_go.Client[sdp_go.ExchangeKeyForTokenRequest, sdp_go.ExchangeKeyForTokenResponse]
}

// CreateAPIKey calls apikeys.ApiKeyService.CreateAPIKey.
func (c *apiKeyServiceClient) CreateAPIKey(ctx context.Context, req *connect_go.Request[sdp_go.CreateAPIKeyRequest]) (*connect_go.Response[sdp_go.CreateAPIKeyResponse], error) {
	return c.createAPIKey.CallUnary(ctx, req)
}

// GetAPIKey calls apikeys.ApiKeyService.GetAPIKey.
func (c *apiKeyServiceClient) GetAPIKey(ctx context.Context, req *connect_go.Request[sdp_go.GetAPIKeyRequest]) (*connect_go.Response[sdp_go.GetAPIKeyResponse], error) {
	return c.getAPIKey.CallUnary(ctx, req)
}

// UpdateAPIKey calls apikeys.ApiKeyService.UpdateAPIKey.
func (c *apiKeyServiceClient) UpdateAPIKey(ctx context.Context, req *connect_go.Request[sdp_go.UpdateAPIKeyRequest]) (*connect_go.Response[sdp_go.UpdateAPIKeyResponse], error) {
	return c.updateAPIKey.CallUnary(ctx, req)
}

// ListAPIKeys calls apikeys.ApiKeyService.ListAPIKeys.
func (c *apiKeyServiceClient) ListAPIKeys(ctx context.Context, req *connect_go.Request[sdp_go.ListAPIKeysRequest]) (*connect_go.Response[sdp_go.ListAPIKeysResponse], error) {
	return c.listAPIKeys.CallUnary(ctx, req)
}

// DeleteAPIKey calls apikeys.ApiKeyService.DeleteAPIKey.
func (c *apiKeyServiceClient) DeleteAPIKey(ctx context.Context, req *connect_go.Request[sdp_go.DeleteAPIKeyRequest]) (*connect_go.Response[sdp_go.DeleteAPIKeyResponse], error) {
	return c.deleteAPIKey.CallUnary(ctx, req)
}

// ExchangeKeyForToken calls apikeys.ApiKeyService.ExchangeKeyForToken.
func (c *apiKeyServiceClient) ExchangeKeyForToken(ctx context.Context, req *connect_go.Request[sdp_go.ExchangeKeyForTokenRequest]) (*connect_go.Response[sdp_go.ExchangeKeyForTokenResponse], error) {
	return c.exchangeKeyForToken.CallUnary(ctx, req)
}

// ApiKeyServiceHandler is an implementation of the apikeys.ApiKeyService service.
type ApiKeyServiceHandler interface {
	// Creates an API key, pending access token generation from Auth0. The key
	// cannot be used until the user has been redirected to the given URL which
	// allows Auth0 to actually generate an access token
	CreateAPIKey(context.Context, *connect_go.Request[sdp_go.CreateAPIKeyRequest]) (*connect_go.Response[sdp_go.CreateAPIKeyResponse], error)
	GetAPIKey(context.Context, *connect_go.Request[sdp_go.GetAPIKeyRequest]) (*connect_go.Response[sdp_go.GetAPIKeyResponse], error)
	UpdateAPIKey(context.Context, *connect_go.Request[sdp_go.UpdateAPIKeyRequest]) (*connect_go.Response[sdp_go.UpdateAPIKeyResponse], error)
	ListAPIKeys(context.Context, *connect_go.Request[sdp_go.ListAPIKeysRequest]) (*connect_go.Response[sdp_go.ListAPIKeysResponse], error)
	DeleteAPIKey(context.Context, *connect_go.Request[sdp_go.DeleteAPIKeyRequest]) (*connect_go.Response[sdp_go.DeleteAPIKeyResponse], error)
	// Exchanges an Overmind API key for an Oauth access token. That token can
	// then be used to access all other Overmind APIs
	ExchangeKeyForToken(context.Context, *connect_go.Request[sdp_go.ExchangeKeyForTokenRequest]) (*connect_go.Response[sdp_go.ExchangeKeyForTokenResponse], error)
}

// NewApiKeyServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewApiKeyServiceHandler(svc ApiKeyServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	apiKeyServiceCreateAPIKeyHandler := connect_go.NewUnaryHandler(
		ApiKeyServiceCreateAPIKeyProcedure,
		svc.CreateAPIKey,
		opts...,
	)
	apiKeyServiceGetAPIKeyHandler := connect_go.NewUnaryHandler(
		ApiKeyServiceGetAPIKeyProcedure,
		svc.GetAPIKey,
		opts...,
	)
	apiKeyServiceUpdateAPIKeyHandler := connect_go.NewUnaryHandler(
		ApiKeyServiceUpdateAPIKeyProcedure,
		svc.UpdateAPIKey,
		opts...,
	)
	apiKeyServiceListAPIKeysHandler := connect_go.NewUnaryHandler(
		ApiKeyServiceListAPIKeysProcedure,
		svc.ListAPIKeys,
		opts...,
	)
	apiKeyServiceDeleteAPIKeyHandler := connect_go.NewUnaryHandler(
		ApiKeyServiceDeleteAPIKeyProcedure,
		svc.DeleteAPIKey,
		opts...,
	)
	apiKeyServiceExchangeKeyForTokenHandler := connect_go.NewUnaryHandler(
		ApiKeyServiceExchangeKeyForTokenProcedure,
		svc.ExchangeKeyForToken,
		opts...,
	)
	return "/apikeys.ApiKeyService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case ApiKeyServiceCreateAPIKeyProcedure:
			apiKeyServiceCreateAPIKeyHandler.ServeHTTP(w, r)
		case ApiKeyServiceGetAPIKeyProcedure:
			apiKeyServiceGetAPIKeyHandler.ServeHTTP(w, r)
		case ApiKeyServiceUpdateAPIKeyProcedure:
			apiKeyServiceUpdateAPIKeyHandler.ServeHTTP(w, r)
		case ApiKeyServiceListAPIKeysProcedure:
			apiKeyServiceListAPIKeysHandler.ServeHTTP(w, r)
		case ApiKeyServiceDeleteAPIKeyProcedure:
			apiKeyServiceDeleteAPIKeyHandler.ServeHTTP(w, r)
		case ApiKeyServiceExchangeKeyForTokenProcedure:
			apiKeyServiceExchangeKeyForTokenHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedApiKeyServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedApiKeyServiceHandler struct{}

func (UnimplementedApiKeyServiceHandler) CreateAPIKey(context.Context, *connect_go.Request[sdp_go.CreateAPIKeyRequest]) (*connect_go.Response[sdp_go.CreateAPIKeyResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("apikeys.ApiKeyService.CreateAPIKey is not implemented"))
}

func (UnimplementedApiKeyServiceHandler) GetAPIKey(context.Context, *connect_go.Request[sdp_go.GetAPIKeyRequest]) (*connect_go.Response[sdp_go.GetAPIKeyResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("apikeys.ApiKeyService.GetAPIKey is not implemented"))
}

func (UnimplementedApiKeyServiceHandler) UpdateAPIKey(context.Context, *connect_go.Request[sdp_go.UpdateAPIKeyRequest]) (*connect_go.Response[sdp_go.UpdateAPIKeyResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("apikeys.ApiKeyService.UpdateAPIKey is not implemented"))
}

func (UnimplementedApiKeyServiceHandler) ListAPIKeys(context.Context, *connect_go.Request[sdp_go.ListAPIKeysRequest]) (*connect_go.Response[sdp_go.ListAPIKeysResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("apikeys.ApiKeyService.ListAPIKeys is not implemented"))
}

func (UnimplementedApiKeyServiceHandler) DeleteAPIKey(context.Context, *connect_go.Request[sdp_go.DeleteAPIKeyRequest]) (*connect_go.Response[sdp_go.DeleteAPIKeyResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("apikeys.ApiKeyService.DeleteAPIKey is not implemented"))
}

func (UnimplementedApiKeyServiceHandler) ExchangeKeyForToken(context.Context, *connect_go.Request[sdp_go.ExchangeKeyForTokenRequest]) (*connect_go.Response[sdp_go.ExchangeKeyForTokenResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("apikeys.ApiKeyService.ExchangeKeyForToken is not implemented"))
}