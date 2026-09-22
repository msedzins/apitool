package runtime_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"apitool/internal/model"
	"apitool/internal/runtime"
)

func TestHistorySearchesSafeMetadataAndExcludesOAuthFailureSecrets(t *testing.T) {
	workspace := t.TempDir()
	store, err := runtime.Open(workspace)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	key := runtime.Key{CollectionPath: "payments", Environment: "prod", RequestID: "users/create"}
	oauthFailure := &model.ExecutionError{
		Stage:       model.StageOAuth,
		Category:    model.CategoryOAuth,
		SafeMessage: "OAuth failed for access-token-123 with client-secret-456 and raw-body-789",
	}
	if err := os.MkdirAll(filepath.Join(workspace, ".apitool"), 0o700); err != nil {
		t.Fatalf("create runtime directory for malformed line: %v", err)
	}
	historyFile, err := os.OpenFile(filepath.Join(workspace, ".apitool", "history.jsonl"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatalf("open history to add malformed line: %v", err)
	}
	if _, err := historyFile.WriteString("not-json\n"); err != nil {
		_ = historyFile.Close()
		t.Fatalf("write malformed history line: %v", err)
	}
	if err := historyFile.Close(); err != nil {
		t.Fatalf("close malformed history line: %v", err)
	}
	if err := store.AppendHistory(key, "POST", model.Response{Duration: 25 * time.Millisecond}, oauthFailure); err != nil {
		t.Fatalf("AppendHistory() error = %v", err)
	}
	if err := store.AppendHistory(runtime.Key{CollectionPath: "users", Environment: "test", RequestID: "list"}, "GET", model.Response{StatusCode: 204, Duration: time.Millisecond}, nil); err != nil {
		t.Fatalf("AppendHistory() success error = %v", err)
	}

	for query, wantRequest := range map[string]string{
		"PAYMENTS": "users/create",
		"post":     "users/create",
		"oauth":    "users/create",
		"204":      "list",
	} {
		entries, err := store.SearchHistory(query)
		if err != nil {
			t.Fatalf("SearchHistory(%q) error = %v", query, err)
		}
		if len(entries) != 1 || entries[0].RequestID != wantRequest {
			t.Errorf("SearchHistory(%q) = %#v, want one %q entry", query, entries, wantRequest)
		}
	}

	raw, err := os.ReadFile(filepath.Join(workspace, ".apitool", "history.jsonl"))
	if err != nil {
		t.Fatalf("read history: %v", err)
	}
	for _, secret := range [][]byte{[]byte("access-token-123"), []byte("client-secret-456"), []byte("raw-body-789")} {
		if bytes.Contains(raw, secret) {
			t.Errorf("history contains secret %q: %s", secret, raw)
		}
	}
}

func TestSearchHistorySkipsOversizedAndStructurallyInvalidLinesNewestFirst(t *testing.T) {
	workspace := t.TempDir()
	store, err := runtime.Open(workspace)
	if err != nil {
		t.Fatal(err)
	}
	corrupt := append(bytes.Repeat([]byte("x"), 1024*1024+1), '\n')
	corrupt = append(corrupt, []byte("{}\n")...)
	if err := os.WriteFile(filepath.Join(workspace, ".apitool", "history.jsonl"), corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	key := runtime.Key{CollectionPath: "payments", Environment: "prod", RequestID: "list"}
	if err := store.AppendHistory(key, "GET", model.Response{StatusCode: 200}, nil); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendHistory(runtime.Key{CollectionPath: "payments", Environment: "prod", RequestID: "create"}, "POST", model.Response{StatusCode: 201}, nil); err != nil {
		t.Fatal(err)
	}
	entries, err := store.SearchHistory("payments")
	if err != nil {
		t.Fatalf("SearchHistory() error = %v, want malformed lines skipped", err)
	}
	if len(entries) != 2 || entries[0].RequestID != "create" || entries[1].RequestID != "list" {
		t.Errorf("SearchHistory() = %#v, want newest valid entries first", entries)
	}
}
