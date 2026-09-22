package runtime_test

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"apitool/internal/runtime"
)

func TestRedactionRemovesSensitiveHeadersAndStructuredValues(t *testing.T) {
	headers := runtime.RedactHeaders(http.Header{
		"Authorization":       {"Bearer token-123"},
		"Proxy-Authorization": {"Basic proxy-secret"},
		"Cookie":              {"session=very-secret"},
		"Set-Cookie":          {"refresh=very-secret"},
		"X-Trace":             {"safe"},
	})
	for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie", "Set-Cookie"} {
		if got := headers.Get(name); got != "[REDACTED]" {
			t.Errorf("RedactHeaders()[%q] = %q, want [REDACTED]", name, got)
		}
	}
	if got := headers.Get("X-Trace"); got != "safe" {
		t.Errorf("RedactHeaders()[X-Trace] = %q, want safe", got)
	}

	got := runtime.RedactData(map[string]any{
		"client_secret": "client-secret-123",
		"nested": map[string]any{
			"access_token": "access-token-456",
			"trace_id":     "safe-trace",
		},
	})
	redacted, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("RedactData() = %T, want map[string]any", got)
	}
	if redacted["client_secret"] != "[REDACTED]" {
		t.Errorf("RedactData() client_secret = %#v, want [REDACTED]", redacted["client_secret"])
	}
	nested, ok := redacted["nested"].(map[string]any)
	if !ok || nested["access_token"] != "[REDACTED]" || nested["trace_id"] != "safe-trace" {
		t.Errorf("RedactData() nested = %#v, want token redacted and trace preserved", redacted["nested"])
	}
}

func TestAppendLogRedactsNestedMetadataHeadersAndSensitiveQueryValues(t *testing.T) {
	workspace := t.TempDir()
	store, err := runtime.Open(workspace)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	err = store.AppendLog(runtime.LogEntry{
		Key:    runtime.Key{CollectionPath: "payments", Environment: "prod", RequestID: "users/create"},
		Method: "POST",
		Host:   "api.example.test",
		Path:   "/users?access_token=access-token-123&trace=safe-trace",
		Data: map[string]any{
			"headers": http.Header{"Authorization": {"Bearer header-token-456"}},
			"oauth":   map[string]any{"client_secret": "client-secret-789", "trace": "safe-trace"},
			"body":    "raw-request-body-000",
		},
	})
	if err != nil {
		t.Fatalf("AppendLog() error = %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(workspace, ".apitool", "logs", "executions.jsonl"))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	for _, secret := range [][]byte{[]byte("access-token-123"), []byte("header-token-456"), []byte("client-secret-789"), []byte("raw-request-body-000")} {
		if bytes.Contains(raw, secret) {
			t.Errorf("log contains secret %q: %s", secret, raw)
		}
	}
	if !bytes.Contains(raw, []byte("safe-trace")) {
		t.Errorf("log omitted safe metadata: %s", raw)
	}
}

func TestAppendLogRejectsUnstructuredDataAndUnsafeURLParts(t *testing.T) {
	workspace := t.TempDir()
	store, err := runtime.Open(workspace)
	if err != nil {
		t.Fatal(err)
	}
	key := runtime.Key{CollectionPath: "payments", Environment: "prod", RequestID: "users/list"}
	for name, entry := range map[string]runtime.LogEntry{
		"scalar data": {Key: key, Method: "GET", Data: "data-token-000"},
		"userinfo":    {Key: key, Method: "GET", Path: "//user:fragment-token-789@example.test/items"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := store.AppendLog(entry); err == nil {
				t.Fatal("AppendLog() error = nil, want unsafe input rejected")
			}
		})
	}
	if err := store.AppendLog(runtime.LogEntry{Key: key, Method: "GET", Path: "/items#access_token=fragment-token-789", Data: map[string]any{"detail": "unknown-value-token-111"}}); err != nil {
		t.Fatalf("AppendLog() fragment error = %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(workspace, ".apitool", "logs", "executions.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range [][]byte{[]byte("data-token-000"), []byte("fragment-token-789"), []byte("unknown-value-token-111")} {
		if bytes.Contains(raw, secret) {
			t.Errorf("log contains unsafe value %q: %s", secret, raw)
		}
	}
}

func TestAppendLogRejectsUnsafeDirectHostFields(t *testing.T) {
	workspace := t.TempDir()
	store, err := runtime.Open(workspace)
	if err != nil {
		t.Fatal(err)
	}
	key := runtime.Key{CollectionPath: "payments", Environment: "prod", RequestID: "users/list"}
	for name, entry := range map[string]runtime.LogEntry{
		"host token":         {Key: key, Method: "GET", Host: "access-token-123"},
		"host URL":           {Key: key, Method: "GET", Host: "https://api.example.test/items?access_token=query-token-456"},
		"oauth authority":    {Key: key, Method: "GET", OAuthEndpointHost: "user:client-secret-789@auth.example.test"},
		"oauth URL fragment": {Key: key, Method: "GET", OAuthEndpointHost: "https://auth.example.test#access_token=fragment-token-000"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := store.AppendLog(entry); err == nil {
				t.Fatal("AppendLog() error = nil, want unsafe host field rejected")
			}
		})
	}
	if err := store.AppendLog(runtime.LogEntry{Key: key, Method: "GET", Host: "API.Example.Test:443", OAuthEndpointHost: "auth.example.test"}); err != nil {
		t.Fatalf("AppendLog() valid hosts error = %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(workspace, ".apitool", "logs", "executions.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, unsafe := range [][]byte{[]byte("access-token-123"), []byte("query-token-456"), []byte("client-secret-789"), []byte("fragment-token-000"), []byte(":443")} {
		if bytes.Contains(raw, unsafe) {
			t.Errorf("log contains unsafe host input %q: %s", unsafe, raw)
		}
	}
	if !bytes.Contains(raw, []byte(`"host":"api.example.test"`)) || !bytes.Contains(raw, []byte(`"oauth_endpoint_host":"auth.example.test"`)) {
		t.Errorf("log = %s, want normalized hostname-only fields", raw)
	}
}
