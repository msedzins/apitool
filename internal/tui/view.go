package tui

import (
	"apitool/internal/collection"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"sort"
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
	if m.approvedShell {
		return m.collectionView()
	}
	left := m.treeLines()
	rightTop := []string{"Request", fmt.Sprintf("Collection: %s", m.collection), fmt.Sprintf("Environment: %s", m.view.Environment), m.selectedRequest()}
	if m.mode == searchMode {
		rightTop = append(rightTop, "Search: "+m.query+"  (Enter select, Esc cancel)")
	}
	rightBottom := []string{"Response / diagnostics", m.message, "Focus: " + m.focusName(), "Tab panes • Ctrl+P collections • Ctrl+E environments • / search"}
	return spatial(left, rightTop, rightBottom, m.explorerWidth(), m.width, m.height)
}

func (m Model) collectionView() string {
	width, height := m.width, m.height
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}
	leftWidth := 28
	if m.explorer > 0 {
		leftWidth = m.explorerWidth()
	}
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

	left, right := make([]string, height), make([]string, height)
	if height > 1 {
		left[1] = " Collections / tree"
		if request, ok := m.activeRequest(); ok {
			right[1] = fmt.Sprintf(" %s | %-48s[ Send ]", request.Request.Method, request.Request.Request.URL)
		}
	}
	if height > 3 {
		right[3] = " Params | Headers | Auth | Body | Settings"
	}
	for row, line := range m.collectionTreeLines() {
		row += 3
		if row >= height-4 {
			break
		}
		left[row] = " " + line
	}
	if height > 5 {
		switch {
		case m.mode == searchMode:
			right[5] = " Search: " + m.query + "  (Enter select, Esc cancel)"
		case m.selectedRequest() != "No request selected":
			right[5] = " " + m.selectedRequest()
		default:
			request, ok := m.previewRequest()
			if !ok {
				break
			}
			right[5] = " Request: " + request.ID
		}
	}
	responseDivider := 8
	if responseDivider < height {
		right[responseDivider+1] = " Response / Diagnostics / Request Log"
	}
	if responseDivider+2 < height {
		right[responseDivider+2] = " Select Send to execute this request."
	}
	statusDivider := height - 4
	if statusDivider+1 < height {
		left[statusDivider+1] = " Collection: " + m.collection
		right[statusDivider+1] = " Environment: " + m.view.Environment
	}
	if statusDivider+2 < height {
		left[statusDivider+2] = " Ctrl+P collections"
		if m.focus != collectionPane {
			right[statusDivider+2] = " Focus: " + m.focusName()
		}
	}

	lines := make([]string, 0, height)
	for row := 0; row < height; row++ {
		switch row {
		case 0:
			lines = append(lines, "┌"+strings.Repeat("─", leftWidth)+"┬"+strings.Repeat("─", rightWidth)+"┐")
		case height - 1:
			lines = append(lines, "└"+strings.Repeat("─", leftWidth)+"┴"+strings.Repeat("─", rightWidth)+"┘")
		case statusDivider:
			lines = append(lines, "├"+strings.Repeat("─", leftWidth)+"┼"+strings.Repeat("─", rightWidth)+"┤")
		case 2, responseDivider:
			lines = append(lines, "│"+pad(left[row], leftWidth)+"├"+strings.Repeat("─", rightWidth)+"┤")
		default:
			lines = append(lines, "│"+pad(left[row], leftWidth)+"│"+pad(right[row], rightWidth)+"│")
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m Model) collectionTreeLines() []string {
	lines := []string{"▾ " + m.collection}
	if m.mode == searchMode {
		for i, row := range m.visibleRows() {
			if row.kind != requestRow {
				continue
			}
			prefix := "  "
			if i == m.treeIndex {
				prefix = "> "
			}
			lines = append(lines, prefix+m.view.Tree.Requests[row.id].Request.Method+" "+row.id)
		}
		return lines
	}
	groups := append([]collection.GroupNode(nil), m.view.Tree.Groups...)
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].ID == m.collection {
			return true
		}
		if groups[j].ID == m.collection {
			return false
		}
		return groups[i].ID < groups[j].ID
	})
	for _, group := range groups {
		if !m.ancestorsExpanded(group.ID) {
			continue
		}
		indent := strings.Repeat("  ", strings.Count(group.ID, "/")+1)
		marker := "▸"
		if m.expanded[group.ID] && m.groupHasContents(group.ID) {
			marker = "▾"
		}
		lines = append(lines, indent+marker+" "+lastSegment(group.ID))
		if !m.expanded[group.ID] {
			continue
		}
		for _, request := range m.groupRequests(group.ID) {
			prefix := "  "
			if m.isSelectedRequest(request.ID) {
				prefix = "> "
			}
			lines = append(lines, indent+prefix+request.Request.Method+" "+lastSegment(request.ID))
		}
	}
	for _, id := range m.view.Tree.RequestIDs {
		request := m.view.Tree.Requests[id]
		if len(request.Groups) != 0 || !m.requestVisible(id) {
			continue
		}
		prefix := "  "
		if m.isSelectedRequest(id) {
			prefix = "> "
		}
		lines = append(lines, prefix+request.Request.Method+" "+id)
	}
	for _, collection := range m.collections {
		if collection != m.collection {
			lines = append(lines, "▸ "+collection)
		}
	}
	return lines
}

func (m Model) isSelectedRequest(id string) bool {
	rows := m.visibleRows()
	return m.treeIndex >= 0 && m.treeIndex < len(rows) && rows[m.treeIndex].kind == requestRow && rows[m.treeIndex].id == id
}

func (m Model) groupRequests(groupID string) []collection.RequestNode {
	var requests []collection.RequestNode
	for _, id := range m.view.Tree.RequestIDs {
		request := m.view.Tree.Requests[id]
		if len(request.Groups) != 0 && request.Groups[len(request.Groups)-1].ID == groupID {
			requests = append(requests, request)
		}
	}
	sort.Slice(requests, func(i, j int) bool {
		if requests[i].Request.Method == requests[j].Request.Method {
			return requests[i].ID < requests[j].ID
		}
		return requests[i].Request.Method < requests[j].Request.Method
	})
	return requests
}

func (m Model) groupHasContents(groupID string) bool {
	if len(m.groupRequests(groupID)) != 0 {
		return true
	}
	for _, group := range m.view.Tree.Groups {
		if parent(group.ID) == groupID {
			return true
		}
	}
	return false
}

func (m Model) previewRequest() (collection.RequestNode, bool) {
	var requests []collection.RequestNode
	for _, id := range m.view.Tree.RequestIDs {
		request := m.view.Tree.Requests[id]
		if len(request.Diagnostics) != 0 {
			continue
		}
		if _, invalid := m.view.Tree.Invalid[id]; invalid {
			continue
		}
		requests = append(requests, request)
	}
	sort.Slice(requests, func(i, j int) bool {
		if requests[i].Request.Method == requests[j].Request.Method {
			return requests[i].ID < requests[j].ID
		}
		return requests[i].Request.Method < requests[j].Request.Method
	})
	if len(requests) == 0 {
		return collection.RequestNode{}, false
	}
	return requests[0], true
}

func (m Model) activeRequest() (collection.RequestNode, bool) {
	rows := m.visibleRows()
	if m.treeIndex >= 0 && m.treeIndex < len(rows) && rows[m.treeIndex].kind == requestRow {
		return m.view.Tree.Requests[rows[m.treeIndex].id], true
	}
	return m.previewRequest()
}

func lastSegment(value string) string {
	if index := strings.LastIndex(value, "/"); index >= 0 {
		return value[index+1:]
	}
	return value
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
	setRow := func(row int, leftText, rightText string) {
		if row >= 0 && row < height {
			left[row], right[row] = leftText, rightText
		}
	}
	setRow(1, " Collections / tree", " apitool")
	setRow(3, "", "  Collection picker")
	setRow(4, "", "  Select a collection to open")
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
