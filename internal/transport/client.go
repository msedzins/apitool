// Package transport builds and executes one resolved HTTP request.
package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"apitool/internal/auth"
	"apitool/internal/model"
)

// Execute sends a fully resolved request. Completed HTTP responses, regardless
// of status, are returned as a Response; only execution failures return an
// ExecutionError.
func Execute(ctx context.Context, effective model.EffectiveRequest, tokenProvider auth.TokenProvider) (model.Response, *model.ExecutionError) {
	started := time.Now()
	executionContext := ctx
	cancel := func() {}
	if effective.Timeout > 0 {
		executionContext, cancel = context.WithTimeout(ctx, effective.Timeout)
	}
	defer cancel()

	request, buildError := buildRequest(executionContext, effective)
	if buildError != nil {
		return model.Response{}, buildError
	}
	if effective.Auth != nil && !effective.Auth.None && effective.Auth.Type == "oauth2" {
		if tokenProvider == nil {
			return model.Response{}, safeError(model.StageOAuth, model.CategoryOAuth, "OAuth token provider unavailable")
		}
		token, err := tokenProvider.Token(executionContext, *effective.Auth)
		if err != nil {
			return model.Response{}, oauthExecutionError(executionContext, err)
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
	if headerError := validateHeaders(effective.Headers); headerError != nil {
		return nil, headerError
	}
	request, err := http.NewRequestWithContext(ctx, effective.Method, parsed.String(), bytes.NewReader(effective.Body))
	if err != nil {
		return nil, safeError(model.StageRequestBuild, model.CategoryRequestBuild, "HTTP request is invalid")
	}
	for key, value := range effective.Headers {
		request.Header.Set(key, value)
	}
	return request, nil
}

func validateHeaders(headers map[string]string) *model.ExecutionError {
	seen := make(map[string]struct{}, len(headers))
	for name := range headers {
		canonical := strings.ToLower(name)
		if _, duplicate := seen[canonical]; duplicate {
			return safeError(model.StageRequestBuild, model.CategoryRequestBuild, "Request headers contain duplicate names")
		}
		seen[canonical] = struct{}{}
	}
	return nil
}

func oauthExecutionError(ctx context.Context, err error) *model.ExecutionError {
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return safeError(model.StageOAuth, model.CategoryCanceled, "OAuth token request canceled")
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return safeError(model.StageOAuth, model.CategoryTimeout, "OAuth token request timed out")
	}
	message := "OAuth token acquisition failed"
	if oauthError, ok := err.(*auth.OAuthError); ok {
		message = oauthError.Error()
	}
	return safeError(model.StageOAuth, model.CategoryOAuth, message)
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
