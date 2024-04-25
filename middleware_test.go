package sdp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	log "github.com/sirupsen/logrus"
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

func BenchmarkAuthMiddleware(b *testing.B) {
	config := AuthConfig{
		Auth0Domain:   "om-dogfood.eu.auth0.com",
		Auth0Audience: "https://api.overmind.tech",
	}

	okHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	handler := NewAuthMiddleware(config, http.HandlerFunc(okHandler))

	// Reduce logging
	log.SetLevel(log.FatalLevel)

	for i := 0; i < b.N; i++ {
		// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
		// pass 'nil' as the third parameter.
		req, err := http.NewRequest("GET", "/", nil)

		if err != nil {
			b.Fatal(err)
		}

		// Set to a known bad JWT (this JWT is garbage don't freak out)
		// req.Header.Set("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InBpQWx1Q1FkQTB4MTNweG1JQzM4dyJ9.eyJodHRwczovL2FwaS5vdmVybWluZC50ZWNoL2FjY291bnQtbmFtZSI6IjYzZDE4NWM3MTQxMjM3OTc4Y2ZkYmFhMiIsImlzcyI6Imh0dHBzOi8vb20tZG9nZm9vZC5ldS5hdXRoMC5jb20vIiwic3ViIjoiYXV0aDB8NjNkMTg1YzcxNDEyMzc5NzhjZmRiYWEyIiwiYXVkIjpbImh0dHBzOi8vYXBpLmRmLm92ZXJtaW5kLWRlbW8uY29tIiwiaHR0cHM6Ly9vbS1kb2dmb29kLmV1LmF1dGgwLmNvbS91c2VyaW5mbyJdLCJpYXQiOjE3MTQwNDIwOTIsImV4cCI6MTcxNDEyODQ5Miwic2NvcGUiOiJvcGVuaWQgcHJvZmlsZSBlbWFpbCBhY2NvdW50OnJlYWQgYWNjb3VudDp3cml0ZSBhcGlfa2V5czpyZWFkIGFwaV9rZXlzOndyaXRlIGFwaTpyZWFkIGFwaTp3cml0ZSBjaGFuZ2VzOnJlYWQgY2hhbmdlczp3cml0ZSBjb25maWc6cmVhZCBjb25maWc6d3JpdGUgZXhwbG9yZTpyZWFkIGdhdGV3YXk6b2JqZWN0cyBnYXRld2F5OnN0cmVhbSByZXF1ZXN0OnJlY2VpdmUgcmVxdWVzdDpzZW5kIHJldmVyc2VsaW5rOnJlcXVlc3Qgc291cmNlOnJlYWQgc291cmNlOndyaXRlIHNvdXJjZXM6cmVhZCBzb3VyY2VzOndyaXRlIiwiYXpwIjoibnh0OUNzWFg3MnRncWlMYnE2Q0ZMOExLVFBXQ0ltQkwifQ.cEEh8jVnEItZel4SoyPybLUg7sArwduCrmSJHMz3YNRfzpRl9lxry39psuDUHFKdgOoNVxUv3Lgm-JWG-9uddCKYOW_zQxEvQvj6o8tcpQkmBZBlc8huG21dLPz7yrPhogVAcApLjdHf1fqii9EHxQegxch9FHlyfF7Xii5t9Hus62l4vdZ5dVWaIuiOLtcbG_hLxl9yqBf5tzN8eEC-Pa1SoAciRPesqH4AARfKyBFBhN774Fu3NzfNtW3wD_ASvnv7aFwzblS8ff5clqdTr2GuuJKdIPcmjQV2LaGSExHg2riCryf5guAhitAuwhugssW__STQmwp8dJmhifs7DA")
		req.Header.Set("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InBpQWx1Q1FkQTB4MTNweG1JQzM4dyJ9.eyJodHRwczovL2FwaS5vdmVybWluZC50ZWNoL2FjY291bnQtbmFtZSI6IlRFU1QiLCJpc3MiOiJodHRwczovL29tLWRvZ2Zvb2QuZXUuYXV0aDAuY29tLyIsInN1YiI6ImF1dGgwfFRFU1QiLCJhdWQiOlsiaHR0cHM6Ly9hcGkuZGYub3Zlcm1pbmQtZGVtby5jb20iLCJodHRwczovL29tLWRvZ2Zvb2QuZXUuYXV0aDAuY29tL3VzZXJpbmZvIl0sImlhdCI6MTcxNDA0MjA5MiwiZXhwIjoxNzE0MTI4NDkyLCJzY29wZSI6Im1hbnkgc2NvcGVzIiwiYXpwIjoiVEVTVCJ9.cEEh8jVnEItZel4SoyPybLUg7sArwduCrmSJHMz3YNRfzpRl9lxry39psuDUHFKdgOoNVxUv3Lgm-JWG-9uddCKYOW_zQxEvQvj6o8tcpQkmBZBlc8huG21dLPz7yrPhogVAcApLjdHf1fqii9EHxQegxch9FHlyfF7Xii5t9Hus62l4vdZ5dVWaIuiOLtcbG_hLxl9yqBf5tzN8eEC-Pa1SoAciRPesqH4AARfKyBFBhN774Fu3NzfNtW3wD_ASvnv7aFwzblS8ff5clqdTr2GuuJKdIPcmjQV2LaGSExHg2riCryf5guAhitAuwhugssW__STQmwp8dJmhifs7DA")

		// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			b.Errorf("expected status code %d, but got %d", http.StatusUnauthorized, rr.Code)
		}
	}
}
