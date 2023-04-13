package sdp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// AuthBypassedContextKey is a key that is stored in the request context when auth is
// actively being bypassed, e.g. in development. When this is set the
// `HasScopes()` function will always return true, and can be set using the
// `BypassAuth()` middleware.
type AuthBypassedContextKey struct{}

// CustomClaimsContextKey is the key that is used to store the custom claims
// from the JWT
type CustomClaimsContextKey struct{}

// AuthConfig Configuration for the auth middleware
type AuthConfig struct {
	// Bypasses all auth checks, meaning that HasScopes() will always return
	// true. This should be used in conjunction with the `AccountOverride` field
	// since there won't be a token to parse the account from
	BypassAuth bool

	// Overrides the account name stored in the CustomClaimsContextKey
	AccountOverride *string

	// Overrides the scope stored in the CustomClaimsContextKey
	ScopeOverride *string
}

// HasScopes checks that the authenticated user in the request context has the
// required scopes. If auth has been bypassed, this will always return true
func HasScopes(ctx context.Context, requiredScopes ...string) bool {
	if ctx.Value(AuthBypassedContextKey{}) == true {
		trace.SpanFromContext(ctx).SetAttributes(attribute.Bool("om.auth.bypass", true))

		// Bypass all auth
		return true
	}

	claims := ctx.Value(CustomClaimsContextKey{}).(*CustomClaims)
	trace.SpanFromContext(ctx).SetAttributes(
		attribute.StringSlice("om.auth.requiredScopes", requiredScopes),
	)
	for _, scope := range requiredScopes {
		if !claims.HasScope(scope) {
			return false
		}
	}
	return true
}

// NewAuthMiddleware Creates new auth middleware. The options allow you to
// bypass the authentication process or not, but either way this middleware will
// set the `CustomClaimsContextKey` in the request context which allows you to
// use the `HasScopes()` function to check the scopes without having to worry
// about whether the server is using auth or not.
//
// If auth is not bypassed, then tokens will be validated using Auth0 and
// therefore the following environment variables must be set: AUTH0_DOMAIN,
// AUTH0_AUDIENCE. If cookie auth is intended to be used, then AUTH_COOKIE_NAME
// must also be set.
func NewAuthMiddleware(config AuthConfig, next http.Handler) http.Handler {
	processOverrides := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := OverrideCustomClaims(r.Context(), config.ScopeOverride, config.AccountOverride)

		r = r.Clone(ctx)

		next.ServeHTTP(w, r)
	})

	if config.BypassAuth {
		return bypassAuthHandler(*config.AccountOverride, processOverrides)
	} else {
		return ensureValidTokenHandler(processOverrides)
	}
}

// AddBypassAuthConfig Adds the requires keys to the context so that
// authentication is bypassed. This is intended to be used in tests
func AddBypassAuthConfig(ctx context.Context) context.Context {
	return context.WithValue(ctx, AuthBypassedContextKey{}, true)
}

// OverrideCustomClaims Overrides the custom claims in the context that have
// been set at CustomClaimsContextKey
func OverrideCustomClaims(ctx context.Context, scope *string, account *string) context.Context {
	// Read existing claims from the context
	i := ctx.Value(CustomClaimsContextKey{})

	var claims *CustomClaims
	var ok bool

	if claims, ok = i.(*CustomClaims); !ok {
		// Create a new object if required
		claims = &CustomClaims{}
	}

	if scope != nil {
		claims.Scope = *scope
	}

	if account != nil {
		claims.AccountName = *account
	}

	// Store the new claims in the context
	ctx = context.WithValue(ctx, CustomClaimsContextKey{}, claims)

	return ctx
}

// bypassAuthHandler is a middleware that will bypass authentication
func bypassAuthHandler(accountName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := AddBypassAuthConfig(r.Context())

		r = r.Clone(ctx)

		next.ServeHTTP(w, r)
	})
}

// ensureValidTokenHandler is a middleware that will check the validity of our JWT.
//
// This requires the following environment variables to be set as per the Auth0
// standards: AUTH0_DOMAIN, AUTH0_AUDIENCE, AUTH_COOKIE_NAME
//
// This middleware also extract custom claims form the token and stores them in
// CustomClaimsContextKey
func ensureValidTokenHandler(next http.Handler) http.Handler {
	issuerURL, err := url.Parse("https://" + os.Getenv("AUTH0_DOMAIN") + "/")
	if err != nil {
		log.Fatalf("Failed to parse the issuer url: %v", err)
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{os.Getenv("AUTH0_AUDIENCE")},
		validator.WithCustomClaims(
			func() validator.CustomClaims {
				return &CustomClaims{}
			},
		),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to set up the jwt validator")
	}

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Encountered error while validating JWT: %v", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Failed to validate JWT."}`))
	}

	// Set up token extractors based on what env vars are available
	extractors := []jwtmiddleware.TokenExtractor{
		jwtmiddleware.AuthHeaderTokenExtractor,
	}

	if name := os.Getenv("AUTH_COOKIE_NAME"); name != "" {
		extractors = append(extractors, jwtmiddleware.CookieTokenExtractor(name))
	}

	middleware := jwtmiddleware.New(
		jwtValidator.ValidateToken,
		jwtmiddleware.WithErrorHandler(errorHandler),
		jwtmiddleware.WithTokenExtractor(jwtmiddleware.MultiTokenExtractor(extractors...)),
	)

	return middleware.CheckJWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// extract account name and setup otel attributes after the JWT was validated, but before the actual handler runs
		claims := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
		customClaims := claims.CustomClaims.(*CustomClaims)

		if customClaims != nil {
			r = r.Clone(context.WithValue(r.Context(), CustomClaimsContextKey{}, customClaims))

			trace.SpanFromContext(r.Context()).SetAttributes(
				attribute.String("om.auth.scopes", customClaims.Scope),
				attribute.Int64("om.auth.expiry", claims.RegisteredClaims.Expiry),
				attribute.String("om.auth.accountName", customClaims.AccountName),
			)

			next.ServeHTTP(w, r)
		} else {
			errorHandler(w, r, fmt.Errorf("couldn't get claims from: %v", claims))
			return
		}
	}))
}

// CustomClaims contains custom data we want from the token.
type CustomClaims struct {
	Scope       string `json:"scope"`
	AccountName string `json:"https://api.overmind.tech/account-name"`
}

// HasScope checks whether our claims have a specific scope.
func (c CustomClaims) HasScope(expectedScope string) bool {
	result := strings.Split(c.Scope, " ")
	for i := range result {
		if result[i] == expectedScope {
			return true
		}
	}

	return false
}

// Validate does nothing for this example, but we need
// it to satisfy validator.CustomClaims interface.
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}
