package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/bufbuild/connect-go"
	josejwt "github.com/go-jose/go-jose/v3/jwt"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	overmind "github.com/overmindtech/api-client"
	"github.com/overmindtech/sdp-go"
	"github.com/overmindtech/sdp-go/sdpconnect"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const UserAgentVersion = "0.1"

// TokenClient Represents something that is capable of getting NATS JWT tokens
// for a given set of NKeys
type TokenClient interface {
	// Returns a NATS token that can be used to connect
	GetJWT() (string, error)

	// Uses the NKeys associated with the token to sign some binary data
	Sign([]byte) ([]byte, error)
}

// BasicTokenClient stores a static token and returns it when called, ignoring
// any provided NKeys or context since it already has the token and doesn't need
// to make any requests
type BasicTokenClient struct {
	staticToken string
	staticKeys  nkeys.KeyPair
}

// NewBasicTokenClient Creates a new basic token client that simply returns a static token
func NewBasicTokenClient(token string, keys nkeys.KeyPair) *BasicTokenClient {
	return &BasicTokenClient{
		staticToken: token,
		staticKeys:  keys,
	}
}

func (b *BasicTokenClient) GetJWT() (string, error) {
	return b.staticToken, nil
}

func (b *BasicTokenClient) Sign(in []byte) ([]byte, error) {
	return b.staticKeys.Sign(in)
}

// ClientCredentialsConfig Authenticates to Overmind using the Client
// Credentials flow
// https://auth0.com/docs/get-started/authentication-and-authorization-flow/client-credentials-flow
type ClientCredentialsConfig struct {
	// The ClientID of the application that we'll be authenticating as
	ClientID string
	// ClientSecret that cirresponds to the ClientID
	ClientSecret string
	// If Account is specified, then the ClientID must have `admin:write`
	// permissions in order to be able to request a token for any account. If
	// this is omitted then the account will be determined based on the account
	// included in the resulting token. This will be stored in the
	// `https://api.overmind.tech/account-name` claim
	Account string
}

// NewOAuthTokenClient Generates a token client that authenticates to OAuth
// using the client credentials flow, then uses that auth to get a NATS token.
// `clientID` and `clientSecret` are used to authenticate using the client
// credentials flow with an API at `oAuthTokenURL`. `overmindAPIURL` is the root
// URL of the NATS token exchange API that will be used e.g.
// https://api.server.test/v1
//
// Tokens will be for the org specified under `org`. Note that the client must
// have admin rights for this
func NewOAuthTokenClient(oAuthTokenURL string, overmindAPIURL string, flowConfig ClientCredentialsConfig) *natsTokenClient {
	conf := &clientcredentials.Config{
		ClientID:     flowConfig.ClientID,
		ClientSecret: flowConfig.ClientSecret,
		TokenURL:     oAuthTokenURL,
		EndpointParams: url.Values{
			"audience": []string{"https://api.overmind.tech"},
		},
	}

	// Get an authenticated client that we can then make more HTTP calls with
	authenticatedClient := conf.Client(context.Background())
	// inject otelhttp propagation
	authenticatedClient.Transport = otelhttp.NewTransport(authenticatedClient.Transport)

	// Configure the token exchange client to use the newly authenticated HTTP
	// client among other things
	tokenExchangeConf := &overmind.Configuration{
		DefaultHeader: make(map[string]string),
		UserAgent:     fmt.Sprintf("Overmind/%v (%v/%v)", UserAgentVersion, runtime.GOOS, runtime.GOARCH),
		Debug:         false,
		Servers: overmind.ServerConfigurations{
			{
				URL:         overmindAPIURL,
				Description: "Overmind API",
			},
		},
		OperationServers: map[string]overmind.ServerConfigurations{},
		HTTPClient:       authenticatedClient,
	}

	nClient := overmind.NewAPIClient(tokenExchangeConf)

	return &natsTokenClient{
		Account:     flowConfig.Account,
		OvermindAPI: nClient,
	}
}

// natsTokenClient A client that is capable of getting NATS JWTs and signing the
// required nonce to prove ownership of the NKeys. Satisfies the `TokenClient`
// interface
type natsTokenClient struct {
	// The name of the account to impersonate. If this is omitted then the
	// account will be determined based on the account included in the resulting
	// token.
	Account string

	// An authenticated client for the Overmind API
	OvermindAPI *overmind.APIClient

	jwt  string
	keys nkeys.KeyPair
}

// generateKeys Generates a new set of keys for the client
func (n *natsTokenClient) generateKeys() error {
	var err error

	n.keys, err = nkeys.CreateUser()

	return err
}

// generateJWT Gets a new JWT from the auth API
func (n *natsTokenClient) generateJWT(ctx context.Context) error {
	if n.OvermindAPI == nil {
		return errors.New("no Overmind API client configured")
	}

	// If we don't yet have keys generate them
	if n.keys == nil {
		err := n.generateKeys()

		if err != nil {
			return err
		}
	}

	var err error
	var pubKey string
	var hostname string
	var response *http.Response

	pubKey, err = n.keys.PublicKey()

	if err != nil {
		return err
	}

	hostname, err = os.Hostname()

	if err != nil {
		return err
	}

	// Create the request for a NATS token
	if n.Account == "" {
		// Use the regular API and let it determine what our org should be
		n.jwt, response, err = n.OvermindAPI.CoreApi.CreateToken(ctx).TokenRequestData(overmind.TokenRequestData{
			UserPubKey: pubKey,
			UserName:   hostname,
		}).Execute()
	} else {
		// Explicitly request an org
		n.jwt, response, err = n.OvermindAPI.AdminApi.AdminCreateToken(ctx, n.Account).TokenRequestData(overmind.TokenRequestData{
			UserPubKey: pubKey,
			UserName:   hostname,
		}).Execute()
	}

	if err != nil {
		errString := fmt.Sprintf("getting NATS token failed: %v", err.Error())

		if response != nil && response.Request != nil && response.Request.URL != nil {
			errString = errString + fmt.Sprintf(". Request URL: %v", response.Request.URL.String())
		}

		return errors.New(errString)
	}

	return nil
}

func (n *natsTokenClient) GetJWT() (string, error) {
	ctx, span := tracer.Start(context.Background(), "connect.GetJWT")
	defer span.End()

	// If we don't yet have a JWT, generate one
	if n.jwt == "" {
		err := n.generateJWT(ctx)

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return "", err
		}
	}

	claims, err := jwt.DecodeUserClaims(n.jwt)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return n.jwt, err
	}

	// Validate to make sure the JWT is valid. If it isn't we'll generate a new
	// one
	var vr jwt.ValidationResults

	claims.Validate(&vr)

	if vr.IsBlocking(true) {
		// Regenerate the token
		err := n.generateJWT(ctx)

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return "", err
		}
	}

	span.SetStatus(codes.Ok, "Completed")
	return n.jwt, nil
}

func (n *natsTokenClient) Sign(in []byte) ([]byte, error) {
	if n.keys == nil {
		err := n.generateKeys()

		if err != nil {
			return []byte{}, err
		}
	}

	return n.keys.Sign(in)
}

// An OAuth2 token source which uses an Overmind API token as a source for OAuth
// tokens
type APIKeyTokenSource struct {
	// The API Key to use to authenticate to the Overmind API
	ApiKey       string
	token        *oauth2.Token
	apiKeyClient sdpconnect.ApiKeyServiceClient
}

func NewAPIKeyTokenSource(apiKey string, overmindAPIURL string) *APIKeyTokenSource {
	httpClient := http.Client{
		Timeout:   10 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	// Create a client that exchanges the API key for a JWT
	apiKeyClient := sdpconnect.NewApiKeyServiceClient(&httpClient, overmindAPIURL)

	return &APIKeyTokenSource{
		ApiKey:       apiKey,
		apiKeyClient: apiKeyClient,
	}
}

// Exchange an API key for an OAuth token
func (ats *APIKeyTokenSource) Token() (*oauth2.Token, error) {
	if ats.token != nil {
		// If we already have a token, and it is valid, return it
		if ats.token.Valid() {
			return ats.token, nil
		}
	}

	// Get a new token
	res, err := ats.apiKeyClient.ExchangeKeyForToken(context.Background(), connect.NewRequest(&sdp.ExchangeKeyForTokenRequest{
		ApiKey: ats.ApiKey,
	}))

	if err != nil {
		return nil, fmt.Errorf("error exchanging API key: %w", err)
	}

	if res.Msg.AccessToken == "" {
		return nil, errors.New("no access token returned")
	}

	// Parse the expiry out of the token
	token, err := josejwt.ParseSigned(res.Msg.AccessToken)

	if err != nil {
		return nil, fmt.Errorf("error parsing JWT: %w", err)
	}

	claims := josejwt.Claims{}

	err = token.UnsafeClaimsWithoutVerification(&claims)

	if err != nil {
		return nil, fmt.Errorf("error parsing JWT claims: %w", err)
	}

	ats.token = &oauth2.Token{
		AccessToken: res.Msg.AccessToken,
		TokenType:   "Bearer",
		Expiry:      claims.Expiry.Time(),
	}

	return ats.token, nil
}

func NewAPIKeyClient(oAuthTokenURL string, overmindAPIURL string, apiKey string) *natsTokenClient {
	// Create a token source that exchanges the API key for an OAuth token
	tokenSource := NewAPIKeyTokenSource(apiKey, overmindAPIURL)
	transport := oauth2.Transport{
		Source: tokenSource,
		Base:   http.DefaultTransport,
	}
	httpClient := http.Client{
		Transport: otelhttp.NewTransport(&transport),
	}

	// Create a client for the token exchange API
	tokenExchangeClient := overmind.NewAPIClient(&overmind.Configuration{
		DefaultHeader: make(map[string]string),
		UserAgent:     fmt.Sprintf("Overmind/%v (%v/%v)", UserAgentVersion, runtime.GOOS, runtime.GOARCH),
		Debug:         false,
		Servers: overmind.ServerConfigurations{
			{
				URL:         overmindAPIURL,
				Description: "Overmind API",
			},
		},
		OperationServers: map[string]overmind.ServerConfigurations{},
		HTTPClient:       &httpClient,
	})

	return &natsTokenClient{
		OvermindAPI: tokenExchangeClient,
	}
}