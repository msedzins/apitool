package app_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"testing"

	"apitool/internal/app"
	apiruntime "apitool/internal/runtime"
)

const (
	fixtureClientID     = "fixture-client-id"
	fixtureClientSecret = "fixture-client-secret-marker"
	fixtureAccessToken  = "fixture-access-token-marker"
	fixtureAPIKey       = "fixture-api-key-marker"
	fixtureCookie       = "fixture-cookie-marker"
	fixtureRawBody      = "fixture-raw-body-marker"
)

func TestFixtureWorkspaceCanDiscoverEditExecuteCacheAndRecall(t *testing.T) {
	server := newAPIServer(t, http.StatusOK, `{"ok":true}`)
	defer server.Close()
	t.Setenv("APITOOL_E2E_API_URL", server.URL)
	t.Setenv("APITOOL_E2E_TOKEN_URL", server.URL+"/oauth/token")
	t.Setenv("APITOOL_E2E_CLIENT_ID", fixtureClientID)
	t.Setenv("APITOOL_E2E_CLIENT_SECRET", fixtureClientSecret)
	t.Setenv("APITOOL_E2E_API_KEY", fixtureAPIKey)
	t.Setenv("APITOOL_E2E_COOKIE", fixtureCookie)
	t.Setenv("APITOOL_E2E_RAW_BODY", fixtureRawBody)

	root := fixtureRoot(t)
	service, err := app.New(fixtureDependencies(t, server))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{Collection: "users", Environment: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(opened.Collections); got != 2 {
		t.Fatalf("discovered %d collections, want 2", got)
	}
	users := opened.Collections["users"]
	if got := len(users.Tree.Invalid); got != 1 {
		t.Fatalf("invalid sibling requests = %d, want 1", got)
	}
	for id, invalid := range users.Tree.Invalid {
		if len(invalid.Diagnostics) == 0 {
			t.Fatalf("invalid sibling request %q has no diagnostic", id)
		}
	}
	request, ok := users.Tree.Requests["users/list"]
	if !ok {
		t.Fatal("users/list request was not discovered")
	}
	request.Request.Name = "List users (edited)"
	if err := service.SaveRequest(context.Background(), app.Selection{Collection: "users", RequestID: "users/list"}, request.Request); err != nil {
		t.Fatalf("SaveRequest() error = %v", err)
	}
	rawDefinition, err := os.ReadFile(filepath.Join(root, "users", ".api", "requests", "users", "list.yaml"))
	if err != nil || !strings.Contains(string(rawDefinition), "List users (edited)") {
		t.Fatalf("edited request definition was not persisted: err=%v, contents=%q", err, rawDefinition)
	}

	result := service.Send(context.Background(), app.Selection{Collection: "users", Environment: "test", RequestID: "users/list", Confirmed: true})
	if result.ExecutionError != nil || result.Response == nil || result.Response.StatusCode != http.StatusOK {
		t.Fatalf("Send() result = %#v", result)
	}
	if server.apiRequests != 1 || server.tokenRequests != 1 {
		t.Fatalf("server request counts = api:%d token:%d, want api:1 token:1", server.apiRequests, server.tokenRequests)
	}
	cache, err := apiruntime.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	key := apiruntime.Key{CollectionPath: "users", Environment: "test", RequestID: "users/list"}
	cached, err := cache.LatestResponse(key)
	if err != nil || cached.StatusCode != http.StatusOK || string(cached.Body) != `{"ok":true}` {
		t.Fatalf("LatestResponse() = %#v, %v", cached, err)
	}
	history, err := service.SearchHistory(context.Background(), "users/list")
	if err != nil || len(history) != 1 || history[0].StatusCode != http.StatusOK {
		t.Fatalf("SearchHistory() = %#v, %v", history, err)
	}
	for _, path := range []string{"history.jsonl", filepath.Join("logs", "executions.jsonl")} {
		if _, err := os.Stat(filepath.Join(root, ".apitool", path)); err != nil {
			t.Errorf("runtime record %s was not written below .apitool: %v", path, err)
		}
	}
	if len(result.Logs) != 1 || result.Logs[0].Key != key {
		t.Fatalf("execution logs = %#v, want one safe record for %#v", result.Logs, key)
	}
	assertRuntimeContainsNoSecrets(t, filepath.Join(root, ".apitool"))
	status, err := service.GitStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(status, "users/.api/requests/users/list.yaml") || strings.Contains(status, ".apitool") {
		t.Fatalf("Git status after runtime writes = %q, want the edited definition and no runtime paths", status)
	}
}

type fixtureServer struct {
	*httptest.Server
	apiRequests   int
	tokenRequests int
}

func newAPIServer(t *testing.T, status int, body string) *fixtureServer {
	t.Helper()
	fixture := &fixtureServer{}
	fixture.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			fixture.tokenRequests++
			clientID, secret, ok := r.BasicAuth()
			requestBody, _ := io.ReadAll(r.Body)
			if !ok || clientID != fixtureClientID || secret != fixtureClientSecret || !strings.Contains(string(requestBody), "grant_type=client_credentials") {
				http.Error(w, "invalid fixture credentials", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"access_token":%q,"token_type":"Bearer","expires_in":3600}`, fixtureAccessToken)
		case "/users":
			fixture.apiRequests++
			requestBody, _ := io.ReadAll(r.Body)
			if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer "+fixtureAccessToken || r.Header.Get("X-API-Key") != fixtureAPIKey || r.Header.Get("Cookie") != "session="+fixtureCookie || string(requestBody) != fixtureRawBody {
				http.Error(w, "fixture request did not match", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = io.WriteString(w, body)
		default:
			http.NotFound(w, r)
		}
	}))
	return fixture
}

func fixtureDependencies(_ *testing.T, _ *fixtureServer) app.Dependencies {
	return app.Dependencies{}
}

func fixtureRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := stdruntime.Caller(0)
	if !ok {
		t.Fatal("could not locate end-to-end test source")
	}
	sourceRoot := filepath.Join(filepath.Dir(source), "..", "..", "testdata", "workspace")
	root := filepath.Join(t.TempDir(), "workspace")
	if err := copyDirectory(sourceRoot, root); err != nil {
		t.Fatalf("copy fixture workspace: %v", err)
	}
	gitCommand(t, root, "init", "-q")
	gitCommand(t, root, "config", "user.email", "fixture@example.test")
	gitCommand(t, root, "config", "user.name", "Fixture Test")
	gitCommand(t, root, "add", ".")
	gitCommand(t, root, "commit", "-qm", "fixture workspace")
	return root
}

func copyDirectory(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func gitCommand(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func assertRuntimeContainsNoSecrets(t *testing.T, root string) {
	t.Helper()
	secrets := []string{fixtureClientSecret, fixtureAccessToken, fixtureAPIKey, fixtureCookie, fixtureRawBody}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, secret := range secrets {
			if strings.Contains(string(data), secret) {
				t.Errorf("runtime file %s contains sensitive fixture value %q", path, secret)
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("inspect runtime files: %v", err)
	}
}
