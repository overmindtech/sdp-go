package sdp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	log "github.com/sirupsen/logrus"
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

// AccountNameContextKey is the key that is used to store the currently acting
// account name
type AccountNameContextKey struct{}

// UserTokenContextKey is the key that is used to store the full JWT token of the user
type UserTokenContextKey struct{}

// CurrentSubjectContextKey is the key that is used to store the current subject attribute.
// This will be the auth0 `user_id` from the tokens `sub` claim.
type CurrentSubjectContextKey struct{}

// AuthConfig Configuration for the auth middleware
type AuthConfig struct {
	Auth0Domain    string
	Auth0Audience  string
	AuthCookieName string // leave this empty to disable cookie auth

	// Bypasses all auth checks, meaning that HasScopes() will always return
	// true. This should be used in conjunction with the `AccountOverride` field
	// since there won't be a token to parse the account from
	BypassAuth bool

	// Bypasses auth for the given paths. This is a regular expression that is
	// matched against the path of the request. If the regex matches then the
	// request will be allowed through without auth. This should be used with
	// `AccountOverride` in order to avoid the required context values not being
	// set and therefore causing issues (probably nil pointer panics)
	BypassAuthForPaths *regexp.Regexp

	// Overrides the account name stored in the CustomClaimsContextKey
	AccountOverride *string

	// Overrides the scope stored in the CustomClaimsContextKey
	ScopeOverride *string
}

// NewAuthConfigFromEnv Creates a new AuthConfig from the default Auth0
// environment variables, namely: AUTH0_DOMAIN, AUTH0_AUDIENCE, and
// AUTH_COOKIE_NAME
func NewAuthConfigFromEnv() AuthConfig {
	return AuthConfig{
		Auth0Domain:    os.Getenv("AUTH0_DOMAIN"),
		Auth0Audience:  os.Getenv("AUTH0_AUDIENCE"),
		AuthCookieName: os.Getenv("AUTH_COOKIE_NAME"),
	}
}

// HasScopes compatibility alias for HasAllScopes
func HasScopes(ctx context.Context, requiredScopes ...string) bool {
	return HasAllScopes(ctx, requiredScopes...)
}

// HasAllScopes checks that the authenticated user in the request context has all the
// required scopes. If auth has been bypassed, this will always return true
func HasAllScopes(ctx context.Context, requiredScopes ...string) bool {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.StringSlice("ovm.auth.requiredScopes.all", requiredScopes),
	)

	if ctx.Value(AuthBypassedContextKey{}) == true {
		// this is always set when auth is bypassed
		// set it here again to capture non-standard auth configs
		span.SetAttributes(attribute.Bool("ovm.auth.bypass", true))

		// Bypass all auth
		return true
	}

	claims, ok := ctx.Value(CustomClaimsContextKey{}).(*CustomClaims)
	if !ok {
		span.SetAttributes(attribute.String("ovm.auth.missingClaims", "all"))
		return false
	}

	for _, scope := range requiredScopes {
		if !claims.HasScope(scope) {
			span.SetAttributes(attribute.String("ovm.auth.missingClaims", scope))
			return false
		}
	}
	return true
}

// HasAnyScopes checks that the authenticated user in the request context has any of the
// required scopes. If auth has been bypassed, this will always return true
func HasAnyScopes(ctx context.Context, requiredScopes ...string) bool {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.StringSlice("ovm.auth.requiredScopes.any", requiredScopes),
	)

	if ctx.Value(AuthBypassedContextKey{}) == true {
		// this is always set when auth is bypassed
		// set it here again to capture non-standard auth configs
		span.SetAttributes(attribute.Bool("ovm.auth.bypass", true))

		// Bypass all auth
		return true
	}

	claims, ok := ctx.Value(CustomClaimsContextKey{}).(*CustomClaims)
	if !ok {
		span.SetAttributes(attribute.String("ovm.auth.missingClaims", "all"))
		return false
	}

	span.SetAttributes(
		attribute.String("ovm.auth.scopes", claims.Scope),
	)

	for _, scope := range requiredScopes {
		if claims.HasScope(scope) {
			span.SetAttributes(attribute.String("ovm.auth.usedClaim", scope))
			return true
		}
	}
	return false
}

var ErrNoClaims = errors.New("error extracting claims from token")

// ExtractAccount Extracts the account name from a context
func ExtractAccount(ctx context.Context) (string, error) {
	claims := ctx.Value(CustomClaimsContextKey{})

	if claims == nil {
		return "", ErrNoClaims
	}

	return claims.(*CustomClaims).AccountName, nil
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

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bypassPath bool
		var accountOverride string

		if config.BypassAuthForPaths != nil {
			bypassPath = config.BypassAuthForPaths.MatchString(r.URL.Path)
		}

		if config.AccountOverride != nil {
			accountOverride = *config.AccountOverride
		}

		if config.BypassAuth || bypassPath {
			// If auth is disabled then bypass
			span := trace.SpanFromContext(r.Context())
			// this is always set when auth is bypassed
			span.SetAttributes(attribute.Bool("ovm.auth.bypass", true))
			if bypassPath {
				span.SetAttributes(attribute.String("ovm.auth.bypassedPath", r.URL.Path))
			}
			bypassAuthHandler(accountOverride, processOverrides).ServeHTTP(w, r)
		} else {
			// Otherwise ensure the token is valid
			ensureValidTokenHandler(config, processOverrides).ServeHTTP(w, r)
		}
	})
}

// AddBypassAuthConfig Adds the requires keys to the context so that
// authentication is bypassed. This is intended to be used in tests
func AddBypassAuthConfig(ctx context.Context) context.Context {
	return context.WithValue(ctx, AuthBypassedContextKey{}, true)
}

// OverrideAuthContext overrides the authentication data and token stored in the context.
// This is mostly useful for testing or delegating access locally into a protected API.
func OverrideAuthContext(ctx context.Context, claims *validator.ValidatedClaims) context.Context {
	customClaims := claims.CustomClaims.(*CustomClaims)
	ctx = context.WithValue(ctx, jwtmiddleware.ContextKey{}, claims)
	ctx = context.WithValue(ctx, CustomClaimsContextKey{}, customClaims)
	ctx = context.WithValue(ctx, CurrentSubjectContextKey{}, claims.RegisteredClaims.Subject)
	ctx = context.WithValue(ctx, AccountNameContextKey{}, customClaims.AccountName)
	return ctx
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
	ctx = context.WithValue(ctx, AccountNameContextKey{}, claims.AccountName)

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

// ensureValidTokenHandler is a middleware that will check the validity of our
// JWT.
//
// # This uses the AuthConfig data to configure token validation. If there is no
// Auth0 values (i.e. Auth0Domain, Auth0Audience and AuthCookieName  are empty
// strings) then this will use the environment variables: AUTH0_DOMAIN,
// AUTH0_AUDIENCE, and AUTH_COOKIE_NAME
//
// This middleware also extract custom claims form the token and stores them in
// CustomClaimsContextKey
func ensureValidTokenHandler(config AuthConfig, next http.Handler) http.Handler {
	if config.Auth0Domain == "" && config.Auth0Audience == "" && config.AuthCookieName == "" {
		config.Auth0Domain = os.Getenv("AUTH0_DOMAIN")
		config.Auth0Audience = os.Getenv("AUTH0_AUDIENCE")
		config.AuthCookieName = os.Getenv("AUTH_COOKIE_NAME")
	}

	issuerURL, err := url.Parse("https://" + config.Auth0Domain + "/")
	if err != nil {
		log.Fatalf("Failed to parse the issuer url: %v", err)
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{config.Auth0Audience},
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
		log.WithContext(r.Context()).WithError(err).Errorf("Encountered error while validating JWT")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Failed to validate JWT."}`))
	}

	// Set up token extractors based on what env vars are available
	extractors := []jwtmiddleware.TokenExtractor{
		jwtmiddleware.AuthHeaderTokenExtractor,
	}

	if config.AuthCookieName != "" {
		extractors = append(extractors, jwtmiddleware.CookieTokenExtractor(config.AuthCookieName))
	}

	tokenExtractor := jwtmiddleware.MultiTokenExtractor(extractors...)

	middleware := jwtmiddleware.New(
		jwtValidator.ValidateToken,
		jwtmiddleware.WithErrorHandler(errorHandler),
		jwtmiddleware.WithTokenExtractor(tokenExtractor),
	)

	return middleware.CheckJWT(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// extract account name and setup otel attributes after the JWT was validated, but before the actual handler runs
		claims := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)

		token, err := tokenExtractor(r)
		// we should never hit this as the middleware wouldn't call the handler
		if err != nil {
			// This is not ErrJWTMissing because an error here means that the
			// tokenExtractor had an error and _not_ that the token was missing.
			errorHandler(w, r, fmt.Errorf("error extracting token: %w", err))
			return
		}

		customClaims := claims.CustomClaims.(*CustomClaims)
		if customClaims == nil {
			errorHandler(w, r, fmt.Errorf("couldn't get claims from: %v", claims))
			return
		}

		ctx := r.Context()

		// note that the values are looked up in last-in-first-out order, so
		// there is an absolutely minor perf optimisation to have the context
		// values set in ascending order of access frequency.
		ctx = context.WithValue(ctx, UserTokenContextKey{}, token)
		ctx = context.WithValue(ctx, CustomClaimsContextKey{}, customClaims)
		ctx = context.WithValue(ctx, CurrentSubjectContextKey{}, claims.RegisteredClaims.Subject)
		ctx = context.WithValue(ctx, AccountNameContextKey{}, customClaims.AccountName)

		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("ovm.auth.accountName", customClaims.AccountName),
			attribute.Int64("ovm.auth.expiry", claims.RegisteredClaims.Expiry),
			attribute.String("ovm.auth.scopes", customClaims.Scope),
			attribute.String("ovm.auth.subject", claims.RegisteredClaims.Subject),
		)

		r = r.Clone(ctx)

		next.ServeHTTP(w, r)
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
