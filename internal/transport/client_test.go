package transport_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"apitool/internal/auth"
	"apitool/internal/model"
	"apitool/internal/transport"
)

func TestExecuteReturnsCompletedUnauthorizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Reason", "missing credentials")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
	}))
	t.Cleanup(server.Close)

	response, executionError := transport.Execute(context.Background(), request(server.URL), nil)
	if executionError != nil {
		t.Fatalf("Execute() execution error = %v", executionError)
	}
	if response.StatusCode != http.StatusUnauthorized || string(response.Body) != "unauthorized" || response.Headers.Get("X-Reason") != "missing credentials" {
		t.Errorf("Execute() response = %#v, want completed 401 response", response)
	}
	if response.ReceivedAt.IsZero() || response.Duration < 0 {
		t.Errorf("Execute() response metadata = %#v, want received time and non-negative duration", response)
	}
}

func TestExecuteReturnsCompletedServerErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("service unavailable"))
	}))
	t.Cleanup(server.Close)

	response, executionError := transport.Execute(context.Background(), request(server.URL), nil)
	if executionError != nil || response.StatusCode != http.StatusInternalServerError || string(response.Body) != "service unavailable" {
		t.Fatalf("Execute() = %#v, %v; want completed 500 response", response, executionError)
	}
}

func TestExecuteDoesNotFollowRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/final", http.StatusFound)
	}))
	t.Cleanup(server.Close)

	response, executionError := transport.Execute(context.Background(), request(server.URL), nil)
	if executionError != nil || response.StatusCode != http.StatusFound {
		t.Fatalf("Execute() = %#v, %v; want observable 302", response, executionError)
	}
}

func TestExecuteClassifiesCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, executionError := transport.Execute(ctx, request("http://127.0.0.1:1"), nil)
	if executionError == nil || executionError.Category != model.CategoryCanceled {
		t.Fatalf("Execute() execution error = %#v, want canceled", executionError)
	}
}

func TestExecuteClassifiesTimeoutWithoutLeakingRequestURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)

	effective := request(server.URL + "?token=very-secret")
	effective.Timeout = 10 * time.Millisecond
	_, executionError := transport.Execute(context.Background(), effective, nil)
	if executionError == nil || executionError.Category != model.CategoryTimeout {
		t.Fatalf("Execute() execution error = %#v, want timeout", executionError)
	}
	if executionError.SafeMessage == "" || strings.Contains(executionError.SafeMessage, "very-secret") {
		t.Errorf("timeout safe message = %q, want redacted diagnostic", executionError.SafeMessage)
	}
}

func TestExecuteAppliesEffectiveTimeoutToTokenAcquisition(t *testing.T) {
	resourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(resourceServer.Close)
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(40 * time.Millisecond):
			_, _ = w.Write([]byte(`{"access_token":"late-token","expires_in":3600}`))
		}
	}))
	t.Cleanup(tokenServer.Close)

	effective := request(resourceServer.URL)
	effective.Timeout = 10 * time.Millisecond
	effective.Auth = &model.Auth{
		Type: "oauth2", Grant: "client_credentials", TokenURL: tokenServer.URL,
		ClientID: "client-id", ClientSecret: "very-secret",
	}
	_, executionError := transport.Execute(context.Background(), effective, auth.NewClientCredentials(tokenServer.Client()))
	if executionError == nil || executionError.Stage != model.StageOAuth || executionError.Category != model.CategoryTimeout {
		t.Fatalf("Execute() token acquisition error = %#v, want OAuth timeout", executionError)
	}
	if strings.Contains(executionError.SafeMessage, "very-secret") {
		t.Errorf("OAuth timeout safe message leaked secret: %q", executionError.SafeMessage)
	}
}

func TestExecuteDoesNotExposeUntrustedOAuthErrorInSafeMessage(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"token-endpoint-secret"}`))
	}))
	t.Cleanup(tokenServer.Close)

	effective := request("http://api.example.test")
	effective.Auth = &model.Auth{
		Type: "oauth2", Grant: "client_credentials", TokenURL: tokenServer.URL,
		ClientID: "client-id", ClientSecret: "very-secret",
	}
	_, executionError := transport.Execute(context.Background(), effective, auth.NewClientCredentials(tokenServer.Client()))
	if executionError == nil || executionError.Category != model.CategoryOAuth {
		t.Fatalf("Execute() OAuth error = %#v, want OAuth diagnostic", executionError)
	}
	if strings.Contains(executionError.SafeMessage, "token-endpoint-secret") || strings.Contains(executionError.SafeMessage, "very-secret") {
		t.Errorf("OAuth safe message leaked token endpoint or client secret: %q", executionError.SafeMessage)
	}
}

func TestExecuteClassifiesCanceledGenericTokenProvider(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	effective := request("http://api.example.test")
	effective.Auth = &model.Auth{Type: "oauth2", Grant: "client_credentials"}
	_, executionError := transport.Execute(ctx, effective, genericErrorProvider{})
	if executionError == nil || executionError.Stage != model.StageOAuth || executionError.Category != model.CategoryCanceled {
		t.Fatalf("Execute() token acquisition error = %#v, want OAuth cancellation", executionError)
	}
	if strings.Contains(executionError.SafeMessage, "very-secret") {
		t.Errorf("OAuth cancellation safe message leaked provider value: %q", executionError.SafeMessage)
	}
}

func TestExecuteRejectsCaseInsensitiveDuplicateHeaders(t *testing.T) {
	effective := request("http://api.example.test")
	effective.Headers = map[string]string{"X-Trace": "one", "x-trace": "two"}
	_, executionError := transport.Execute(context.Background(), effective, nil)
	if executionError == nil || executionError.Stage != model.StageRequestBuild || executionError.Category != model.CategoryRequestBuild {
		t.Fatalf("Execute() duplicate-header error = %#v, want request-build diagnostic", executionError)
	}
	if strings.Contains(executionError.SafeMessage, "one") || strings.Contains(executionError.SafeMessage, "two") {
		t.Errorf("duplicate-header safe message leaked values: %q", executionError.SafeMessage)
	}
}

func TestExecuteAuthNoneOmitsAuthorizationAndPreservesRawBody(t *testing.T) {
	wantBody := []byte("<entry> unchanged \n</entry>")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization = %q, want absent for auth none", got)
		}
		gotBody, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(gotBody) != string(wantBody) {
			t.Errorf("body = %q, want unchanged %q", gotBody, wantBody)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	effective := request(server.URL)
	effective.Method = http.MethodPost
	effective.Body = wantBody
	effective.Auth = &model.Auth{None: true}
	response, executionError := transport.Execute(context.Background(), effective, failingProvider{})
	if executionError != nil || response.StatusCode != http.StatusNoContent {
		t.Fatalf("Execute() = %#v, %v", response, executionError)
	}
}

func TestExecuteUsesOAuthTokenAndQueryParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Errorf("Authorization = %q, want bearer token", got)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("query page = %q, want 2", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	effective := request(server.URL)
	effective.Params = map[string]string{"page": "2"}
	effective.Auth = &model.Auth{Type: "oauth2", Grant: "client_credentials"}
	_, executionError := transport.Execute(context.Background(), effective, staticProvider{})
	if executionError != nil {
		t.Fatalf("Execute() execution error = %v", executionError)
	}
}

func TestExecuteRejectsSelfSignedTLSUnlessExplicitlyEnabled(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	effective := request(server.URL)
	_, executionError := transport.Execute(context.Background(), effective, nil)
	if executionError == nil || executionError.Category != model.CategoryTLS {
		t.Fatalf("verified TLS execution error = %#v, want TLS failure", executionError)
	}
	effective.InsecureSkipVerify = true
	response, executionError := transport.Execute(context.Background(), effective, nil)
	if executionError != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("explicit insecure TLS Execute() = %#v, %v", response, executionError)
	}
}

func request(rawURL string) model.EffectiveRequest {
	return model.EffectiveRequest{Method: http.MethodGet, URL: rawURL, Headers: map[string]string{"X-Test": "true"}, Timeout: time.Second}
}

type staticProvider struct{}

func (staticProvider) Token(context.Context, model.Auth) (auth.Token, error) {
	return auth.Token{AccessToken: "access-token", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}, nil
}

type failingProvider struct{}

func (failingProvider) Token(context.Context, model.Auth) (auth.Token, error) {
	return auth.Token{}, errors.New("provider must not be called")
}

type genericErrorProvider struct{}

func (genericErrorProvider) Token(context.Context, model.Auth) (auth.Token, error) {
	return auth.Token{}, errors.New("very-secret")
}

var _ = tls.VersionTLS13
