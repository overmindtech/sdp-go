[![Go Reference](https://pkg.go.dev/badge/github.com/overmindtech/sdp-go.svg)](https://pkg.go.dev/github.com/overmindtech/sdp-go)

# SDP Go Libraries

A set of Golang libraries for [State Description Protocol](https://github.com/overmindtech/sdp)

## Auth

These libraries contain an `EnsureValidToken` HTTP middleware that can be used as follows:

```go
router := http.NewServeMux()
router.Handle(withCORS(
    sdp.EnsureValidTokenWithPattern(
        sdpconnect.NewBookmarksServiceHandler(
            &bookmarkHandler,
            connect.WithInterceptors(otelconnect.NewInterceptor(otelconnect.WithTrustRemote(), otelconnect.WithoutTraceEvents())),
        ))))
router.Handle(withCORS(
    sdp.EnsureValidTokenWithPattern(
        sdpconnect.NewSnapshotsServiceHandler(
            &snapshotHandler,
            connect.WithInterceptors(otelconnect.NewInterceptor(otelconnect.WithTrustRemote(), otelconnect.WithoutTraceEvents())),
        ))))

serverAddress := fmt.Sprintf(":%v", "8080")
gatewayHTTPServer = &http.Server{
    Addr:    serverAddress,
    Handler: router,
}

err := gatewayHTTPServer.ListenAndServe()
```

Note however that using this will require the following environment variables to be present:

|Name|Description|
|----|-----------|
|`AUTH0_DOMAIN`| The domain to validate token against e.g. `om-dogfood.eu.auth0.com`|
|`AUTH0_AUDIENCE`| The audience e.g. `https://api.overmind.tech`|
|`AUTH_COOKIE_NAME`| *(Optional)* The name of the cookie to extract a token from if not present in the `Authorization` header|