package tui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type rowKind int

const (
	groupRow rowKind = iota
	requestRow
	invalidRow
)

type treeRow struct {
	id   string
	kind rowKind
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch x := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = x.Width, x.Height
		m.explorer = clampExplorer(m.explorer, m.width)
		m.savePreferences()
	case tea.KeyMsg:
		m.handleKey(x)
	case tea.MouseMsg:
		m.handleMouse(x)
	}
	return m, nil
}
func (m *Model) handleKey(k tea.KeyMsg) {
	if m.mode == collectionPickerMode {
		m.handleCollectionPicker(k)
		return
	}
	if m.mode == environmentPickerMode {
		m.handleEnvironmentPicker(k)
		return
	}
	if m.mode == searchMode {
		m.handleSearch(k)
		return
	}
	switch k.Type {
	case tea.KeyTab:
		m.focus = (m.focus + 1) % 3
	case tea.KeyCtrlP:
		m.mode = collectionPickerMode
		m.collectionIndex = indexOf(m.collections, m.collection)
	case tea.KeyCtrlE:
		if m.collection != "" {
			m.mode = environmentPickerMode
			m.environmentIndex = indexOf(m.environmentNames(), m.view.Environment)
		}
	case tea.KeyCtrlLeft:
		m.resizeExplorer(-2)
	case tea.KeyCtrlRight:
		m.resizeExplorer(2)
	case tea.KeyEsc:
		m.message = ""
	case tea.KeyEnter:
		m.openSelected()
	case tea.KeyUp:
		m.moveTree(-1)
	case tea.KeyDown:
		m.moveTree(1)
	case tea.KeyLeft:
		m.collapseSelected()
	case tea.KeyRight:
		m.expandSelected()
	case tea.KeyRunes:
		m.handleRune(string(k.Runes))
	}
}

func (m *Model) resizeExplorer(delta int) {
	m.explorer = clampExplorer(m.explorer+delta, m.width)
	m.savePreferences()
}
func (m *Model) savePreferences() {
	if m.service != nil {
		_ = m.service.SaveUIPreferences(m.collection, m.explorer)
	}
}
func (m *Model) handleRune(s string) {
	if s == "/" {
		m.mode, m.query, m.treeIndex = searchMode, "", 0
		return
	}
	if !m.vim {
		return
	}
	switch s {
	case "j":
		m.moveTree(1)
	case "k":
		m.moveTree(-1)
	case "h":
		m.collapseSelected()
	case "l":
		m.expandSelected()
	}
}
func (m *Model) handleSearch(k tea.KeyMsg) {
	switch k.Type {
	case tea.KeyEsc:
		m.mode, m.query, m.treeIndex = browseMode, "", 0
	case tea.KeyEnter:
		m.openSearchSelection()
	case tea.KeyBackspace:
		if n := len(m.query); n > 0 {
			m.query = m.query[:n-1]
			m.clampTreeIndex()
		}
	case tea.KeyUp:
		m.moveTree(-1)
	case tea.KeyDown:
		m.moveTree(1)
	case tea.KeyRunes:
		m.query += string(k.Runes)
		m.clampTreeIndex()
	}
}
func (m *Model) handleCollectionPicker(k tea.KeyMsg) {
	switch k.Type {
	case tea.KeyEsc:
		m.mode = browseMode
	case tea.KeyEnter:
		if len(m.collections) > 0 {
			m.openCollection(m.collections[m.collectionIndex])
		}
	case tea.KeyUp:
		m.collectionIndex = wrap(m.collectionIndex-1, len(m.collections))
	case tea.KeyDown:
		m.collectionIndex = wrap(m.collectionIndex+1, len(m.collections))
	}
}
func (m *Model) handleEnvironmentPicker(k tea.KeyMsg) {
	envs := m.environmentNames()
	switch k.Type {
	case tea.KeyEsc:
		m.mode = browseMode
	case tea.KeyEnter:
		if len(envs) > 0 {
			m.selectEnvironment(envs[m.environmentIndex])
		}
	case tea.KeyUp:
		m.environmentIndex = wrap(m.environmentIndex-1, len(envs))
	case tea.KeyDown:
		m.environmentIndex = wrap(m.environmentIndex+1, len(envs))
	}
}
func (m *Model) moveTree(d int) { m.treeIndex = wrap(m.treeIndex+d, len(m.visibleRows())) }
func (m *Model) clampTreeIndex() {
	n := len(m.visibleRows())
	if n == 0 {
		m.treeIndex = 0
	} else if m.treeIndex >= n {
		m.treeIndex = n - 1
	}
}
func (m *Model) openSelected() {
	rows := m.visibleRows()
	if m.focus != collectionPane || m.treeIndex < 0 || m.treeIndex >= len(rows) {
		return
	}
	row := rows[m.treeIndex]
	if row.kind == groupRow {
		m.expanded[row.id] = !m.expanded[row.id]
		m.clampTreeIndex()
	} else if row.kind == requestRow {
		m.focus = requestPane
		m.message = "Request: " + row.id
	}
}
func (m *Model) collapseSelected() {
	rows := m.visibleRows()
	if m.treeIndex >= 0 && m.treeIndex < len(rows) && rows[m.treeIndex].kind == groupRow {
		m.expanded[rows[m.treeIndex].id] = false
		m.clampTreeIndex()
	}
}
func (m *Model) expandSelected() {
	rows := m.visibleRows()
	if m.treeIndex >= 0 && m.treeIndex < len(rows) && rows[m.treeIndex].kind == groupRow {
		m.expanded[rows[m.treeIndex].id] = true
	}
}
func (m *Model) handleMouse(x tea.MouseMsg) {
	if x.Action != tea.MouseActionPress || x.Button != tea.MouseButtonLeft {
		return
	}
	if x.X < m.explorerWidth() {
		m.focus = collectionPane
		row := x.Y - 2
		rows := m.visibleRows()
		if row >= 0 && row < len(rows) {
			m.treeIndex = row
			m.openSelected()
		}
	} else if x.Y < m.height/2 {
		m.focus = requestPane
	} else {
		m.focus = responsePane
	}
}
func (m *Model) openSearchSelection() {
	rows := m.searchRows()
	if m.treeIndex < 0 || m.treeIndex >= len(rows) {
		m.mode = browseMode
		return
	}
	id := rows[m.treeIndex].id
	for _, group := range m.view.Tree.Requests[id].Groups {
		m.expanded[group.ID] = true
	}
	m.mode = browseMode
	m.treeIndex = indexRow(m.visibleRows(), id)
	m.focus, m.message = requestPane, "Request: "+id
}
func (m Model) environmentNames() []string {
	r := make([]string, 0, len(m.view.Environments))
	for n := range m.view.Environments {
		r = append(r, n)
	}
	sort.Strings(r)
	return r
}
func (m Model) visibleRows() []treeRow {
	if m.mode == searchMode {
		return m.searchRows()
	}
	var r []treeRow
	for _, g := range m.view.Tree.Groups {
		if !m.ancestorsExpanded(g.ID) {
			continue
		}
		r = append(r, treeRow{g.ID, groupRow})
	}
	for _, id := range m.view.Tree.RequestIDs {
		if !m.requestVisible(id) {
			continue
		}
		r = append(r, treeRow{id, requestRow})
	}
	bad := make([]string, 0, len(m.view.Tree.Invalid))
	for id := range m.view.Tree.Invalid {
		bad = append(bad, id)
	}
	sort.Strings(bad)
	for _, id := range bad {
		if !m.ancestorsExpanded(id) {
			continue
		}
		r = append(r, treeRow{id, invalidRow})
	}
	return r
}
func (m Model) searchRows() []treeRow {
	var r []treeRow
	for _, id := range m.view.Tree.RequestIDs {
		if strings.Contains(strings.ToLower(id), strings.ToLower(m.query)) {
			r = append(r, treeRow{id, requestRow})
		}
	}
	return r
}
func (m Model) ancestorsExpanded(id string) bool {
	for p := parent(id); p != ""; p = parent(p) {
		if !m.expanded[p] {
			return false
		}
	}
	return true
}
func indexRow(rows []treeRow, id string) int {
	for i, row := range rows {
		if row.id == id {
			return i
		}
	}
	return 0
}
func clampExplorer(value, width int) int {
	if width <= 0 {
		if value < 24 {
			return 24
		}
		return value
	}
	max := width - 10
	if max < 1 {
		max = 1
	}
	if value == 0 {
		value = width / 3
	}
	if value < 24 && width >= 34 {
		value = 24
	}
	if value > max {
		value = max
	}
	return value
}
func parent(id string) string {
	if i := strings.LastIndex(id, "/"); i >= 0 {
		return id[:i]
	}
	return ""
}
func wrap(i, n int) int {
	if n == 0 {
		return 0
	}
	return (i%n + n) % n
}
func indexOf(xs []string, w string) int {
	for i, x := range xs {
		if x == w {
			return i
		}
	}
	return 0
}
