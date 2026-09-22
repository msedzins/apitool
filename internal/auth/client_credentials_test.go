package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"apitool/internal/auth"
	"apitool/internal/model"
)

func TestClientCredentialsCachesUnexpiredTokenAndSendsScopes(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost {
			t.Errorf("token request method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.Form.Get("grant_type"); got != "client_credentials" {
			t.Errorf("grant_type = %q, want client_credentials", got)
		}
		if got := r.Form.Get("scope"); got != "users.read users.write" {
			t.Errorf("scope = %q, want requested scopes", got)
		}
		if user, password, ok := r.BasicAuth(); !ok || user != "client-id" || password != "very-secret" {
			t.Error("token request did not use the configured client credentials")
		}
		_, _ = w.Write([]byte(`{"access_token":"token-123","token_type":"Bearer","expires_in":3600}`))
	}))
	t.Cleanup(server.Close)

	provider := auth.NewClientCredentials(server.Client())
	config := oauthConfig(server.URL, []string{"users.read", "users.write"})
	for range 2 {
		token, err := provider.Token(context.Background(), config)
		if err != nil {
			t.Fatalf("Token() error = %v", err)
		}
		if token.AccessToken != "token-123" {
			t.Errorf("Token().AccessToken = %q, want token-123", token.AccessToken)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("token endpoint calls = %d, want 1", got)
	}
}

func TestClientCredentialsRefreshesExpiredToken(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sequence := calls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "token-" + string(rune('0'+sequence)), "token_type": "Bearer", "expires_in": 0,
		})
	}))
	t.Cleanup(server.Close)

	provider := auth.NewClientCredentials(server.Client())
	config := oauthConfig(server.URL, nil)
	first, err := provider.Token(context.Background(), config)
	if err != nil {
		t.Fatalf("first Token() error = %v", err)
	}
	second, err := provider.Token(context.Background(), config)
	if err != nil {
		t.Fatalf("second Token() error = %v", err)
	}
	if first.AccessToken == second.AccessToken || calls.Load() != 2 {
		t.Errorf("expired token was reused: first=%q second=%q calls=%d", first.AccessToken, second.AccessToken, calls.Load())
	}
}

func TestOAuthInvalidClientDiagnosticIsRedacted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client","error_description":"very-secret"}`))
	}))
	t.Cleanup(server.Close)

	_, err := auth.NewClientCredentials(server.Client()).Token(context.Background(), oauthConfig(server.URL, nil))
	if err == nil {
		t.Fatal("Token() error = nil, want invalid_client error")
	}
	if got := err.Error(); containsAny(got, "very-secret", "client-id", "token-123") {
		t.Errorf("Token() error leaked credential or token: %q", got)
	}
	if got := err.Error(); !strings.Contains(got, "invalid_client") {
		t.Errorf("Token() error = %q, want invalid_client code", got)
	}
}

func TestOAuthUntrustedErrorValueIsRedacted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"very-secret-token-123"}`))
	}))
	t.Cleanup(server.Close)

	_, err := auth.NewClientCredentials(server.Client()).Token(context.Background(), oauthConfig(server.URL, nil))
	if err == nil {
		t.Fatal("Token() error = nil, want safe OAuth error")
	}
	if got := err.Error(); strings.Contains(got, "very-secret-token-123") {
		t.Errorf("Token() error leaked an untrusted token endpoint value: %q", got)
	}
	if got := err.Error(); !strings.Contains(got, "oauth_error") {
		t.Errorf("Token() error = %q, want generic oauth_error code", got)
	}
}

func TestClientCredentialsFormEncodesBasicCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok {
			t.Error("token request has no Basic authorization")
		}
		if user != "client%3A+id%2B%25%C3%A9" || password != "secret%3A+value%2B%25%C5%BC" {
			t.Errorf("Basic credentials = %q, %q; want form-encoded values", user, password)
		}
		_, _ = w.Write([]byte(`{"access_token":"token-123","expires_in":3600}`))
	}))
	t.Cleanup(server.Close)

	config := oauthConfig(server.URL, nil)
	config.ClientID = "client: id+%é"
	config.ClientSecret = "secret: value+%ż"
	if _, err := auth.NewClientCredentials(server.Client()).Token(context.Background(), config); err != nil {
		t.Fatalf("Token() error = %v", err)
	}
}

func TestMaskDoesNotRevealValue(t *testing.T) {
	if got := auth.Mask("token-123"); got == "token-123" || got == "" {
		t.Errorf("Mask() = %q, want a non-empty masked value", got)
	}
}

func oauthConfig(tokenURL string, scopes []string) model.Auth {
	return model.Auth{Type: "oauth2", Grant: "client_credentials", TokenURL: tokenURL, ClientID: "client-id", ClientSecret: "very-secret", Scopes: scopes}
}

func containsAny(value string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(value, part) {
			return true
		}
	}
	return false
}
