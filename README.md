# SDP Go Libraries

[![Go Reference](https://pkg.go.dev/badge/github.com/overmindtech/sdp-go.svg)](https://pkg.go.dev/github.com/overmindtech/sdp-go)

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

## Linked Item Query Extraction

This package provides some helper methods to extract linked items from unknown data structures. This is intended to be used for sections of config that are likely to have interesting data, but in a format that we don't know about. A good example would be a the env vars of a kubernetes pod.

This supports extracting the following formats:

- IP addresses
- HTTP/HTTPS URLs
- DNS names
- AWS ARNs

### `ExtractLinksFromAttributes`

```go
func ExtractLinksFromAttributes(attributes *ItemAttributes) []*LinkedItemQuery
```

This function attempts to extract linked item queries from the attributes of an item. It is designed to be used on items known to potentially contain references that can be discovered, but are in an unstructured format from which linked item queries cannot be directly constructed.

### `ExtractLinksViaJSON`

```go
func ExtractLinksViaJSON(i any) ([]*LinkedItemQuery, error)
```

This function performs the same operation as `ExtractLinksFromAttributes`, but takes any input format and converts it to a `map[string]interface{}` via JSON. It then extracts the linked item queries in a similar manner to `ExtractLinksFromAttributes`.
