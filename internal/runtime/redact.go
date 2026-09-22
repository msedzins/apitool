package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"apitool/internal/model"
)

const redacted = "[REDACTED]"

// LogEntry is safe execution metadata. Data is accepted for diagnostics but
// recursively redacted before it reaches disk.
type LogEntry struct {
	Timestamp         time.Time               `json:"timestamp"`
	Key               Key                     `json:"key"`
	Method            string                  `json:"method"`
	Host              string                  `json:"host,omitempty"`
	Path              string                  `json:"path,omitempty"`
	StatusCode        int                     `json:"status_code,omitempty"`
	Duration          time.Duration           `json:"duration_ns,omitempty"`
	ErrorCategory     model.ExecutionCategory `json:"error_category,omitempty"`
	OAuthEndpointHost string                  `json:"oauth_endpoint_host,omitempty"`
	OAuthGrant        string                  `json:"oauth_grant,omitempty"`
	OAuthScopes       []string                `json:"oauth_scopes,omitempty"`
	Data              any                     `json:"data,omitempty"`
}

// AppendLog appends a redacted execution log record.
func (s *Store) AppendLog(entry LogEntry) error {
	if _, err := entry.Key.segments(); err != nil {
		return err
	}
	entry.Method = strings.ToUpper(strings.TrimSpace(entry.Method))
	if entry.Method == "" {
		return fmt.Errorf("log method is required")
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	// These exported fields are caller-controlled strings. A host-looking token
	// is indistinguishable from a DNS name, so logging either would violate the
	// no-secret persistence guarantee. Keep host metadata out of this contract.
	entry.Host = ""
	entry.OAuthEndpointHost = ""
	path, err := redactPath(entry.Path)
	if err != nil {
		return err
	}
	entry.Path = path
	data, err := structuredRedaction(entry.Data)
	if err != nil {
		return err
	}
	entry.Data = data
	encoded, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encode log entry: %w", err)
	}
	encoded = append(encoded, '\n')
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.appendFile(filepath.Join("logs", "executions.jsonl"), encoded)
}

// RedactHeaders returns a copy of headers with credentials and cookies removed.
func RedactHeaders(headers http.Header) http.Header {
	if headers == nil {
		return nil
	}
	result := make(http.Header, len(headers))
	for name, values := range headers {
		if sensitiveKey(name) {
			result[name] = []string{redacted}
			continue
		}
		result[name] = append([]string(nil), values...)
	}
	return result
}

// RedactData copies JSON-like structured data and redacts sensitive values at
// every nesting level. It is intended for safe log metadata.
func RedactData(value any) any {
	switch typed := value.(type) {
	case http.Header:
		return RedactHeaders(typed)
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			if sensitiveKey(key) {
				result[key] = redacted
				continue
			}
			result[key] = RedactData(item)
		}
		return result
	case map[string]string:
		result := make(map[string]string, len(typed))
		for key, item := range typed {
			if sensitiveKey(key) {
				result[key] = redacted
				continue
			}
			result[key] = item
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			result[index] = RedactData(item)
		}
		return result
	case []string:
		return append([]string(nil), typed...)
	default:
		return value
	}
}

func sensitiveKey(key string) bool {
	normalized := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(key))
	switch normalized {
	case "authorization", "proxyauthorization", "cookie", "setcookie", "clientsecret", "accesstoken", "refreshtoken", "idtoken", "token", "secret", "password", "body", "rawbody", "requestbody", "content", "apikey", "xapikey":
		return true
	default:
		return strings.Contains(normalized, "token") || strings.Contains(normalized, "secret") || strings.Contains(normalized, "apikey")
	}
}

func structuredRedaction(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	if _, ok := value.(map[string]any); !ok {
		return nil, errors.New("log metadata must be an object")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode log metadata: %w", err)
	}
	var generic any
	if err := json.Unmarshal(encoded, &generic); err != nil {
		return nil, fmt.Errorf("decode log metadata: %w", err)
	}
	object, ok := generic.(map[string]any)
	if !ok {
		return nil, errors.New("log metadata must be an object")
	}
	return redactLogObject(object), nil
}

func redactLogObject(object map[string]any) map[string]any {
	result := make(map[string]any, len(object))
	for key, value := range object {
		if sensitiveKey(key) {
			result[key] = redacted
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			result[key] = redactLogObject(typed)
		case []any:
			if safeLogValueKey(key) {
				result[key] = typed
			} else {
				result[key] = redacted
			}
		default:
			if safeLogValueKey(key) {
				result[key] = typed
			} else {
				result[key] = redacted
			}
		}
	}
	return result
}

func safeLogValueKey(key string) bool {
	switch strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(key)) {
	case "trace", "traceid", "stage", "status", "statuscode", "errorcode", "grant", "scope", "scopes", "expiry", "expiresat", "method", "host", "path", "duration":
		return true
	default:
		return false
	}
}

func redactPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	parsed, err := url.Parse(path)
	if err != nil {
		return "", errors.New("log path is invalid")
	}
	if parsed.IsAbs() || parsed.Host != "" || parsed.User != nil {
		return "", errors.New("log path must not include an authority")
	}
	parsed.Fragment = ""
	query := parsed.Query()
	for key := range query {
		if sensitiveKey(key) {
			query.Set(key, redacted)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func responseBodySensitive(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	var decoded any
	if json.Unmarshal(body, &decoded) == nil {
		return responseValueSensitive(decoded)
	}
	lower := strings.ToLower(string(body))
	for _, marker := range []string{"access_token", "refresh_token", "client_secret", "api_key", "authorization", "set-cookie"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func responseValueSensitive(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if credentialKey(key) || responseValueSensitive(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if responseValueSensitive(child) {
				return true
			}
		}
	}
	return false
}

func credentialKey(key string) bool {
	normalized := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(key))
	return strings.Contains(normalized, "token") || strings.Contains(normalized, "secret") || strings.Contains(normalized, "apikey") || normalized == "authorization" || normalized == "cookie" || normalized == "password"
}
