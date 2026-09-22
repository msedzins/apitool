package runtime_test

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"apitool/internal/model"
	"apitool/internal/runtime"
)

func TestResponseCacheIsIsolatedByEnvironment(t *testing.T) {
	store, err := runtime.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	key := runtime.Key{CollectionPath: "payments", RequestID: "users/list", Environment: "prod"}
	response := model.Response{StatusCode: 200, Body: []byte(`{"source":"prod"}`)}
	if err := store.SaveResponse(key, response); err != nil {
		t.Fatalf("SaveResponse() error = %v", err)
	}

	_, err = store.LatestResponse(runtime.Key{CollectionPath: "payments", RequestID: "users/list", Environment: "test"})
	if !errors.Is(err, runtime.ErrNotFound) {
		t.Fatalf("LatestResponse() error = %v, want ErrNotFound", err)
	}
}

func TestOpenCreatesOwnerOnlyRuntimeAndLogDirectories(t *testing.T) {
	workspace := t.TempDir()
	if _, err := runtime.Open(workspace); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	for _, path := range []string{
		filepath.Join(workspace, ".apitool"),
		filepath.Join(workspace, ".apitool", "logs"),
	} {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
			t.Errorf("runtime directory %q = %v, %v; want owner-only directory", path, info, err)
		}
	}
}

func TestResponseCachePersistsIdentityMetadataAndRedactedHeaders(t *testing.T) {
	workspace := t.TempDir()
	store, err := runtime.Open(workspace)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	key := runtime.Key{CollectionPath: "payments", Environment: "prod", RequestID: "users/list"}
	response := model.Response{
		StatusCode: 201,
		Headers: http.Header{
			"Authorization":   {"Bearer token-123"},
			"X-Client-Secret": {"client-secret-456"},
			"X-Trace":         {"safe-trace"},
		},
		Body:       []byte(`{"created":true}`),
		Duration:   125 * time.Millisecond,
		ReceivedAt: time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC),
	}
	if err := store.SaveResponse(key, response); err != nil {
		t.Fatalf("SaveResponse() error = %v", err)
	}

	cachePath := filepath.Join(workspace, ".apitool", "responses", "payments", "prod", "users", "list", "latest.json")
	raw, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read cache at expected isolated path: %v", err)
	}
	if bytes.Contains(raw, []byte("token-123")) || bytes.Contains(raw, []byte("client-secret-456")) {
		t.Errorf("cache contains secret: %s", raw)
	}
	if info, err := os.Stat(cachePath); err != nil || info.Mode().Perm()&0o077 != 0 {
		t.Errorf("cache permissions = %v, %v; want owner-only", info, err)
	}

	cached, err := store.LatestResponse(key)
	if err != nil {
		t.Fatalf("LatestResponse() error = %v", err)
	}
	if cached.StatusCode != 201 || string(cached.Body) != `{"created":true}` || cached.Duration != 125*time.Millisecond || !cached.ReceivedAt.Equal(response.ReceivedAt) {
		t.Errorf("LatestResponse() = %#v, want persisted response metadata", cached)
	}
	if cached.Headers.Get("Authorization") != "[REDACTED]" || cached.Headers.Get("X-Client-Secret") != "[REDACTED]" || cached.Headers.Get("X-Trace") != "safe-trace" {
		t.Errorf("LatestResponse() headers = %#v, want sensitive values redacted", cached.Headers)
	}
}

func TestResponseCacheRejectsTraversalInEveryKeyComponent(t *testing.T) {
	store, err := runtime.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	for name, key := range map[string]runtime.Key{
		"collection":  {CollectionPath: "../outside", Environment: "prod", RequestID: "users/list"},
		"environment": {CollectionPath: "payments", Environment: "../prod", RequestID: "users/list"},
		"request":     {CollectionPath: "payments", Environment: "prod", RequestID: "../../outside"},
	} {
		t.Run(name, func(t *testing.T) {
			err := store.SaveResponse(key, model.Response{StatusCode: 200})
			if err == nil {
				t.Fatal("SaveResponse() error = nil, want invalid key rejected")
			}
		})
	}
}

func TestStateIsEmptyWhenMissingAndPersistsOnlySessionPreferences(t *testing.T) {
	workspace := t.TempDir()
	store, err := runtime.Open(workspace)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	empty, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState() before save error = %v", err)
	}
	if len(empty.LastActiveEnvironment) != 0 || len(empty.PanelPreferences) != 0 {
		t.Errorf("LoadState() before save = %#v, want empty state", empty)
	}

	want := runtime.State{
		LastActiveEnvironment: map[string]string{"payments": "prod"},
		PanelPreferences:      map[string]bool{"response_headers_open": true},
	}
	if err := store.SaveState(want); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}
	got, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState() after save error = %v", err)
	}
	if got.LastActiveEnvironment["payments"] != "prod" || !got.PanelPreferences["response_headers_open"] {
		t.Errorf("LoadState() = %#v, want %#v", got, want)
	}
	if raw, err := os.ReadFile(filepath.Join(workspace, ".apitool", "state.json")); err != nil || bytes.Contains(raw, []byte("access-token")) || bytes.Contains(raw, []byte("client-secret")) {
		t.Errorf("state persistence = %q, %v; want non-secret preferences only", raw, err)
	}
}

func TestOpenRejectsRuntimeDirectorySymlink(t *testing.T) {
	workspace := t.TempDir()
	external := t.TempDir()
	if err := os.Symlink(external, filepath.Join(workspace, ".apitool")); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Open(workspace); err == nil {
		t.Fatal("Open() error = nil, want runtime directory symlink rejected")
	}
	if entries, err := os.ReadDir(external); err != nil || len(entries) != 0 {
		t.Errorf("external directory after Open() = %#v, %v; want untouched", entries, err)
	}
}

func TestResponseCacheRejectsNestedSymlink(t *testing.T) {
	workspace := t.TempDir()
	store, err := runtime.Open(workspace)
	if err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	responses := filepath.Join(workspace, ".apitool", "responses")
	if err := os.MkdirAll(responses, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(responses, "payments")); err != nil {
		t.Fatal(err)
	}
	err = store.SaveResponse(runtime.Key{CollectionPath: "payments", Environment: "prod", RequestID: "list"}, model.Response{StatusCode: 200})
	if err == nil {
		t.Fatal("SaveResponse() error = nil, want nested cache symlink rejected")
	}
	if entries, err := os.ReadDir(external); err != nil || len(entries) != 0 {
		t.Errorf("external directory after SaveResponse() = %#v, %v; want untouched", entries, err)
	}
}

func TestResponseCacheRejectsTokenBearingBodyAndAPIKeyHeader(t *testing.T) {
	workspace := t.TempDir()
	store, err := runtime.Open(workspace)
	if err != nil {
		t.Fatal(err)
	}
	key := runtime.Key{CollectionPath: "payments", Environment: "prod", RequestID: "token"}
	err = store.SaveResponse(key, model.Response{StatusCode: 200, Body: []byte(`{"access_token":"body-token-456"}`)})
	if err == nil {
		t.Fatal("SaveResponse() error = nil, want token-bearing body rejected")
	}
	if _, err := store.LatestResponse(key); !errors.Is(err, runtime.ErrNotFound) {
		t.Errorf("LatestResponse() error = %v, want ErrNotFound after rejected cache body", err)
	}

	safeKey := runtime.Key{CollectionPath: "payments", Environment: "prod", RequestID: "headers"}
	if err := store.SaveResponse(safeKey, model.Response{StatusCode: 200, Headers: http.Header{"X-API-Key": {"api-key-123"}}}); err != nil {
		t.Fatalf("SaveResponse() safe body error = %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(workspace, ".apitool", "responses", "payments", "prod", "headers", "latest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("api-key-123")) {
		t.Errorf("cached headers contain API key: %s", raw)
	}
}

func TestSaveStateMergesTwoStoresPreferences(t *testing.T) {
	workspace := t.TempDir()
	first, err := runtime.Open(workspace)
	if err != nil {
		t.Fatal(err)
	}
	second, err := runtime.Open(workspace)
	if err != nil {
		t.Fatal(err)
	}
	stateA, err := first.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	stateB, err := second.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	stateA.LastActiveEnvironment["payments"] = "prod"
	stateB.PanelPreferences["response_headers_open"] = true
	if err := first.SaveState(stateA); err != nil {
		t.Fatal(err)
	}
	if err := second.SaveState(stateB); err != nil {
		t.Fatal(err)
	}
	got, err := first.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if got.LastActiveEnvironment["payments"] != "prod" || !got.PanelPreferences["response_headers_open"] {
		t.Errorf("merged state = %#v, want both stores' updates", got)
	}
}
