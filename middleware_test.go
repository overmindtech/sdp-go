package sdp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
)

func TestHasScopes(t *testing.T) {
	t.Run("with auth bypassed", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), AuthBypassed{}, true)

		pass := HasScopes(ctx, "test")

		if !pass {
			t.Error("expected to allow since auth is bypassed")
		}
	})

	t.Run("with good scopes", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), jwtmiddleware.ContextKey{}, &validator.ValidatedClaims{
			CustomClaims: &CustomClaims{
				Scope: "test foo bar",
			},
		})

		pass := HasScopes(ctx, "test")

		if !pass {
			t.Error("expected to allow since scopes are present")
		}
	})

	t.Run("with bad scopes", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), jwtmiddleware.ContextKey{}, &validator.ValidatedClaims{
			CustomClaims: &CustomClaims{
				Scope: "test foo bar",
			},
		})

		pass := HasScopes(ctx, "baz")

		if pass {
			t.Error("expected to deny since scopes are not present")
		}
	})
}

func TestBypassAuth(t *testing.T) {
	handler := BypassAuth("test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(AuthBypassed{}) != true {
			t.Error("expected auth bypassed to be set")
		}

		if r.Context().Value(AccountNameContextKey{}) != "test" {
			t.Error("expected account name to be set")
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
