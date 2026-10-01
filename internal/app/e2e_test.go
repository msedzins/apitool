package app_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"apitool/internal/app"
	"apitool/internal/runtime"
)

func TestFixtureWorkspaceCanDiscoverEditExecuteCacheAndRecall(t *testing.T) {
	var executions atomic.Int32
	const accessToken = "fixture-access-token-value"
	const fixtureSecret = "fixture-client-secret-value"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executions.Add(1)
		if r.URL.Path != "/users" {
			t.Errorf("request path = %q, want /users", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+accessToken {
			t.Errorf("authorization = %q, want fixture token", got)
		}
		if got := r.Header.Get("X-Fixture-Edit"); got != "saved" {
			t.Errorf("edited header = %q, want saved", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Set-Cookie", "session="+fixtureSecret)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	t.Setenv("APITOOL_TEST_SERVER", server.URL)
	t.Setenv("APITOOL_FIXTURE_ACCESS_TOKEN", accessToken)
	t.Setenv("APITOOL_FIXTURE_SECRET", fixtureSecret)
	root := copyFixtureWorkspace(t)
	initializeFixtureGit(t, root)

	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{Collection: "users", Environment: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(opened.Collections) != 2 {
		t.Fatalf("collections = %v, want users and payments", collectionNames(opened.Collections))
	}
	users, ok := opened.Collections["users"]
	if !ok || users.Environment != "test" {
		t.Fatalf("users collection/environment = %#v, want test environment", users)
	}
	if _, ok := opened.Collections["payments"]; !ok {
		t.Fatal("payments collection was not discovered")
	}
	if invalid := users.Tree.Invalid; len(invalid) == 0 {
		t.Fatal("malformed sibling request was not reported as invalid")
	}
	request := users.Tree.Requests["users/list"].Request
	if request.Request.Headers == nil {
		request.Request.Headers = map[string]string{}
	}
	request.Request.Headers["X-Fixture-Edit"] = "saved"
	if err := service.SaveRequest(context.Background(), app.Selection{Collection: "users", RequestID: "users/list"}, request); err != nil {
		t.Fatalf("save edited request: %v", err)
	}
	savedDefinition, err := os.ReadFile(filepath.Join(root, "users", ".api", "requests", "users", "list.yaml"))
	if err != nil || !strings.Contains(string(savedDefinition), "X-Fixture-Edit: saved") {
		t.Fatalf("saved request definition = %q, %v", savedDefinition, err)
	}

	result := service.Send(context.Background(), app.Selection{Collection: "users", Environment: "test", RequestID: "users/list"})
	if result.ExecutionError != nil || result.Response == nil || result.Response.StatusCode != http.StatusOK {
		t.Fatalf("send result = %#v", result)
	}
	if got := executions.Load(); got != 1 {
		t.Fatalf("local server executions = %d, want exactly one", got)
	}

	store, err := runtime.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	key := runtime.Key{CollectionPath: "users", Environment: "test", RequestID: "users/list"}
	cached, err := store.LatestResponse(key)
	if err != nil || cached.StatusCode != http.StatusOK || string(cached.Body) != `{"ok":true}` {
		t.Fatalf("cached response = %#v, %v", cached, err)
	}
	assertBytesDoNotContainFixtureSecrets(t, "decoded cached response body", cached.Body, accessToken, fixtureSecret)
	history, err := service.SearchHistory(context.Background(), "users/list")
	if err != nil || len(history) != 1 || history[0].StatusCode != http.StatusOK {
		t.Fatalf("history = %#v, %v", history, err)
	}
	reopened, err := service.OpenCollection(context.Background(), history[0].CollectionPath)
	if err != nil || reopened.Tree.Requests[history[0].RequestID].Request.Request.Headers["X-Fixture-Edit"] != "saved" {
		t.Fatalf("history request recall = %#v, %v", reopened.Tree.Requests[history[0].RequestID], err)
	}
	for _, path := range []string{
		filepath.Join(root, ".apitool", "history.jsonl"),
		filepath.Join(root, ".apitool", "logs", "executions.jsonl"),
		filepath.Join(root, ".apitool", "responses"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("runtime artifact %q missing: %v", path, err)
		}
	}
	assertNoFixtureSecrets(t, filepath.Join(root, ".apitool"), accessToken, fixtureSecret)

	status := exec.Command("git", "status", "--short", "--untracked-files=all")
	status.Dir = root
	output, err := status.CombinedOutput()
	if err != nil {
		t.Fatalf("git status: %v\n%s", err, output)
	}
	if strings.Contains(string(output), ".apitool") {
		t.Fatalf("runtime path appears in git status:\n%s", output)
	}
}

func copyFixtureWorkspace(t *testing.T) string {
	t.Helper()
	source := filepath.Join("..", "..", "testdata", "workspace")
	root := t.TempDir()
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil || rel == "." {
			return err
		}
		destination := filepath.Join(root, rel)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy fixture workspace: %v", err)
	}
	return root
}

func initializeFixtureGit(t *testing.T, root string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	run("init", "-q")
	gitignore := filepath.Join(root, ".gitignore")
	if err := os.WriteFile(gitignore, []byte(".apitool/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("-c", "user.name=apitool test", "-c", "user.email=apitool@example.test", "commit", "-qm", "fixture")
}

func assertNoFixtureSecrets(t *testing.T, root string, secrets ...string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, secret := range secrets {
			if containsFixtureSecret(data, secret) {
				t.Errorf("runtime file %q contains a fixture secret (plain or base64 encoded)", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("inspect runtime files: %v", err)
	}
}

func containsFixtureSecret(data []byte, secret string) bool {
	return strings.Contains(string(data), secret) || strings.Contains(string(data), base64.StdEncoding.EncodeToString([]byte(secret)))
}

func TestFixtureSecretScanDetectsBase64EncodedValues(t *testing.T) {
	secret := "fixture-access-token-value"
	encoded := []byte(base64.StdEncoding.EncodeToString([]byte(secret)))
	if !containsFixtureSecret(encoded, secret) {
		t.Fatal("base64-encoded fixture secret was not detected")
	}
}

func assertBytesDoNotContainFixtureSecrets(t *testing.T, label string, data []byte, secrets ...string) {
	t.Helper()
	for _, secret := range secrets {
		if strings.Contains(string(data), secret) {
			t.Errorf("%s contains a fixture secret", label)
		}
	}
}

func collectionNames(collections map[string]app.CollectionView) []string {
	names := make([]string, 0, len(collections))
	for name := range collections {
		names = append(names, name)
	}
	return names
}
