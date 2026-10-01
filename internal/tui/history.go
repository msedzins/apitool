package tui

import (
	"context"
	"fmt"
	"strings"

	"apitool/internal/app"
	"apitool/internal/runtime"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) openHistory() {
	if m.service == nil {
		m.message = "No workspace is open"
		return
	}
	m.historyReturnMode = m.mode
	m.historyQuery = ""
	m.historyIndex = 0
	m.mode = historyMode
	m.loadHistory()
}

func (m *Model) loadHistory() {
	entries, err := m.service.SearchHistory(context.Background(), m.historyQuery)
	if err != nil {
		m.historyEntries = nil
		m.message = "Could not read request history: " + err.Error()
		return
	}
	m.historyEntries = entries
	if m.historyIndex >= len(entries) {
		m.historyIndex = max(0, len(entries)-1)
	}
	m.message = ""
}

func (m Model) historyView() string {
	lines := []string{"Request history", "Filter: " + m.historyQuery, "Type to filter • ↑/↓ select • Enter reopen • Esc return"}
	if len(m.historyEntries) == 0 {
		lines = append(lines, "", "No request history")
		if m.message != "" {
			lines = append(lines, "", m.message)
		}
		return strings.Join(lines, "\n") + "\n"
	}
	for index, entry := range m.historyEntries {
		marker := "  "
		if index == m.historyIndex {
			marker = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s  %s %s  %s  %s", marker, entry.Timestamp.Local().Format("2006-01-02 15:04"), entry.Method, entry.CollectionPath+"/"+entry.RequestID, entry.Environment, entry.Result))
	}
	if m.message != "" {
		lines = append(lines, "", m.message)
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m *Model) handleHistoryKey(key tea.KeyMsg) {
	switch key.Type {
	case tea.KeyEsc:
		m.mode, m.message = m.historyReturnMode, ""
	case tea.KeyEnter:
		if m.historyIndex >= 0 && m.historyIndex < len(m.historyEntries) {
			m.reopenHistoryEntry(m.historyEntries[m.historyIndex])
		}
	case tea.KeyUp:
		if len(m.historyEntries) > 0 {
			m.historyIndex = max(0, m.historyIndex-1)
		}
	case tea.KeyDown:
		if len(m.historyEntries) > 0 {
			m.historyIndex = min(len(m.historyEntries)-1, m.historyIndex+1)
		}
	case tea.KeyBackspace:
		runes := []rune(m.historyQuery)
		if len(runes) > 0 {
			m.historyQuery = string(runes[:len(runes)-1])
			m.historyIndex = 0
			m.loadHistory()
		}
	case tea.KeyCtrlU:
		m.historyQuery = ""
		m.historyIndex = 0
		m.loadHistory()
	case tea.KeyRunes:
		m.historyQuery += string(key.Runes)
		m.historyIndex = 0
		m.loadHistory()
	}
}

func (m *Model) reopenHistoryEntry(entry runtime.HistoryEntry) {
	if m.editor != nil && m.editor.Dirty() {
		m.message = "Unsaved request changes; return to the editor to save or discard before reopening history"
		return
	}
	view, err := m.service.OpenCollection(context.Background(), entry.CollectionPath)
	if err != nil {
		m.message = "Could not reopen collection: " + err.Error()
		return
	}
	if _, ok := view.Environments[entry.Environment]; !ok {
		m.message = fmt.Sprintf("Environment %q no longer exists for %q", entry.Environment, entry.CollectionPath)
		return
	}
	if _, ok := view.Tree.Requests[entry.RequestID]; !ok {
		if _, invalid := view.Tree.Invalid[entry.RequestID]; !invalid {
			m.message = fmt.Sprintf("Referenced request %q no longer exists", entry.RequestID)
			return
		}
	}
	view, err = m.service.SelectEnvironment(context.Background(), entry.CollectionPath, entry.Environment)
	if err != nil {
		m.message = "Could not select history environment: " + err.Error()
		return
	}
	m.collection, m.view = entry.CollectionPath, view
	m.collectionIndex = indexOf(m.collections, entry.CollectionPath)
	m.expanded = map[string]bool{}
	for _, group := range view.Tree.Groups {
		m.expanded[group.ID] = true
	}
	m.mode, m.focus, m.message = browseMode, collectionPane, ""
	m.pendingEnvironment = ""
	m.result = app.SendResult{}
	m.gitAction, m.gitOutput = "", ""
	rows := m.visibleRows()
	for index, row := range rows {
		if (row.kind == requestRow || row.kind == invalidRow) && row.id == entry.RequestID {
			m.treeIndex = index
			m.selectedID = entry.RequestID
			m.treeOffset = 0
			m.syncViewport()
			if row.kind == invalidRow {
				m.openSelected()
			} else {
				m.focus = requestPane
			}
			m.savePreferences()
			return
		}
	}
	m.mode = historyMode
	m.message = fmt.Sprintf("Referenced request %q could not be selected", entry.RequestID)
}
