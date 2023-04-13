package sdp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHasScopes(t *testing.T) {
	t.Run("with auth bypassed", func(t *testing.T) {
		t.Parallel()

		ctx := AddBypassAuthConfig(context.Background(), "foo")

		pass := HasScopes(ctx, "test")

		if !pass {
			t.Error("expected to allow since auth is bypassed")
		}
	})

	t.Run("with good scopes", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(context.Background(), CustomClaimsContextKey{}, &CustomClaims{
			Scope: "test foo bar",
		})

		pass := HasScopes(ctx, "test")

		if !pass {
			t.Error("expected to allow since `test` scope is present")
		}
	})

	t.Run("with bad scopes", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(context.Background(), CustomClaimsContextKey{}, &CustomClaims{
			Scope: "test foo bar",
		})

		pass := HasScopes(ctx, "baz")

		if pass {
			t.Error("expected to deny since `baz` scope is not present")
		}
	})
}

func TestNewAuthMiddleware(t *testing.T) {
	t.Parallel()

	account := "foo"
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
}
