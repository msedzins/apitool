// Package transport builds and executes one resolved HTTP request.
package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"time"

	"apitool/internal/auth"
	"apitool/internal/model"
)

// Execute sends a fully resolved request. Completed HTTP responses, regardless
// of status, are returned as a Response; only execution failures return an
// ExecutionError.
func Execute(ctx context.Context, effective model.EffectiveRequest, tokenProvider auth.TokenProvider) (model.Response, *model.ExecutionError) {
	started := time.Now()
	request, buildError := buildRequest(ctx, effective)
	if buildError != nil {
		return model.Response{}, buildError
	}
	if effective.Auth != nil && !effective.Auth.None && effective.Auth.Type == "oauth2" {
		if tokenProvider == nil {
			return model.Response{}, safeError(model.StageOAuth, model.CategoryOAuth, "OAuth token provider unavailable")
		}
		token, err := tokenProvider.Token(ctx, *effective.Auth)
		if err != nil {
			message := "OAuth token acquisition failed"
			if oauthError, ok := err.(*auth.OAuthError); ok {
				message = oauthError.Error()
			}
			return model.Response{}, safeError(model.StageOAuth, model.CategoryOAuth, message)
		}
		request.Header.Set("Authorization", "Bearer "+token.AccessToken)
	}

	client := httpClient(effective)
	response, err := client.Do(request)
	if err != nil {
		return model.Response{}, transportError(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return model.Response{}, transportError(err)
	}
	received := time.Now()
	return model.Response{
		StatusCode: response.StatusCode,
		Headers:    response.Header.Clone(),
		Body:       body,
		Duration:   received.Sub(started),
		ReceivedAt: received,
	}, nil
}

func buildRequest(ctx context.Context, effective model.EffectiveRequest) (*http.Request, *model.ExecutionError) {
	parsed, err := url.Parse(effective.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, safeError(model.StageRequestBuild, model.CategoryRequestBuild, "Request URL is invalid")
	}
	query := parsed.Query()
	for key, value := range effective.Params {
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, effective.Method, parsed.String(), bytes.NewReader(effective.Body))
	if err != nil {
		return nil, safeError(model.StageRequestBuild, model.CategoryRequestBuild, "HTTP request is invalid")
	}
	for key, value := range effective.Headers {
		request.Header.Set(key, value)
	}
	return request, nil
}

func httpClient(effective model.EffectiveRequest) *http.Client {
	base := http.DefaultTransport.(*http.Transport).Clone()
	base.TLSClientConfig = &tls.Config{InsecureSkipVerify: effective.InsecureSkipVerify} // #nosec G402 -- enabled only by explicit effective configuration.
	return &http.Client{
		Transport: base,
		Timeout:   effective.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
