package tui

import (
	"apitool/internal/app"
	"apitool/internal/auth"
	"apitool/internal/model"
	"apitool/internal/runtime"
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHTTP401RendersResponseNotFailure(t *testing.T) {
	m, _ := requestScreen(t, "GET", false)
	next := finishSend(m, app.SendResult{Response: &model.Response{StatusCode: 401}})
	if !strings.Contains(next.View(), "401 Unauthorized") || strings.Contains(next.View(), "Request failed") {
		t.Fatal(next.View())
	}
}
func TestResponseViewsAndDiagnostics(t *testing.T) {
	for _, code := range []int{302, 401, 500} {
		m, _ := requestScreen(t, "GET", false)
		m.width, m.height = 110, 32
		next := finishSend(m, app.SendResult{Response: &model.Response{StatusCode: code, Duration: time.Millisecond, Headers: http.Header{"Set-Cookie": {"secret"}, "Content-Type": {"application/json"}}, Body: []byte(`{"ok":true}`)}})
		view := next.View()
		for _, want := range []string{http.StatusText(code), "1ms", "11 bytes", "Content-Type", "[REDACTED]", "\"ok\": true"} {
			if !strings.Contains(view, want) {
				t.Fatalf("missing %s: %s", want, view)
			}
		}
		if strings.Contains(view, "secret") {
			t.Fatal(view)
		}
	}
	for _, x := range []*model.ExecutionError{{Stage: model.StageOAuth, Category: model.CategoryOAuth, SafeMessage: "OAuth failed"}, {Stage: model.StageTransport, Category: model.CategoryDNS, SafeMessage: "DNS failed"}} {
		m, _ := requestScreen(t, "GET", false)
		next := finishSend(m, app.SendResult{ExecutionError: x, Logs: []runtime.LogEntry{{Method: "GET", Data: map[string]any{"access_token": "secret"}}}})
		view := next.View()
		for _, want := range []string{string(x.Stage), string(x.Category), x.SafeMessage, "Request Log"} {
			if !strings.Contains(view, want) {
				t.Fatalf("missing %s: %s", want, view)
			}
		}
		if strings.Contains(view, "secret") {
			t.Fatal(view)
		}
	}
}

func finishSend(m Model, result app.SendResult) tea.Model {
	selection, ok := m.executionSelection()
	if !ok {
		return m
	}
	m.sending = true
	m.sendSelection = selection
	next, _ := m.Update(sendFinishedMsg{selection: selection, result: result})
	return next
}

func TestResponseResultStaysScopedToSelectedEnvironment(t *testing.T) {
	m, _ := requestScreen(t, "GET", false)
	workspace, err := m.service.Workspace()
	if err != nil {
		t.Fatal(err)
	}
	prodPath := filepath.Join(workspace.Root, "demo", ".api", "environments", "prod.yaml")
	if err := os.WriteFile(prodPath, []byte("name: prod\nvariables:\n  host: https://prod.example.test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := m.service.OpenWorkspace(context.Background(), workspace.Root, app.OpenOptions{Collection: "demo", Environment: "test"}); err != nil {
		t.Fatal(err)
	}
	m.selectEnvironment("prod")
	finished := finishSend(m, app.SendResult{
		Response: &model.Response{StatusCode: http.StatusOK, Body: []byte(`{"value":"prod-response-leak-marker"}`)},
		Logs:     []runtime.LogEntry{{Method: "GET", Path: "/prod-log-leak-marker", StatusCode: http.StatusOK}},
	})
	m = finished.(Model)
	if !strings.Contains(strings.Join(m.responseLines(), "\n"), "prod-response-leak-marker") {
		t.Fatal("expected the completed prod response to be visible under prod")
	}
	m.responseTab = 2
	if !strings.Contains(strings.Join(m.responseLines(), "\n"), "prod-log-leak-marker") {
		t.Fatal("expected the completed prod request log to be visible under prod")
	}
	m.selectEnvironment("test")
	for _, marker := range []string{"prod-response-leak-marker", "prod-log-leak-marker"} {
		if strings.Contains(strings.Join(m.responseLines(), "\n"), marker) {
			t.Fatalf("prod data %q remained visible under test: %v", marker, m.responseLines())
		}
	}
}

func TestCachedResponseRecallStaysScopedToEnvironment(t *testing.T) {
	m, _ := requestScreen(t, "GET", false)
	workspace, err := m.service.Workspace()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace.Root, "demo", ".api", "environments", "prod.yaml"), []byte("name: prod\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := runtime.Open(workspace.Root)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := m.service.OpenWorkspace(context.Background(), workspace.Root, app.OpenOptions{Collection: "demo", Environment: "test"})
	if err != nil {
		t.Fatal(err)
	}
	m.view = opened.Collections["demo"]
	selection, ok := m.executionSelection()
	if !ok {
		t.Fatal("request selection is unavailable")
	}
	for environment, body := range map[string]string{
		"prod": `{"source":"prod-cache-marker"}`,
		"test": `{"source":"test-cache-marker"}`,
	} {
		key := runtime.Key{CollectionPath: selection.Collection, Environment: environment, RequestID: selection.RequestID}
		if err := store.SaveResponse(key, model.Response{StatusCode: http.StatusOK, Body: []byte(body)}); err != nil {
			t.Fatalf("SaveResponse(%s) error = %v", environment, err)
		}
	}

	m.selectEnvironment("prod")
	prodLines := strings.Join(m.responseLines(), "\n")
	if !strings.Contains(prodLines, "prod-cache-marker") || strings.Contains(prodLines, "test-cache-marker") {
		t.Fatalf("prod cache view = %q selection=%#v", prodLines, selection)
	}
	m.selectEnvironment("test")
	testLines := strings.Join(m.responseLines(), "\n")
	if !strings.Contains(testLines, "test-cache-marker") || strings.Contains(testLines, "prod-cache-marker") {
		t.Fatalf("test cache view = %s", testLines)
	}
}

func TestStaleSendCompletionCannotReplaceCurrentEnvironmentResponse(t *testing.T) {
	m, _ := requestScreen(t, "GET", false)
	workspace, _ := m.service.Workspace()
	if err := os.WriteFile(filepath.Join(workspace.Root, "demo", ".api", "environments", "prod.yaml"), []byte("name: prod\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := m.service.OpenWorkspace(context.Background(), workspace.Root, app.OpenOptions{Collection: "demo", Environment: "test"}); err != nil {
		t.Fatal(err)
	}
	m.selectEnvironment("prod")
	prod := app.Selection{Collection: "demo", Environment: "prod", RequestID: "item"}
	m.sending, m.sendSelection = true, prod
	m.selectEnvironment("test")
	next, _ := m.Update(sendFinishedMsg{selection: prod, result: app.SendResult{Response: &model.Response{StatusCode: 200, Body: []byte("stale-prod-response-marker")}}})
	m = next.(Model)
	if strings.Contains(strings.Join(m.responseLines(), "\n"), "stale-prod-response-marker") {
		t.Fatal("stale prod completion replaced the current test response")
	}
}

func TestSuccessfulResponseShowsStorageDiagnostics(t *testing.T) {
	m, _ := requestScreen(t, "GET", false)
	workspace, err := m.service.Workspace()
	if err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(workspace.Root, ".apitool", "logs", "executions.jsonl")
	if err := os.Mkdir(logPath, 0o755); err != nil {
		t.Fatal(err)
	}
	next, command := m.Update(sendKey())
	if command == nil {
		t.Fatal("send command was not created")
	}
	next, _ = next.Update(command())
	m = next.(Model)
	if lines := strings.Join(m.responseLines(), "\n"); !strings.Contains(lines, "401 Unauthorized") || !strings.Contains(lines, "could not write execution log") {
		t.Fatalf("response pane did not retain the response and expose its storage warning: %s", lines)
	}
	m.responseTab = 1
	if lines := strings.Join(m.responseLines(), "\n"); !strings.Contains(lines, "runtime_log_write") || !strings.Contains(lines, "could not write execution log") {
		t.Fatalf("diagnostic tab hid the successful-send warning: %s", lines)
	}
}

func TestCachedResponseReloadedWhenOpeningCollection(t *testing.T) {
	m, _ := requestScreen(t, "GET", false)
	next, command := m.Update(sendKey())
	if command == nil {
		t.Fatal("send command was not created")
	}
	nextModel, _ := next.Update(command())
	m = nextModel.(Model)
	if !strings.Contains(strings.Join(m.responseLines(), "\n"), "401 Unauthorized") {
		t.Fatal("fresh response was not displayed")
	}
	workspace, err := m.service.Workspace()
	if err != nil {
		t.Fatal(err)
	}
	restartedService, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restartedService.OpenWorkspace(context.Background(), workspace.Root, app.OpenOptions{Collection: "demo", Environment: "test"}); err != nil {
		t.Fatal(err)
	}
	reopened := New(restartedService, Options{StartingCollection: "demo"}).(Model)
	lines := strings.Join(reopened.responseLines(), "\n")
	if !strings.Contains(lines, "Cached") || !strings.Contains(lines, "denied") {
		t.Fatalf("cached response was not reloaded and identified as cached: %s", lines)
	}
}
func TestPrettyBodyFallsBackToRaw(t *testing.T) {
	if got := prettyBody([]byte("not json")); got != "not json" {
		t.Fatal(got)
	}
	if got := prettyBody([]byte(`{"x":1}`)); got != "{\n  \"x\": 1\n}" {
		t.Fatal(got)
	}
}

type sessionTokenProvider struct{}

func (sessionTokenProvider) Token(_ context.Context, _ model.Auth) (auth.Token, error) {
	return auth.Token{AccessToken: "session-secret", TokenType: "Bearer", Expiry: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)}, nil
}
func TestAuthTokenMaskedAndRevealIsSessionOnly(t *testing.T) {
	m, _ := requestScreenWithAuth(t, "GET", false, sessionTokenProvider{})
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if cmd == nil {
		t.Fatalf("Auth must load asynchronously: %s", next.View())
	}
	next, _ = next.Update(cmd())
	view := next.View()
	for _, want := range []string{"client_credentials", "oauth.example.test/token", "read, write", "2030", "[REDACTED]", "Show token"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %s: %s", want, view)
		}
	}
	if strings.Contains(view, "session-secret") || strings.Contains(view, "hidden-client-secret") {
		t.Fatal(view)
	}
	next, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if !strings.Contains(next.View(), "session-secret") {
		t.Fatal("explicit reveal failed")
	}
	fresh := New(m.service, Options{StartingCollection: "demo"})
	fresh, cmd = fresh.Update(tea.KeyMsg{Type: tea.KeyTab})
	fresh, cmd = fresh.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if cmd == nil {
		t.Fatal("fresh Auth load missing")
	}
	fresh, _ = fresh.Update(cmd())
	if strings.Contains(fresh.View(), "session-secret") {
		t.Fatal("reveal leaked to new session")
	}
	workspace, _ := m.service.Workspace()
	err := filepath.WalkDir(filepath.Join(workspace.Root, ".apitool"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(data), "session-secret") {
				t.Errorf("persisted token in %s", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestResponsePaneScrollAndRawToggle(t *testing.T) {
	m, _ := requestScreen(t, "GET", false)
	m.width, m.height = 100, 24
	m.focus = responsePane
	next := finishSend(m, app.SendResult{Response: &model.Response{StatusCode: 500, Body: []byte(`{"many":[1,2,3,4,5,6,7,8,9]}`)}})
	if strings.Contains(next.View(), "9") {
		t.Fatal("body should exceed pane")
	}
	for i := 0; i < 20; i++ {
		next, _ = next.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if !strings.Contains(next.View(), "9") {
		t.Fatal("body cannot scroll")
	}
	next, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	if !strings.Contains(next.View(), `{"many":[1,2,3,4,5,6,7,8,9]}`) {
		t.Fatal("raw body lost")
	}
}

func TestCompactResponseScrollsThroughWrappedBody(t *testing.T) {
	m, _ := requestScreen(t, "GET", false)
	m.width, m.height = 60, 10
	m.focus = responsePane
	next := finishSend(m, app.SendResult{Response: &model.Response{StatusCode: 200, Body: []byte("first\nsecond\nthird\nfourth\nfifth\nlast-visible-body-line")}})
	before := next.View()
	for i := 0; i < 30; i++ {
		next, _ = next.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if next.View() == before || !strings.Contains(next.View(), "last-visible-body-line") {
		t.Fatalf("compact scroll did not reveal body tail: %s", next.View())
	}
}
