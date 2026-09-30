package tui

import (
	"context"
	"fmt"
	"strings"

	"apitool/internal/app"
	"apitool/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

type confirmationKind uint8

const (
	confirmDirtyNavigation confirmationKind = iota
	confirmDeleteTarget
	commandPalette
)

type confirmation struct {
	kind         confirmationKind
	targetID     string
	group        bool
	paths        []string
	deleteTarget app.DeleteTarget
	choice       string
	saving       bool
}

type requestSavedMsg struct {
	selection app.Selection
	request   model.Request
	err       error
}

type deleteFinishedMsg struct {
	target app.DeleteTarget
	err    error
}

func (m *Model) beginDuplicate() {
	if m.editor == nil {
		return
	}
	selection := m.editor.Selection()
	request := m.editor.Request()
	targetID := selection.RequestID + "-copy"
	for suffix := 2; m.requestIDExists(targetID); suffix++ {
		targetID = fmt.Sprintf("%s-copy-%d", selection.RequestID, suffix)
	}
	m.editor.SetSelection(app.Selection{Collection: selection.Collection, Environment: selection.Environment, RequestID: targetID, CreateOnly: true})
	m.editor.SetName("Copy of " + request.Name)
	m.editorField, m.replaceField = 7, true
	m.fieldDraft, m.fieldDraftDirty = targetID, false
	m.fieldCursor = len([]rune(targetID))
	m.duplicateFlow = true
	m.duplicateTarget = targetID
	m.message = ""
}

func (m *Model) beginDelete(id string, group bool) {
	if m.service == nil {
		return
	}
	target, err := m.service.Delete(context.Background(), m.collection, id, group, false)
	if err != nil {
		m.message = err.Error()
		return
	}
	m.prompt = &confirmation{kind: confirmDeleteTarget, targetID: id, group: group, paths: target.Paths, deleteTarget: target}
}

func (m Model) confirmationView() string {
	if m.prompt == nil {
		return ""
	}
	switch m.prompt.kind {
	case confirmDirtyNavigation:
		action := "continue"
		if m.prompt.targetID == "" {
			action = "close"
		}
		view := "Unsaved request changes\nSave and " + action + " (s)\nDiscard changes and " + action + " (d)\nCancel (c/Esc)\n"
		if m.message != "" {
			view += "Status: " + m.message + "\n"
		}
		return view
	case confirmDeleteTarget:
		lines := []string{"Confirm delete", "Affected paths:"}
		lines = append(lines, m.prompt.paths...)
		lines = append(lines, "Press y or Enter to confirm; Esc cancels.")
		return strings.Join(lines, "\n") + "\n"
	case commandPalette:
		return "Request actions\n[d] Duplicate request\n[x] Delete request\n[g] Delete group\n[Esc] Cancel\n"
	default:
		return ""
	}
}

func (m *Model) handleConfirmation(key tea.KeyMsg) tea.Cmd {
	if m.prompt == nil {
		return nil
	}
	if key.Type == tea.KeyEsc || (key.Type == tea.KeyRunes && string(key.Runes) == "c") {
		m.prompt = nil
		return nil
	}
	value := ""
	if key.Type == tea.KeyRunes {
		value = string(key.Runes)
	}
	if key.Type == tea.KeyEnter {
		value = "y"
	}
	switch m.prompt.kind {
	case confirmDirtyNavigation:
		switch strings.ToLower(value) {
		case "s":
			cmd := m.Save()
			if cmd != nil {
				m.prompt.saving = true
			}
			return cmd
		case "d":
			target := m.prompt.targetID
			m.prompt = nil
			m.discardEdits()
			if target != "" {
				m.beginEdit(target)
			}
		}
	case confirmDeleteTarget:
		if strings.ToLower(value) == "y" {
			target := m.prompt.deleteTarget
			m.prompt = nil
			m.saving = true
			return func() tea.Msg {
				_, err := m.service.Delete(context.Background(), target.Collection, target.RequestID, target.Group, true)
				return deleteFinishedMsg{target: target, err: err}
			}
		}
	case commandPalette:
		m.prompt = nil
		switch strings.ToLower(value) {
		case "d":
			m.beginDuplicate()
		case "x":
			if m.editor != nil {
				selection := m.editor.Selection()
				m.beginDelete(selection.RequestID, false)
			}
		case "g":
			if m.editor != nil {
				selection := m.editor.Selection()
				m.beginDelete(parent(selection.RequestID), true)
			}
		}
	}
	return nil
}

func (m *Model) requestIDExists(id string) bool {
	if _, ok := m.view.Tree.Requests[id]; ok {
		return true
	}
	_, ok := m.view.Tree.Invalid[id]
	return ok
}
