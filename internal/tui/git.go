package tui

import (
	"context"
	"sort"
	"strings"

	"apitool/internal/app"
	"apitool/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

type gitActionFinishedMsg struct {
	action string
	output string
	err    error
}

func (m *Model) startGitAction(action, message string) tea.Cmd {
	if m.service == nil || m.gitBusy {
		return nil
	}
	if action == "Pull" && m.editor != nil && (m.editor.Dirty() || m.fieldDraftDirty || m.jsonScalarDirty) {
		m.message = "Save or discard request edits before Git Pull"
		return nil
	}
	if m.mode != gitCommitMode {
		m.paletteReturnMode = m.mode
	}
	m.gitBusy = true
	m.gitAction, m.gitOutput = "", ""
	m.message = "Running Git " + action + "…"
	service := m.service
	return func() tea.Msg {
		ctx := context.Background()
		var output string
		var err error
		switch action {
		case "Status":
			output, err = service.GitStatus(ctx)
		case "Diff":
			output, err = service.GitDiff(ctx)
		case "Pull":
			output, err = service.GitPull(ctx)
		case "Push":
			output, err = service.GitPush(ctx)
		case "Commit":
			output, err = service.GitCommit(ctx, message)
		}
		return gitActionFinishedMsg{action: action, output: output, err: err}
	}
}

func (m *Model) reloadAfterPull() error {
	previous, hadSelection := m.executionSelection()
	workspace, err := m.service.ReloadWorkspace(context.Background())
	if err != nil {
		return err
	}
	m.collections = m.collections[:0]
	for path := range workspace.Collections {
		m.collections = append(m.collections, path)
	}
	sort.Strings(m.collections)
	m.collectionIndex = -1

	collectionPath := m.collection
	view, exists := workspace.Collections[collectionPath]
	if !exists {
		collectionPath = workspace.ActiveCollection
		view, exists = workspace.Collections[collectionPath]
	}
	if !exists && len(m.collections) > 0 {
		collectionPath = m.collections[0]
		view, exists = workspace.Collections[collectionPath]
	}
	m.clearResult()
	m.expanded = map[string]bool{}
	if !exists {
		m.collection, m.view = "", app.CollectionView{}
		m.collectionIndex, m.treeIndex, m.focus = 0, 0, collectionPane
		m.mode = collectionPickerMode
		return nil
	}
	if diagnostic := collectionLoadDiagnostic(view); diagnostic != nil {
		m.collection, m.view = collectionPath, view
		m.collectionIndex = indexOf(m.collections, collectionPath)
		m.treeIndex, m.focus = 0, collectionPane
		m.mode = collectionPickerMode
		m.message = diagnostic.Path + ": " + diagnostic.Message
		return nil
	}
	m.collection, m.view = collectionPath, view
	m.collectionIndex = indexOf(m.collections, collectionPath)
	for _, group := range view.Tree.Groups {
		m.expanded[group.ID] = true
	}
	m.treeIndex, m.focus = 0, collectionPane
	m.mode = browseMode
	if hadSelection && previous.Collection == collectionPath {
		for index, row := range m.visibleRows() {
			if row.id == previous.RequestID && (row.kind == requestRow || row.kind == invalidRow) {
				m.treeIndex = index
				if row.kind == requestRow {
					m.focus = requestPane
				}
				break
			}
		}
	}
	m.loadCachedResponse()
	return nil
}

func collectionLoadDiagnostic(view app.CollectionView) *model.Diagnostic {
	for i := range view.Diagnostics {
		if view.Diagnostics[i].Code == "collection_load" {
			return &view.Diagnostics[i]
		}
	}
	return nil
}

func (m *Model) handleGitCommitKey(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyEsc:
		m.mode, m.gitCommitMessage, m.message = m.paletteReturnMode, "", ""
	case tea.KeyEnter:
		if strings.TrimSpace(m.gitCommitMessage) == "" {
			m.message = "Commit message must not be blank"
			return nil
		}
		message := strings.TrimSpace(m.gitCommitMessage)
		return m.startGitAction("Commit", message)
	case tea.KeyBackspace:
		runes := []rune(m.gitCommitMessage)
		if len(runes) > 0 {
			m.gitCommitMessage = string(runes[:len(runes)-1])
		}
	case tea.KeyCtrlU:
		m.gitCommitMessage = ""
	case tea.KeyRunes:
		m.gitCommitMessage += string(key.Runes)
	}
	return nil
}

func (m Model) gitCommitView() string {
	lines := []string{"Git: Commit", "Commit message: " + m.gitCommitMessage, "Enter commit • Esc cancel"}
	if m.message != "" {
		lines = append(lines, "", m.message)
	}
	return strings.Join(lines, "\n") + "\n"
}
