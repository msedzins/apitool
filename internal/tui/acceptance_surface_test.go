package tui_test

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"apitool/internal/app"
	"apitool/internal/auth"
	"apitool/internal/model"
	"apitool/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAcceptanceUI004NavigatesToServerErrorWithoutColor(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	for path, contents := range map[string]string{
		".git/placeholder":                 "",
		"demo/.api/collection.yaml":        "name: Demo\n",
		"demo/.api/environments/test.yaml": "name: test\n",
		"demo/.api/requests/item.yaml":     "name: Item\nmethod: GET\nrequest:\n  url: https://example.test/item\n",
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	service, err := app.New(app.Dependencies{Execute: func(_ context.Context, _ model.EffectiveRequest, _ auth.TokenProvider) (model.Response, *model.ExecutionError) {
		return model.Response{StatusCode: http.StatusInternalServerError, Body: []byte(`{"error":"fixture failure"}`)}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{Collection: "demo", Environment: "test"}); err != nil {
		t.Fatal(err)
	}
	m := tui.New(service, tui.Options{StartingCollection: "demo", Color: false})
	m, _ = acceptanceSurfaceUpdate(m, tea.KeyMsg{Type: tea.KeyDown})
	m, _ = acceptanceSurfaceUpdate(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(m.View(), "Request: item") || !strings.Contains(m.View(), "Focus: request") {
		t.Fatalf("keyboard navigation did not reach request pane: %s", m.View())
	}
	next, command := m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	if command == nil {
		t.Fatalf("request was not executable: %s", next.View())
	}
	next, _ = next.Update(command())
	view := next.View()
	if !strings.Contains(view, "500 Internal Server Error") {
		t.Fatalf("no-color response did not identify the server-error status: %s", view)
	}
	if strings.Contains(view, "\x1b[") {
		t.Fatalf("no-color response contains ANSI styling: %q", view)
	}
}

func TestAcceptanceUI014RunsSafeGitActionsFromPalette(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	for path, contents := range map[string]string{
		"users/.api/collection.yaml":        "name: Users\n",
		"users/.api/environments/test.yaml": "name: test\n",
		"users/.api/requests/list.yaml":     "name: List\nmethod: GET\nrequest:\n  url: https://example.test/users\n",
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	acceptanceSurfaceGit(t, root, "init", "-q")
	acceptanceSurfaceGit(t, root, "config", "user.name", "Acceptance")
	acceptanceSurfaceGit(t, root, "config", "user.email", "acceptance@example.test")
	acceptanceSurfaceGit(t, root, "add", ".")
	acceptanceSurfaceGit(t, root, "commit", "-qm", "fixture")
	bare := filepath.Join(t.TempDir(), "remote.git")
	acceptanceSurfaceGit(t, root, "branch", "-M", "main")
	acceptanceSurfaceGit(t, t.TempDir(), "init", "--bare", "--initial-branch=main", bare)
	acceptanceSurfaceGit(t, root, "remote", "add", "origin", bare)
	acceptanceSurfaceGit(t, root, "push", "-u", "origin", "main")
	request := filepath.Join(root, "users/.api/requests/list.yaml")
	if err := os.WriteFile(request, []byte("name: Changed\nmethod: GET\nrequest:\n  url: https://example.test/changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	unstaged := filepath.Join(root, "users/.api/requests/unstaged.yaml")
	if err := os.WriteFile(unstaged, []byte("name: Unstaged\nmethod: GET\nrequest:\n  url: https://example.test/unstaged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	acceptanceSurfaceGit(t, root, "add", "users/.api/requests/list.yaml")

	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{Collection: "users", Environment: "test"}); err != nil {
		t.Fatal(err)
	}
	m := tui.New(service, tui.Options{StartingCollection: "users"})
	for _, action := range []struct{ key, want, name string }{{"s", "users/.api/requests/list.yaml", "Status"}, {"f", "name: Changed", "Diff"}} {
		var command tea.Cmd
		m, command = acceptanceSurfacePaletteAction(m, action.key)
		if command == nil {
			t.Fatalf("Git %s did not start", action.name)
		}
		m, _ = m.Update(command())
		if !strings.Contains(m.View(), action.want) {
			t.Fatalf("Git %s output does not show %q: %s", action.name, action.want, m.View())
		}
	}

	// A blank message must leave the commit prompt active and must not commit.
	m, _ = acceptanceSurfacePaletteAction(m, "c")
	if !strings.Contains(m.View(), "Commit message") {
		t.Fatalf("Git Commit did not ask for a message: %s", m.View())
	}
	m, _ = acceptanceSurfaceUpdate(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(m.View(), "Commit message") || !strings.Contains(m.View(), "must not be blank") {
		t.Fatalf("blank commit message was not rejected: %s", m.View())
	}
	m, _ = acceptanceSurfaceUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("acceptance change")})
	m, command := acceptanceSurfaceUpdate(m, tea.KeyMsg{Type: tea.KeyEnter})
	if command == nil {
		t.Fatalf("nonblank commit did not start: %s", m.View())
	}
	m, _ = m.Update(command())
	if !strings.Contains(m.View(), "acceptance change") {
		t.Fatalf("commit result was not shown: %s", m.View())
	}
	if _, err := os.Stat(unstaged); err != nil {
		t.Fatalf("Git action altered the unstaged definition: %v", err)
	}
	statusCommand := exec.Command("git", "status", "--short")
	statusCommand.Dir = root
	status, err := statusCommand.CombinedOutput()
	if err != nil || !strings.Contains(string(status), "unstaged.yaml") || strings.Contains(string(status), "list.yaml") {
		t.Fatalf("Git actions staged or lost files: status=%q err=%v", status, err)
	}
	for _, action := range []struct{ key, want, name string }{{"u", "main", "Push"}, {"p", "Already up to date", "Pull"}} {
		var command tea.Cmd
		m, command = acceptanceSurfacePaletteAction(m, action.key)
		if command == nil {
			t.Fatalf("Git %s did not start: %s", action.name, m.View())
		}
		m, _ = m.Update(command())
		if !strings.Contains(m.View(), action.want) {
			t.Fatalf("Git %s result does not show %q: %s", action.name, action.want, m.View())
		}
	}
}

func acceptanceSurfaceUpdate(m tea.Model, key tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.Update(key)
}

func acceptanceSurfacePaletteAction(m tea.Model, key string) (tea.Model, tea.Cmd) {
	m, _ = acceptanceSurfaceUpdate(m, tea.KeyMsg{Type: tea.KeyCtrlK})
	return acceptanceSurfaceUpdate(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
}

func acceptanceSurfaceGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	command.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_AUTHOR_NAME=Acceptance", "GIT_AUTHOR_EMAIL=acceptance@example.test", "GIT_COMMITTER_NAME=Acceptance", "GIT_COMMITTER_EMAIL=acceptance@example.test")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}
