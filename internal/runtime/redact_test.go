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
