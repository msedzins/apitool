// Package auth obtains OAuth credentials without persisting secret material.
package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"apitool/internal/model"
)

const refreshSkew = 30 * time.Second

// Token is an in-memory OAuth access token.
type Token struct {
	AccessToken string
	TokenType   string
	Expiry      time.Time
}

// TokenProvider obtains a token for a resolved OAuth configuration.
type TokenProvider interface {
	Token(ctx context.Context, config model.Auth) (Token, error)
}

// ClientCredentials provides OAuth2 client_credentials tokens with in-memory
// reuse. Its cache key is a hash of the complete effective configuration.
type ClientCredentials struct {
	client *http.Client
	mu     sync.Mutex
	cache  map[string]Token
}

func NewClientCredentials(client *http.Client) *ClientCredentials {
	if client == nil {
		client = http.DefaultClient
	}
	return &ClientCredentials{client: client, cache: make(map[string]Token)}
}

func (p *ClientCredentials) Token(ctx context.Context, config model.Auth) (Token, error) {
	key := cacheKey(config)
	p.mu.Lock()
	if cached, ok := p.cache[key]; ok && time.Until(cached.Expiry) > refreshSkew {
		p.mu.Unlock()
		return cached, nil
	}
	p.mu.Unlock()

	token, err := p.acquire(ctx, config)
	if err != nil {
		return Token{}, err
	}
	p.mu.Lock()
	p.cache[key] = token
	p.mu.Unlock()
	return token, nil
}

func (p *ClientCredentials) acquire(ctx context.Context, config model.Auth) (Token, error) {
	endpoint, err := url.Parse(config.TokenURL)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return Token{}, &OAuthError{Code: "token_endpoint_invalid"}
	}
	if config.ClientID == "" || config.ClientSecret == "" {
		return Token{}, &OAuthError{Code: "client_credentials_missing"}
	}
	form := url.Values{"grant_type": {"client_credentials"}}
	if len(config.Scopes) > 0 {
		form.Set("scope", strings.Join(config.Scopes, " "))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, &OAuthError{Code: "token_request_invalid"}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(config.ClientID, config.ClientSecret)
	response, err := p.client.Do(req)
	if err != nil {
		return Token{}, &OAuthError{Code: oauthTransportCode(err)}
	}
	defer response.Body.Close()

	var payload struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return Token{}, &OAuthError{Code: "token_response_invalid"}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		if payload.Error != "" {
			return Token{}, &OAuthError{Code: payload.Error}
		}
		return Token{}, &OAuthError{Code: fmt.Sprintf("token_http_%d", response.StatusCode)}
	}
	if payload.AccessToken == "" {
		return Token{}, &OAuthError{Code: "token_missing"}
	}
	if payload.TokenType == "" {
		payload.TokenType = "Bearer"
	}
	return Token{AccessToken: payload.AccessToken, TokenType: payload.TokenType, Expiry: time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)}, nil
}

func cacheKey(config model.Auth) string {
	values := []string{config.Type, config.Grant, config.TokenURL, config.ClientID, config.ClientSecret, strings.Join(config.Scopes, "\x00")}
	sum := sha256.Sum256([]byte(strings.Join(values, "\x01")))
	return hex.EncodeToString(sum[:])
}

func oauthTransportCode(err error) string {
	if errors.Is(err, context.Canceled) {
		return "token_request_canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "token_request_timeout"
	}
	return "token_request_failed"
}

// OAuthError intentionally exposes just an OAuth-safe code, never an endpoint,
// client credential, response body, or access token.
type OAuthError struct{ Code string }

func (e *OAuthError) Error() string { return "OAuth token request failed: " + e.Code }

// Mask returns a stable placeholder that does not disclose any token bytes.
func Mask(value string) string {
	if value == "" {
		return ""
	}
	return "[REDACTED]"
}
