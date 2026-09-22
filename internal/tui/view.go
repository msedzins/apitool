package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Status is the response metadata rendered by the response pane.
type Status struct {
	Code     int
	Duration time.Duration
}

// StatusView always contains an explicit category when terminal color is off.
func StatusView(status Status, color bool) string {
	text := fmt.Sprintf("%d %s (%s)", status.Code, statusCategory(status.Code), status.Duration)
	if !color {
		return text
	}
	return lipgloss.NewStyle().Foreground(statusColor(status.Code)).Render(text)
}

func statusCategory(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "success"
	case code >= 300 && code < 400:
		return "redirect"
	case code >= 400 && code < 500:
		return "client error"
	case code >= 500 && code < 600:
		return "server error"
	default:
		return "warning"
	}
}
func statusColor(code int) lipgloss.Color {
	if code >= 500 {
		return lipgloss.Color("9")
	}
	if code >= 400 {
		return lipgloss.Color("11")
	}
	return lipgloss.Color("10")
}

func (m Model) View() string {
	if m.mode == collectionPickerMode {
		return m.collectionPickerView()
	}
	if m.mode == environmentPickerMode {
		return m.environmentPickerView()
	}
	left := m.treeView()
	upper := fmt.Sprintf("Collection: %s\nEnvironment: %s\n%s", m.collection, m.view.Environment, m.selectedRequest())
	lower := "Response / diagnostics\n" + m.message
	return strings.Join([]string{left, upper, lower, "Focus: " + m.focusName(), "Tab panes • Ctrl+P collections • Ctrl+E environments • / search"}, "\n\n")
}
func (m Model) focusName() string { return []string{"collection", "request", "response"}[m.focus] }
func (m Model) collectionPickerView() string {
	if len(m.collections) == 0 {
		return "Collections\n(no collection selected)"
	}
	return "Collections\n> " + strings.Join(m.collections, "\n  ")
}
func (m Model) environmentPickerView() string {
	return "Environments\n> " + strings.Join(m.environmentNames(), "\n  ")
}
func (m Model) selectedRequest() string {
	if len(m.requestIDs) == 0 {
		return "No request selected"
	}
	return "Request: " + m.requestIDs[m.selected%len(m.requestIDs)]
}
func (m Model) treeView() string {
	lines := []string{"Collections"}
	if m.collection != "" {
		lines = append(lines, "> "+m.collection)
	}
	for _, group := range m.view.Tree.Groups {
		marker := "+"
		if m.expanded[group.ID] {
			marker = "-"
		}
		lines = append(lines, "  "+marker+" "+group.ID)
	}
	for _, id := range m.requestIDs {
		if !m.requestVisible(id) {
			continue
		}
		lines = append(lines, "  "+id)
	}
	invalid := make([]string, 0, len(m.view.Tree.Invalid))
	for id := range m.view.Tree.Invalid {
		invalid = append(invalid, id)
	}
	sort.Strings(invalid)
	for _, id := range invalid {
		lines = append(lines, "  ! "+id+" (warning)")
	}
	return strings.Join(lines, "\n")
}

func (m Model) requestVisible(id string) bool {
	node, ok := m.view.Tree.Requests[id]
	if !ok {
		return false
	}
	for _, group := range node.Groups {
		if !m.expanded[group.ID] {
			return false
		}
	}
	return true
}
