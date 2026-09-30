package app_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"apitool/internal/app"
	"apitool/internal/auth"
	"apitool/internal/git"
	"apitool/internal/model"
	"apitool/internal/runtime"
)

func TestGitCommandsExposeSafeTextResultsWithoutStagingRuntimeFiles(t *testing.T) {
	root := realGitWorkspace(t)
	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}

	status, err := service.GitStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(status, " M ") || strings.Contains(status, ".apitool") {
		t.Fatalf("status = %q, want tracked modification without runtime files", status)
	}
	diff, err := service.GitDiff(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "workspace change") {
		t.Fatalf("diff = %q, want workspace change", diff)
	}
}

func TestGitCommitUsesOnlyAlreadyStagedChanges(t *testing.T) {
	root := realGitWorkspace(t)
	service, _ := app.New(app.Dependencies{})
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}

	if _, err := service.GitCommit(context.Background(), "unstaged"); err == nil {
		t.Fatal("GitCommit accepted an unstaged change")
	}
	stage := exec.Command("git", "add", "workspace.txt")
	stage.Dir = root
	if output, err := stage.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, output)
	}
	if _, err := service.GitCommit(context.Background(), "staged change"); err != nil {
		t.Fatal(err)
	}
	if _, err := git.New(root, exec.CommandContext).Status(context.Background()); err != nil {
		t.Fatal(err)
	}
}

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

func TestOpenWorkspaceRestoresActiveCollectionPreference(t *testing.T) {
	root := workspaceFixture(t)
	writeCollection(t, root, "orders", "dev")
	store, err := runtime.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(runtime.State{ActiveCollection: "orders", ExplorerWidth: 31}); err != nil {
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
	if opened.ActiveCollection != "orders" {
		t.Fatalf("active collection = %q, want orders", opened.ActiveCollection)
	}
	preferences, err := service.UIPreferences()
	if err != nil || preferences.ExplorerWidth != 31 {
		t.Fatalf("preferences = %#v, %v", preferences, err)
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
	view, err := service.OpenCollection(context.Background(), "payments")
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range view.Tree.Groups {
		if group.ID == "copies" {
			t.Fatal("deleted group remains in application tree")
		}
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

func realGitWorkspace(t *testing.T) string {
	t.Helper()
	root := workspaceFixture(t)
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
	write(".gitignore", ".apitool/\n")
	write("workspace.txt", "original\n")
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	run("init", "-q")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test User")
	run("add", ".")
	run("commit", "-qm", "initial")
	write("workspace.txt", "workspace change\n")
	return root
}

func TestSaveAndDuplicateRejectSymlinkedRequestDirectory(t *testing.T) {
	root := requestWorkspace(t, "http://127.0.0.1:1")
	external := t.TempDir()
	sentinel := filepath.Join(external, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "payments/.api/requests/linked")
	if err := os.Symlink(external, link); err != nil {
		t.Fatal(err)
	}
	service, _ := app.New(app.Dependencies{})
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}

	request := model.Request{Name: "New", Method: "GET", Request: model.RequestConfig{URL: "https://api.example.test/new"}}
	if err := service.SaveRequest(context.Background(), app.Selection{Collection: "payments", RequestID: "linked/save"}, request); err == nil {
		t.Fatal("SaveRequest accepted a symlinked request directory")
	}
	if _, err := service.DuplicateRequest(context.Background(), app.Selection{Collection: "payments", RequestID: "check"}, "linked/copy"); err == nil {
		t.Fatal("DuplicateRequest accepted a symlinked request directory")
	}
	entries, err := os.ReadDir(external)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "sentinel.txt" {
		t.Fatalf("external directory changed: %#v", entries)
	}
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != "keep" {
		t.Fatalf("sentinel changed: %q, %v", data, err)
	}
}

func TestSendRejectsInvalidAncestorGroup(t *testing.T) {
	for _, test := range []struct {
		name     string
		group    string
		envName  string
		envValue string
	}{
		{name: "malformed YAML", group: "auth: [not-a-mapping\n"},
		{name: "unsupported inherited auth", group: "auth:\n  type: oauth2\n  grant: authorization_code\n  token_url: https://auth.example.test/token\n  client_id: client\n  client_secret: ${API_CLIENT_SECRET}\n"},
		{name: "invalid resolved inherited auth", group: "auth:\n  type: oauth2\n  grant: client_credentials\n  token_url: ${EMPTY_TOKEN_URL}\n  client_id: client\n  client_secret: ${API_CLIENT_SECRET}\n", envName: "EMPTY_TOKEN_URL", envValue: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.envName != "" {
				t.Setenv(test.envName, test.envValue)
				t.Setenv("API_CLIENT_SECRET", "resolved-top-secret")
			}
			root := requestWorkspace(t, "http://api.example.test/admin")
			groupPath := filepath.Join(root, "payments/.api/requests/admin/_group.yaml")
			if err := os.MkdirAll(filepath.Dir(groupPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(groupPath, []byte(test.group), 0o644); err != nil {
				t.Fatal(err)
			}
			requestPath := filepath.Join(root, "payments/.api/requests/admin/list.yaml")
			if err := os.WriteFile(requestPath, []byte("name: List admin\nmethod: GET\nrequest:\n  url: http://api.example.test/admin\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			tokenProvider := &recordingTokenProvider{}
			executeCalls := 0
			service, _ := app.New(app.Dependencies{
				TokenProvider: tokenProvider,
				Execute: func(ctx context.Context, effective model.EffectiveRequest, provider auth.TokenProvider) (model.Response, *model.ExecutionError) {
					executeCalls++
					if effective.Auth != nil && !effective.Auth.None {
						_, _ = provider.Token(ctx, *effective.Auth)
					}
					return model.Response{StatusCode: http.StatusOK}, nil
				},
			})
			if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
				t.Fatal(err)
			}
			result := service.Send(context.Background(), app.Selection{Collection: "payments", Environment: "test", RequestID: "admin/list"})
			if executeCalls != 0 || tokenProvider.calls != 0 {
				t.Fatalf("invalid ancestor executed: execute calls=%d, token calls=%d", executeCalls, tokenProvider.calls)
			}
			if result.ExecutionError == nil || len(result.Diagnostics) == 0 {
				t.Fatalf("invalid ancestor result = %#v, want safe diagnostics and request error", result)
			}
			if test.envName != "" && !hasAppDiagnostic(result.Diagnostics, "auth_token_url_required") {
				t.Fatalf("resolved-auth diagnostics = %#v, want missing effective token URL", result.Diagnostics)
			}
		})
	}
}

func TestSendReportsRuntimePersistenceFailures(t *testing.T) {
	root := requestWorkspace(t, "http://api.example.test")
	service, _ := app.New(app.Dependencies{Execute: func(context.Context, model.EffectiveRequest, auth.TokenProvider) (model.Response, *model.ExecutionError) {
		return model.Response{StatusCode: http.StatusOK}, nil
	}})
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}

	runtimeDir := filepath.Join(root, ".apitool")
	if err := os.RemoveAll(filepath.Join(runtimeDir, "logs")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, "logs"), []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(runtimeDir, "history.jsonl"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, "responses"), []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}

	result := service.Send(context.Background(), app.Selection{Collection: "payments", Environment: "test", RequestID: "check"})
	if result.Response == nil || result.Response.StatusCode != http.StatusOK || result.ExecutionError != nil {
		t.Fatalf("persistence failure changed completed response: %#v", result)
	}
	for _, code := range []string{"runtime_log_write", "runtime_history_write", "runtime_cache_write"} {
		if !hasAppDiagnostic(result.Diagnostics, code) {
			t.Fatalf("diagnostics = %#v, want %s", result.Diagnostics, code)
		}
	}
}

type recordingTokenProvider struct{ calls int }

func (p *recordingTokenProvider) Token(context.Context, model.Auth) (auth.Token, error) {
	p.calls++
	return auth.Token{AccessToken: "test-token", TokenType: "Bearer"}, nil
}

func hasAppDiagnostic(diagnostics []model.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
