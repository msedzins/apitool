package tui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"apitool/internal/app"
	"apitool/internal/auth"
	"apitool/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func TestGitPullReloadsDefinitionsBeforeNextSend(t *testing.T) {
	t.Setenv("APITOOL_E2E_API_URL", "https://old.example.test")
	t.Setenv("APITOOL_E2E_TOKEN_URL", "https://oauth.example.test/token")
	t.Setenv("APITOOL_E2E_CLIENT_ID", "fixture-client")
	t.Setenv("APITOOL_E2E_CLIENT_SECRET", "fixture-secret")
	t.Setenv("APITOOL_E2E_API_KEY", "fixture-key")
	t.Setenv("APITOOL_E2E_COOKIE", "fixture-cookie")
	t.Setenv("APITOOL_E2E_RAW_BODY", "fixture-body")

	root := fixtureWorkflowWorkspace(t)
	remote := filepath.Join(t.TempDir(), "origin.git")
	workflowGitCommand(t, root, "branch", "-M", "main")
	workflowGitCommand(t, t.TempDir(), "init", "--bare", "--initial-branch=main", remote)
	workflowGitCommand(t, root, "remote", "add", "origin", remote)
	workflowGitCommand(t, root, "push", "-u", "origin", "main")

	clone := filepath.Join(t.TempDir(), "upstream")
	workflowGitCommand(t, t.TempDir(), "clone", remote, clone)
	requestPath := filepath.Join(clone, "users", ".api", "requests", "users", "list.yaml")
	request := "name: List users after pull\nmethod: PATCH\nrequest:\n  url: '{{base_url}}/v2/users'\n  body:\n    type: raw\n    content: pulled-body\n"
	if err := os.WriteFile(requestPath, []byte(request), 0o644); err != nil {
		t.Fatal(err)
	}
	environmentPath := filepath.Join(clone, "users", ".api", "environments", "test.yaml")
	environment := "name: test\nvariables:\n  base_url: https://new.example.test\n  token_url: https://oauth.example.test/token\n"
	if err := os.WriteFile(environmentPath, []byte(environment), 0o644); err != nil {
		t.Fatal(err)
	}
	workflowGitCommand(t, clone, "add", "users/.api/requests/users/list.yaml", "users/.api/environments/test.yaml")
	workflowGitCommand(t, clone, "commit", "-m", "update users request")
	workflowGitCommand(t, clone, "push", "origin", "main")

	var sent model.EffectiveRequest
	sends := 0
	service, err := app.New(app.Dependencies{Execute: func(_ context.Context, request model.EffectiveRequest, _ auth.TokenProvider) (model.Response, *model.ExecutionError) {
		sent = request
		sends++
		return model.Response{StatusCode: 200, Body: []byte(`{"ok":true}`)}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{Collection: "users", Environment: "test"}); err != nil {
		t.Fatal(err)
	}
	m := New(service, Options{StartingCollection: "users"}).(Model)
	m.width, m.height = 100, 30
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

	// Pull through the same command palette action exposed to users.
	m, _ = workflowUpdate(m, tea.KeyMsg{Type: tea.KeyCtrlK})
	m, command := workflowUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	if command == nil {
		t.Fatal("Git Pull did not start")
	}
	next, _ := m.Update(command())
	m = next.(Model)
	if !strings.Contains(m.gitOutput, "update users request") && !strings.Contains(m.message, "Git: Pull") {
		t.Fatalf("Git Pull result was not shown: %s", m.View())
	}

	m.focus = requestPane
	next, command = m.Update(sendKey())
	if command == nil {
		t.Fatalf("send after Git Pull did not start: %s", next.View())
	}
	next, _ = next.Update(command())
	m = next.(Model)
	if sends != 1 || sent.Method != "PATCH" || sent.URL != "https://new.example.test/v2/users" || string(sent.Body) != "pulled-body" {
		t.Fatalf("request after pull: count=%d method=%q url=%q body=%q", sends, sent.Method, sent.URL, sent.Body)
	}

}

func workflowGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}
