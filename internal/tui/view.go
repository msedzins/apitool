package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"time"
)

type Status struct {
	Code     int
	Duration time.Duration
}

func StatusView(s Status, color bool) string {
	v := fmt.Sprintf("%d %s (%s)", s.Code, statusCategory(s.Code), s.Duration)
	if !color {
		return v
	}
	return lipgloss.NewStyle().Foreground(statusColor(s.Code)).Render(v)
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
func statusColor(c int) lipgloss.Color {
	if c >= 500 {
		return "9"
	}
	if c >= 400 {
		return "11"
	}
	return "10"
}
func (m Model) View() string {
	if m.mode == collectionPickerMode {
		return m.collectionPickerView()
	}
	if m.mode == environmentPickerMode {
		return m.environmentPickerView()
	}
	left := m.treeLines()
	rightTop := []string{"Request", fmt.Sprintf("Collection: %s", m.collection), fmt.Sprintf("Environment: %s", m.view.Environment), m.selectedRequest()}
	if m.mode == searchMode {
		rightTop = append(rightTop, "Search: "+m.query+"  (Enter select, Esc cancel)")
	}
	rightBottom := []string{"Response / diagnostics", m.message, "Focus: " + m.focusName(), "Tab panes • Ctrl+P collections • Ctrl+E environments • / search"}
	return spatial(left, rightTop, rightBottom, m.explorerWidth(), m.width, m.height)
}
func (m Model) explorerWidth() int {
	if m.explorer > 0 {
		return clampExplorer(m.explorer, m.width)
	}
	w := m.width
	if w == 0 {
		w = 80
	}
	n := w / 3
	if n < 24 {
		n = 24
	}
	if n > w-20 {
		n = w / 3
	}
	return n
}
func spatial(left, top, bottom []string, leftW, width, height int) string {
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}
	rightW := width - leftW - 1
	if rightW < 1 {
		rightW = 1
	}
	topH := height / 2
	lines := make([]string, 0, height)
	for i := 0; i < height; i++ {
		l := ""
		if i < len(left) {
			l = left[i]
		}
		r := ""
		if i < topH {
			if i < len(top) {
				r = top[i]
			}
		} else if j := i - topH; j < len(bottom) {
			r = bottom[j]
		}
		lines = append(lines, pad(l, leftW)+"│"+pad(r, rightW))
	}
	return strings.Join(lines, "\n")
}
func pad(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s + strings.Repeat(" ", n-len(r))
}
func (m Model) focusName() string { return []string{"collection", "request", "response"}[m.focus] }
func (m Model) collectionPickerView() string {
	width, height := m.width, m.height
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}
	leftWidth := 28
	if leftWidth > width-12 {
		leftWidth = width / 3
	}
	if leftWidth < 1 {
		leftWidth = 1
	}
	rightWidth := width - leftWidth - 3
	if rightWidth < 1 {
		rightWidth = 1
	}
	divider := height - 5
	if divider < 1 {
		divider = 1
	}

	left, right := make([]string, height), make([]string, height)
	left[1], right[1] = " Collections / tree", " apitool"
	right[3], right[4] = "  Collection picker", "  Select a collection to open"
	for i, collection := range m.collections {
		if 3+i >= divider {
			break
		}
		left[3+i] = "  ▸ " + collection
	}
	if divider+1 < height {
		left[divider+1], right[divider+1] = " Ctrl+P collections", " No collection selected"
	}
	if divider+2 < height {
		left[divider+2] = " ↑/↓ navigate • Enter open"
	}

	picker := m.collectionPickerLines(rightWidth)
	start := (rightWidth - len([]rune(picker[0]))) / 2
	if start < 0 {
		start = 0
	}
	for i, line := range picker {
		row := 6 + i
		if row >= divider {
			break
		}
		right[row] = strings.Repeat(" ", start) + line
	}

	lines := make([]string, 0, height)
	for row := 0; row < height; row++ {
		switch row {
		case 0:
			lines = append(lines, "┌"+strings.Repeat("─", leftWidth)+"┬"+strings.Repeat("─", rightWidth)+"┐")
		case divider:
			lines = append(lines, "├"+strings.Repeat("─", leftWidth)+"┼"+strings.Repeat("─", rightWidth)+"┤")
		case height - 1:
			lines = append(lines, "└"+strings.Repeat("─", leftWidth)+"┴"+strings.Repeat("─", rightWidth)+"┘")
		default:
			lines = append(lines, "│"+pad(left[row], leftWidth)+"│"+pad(right[row], rightWidth)+"│")
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m Model) collectionPickerLines(available int) []string {
	const pickerWidth = 36
	width := pickerWidth
	if width > available {
		width = available
	}
	if width < 2 {
		return []string{strings.Repeat(" ", width)}
	}
	contentWidth := width - 2
	line := func(value string) string { return "│" + pad(value, contentWidth) + "│" }
	lines := []string{"┌" + strings.Repeat("─", contentWidth) + "┐", line(" Collections")}
	for i, path := range m.collections {
		lines = append(lines, line(""))
		marker := "   "
		if i == m.collectionIndex {
			marker = " > "
		}
		lines = append(lines, line(marker+path))
		lines = append(lines, line("  "+m.collectionDisplayName(path)))
		lines = append(lines, line("  "+path+"/.api"))
	}
	lines = append(lines, line(""), line(" ↑/↓ move   Enter open   Esc close"), "└"+strings.Repeat("─", contentWidth)+"┘")
	return lines
}

func (m Model) collectionDisplayName(path string) string {
	if m.service != nil {
		if workspace, err := m.service.Workspace(); err == nil {
			if view, ok := workspace.Collections[path]; ok && view.Collection.Name != "" {
				return view.Collection.Name
			}
		}
	}
	return path
}
func (m Model) environmentPickerView() string {
	lines := []string{"Environments"}
	for i, n := range m.environmentNames() {
		p := "  "
		if i == m.environmentIndex {
			p = "> "
		}
		lines = append(lines, p+n)
	}
	return strings.Join(lines, "\n")
}
func (m Model) selectedRequest() string {
	rows := m.visibleRows()
	if m.treeIndex >= 0 && m.treeIndex < len(rows) && rows[m.treeIndex].kind == requestRow {
		return "Request: " + rows[m.treeIndex].id
	}
	return "No request selected"
}
func (m Model) treeLines() []string {
	lines := []string{"Explorer", "> " + m.collection}
	rows := m.visibleRows()
	end := m.treeOffset + m.explorerCapacity()
	if end > len(rows) {
		end = len(rows)
	}
	for i := m.treeOffset; i < end; i++ {
		row := rows[i]
		p := "  "
		if i == m.treeIndex {
			p = "> "
		}
		switch row.kind {
		case groupRow:
			mark := "+"
			if m.expanded[row.id] {
				mark = "-"
			}
			lines = append(lines, p+mark+" "+row.id)
		case invalidRow:
			lines = append(lines, p+"! "+row.id+" (warning)")
		default:
			lines = append(lines, p+row.id)
		}
	}
	return lines
}
func (m Model) requestVisible(id string) bool {
	node, ok := m.view.Tree.Requests[id]
	if !ok {
		return false
	}
	for _, g := range node.Groups {
		if !m.expanded[g.ID] {
			return false
		}
	}
	return true
}
