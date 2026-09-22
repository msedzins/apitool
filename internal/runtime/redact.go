package runtime

import (
	"encoding/json"
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
	entry.Path = redactPath(entry.Path)
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
	case "authorization", "proxyauthorization", "cookie", "setcookie", "clientsecret", "accesstoken", "refreshtoken", "idtoken", "token", "secret", "password", "body", "rawbody", "requestbody", "content":
		return true
	default:
		return strings.Contains(normalized, "token") || strings.Contains(normalized, "secret")
	}
}

func structuredRedaction(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode log metadata: %w", err)
	}
	var generic any
	if err := json.Unmarshal(encoded, &generic); err != nil {
		return nil, fmt.Errorf("decode log metadata: %w", err)
	}
	return RedactData(generic), nil
}

func redactPath(path string) string {
	if path == "" {
		return ""
	}
	parsed, err := url.Parse(path)
	if err != nil {
		return redacted
	}
	query := parsed.Query()
	for key := range query {
		if sensitiveKey(key) {
			query.Set(key, redacted)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
