package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestGitPullKeepsDirtyEditorDraftSafe(t *testing.T) {
	m := editorWithJSON(t, `{"before":true}`)
	m.editor.SetURL("https://edited.example.test")
	command := m.startGitAction("Pull", "")
	if command != nil {
		t.Fatal("Git Pull started while the request editor had unsaved changes")
	}
	if !m.editor.Dirty() || m.editor.Request().Request.URL != "https://edited.example.test" {
		t.Fatal("Git Pull discarded the unsaved request draft")
	}
	if m.message != "Save or discard request edits before Git Pull" {
		t.Fatalf("Git Pull guard message = %q", m.message)
	}
}

func TestGitPullDoesNotExposeRequestsFromInvalidCollection(t *testing.T) {
	m, _ := requestScreen(t, "GET", false)
	workspace, err := m.service.Workspace()
	if err != nil {
		t.Fatal(err)
	}
	collectionPath := filepath.Join(workspace.Root, "demo", ".api", "collection.yaml")
	if err := os.WriteFile(collectionPath, []byte("name: [invalid\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := m.reloadAfterPull(); err != nil {
		t.Fatal(err)
	}
	if m.mode != collectionPickerMode || m.message == "" || !strings.Contains(m.message, "collection.yaml") {
		t.Fatalf("invalid collection mode=%v message=%q view=%s", m.mode, m.message, m.View())
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	if m.mode != collectionPickerMode {
		t.Fatalf("Esc left the invalid collection picker in mode %v", m.mode)
	}
	if _, command := m.Update(sendKey()); command != nil {
		t.Fatal("request execution remained available after Esc on an invalid collection")
	}
}

func TestGitOperationBlocksEditorInputWhileRunning(t *testing.T) {
	m := editorWithJSON(t, `{"before":true}`)
	m.gitBusy = true
	m.editor.SetURL("https://saved.example.test")
	before := m.editor.Request().Request.URL
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = next.(Model)
	if got := m.editor.Request().Request.URL; got != before {
		t.Fatalf("request editor changed from %q to %q while Git was running", before, got)
	}
}
