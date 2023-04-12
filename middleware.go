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

// AccountNameContextKey is the key used in the request
// context where the account name from a
// validated JWT will be stored.
type AccountNameContextKey struct{}

// AuthBypassed is a key that is stored in the request context when authis us
// actively being bypassed, e.g. in development. When this is set the
// `HasScopes()` function will always return true, and can be set using the
// `BypassAuth()` middleware.
type AuthBypassed struct{}

// TODO: return connect_go.Response with error
func HasScopes(ctx context.Context, requiredScopes ...string) bool {
	if ctx.Value(AuthBypassed{}) == true {
		// Bypass all auth
		return true
	}

	token := ctx.Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*CustomClaims)
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

// BypassAuth is a middleware that will bypass authentication and set the
// account name to the given string
func BypassAuth(accountName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the account name to "unauthenticated"
		r = r.Clone(context.WithValue(r.Context(), AccountNameContextKey{}, accountName))
		r = r.Clone(context.WithValue(r.Context(), AuthBypassed{}, true))

		next.ServeHTTP(w, r)
	})
}

// EnsureValidTokenWithPattern is a middleware that will check the validity of
// our JWT, returning the pattern and a handler
func EnsureValidTokenWithPattern(pattern string, next http.Handler) (string, http.Handler) {
	return pattern, EnsureValidToken(next)
}

// EnsureValidToken is a middleware that will check the validity of our JWT.
//
// This requires the following environment variables to be set as per the Auth0
// standards:
//
// - AUTH0_DOMAIN
// - AUTH0_AUDIENCE
// - AUTH_COOKIE_NAME
func EnsureValidToken(next http.Handler) http.Handler {
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
			var accountName string
			if customClaims.AccountName != "" {
				accountName = customClaims.AccountName
			}
			if accountName != "" {
				r = r.Clone(context.WithValue(r.Context(), AccountNameContextKey{}, accountName))
			} else {
				errorHandler(w, r, fmt.Errorf("couldn't get 'https://api.overmind.tech/account-name' claim from: %v", claims.CustomClaims))
				return
			}
			trace.SpanFromContext(r.Context()).SetAttributes(
				attribute.String("om.auth.scopes", customClaims.Scope),
				attribute.Int64("om.auth.expiry", claims.RegisteredClaims.Expiry),
				attribute.String("om.auth.accountName", accountName),
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
