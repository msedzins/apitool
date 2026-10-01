package tui

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"testing"

	"apitool/internal/app"
	"apitool/internal/runtime"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFixtureWorkspaceTUIWorkflow(t *testing.T) {
	const clientID, clientSecret = "tui-fixture-client", "tui-fixture-secret-marker"
	const accessToken, apiKey, cookie, rawBody = "tui-fixture-token-marker", "tui-fixture-api-key-marker", "tui-fixture-cookie-marker", "tui-fixture-body-marker"
	var tokenRequests, apiRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			tokenRequests++
			id, secret, ok := r.BasicAuth()
			body, _ := io.ReadAll(r.Body)
			values, _ := url.ParseQuery(string(body))
			if !ok || id != clientID || secret != clientSecret || values.Get("grant_type") != "client_credentials" {
				http.Error(w, "invalid fixture credentials", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"access_token":%q,"token_type":"Bearer","expires_in":3600}`, accessToken)
		case "/users":
			apiRequests++
			body, _ := io.ReadAll(r.Body)
			if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer "+accessToken || r.Header.Get("X-API-Key") != apiKey || r.Header.Get("Cookie") != "session="+cookie || string(body) != rawBody {
				http.Error(w, "fixture request did not match", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"workflow":"completed"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("APITOOL_E2E_API_URL", server.URL)
	t.Setenv("APITOOL_E2E_TOKEN_URL", server.URL+"/oauth/token")
	t.Setenv("APITOOL_E2E_CLIENT_ID", clientID)
	t.Setenv("APITOOL_E2E_CLIENT_SECRET", clientSecret)
	t.Setenv("APITOOL_E2E_API_KEY", apiKey)
	t.Setenv("APITOOL_E2E_COOKIE", cookie)
	t.Setenv("APITOOL_E2E_RAW_BODY", rawBody)

	root := fixtureWorkflowWorkspace(t)
	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{Collection: "users"}); err != nil {
		t.Fatal(err)
	}
	m := New(service, Options{StartingCollection: "users", Color: false}).(Model)
	m.width, m.height = 100, 30
	if got := m.View(); !strings.Contains(got, "broken (warning)") {
		t.Fatalf("TUI did not visibly mark the invalid sibling: %s", got)
	}

	// Select the active environment through the environment picker.
	m, _ = workflowUpdate(m, tea.KeyMsg{Type: tea.KeyCtrlE})
	m, _ = workflowUpdate(m, tea.KeyMsg{Type: tea.KeyDown})
	m, _ = workflowUpdate(m, tea.KeyMsg{Type: tea.KeyEnter})
	rows := m.visibleRows()
	target := -1
	for i, row := range rows {
		if row.kind == requestRow && row.id == "users/list" {
			target = i
			break
		}
	}
	if target < 0 {
		t.Fatal("users/list is not visible in the request tree")
	}
	for m.treeIndex < target {
		m, _ = workflowUpdate(m, tea.KeyMsg{Type: tea.KeyDown})
	}
	for m.treeIndex > target {
		m, _ = workflowUpdate(m, tea.KeyMsg{Type: tea.KeyUp})
	}
	m, _ = workflowUpdate(m, tea.KeyMsg{Type: tea.KeyEnter})
	selection, ok := m.executionSelection()
	if !ok || selection.RequestID != "users/list" || selection.Environment != "test" || m.focus != requestPane {
		t.Fatalf("opened request selection = %#v, valid=%v focus=%v", selection, ok, m.focus)
	}
	next, command := m.Update(sendKey())
	if command == nil {
		t.Fatalf("TUI did not start the request: %s", next.View())
	}
	next, _ = next.Update(command())
	m = next.(Model)
	if apiRequests != 1 || tokenRequests != 1 || !strings.Contains(m.View(), `"workflow": "completed"`) {
		t.Fatalf("TUI execution api=%d token=%d: %s", apiRequests, tokenRequests, m.View())
	}
	if got := m.View(); !strings.Contains(got, "broken (warning)") {
		t.Fatalf("invalid sibling marker disappeared after execution: %s", got)
	}
	m.responseTab = 2
	logView := strings.Join(m.responseLines(), "\n")
	if !strings.Contains(logView, "Request Log") {
		t.Fatalf("TUI request log was unavailable: %s", logView)
	}
	for _, secret := range []string{clientSecret, accessToken, apiKey, cookie, rawBody} {
		if strings.Contains(logView, secret) {
			t.Errorf("TUI request log contains sensitive fixture value %q: %s", secret, logView)
		}
	}
	m.responseTab = 0

	// Open the recorded result through the TUI history palette.
	m, _ = workflowUpdate(m, tea.KeyMsg{Type: tea.KeyCtrlK})
	m, _ = workflowUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if got := m.View(); !strings.Contains(got, "users/list") || !strings.Contains(got, "test") {
		t.Fatalf("TUI history did not show the completed request: %s", got)
	}
	m, _ = workflowUpdate(m, tea.KeyMsg{Type: tea.KeyEsc})

	// Inspect Git status through the same palette; runtime data must stay ignored.
	m, command = workflowUpdate(m, tea.KeyMsg{Type: tea.KeyCtrlK})
	m, command = workflowUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if command == nil {
		t.Fatal("Git Status did not start from the command palette")
	}
	next, _ = m.Update(command())
	m = next.(Model)
	if got := strings.Join(m.responseLines(), "\n"); !strings.Contains(got, "No output.") && !strings.Contains(got, "nothing to commit") && !strings.Contains(got, "working tree clean") {
		t.Fatalf("Git status = %q, want a clean fixture tree", got)
	}
	if strings.Contains(strings.Join(m.responseLines(), "\n"), ".apitool") {
		t.Fatalf("Git status exposed runtime data: %s", m.View())
	}

	store, err := runtime.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	key := runtime.Key{CollectionPath: "users", Environment: "test", RequestID: "users/list"}
	if response, err := store.LatestResponse(key); err != nil || !strings.Contains(string(response.Body), "workflow") {
		t.Fatalf("cached response = %#v, %v", response, err)
	}
	history, err := service.SearchHistory(context.Background(), "users/list")
	if err != nil || len(history) != 1 {
		t.Fatalf("history entries = %#v, %v", history, err)
	}
	for _, path := range []string{"history.jsonl", filepath.Join("logs", "executions.jsonl")} {
		contents, err := os.ReadFile(filepath.Join(root, ".apitool", path))
		if err != nil {
			t.Fatalf("read runtime record %s: %v", path, err)
		}
		for _, secret := range []string{clientSecret, accessToken, apiKey, cookie, rawBody} {
			if strings.Contains(string(contents), secret) {
				t.Errorf("runtime record %s contains sensitive fixture value %q", path, secret)
			}
		}
	}
}

func workflowUpdate(m Model, key tea.KeyMsg) (Model, tea.Cmd) {
	next, command := m.Update(key)
	return next.(Model), command
}

func fixtureWorkflowWorkspace(t *testing.T) string {
	t.Helper()
	_, source, _, ok := stdruntime.Caller(0)
	if !ok {
		t.Fatal("could not locate TUI workflow test source")
	}
	sourceRoot := filepath.Join(filepath.Dir(source), "..", "..", "testdata", "workspace")
	root := filepath.Join(t.TempDir(), "workspace")
	if err := copyWorkflowDirectory(sourceRoot, root); err != nil {
		t.Fatalf("copy workflow fixture: %v", err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "fixture@example.test"}, {"config", "user.name", "Fixture Test"}, {"add", "."}, {"commit", "-qm", "fixture workspace"}} {
		command := exec.Command("git", args...)
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
		}
	}
	return root
}

func copyWorkflowDirectory(source, destination string) error {
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
