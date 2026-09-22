package app_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"apitool/internal/app"
	"apitool/internal/auth"
	"apitool/internal/model"
	"apitool/internal/runtime"
)

func TestOpenWorkspaceRestoresCollectionEnvironmentUnlessCLIOverride(t *testing.T) {
	root := workspaceFixture(t)
	store, err := runtime.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(runtime.State{LastActiveEnvironment: map[string]string{"payments": "test"}}); err != nil {
		t.Fatal(err)
	}
	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}

	opened, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := opened.Collections["payments"].Environment; got != "test" {
		t.Fatalf("restored environment = %q, want test", got)
	}
	opened, err = service.OpenWorkspace(context.Background(), root, app.OpenOptions{Collection: "payments", Environment: "prod"})
	if err != nil {
		t.Fatal(err)
	}
	if got := opened.Collections["payments"].Environment; got != "prod" {
		t.Fatalf("override environment = %q, want prod", got)
	}
}

func TestOpenWorkspaceIgnoresStaleStateAndScopesOverrideToSelectedCollection(t *testing.T) {
	root := workspaceFixture(t)
	writeCollection(t, root, "orders", "test")
	store, err := runtime.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(runtime.State{LastActiveEnvironment: map[string]string{"payments": "missing", "orders": "test"}}); err != nil {
		t.Fatal(err)
	}
	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{Collection: "payments", Environment: "prod"})
	if err != nil {
		t.Fatal(err)
	}
	if got := opened.Collections["payments"].Environment; got != "prod" {
		t.Fatalf("payments = %q, want prod", got)
	}
	if got := opened.Collections["orders"].Environment; got != "test" {
		t.Fatalf("orders = %q, want test", got)
	}
}

func TestSendTreatsHTTP500AsResponseAndRecordsConnectionFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }))
	defer server.Close()
	root := requestWorkspace(t, server.URL)
	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}
	result := service.Send(context.Background(), app.Selection{Collection: "payments", Environment: "test", RequestID: "check"})
	if result.Response == nil || result.Response.StatusCode != http.StatusInternalServerError || result.ExecutionError != nil {
		t.Fatalf("500 result = %#v", result)
	}

	if err := os.WriteFile(filepath.Join(root, "payments/.api/requests/check.yaml"), []byte("name: Check\nmethod: GET\nrequest:\n  url: http://127.0.0.1:1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}
	result = service.Send(context.Background(), app.Selection{Collection: "payments", Environment: "test", RequestID: "check"})
	if result.ExecutionError == nil || result.Response != nil {
		t.Fatalf("connection result = %#v", result)
	}
	store, err := runtime.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	history, err := store.SearchHistory("")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].ErrorCategory == "" {
		t.Fatalf("history = %#v", history)
	}
}

func TestDangerousSendRequiresConfirmationButGETDoesNot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	defer server.Close()
	root := requestWorkspace(t, server.URL)
	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{ConfirmDangerous: true}); err != nil {
		t.Fatal(err)
	}
	if result := service.Send(context.Background(), app.Selection{Collection: "payments", Environment: "test", RequestID: "remove"}); !result.ConfirmationRequired {
		t.Fatalf("DELETE should require confirmation: %#v", result)
	}
	if result := service.Send(context.Background(), app.Selection{Collection: "payments", Environment: "test", RequestID: "check"}); result.ConfirmationRequired || result.Response == nil {
		t.Fatalf("GET should send: %#v", result)
	}
}

func TestDuplicateAndDeleteExposeDistinctPathsBeforeMutation(t *testing.T) {
	root := requestWorkspace(t, "http://127.0.0.1:1")
	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}
	copy, err := service.DuplicateRequest(context.Background(), app.Selection{Collection: "payments", RequestID: "check"}, "copies/check-copy")
	if err != nil || copy.RequestID != "copies/check-copy" {
		t.Fatalf("duplicate = %#v, %v", copy, err)
	}
	if _, err := os.Stat(filepath.Join(root, "payments/.api/requests/copies/check-copy.yaml")); err != nil {
		t.Fatal(err)
	}
	target, err := service.Delete(context.Background(), "payments", "copies", true, false)
	if err != nil || len(target.Paths) != 2 {
		t.Fatalf("delete preview = %#v, %v", target, err)
	}
	if _, err := os.Stat(filepath.Join(root, "payments/.api/requests/copies/check-copy.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Delete(context.Background(), "payments", "copies", true, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "payments/.api/requests/copies")); !os.IsNotExist(err) {
		t.Fatalf("group still exists: %v", err)
	}
}

func TestDeleteRejectsSymlinkedGroupWithoutTouchingExternalFiles(t *testing.T) {
	root := requestWorkspace(t, "http://127.0.0.1:1")
	external := t.TempDir()
	sentinel := filepath.Join(external, "keep.yaml")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, "payments/.api/requests/linked")); err != nil {
		t.Fatal(err)
	}
	service, _ := app.New(app.Dependencies{})
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Delete(context.Background(), "payments", "linked", true, true); err == nil {
		t.Fatal("Delete accepted symlink")
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("external file changed: %v", err)
	}
}

func TestSaveAndDuplicateAreImmediatelySendable(t *testing.T) {
	root := requestWorkspace(t, "http://old.example")
	var urls []string
	service, _ := app.New(app.Dependencies{Execute: func(_ context.Context, e model.EffectiveRequest, _ auth.TokenProvider) (model.Response, *model.ExecutionError) {
		urls = append(urls, e.URL)
		return model.Response{StatusCode: 200}, nil
	}})
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}
	request := model.Request{Name: "New", Method: "GET", Request: model.RequestConfig{URL: "http://new.example"}}
	if err := service.SaveRequest(context.Background(), app.Selection{Collection: "payments", RequestID: "new"}, request); err != nil {
		t.Fatal(err)
	}
	if result := service.Send(context.Background(), app.Selection{Collection: "payments", Environment: "test", RequestID: "new"}); result.Response == nil {
		t.Fatalf("save send=%#v", result)
	}
	if _, err := service.DuplicateRequest(context.Background(), app.Selection{Collection: "payments", RequestID: "new"}, "copy"); err != nil {
		t.Fatal(err)
	}
	if result := service.Send(context.Background(), app.Selection{Collection: "payments", Environment: "test", RequestID: "copy"}); result.Response == nil {
		t.Fatalf("copy send=%#v", result)
	}
	if len(urls) != 2 || urls[0] != "http://new.example" || urls[1] != "http://new.example" {
		t.Fatalf("urls=%v", urls)
	}
}

func TestSendReturnsRedactedLogPath(t *testing.T) {
	root := requestWorkspace(t, "http://example.test/path?access_token=secret&trace=ok")
	service, _ := app.New(app.Dependencies{Execute: func(_ context.Context, _ model.EffectiveRequest, _ auth.TokenProvider) (model.Response, *model.ExecutionError) {
		return model.Response{StatusCode: 200}, nil
	}})
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}
	result := service.Send(context.Background(), app.Selection{Collection: "payments", Environment: "test", RequestID: "check"})
	if len(result.Logs) != 1 || strings.Contains(result.Logs[0].Path, "secret") || !strings.Contains(result.Logs[0].Path, "%5BREDACTED%5D") {
		t.Fatalf("log=%#v", result.Logs)
	}
}

func workspaceFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, data string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("payments/.api/collection.yaml", "name: Payments\n")
	write("payments/.api/environments/test.yaml", "name: test\n")
	write("payments/.api/environments/prod.yaml", "name: prod\n")
	return root
}

func requestWorkspace(t *testing.T, endpoint string) string {
	t.Helper()
	root := workspaceFixture(t)
	write := func(name, data string) {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("payments/.api/requests/check.yaml", "name: Check\nmethod: GET\nrequest:\n  url: "+endpoint+"\n")
	write("payments/.api/requests/remove.yaml", "name: Remove\nmethod: DELETE\nrequest:\n  url: "+endpoint+"\n")
	return root
}

func writeCollection(t *testing.T, root, name, environment string) {
	t.Helper()
	dir := filepath.Join(root, name, ".api", "environments")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, name, ".api", "collection.yaml"), []byte("name: "+name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, environment+".yaml"), []byte("name: "+environment+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
