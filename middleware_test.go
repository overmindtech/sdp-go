package sdp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestHasScopes(t *testing.T) {
	t.Run("with auth bypassed", func(t *testing.T) {
		t.Parallel()

		ctx := AddBypassAuthConfig(context.Background())

		pass := HasAllScopes(ctx, "test")

		if !pass {
			t.Error("expected to allow since auth is bypassed")
		}
	})

	t.Run("with good scopes", func(t *testing.T) {
		t.Parallel()

		account := "foo"
		scope := "test foo bar"
		ctx := OverrideCustomClaims(context.Background(), &scope, &account)

		pass := HasAllScopes(ctx, "test")

		if !pass {
			t.Error("expected to allow since `test` scope is present")
		}
	})

	t.Run("with multiple good scopes", func(t *testing.T) {
		t.Parallel()

		account := "foo"
		scope := "test foo bar"
		ctx := OverrideCustomClaims(context.Background(), &scope, &account)

		pass := HasAllScopes(ctx, "test", "foo")

		if !pass {
			t.Error("expected to allow since `test` scope is present")
		}
	})

	t.Run("with bad scopes", func(t *testing.T) {
		t.Parallel()

		account := "foo"
		scope := "test foo bar"
		ctx := OverrideCustomClaims(context.Background(), &scope, &account)

		pass := HasAllScopes(ctx, "baz")

		if pass {
			t.Error("expected to deny since `baz` scope is not present")
		}
	})

	t.Run("with one scope missing", func(t *testing.T) {
		t.Parallel()

		account := "foo"
		scope := "test foo bar"
		ctx := OverrideCustomClaims(context.Background(), &scope, &account)

		pass := HasAllScopes(ctx, "test", "baz")

		if pass {
			t.Error("expected to deny since `baz` scope is not present")
		}
	})

	t.Run("with any scopes", func(t *testing.T) {
		t.Parallel()

		account := "foo"
		scope := "test foo bar"
		ctx := OverrideCustomClaims(context.Background(), &scope, &account)

		pass := HasAnyScopes(ctx, "fail", "foo")

		if !pass {
			t.Error("expected to allow since `foo` scope is present")
		}
	})

	t.Run("without any scopes", func(t *testing.T) {
		t.Parallel()

		account := "foo"
		scope := "test foo bar"
		ctx := OverrideCustomClaims(context.Background(), &scope, &account)

		pass := HasAnyScopes(ctx, "fail", "fail harder")

		if pass {
			t.Error("expected to deny since no matching scope is present")
		}
	})
}

func TestNewAuthMiddleware(t *testing.T) {
	t.Parallel()

	account := "foo"

	t.Run("with bypass auth", func(t *testing.T) {
		config := AuthConfig{
			BypassAuth:      true,
			AccountOverride: &account,
		}

		handler := NewAuthMiddleware(config, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Context().Value(AuthBypassedContextKey{}) != true {
				t.Error("expected auth bypassed to be set")
			}

			// Read the custom claims from the context
			claims := r.Context().Value(CustomClaimsContextKey{}).(*CustomClaims)
			if claims.AccountName != account {
				t.Errorf("expected account to be %s, but was %s", account, claims.AccountName)
			}
		}))

		// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
		// pass 'nil' as the third parameter.
		req, err := http.NewRequest("GET", "/", nil)

		if err != nil {
			t.Fatal(err)
		}

		// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
	})

	t.Run("with bypass auth for paths", func(t *testing.T) {
		config := AuthConfig{
			BypassAuthForPaths: regexp.MustCompile("/health"),
			AccountOverride:    &account,
		}

		handler := NewAuthMiddleware(config, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Context().Value(AuthBypassedContextKey{}) != true {
				t.Error("expected auth bypassed to be set")
			}

			// Read the custom claims from the context
			claims := r.Context().Value(CustomClaimsContextKey{}).(*CustomClaims)
			if claims.AccountName != account {
				t.Errorf("expected account to be %s, but was %s", account, claims.AccountName)
			}
		}))

		// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
		// pass 'nil' as the third parameter.
		req, err := http.NewRequest("GET", "/health", nil)

		if err != nil {
			t.Fatal(err)
		}

		// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
	})
}
