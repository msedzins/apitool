package tui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"apitool/internal/app"
	"apitool/internal/collection"
	"apitool/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

// These scenarios exercise the editor through the TUI model and inspect the
// definition files users review and commit.
func TestAcceptanceConfig001SaveValidRequestEdit(t *testing.T) {
	m := editorWithJSON(t, `{"old":true}`)
	acceptanceConfigRunGit(t, m.view.Root, "init", "-q")
	acceptanceConfigRunGit(t, m.view.Root, "config", "user.email", "acceptance@example.test")
	acceptanceConfigRunGit(t, m.view.Root, "config", "user.name", "Acceptance")
	acceptanceConfigRunGit(t, m.view.Root, "add", ".api")
	acceptanceConfigRunGit(t, m.view.Root, "commit", "-qm", "baseline")
	m.editor.SetMethod("patch")
	m.editor.SetURL("https://edited.example.test/items")
	m.editor.SetHeaders(map[string]string{"Accept": "application/json", "X-Review": "yes"})
	m.editor.SetBodyText(`{"saved":true,"count":2}`)

	cmd := m.Save()
	if cmd == nil {
		t.Fatalf("explicit save rejected valid edits: %s", m.message)
	}
	m, _ = updateModel(m, cmd())
	request, err := collection.LoadRequest(filepath.Join(m.view.Root, ".api", "requests", "one.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if request.Method != "PATCH" || request.Request.URL != "https://edited.example.test/items" || request.Request.Headers["X-Review"] != "yes" || request.Request.Body.Content.(map[string]any)["saved"] != true {
		t.Fatalf("persisted request does not reflect saved edits: %#v", request)
	}
	if got := acceptanceConfigRunGit(t, m.view.Root, "status", "--short"); !strings.Contains(got, "one.yaml") {
		t.Fatalf("saved request is not visible as a Git definition change: %q", got)
	}
}

func TestAcceptanceConfig002InvalidBodySaveIsBlocked(t *testing.T) {
	for _, scenario := range []struct {
		name string
		edit func(*Model)
		want string
	}{
		{name: "invalid JSON body", edit: func(m *Model) { m.editor.SetBodyText(`{"broken":`) }, want: "json"},
		{name: "missing required URL", edit: func(m *Model) { m.editor.SetURL("") }, want: "url"},
		{name: "unsupported auth type", edit: func(m *Model) { m.editor.SetAuth(&model.Auth{Type: "basic"}) }, want: "oauth2"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			m := editorWithJSON(t, `{"original":true}`)
			path := filepath.Join(m.view.Root, ".api", "requests", "one.yaml")
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			scenario.edit(&m)
			if cmd := m.Save(); cmd != nil {
				t.Fatal("save command created for invalid request definition")
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != string(before) {
				t.Fatal("blocked save changed persisted definition")
			}
			feedback := strings.ToLower(m.message + m.editor.Validation())
			if !strings.Contains(feedback, scenario.want) {
				t.Fatalf("validation feedback is not actionable: message=%q validation=%q", m.message, m.editor.Validation())
			}
		})
	}
}

func TestAcceptanceConfig002UnsupportedBodyAndTimeoutBlockSend(t *testing.T) {
	for _, scenario := range []struct {
		name, file, contents, want string
	}{
		{
			name:     "unsupported body type",
			file:     "demo/.api/requests/item.yaml",
			contents: "name: Item\nmethod: GET\nrequest:\n  url: https://example.test/items\n  body:\n    type: xml\n    content: '<item/>'\n",
			want:     "body type must be json or raw",
		},
		{
			name:     "invalid timeout",
			file:     "demo/.api/collection.yaml",
			contents: "name: Demo\nhttp:\n  timeout: 0s\n",
			want:     "timeout must be a positive Go duration",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			m, calls := requestScreen(t, "GET", false)
			workspace, err := m.service.Workspace()
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(workspace.Root, filepath.FromSlash(scenario.file))
			if err := os.WriteFile(path, []byte(scenario.contents), 0o600); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := m.service.OpenWorkspace(context.Background(), workspace.Root, app.OpenOptions{Collection: "demo", Environment: "test"}); err != nil {
				t.Fatal(err)
			}
			updated, cmd := m.Update(sendKey())
			if cmd == nil {
				t.Fatalf("invalid definition was not rejected with a diagnostic: %s", updated.View())
			}
			updated, _ = updated.Update(cmd())
			m = updated.(Model)
			view := strings.Join(m.responseLines(), "\n")
			if calls == nil || *calls != 0 {
				t.Fatalf("invalid definition reached executor: calls=%v", calls)
			}
			if !strings.Contains(strings.ToLower(view), strings.ToLower(scenario.want)) {
				t.Fatalf("diagnostic lacks actionable feedback %q: %s", scenario.want, view)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != string(before) {
				t.Fatal("blocked send changed the persisted invalid definition")
			}
		})
	}
}

func TestAcceptanceConfig003LiteralClientSecretIsNotSaved(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	acceptanceConfigRunGit(t, m.view.Root, "init", "-q")
	acceptanceConfigRunGit(t, m.view.Root, "config", "user.email", "acceptance@example.test")
	acceptanceConfigRunGit(t, m.view.Root, "config", "user.name", "Acceptance")
	acceptanceConfigRunGit(t, m.view.Root, "add", ".api")
	acceptanceConfigRunGit(t, m.view.Root, "commit", "-qm", "baseline")
	secret := "acceptance-literal-secret"
	auth := &model.Auth{Type: "oauth2", Grant: "client_credentials", TokenURL: "https://auth.example.test/token", ClientID: "client", ClientSecret: secret}
	m.editor.SetAuth(auth)
	path := filepath.Join(m.view.Root, ".api", "requests", "one.yaml")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cmd := m.Save(); cmd != nil {
		t.Fatal("save command created for a literal client secret")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) || strings.Contains(string(after), secret) {
		t.Fatalf("literal secret reached the persisted definition: %s", after)
	}
	if strings.Contains(m.View(), secret) || !strings.Contains(strings.ToLower(m.message+m.editor.Validation()), "secret") {
		t.Fatalf("secret rejection must be actionable without echoing the value: %q", m.View())
	}
	if got := acceptanceConfigRunGit(t, m.view.Root, "status", "--short"); got != "" {
		t.Fatalf("blocked save left a Git-visible definition change: %q", got)
	}
}

func TestAcceptanceConfig005StartupEnvironmentOverridesRememberedEnvironment(t *testing.T) {
	prior := editorWithJSON(t, `{"a":1}`)
	root := filepath.Dir(prior.view.Root)
	writeEditorFile(t, root, "test-api/.api/environments/prod.yaml", "name: prod\n")
	definitions := acceptanceConfigDefinitionSnapshot(t, root)
	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SelectEnvironment(context.Background(), "test-api", "test"); err != nil {
		t.Fatal(err)
	}
	m := ModelFrom(New(service, Options{StartingCollection: "test-api", StartingEnvironment: "prod"}))
	if !strings.Contains(m.View(), "Environment: prod") {
		t.Fatalf("startup override was not displayed: %s", m.View())
	}
	if got := acceptanceConfigDefinitionSnapshot(t, root); got != definitions {
		t.Fatal("startup environment override changed repository definitions")
	}
}

func TestAcceptanceUI005DirtyNavigationOffersSaveDiscardCancel(t *testing.T) {
	m := editorWithChangedURL(t)
	path := filepath.Join(m.view.Root, ".api", "requests", "one.yaml")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyDown})
	if !strings.Contains(m.View(), "Save and continue") || !strings.Contains(m.View(), "Discard changes") || !strings.Contains(m.View(), "Cancel") {
		t.Fatalf("dirty navigation choices missing: %s", m.View())
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) || m.editor.Selection().RequestID != "one" {
		t.Fatal("canceling dirty navigation saved changes or left the current request")
	}
}

func TestAcceptanceUI006UndoRedoHistoryResetsForOpenedRequest(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.editor.SetURL("https://first-edit.example.test")
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyDown})
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if m.editor.Selection().RequestID != "nested/one" {
		t.Fatalf("navigation opened %q, want nested/one", m.editor.Selection().RequestID)
	}
	original := m.editor.Request().Request.URL
	for _, key := range []tea.KeyMsg{{Type: tea.KeyCtrlZ}, {Type: tea.KeyCtrlY}} {
		m, _ = updateModel(m, key)
		if got := m.editor.Request().Request.URL; got != original || !strings.Contains(m.View(), original) {
			t.Fatalf("undo/redo in the new editing session changed rendered request URL to %q; want %q. View: %s", got, original, m.View())
		}
	}
	m.editor.SetURL("https://second-edit.example.test")
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlZ})
	if got := m.editor.Request().Request.URL; got != original || !strings.Contains(m.View(), original) {
		t.Fatalf("undo did not render current request's original URL: %q. View: %s", got, m.View())
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlY})
	if got := m.editor.Request().Request.URL; got != "https://second-edit.example.test" || !strings.Contains(m.View(), got) {
		t.Fatalf("redo did not render current request's edit: %q. View: %s", got, m.View())
	}
}

func TestAcceptanceUI007SaveJSONAndRawBodyModes(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.editor.SetBodyText(`{"b":2,"a":1}`)
	cmd := m.Save()
	if cmd == nil {
		t.Fatalf("save rejected valid JSON body: %s", m.message)
	}
	m, _ = updateModel(m, cmd())
	request, err := collection.LoadRequest(filepath.Join(m.view.Root, ".api", "requests", "one.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if request.Request.Body.Type != "json" || !strings.Contains(m.editor.FieldText(6), "\n") {
		t.Fatalf("JSON was not saved as formatted structured data: type=%q body=%q", request.Request.Body.Type, m.editor.FieldText(6))
	}
	m.editor.SwitchBodyMode(BodyModeRaw)
	raw := "  first line\r\nsecond line \n"
	m.editor.SetBodyText(raw)
	cmd = m.Save()
	if cmd == nil {
		t.Fatalf("save rejected raw body: %s", m.message)
	}
	m, _ = updateModel(m, cmd())
	request, err = collection.LoadRequest(filepath.Join(m.view.Root, ".api", "requests", "one.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if request.Request.Body.Type != "raw" || request.Request.Body.Content != raw {
		t.Fatalf("raw body did not round-trip exactly: %#v", request.Request.Body)
	}
}

func TestAcceptanceConfig007SaveFailureKeepsDraftOpen(t *testing.T) {
	m := editorWithChangedURL(t)
	path := filepath.Join(m.view.Root, ".api", "requests", "one.yaml")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Make the request's parent path unusable after loading the editor, so the
	// save attempt fails independently of permission behavior (including root).
	requestDir := filepath.Dir(path)
	if err := os.Rename(requestDir, requestDir+"-saved"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(requestDir, []byte("blocked"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyDown})
	if !strings.Contains(m.View(), "Save and continue") {
		t.Fatalf("dirty navigation prompt missing: %s", m.View())
	}
	m, cmd := updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if cmd == nil {
		t.Fatalf("Save and continue did not attempt saving: %s", m.View())
	}
	m, _ = updateModel(m, cmd())
	if m.editor.Selection().RequestID != "one" || !m.editor.Dirty() || !strings.Contains(m.View(), "Save and continue") {
		t.Fatalf("failed save did not retain the open dirty draft: %s", m.View())
	}
	if m.message == "" {
		t.Fatal("failed save did not report its error")
	}
	if err := os.Remove(requestDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(requestDir+"-saved", requestDir); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("failed save changed the persisted request definition")
	}
}

func acceptanceConfigDefinitionSnapshot(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	paths := []string{
		"test-api/.api/collection.yaml",
		"test-api/.api/environments/test.yaml",
		"test-api/.api/environments/prod.yaml",
		"test-api/.api/requests/one.yaml",
	}
	for _, rel := range paths {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatal(err)
		}
		b.WriteString(rel)
		b.Write(data)
	}
	return b.String()
}

func acceptanceConfigRunGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}
