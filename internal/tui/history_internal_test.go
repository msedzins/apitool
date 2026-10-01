package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"apitool/internal/app"
	"apitool/internal/model"
	"apitool/internal/runtime"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHistoryDoesNotLeaveDirtyEditorBehind(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.editor.SetURL("https://changed.example.test")
	workspace, err := m.service.Workspace()
	if err != nil {
		t.Fatal(err)
	}
	store, err := runtime.Open(workspace.Root)
	if err != nil {
		t.Fatal(err)
	}
	key := runtime.Key{CollectionPath: m.collection, Environment: "test", RequestID: "one"}
	if err := store.AppendHistory(key, "POST", model.Response{StatusCode: 200}, nil); err != nil {
		t.Fatal(err)
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlK})
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("one")})
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != historyMode || m.editor == nil || !m.editor.Dirty() || m.message == "" {
		t.Fatalf("history selection mode=%v dirty=%v message=%q entries=%#v view=%q, want selection blocked with edit preserved", m.mode, m.editor != nil && m.editor.Dirty(), m.message, m.historyEntries, m.View())
	}
}

func TestEditorSplitKeepsMinimumResponseRowsAtTallLayoutThreshold(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.width, m.height = 90, 22
	_, maximum := m.responseSplitBounds(m.height)
	if got := m.responseDivider(); got > maximum {
		t.Fatalf("editor response split = %d, beyond maximum %d at height %d", got, maximum, m.height)
	}
}

func TestSendReplacesPreviousGitOutput(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.mode = browseMode
	m.width, m.height = 90, 24
	m.gitAction, m.gitOutput = "Status", "working tree clean"
	result := app.SendResult{Response: &model.Response{StatusCode: 201, Body: []byte("created")}}
	next, _ := m.Update(sendFinishedMsg{result: result})
	m = next.(Model)
	if got := m.View(); !strings.Contains(got, "201 Created") || !strings.Contains(got, "created") || strings.Contains(got, "working tree clean") {
		t.Fatalf("response after Git action = %q, want the new HTTP response only", got)
	}
}

func TestHistoryReopensCurrentRequestThatNowHasDiagnostics(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	workspace, err := m.service.Workspace()
	if err != nil {
		t.Fatal(err)
	}
	store, err := runtime.Open(workspace.Root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AppendHistory(runtime.Key{CollectionPath: m.collection, Environment: "test", RequestID: "one"}, "POST", model.Response{StatusCode: 201}, nil); err != nil {
		t.Fatal(err)
	}
	requestPath := filepath.Join(m.view.Root, ".api", "requests", "one.yaml")
	if err := os.WriteFile(requestPath, []byte("name: [invalid\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := m.service.OpenWorkspace(context.Background(), workspace.Root, app.OpenOptions{Collection: m.collection}); err != nil {
		t.Fatal(err)
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlK})
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("one")})
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	rows := m.visibleRows()
	if m.mode != browseMode || m.treeIndex < 0 || m.treeIndex >= len(rows) || rows[m.treeIndex].kind != invalidRow || m.message == "" {
		t.Fatalf("history reopen mode=%v selection=%#v treeIndex=%d message=%q, want current invalid definition selected", m.mode, rows, m.treeIndex, m.message)
	}
}
