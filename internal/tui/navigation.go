package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"apitool/internal/app"
	"apitool/internal/collection"

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
	case requestSavedMsg:
		return m, m.handleRequestSaved(x)
	case deleteFinishedMsg:
		m.handleDeleteFinished(x)
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = x.Width, x.Height
		if m.explorer > 0 {
			m.explorer = clampExplorer(m.explorer, m.width)
		}
		m.savePreferences()
	case tea.KeyMsg:
		if x.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if m.help {
			if x.Type == tea.KeyEsc || (x.Type == tea.KeyRunes && string(x.Runes) == "?") {
				m.help = false
			}
			return m, nil
		}
		if m.prompt != nil {
			return m, m.handleConfirmation(x)
		}
		if m.mode == requestEditMode {
			return m, m.handleEditorKey(x)
		}
		if x.Type == tea.KeyRunes && string(x.Runes) == "?" {
			m.help = true
			return m, nil
		}
		m.handleKey(x)
	case tea.MouseMsg:
		if !m.help {
			m.handleMouse(x)
		}
	}
	m.syncViewport()
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
	width := m.explorer
	if width == 0 {
		width = 28
	}
	m.explorer = clampExplorer(width+delta, m.width)
	m.savePreferences()
}
func (m *Model) savePreferences() {
	if m.service != nil {
		_ = m.service.SaveUIPreferences(m.collection, m.explorer)
	}
}
func (m *Model) handleRune(s string) {
	if s == "e" {
		rows := m.visibleRows()
		if m.treeIndex >= 0 && m.treeIndex < len(rows) && rows[m.treeIndex].kind == requestRow {
			m.beginEdit(rows[m.treeIndex].id)
		}
		return
	}
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
		if m.pendingEnvironment != "" {
			m.mode = collectionPickerMode
		} else {
			m.mode = browseMode
		}
	case tea.KeyEnter:
		if m.environmentIndex >= 0 && m.environmentIndex < len(envs) {
			m.selectEnvironment(envs[m.environmentIndex])
		}
	case tea.KeyUp:
		m.environmentIndex = wrap(m.environmentIndex-1, len(envs))
	case tea.KeyDown:
		m.environmentIndex = wrap(m.environmentIndex+1, len(envs))
	}
}
func (m *Model) moveTree(d int) {
	m.treeIndex = wrap(m.treeIndex+d, len(m.visibleRows()))
	m.syncViewport()
}
func (m *Model) clampTreeIndex() {
	n := len(m.visibleRows())
	if n == 0 {
		m.treeIndex = 0
	} else if m.treeIndex >= n {
		m.treeIndex = n - 1
	}
	m.syncViewport()
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
	} else if row.kind == invalidRow {
		if invalid, ok := m.view.Tree.Invalid[row.id]; ok {
			details := make([]string, 0, len(invalid.Diagnostics))
			for _, diagnostic := range invalid.Diagnostics {
				if diagnostic.Path != "" && diagnostic.Message != "" {
					details = append(details, diagnostic.Path+": "+diagnostic.Message)
				} else if diagnostic.Path != "" {
					details = append(details, diagnostic.Path)
				} else if diagnostic.Message != "" {
					details = append(details, diagnostic.Message)
				}
			}
			m.message = strings.Join(details, " | ")
		}
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
		row := -1
		if m.height > 0 && m.height < 12 {
			row = x.Y - 2 + m.treeOffset
		} else if x.Y >= 4 && x.Y < m.height-4 {
			row = x.Y - 4 + m.treeOffset
		}
		rows := m.visibleRows()
		if row >= 0 && row < len(rows) {
			m.treeIndex = row
			m.openSelected()
		}
	} else if m.height > 0 && m.height < 12 {
		if x.Y < m.height/2 {
			m.focus = requestPane
		} else {
			m.focus = responsePane
		}
	} else if x.Y >= 9 {
		m.focus = responsePane
	} else {
		m.focus = requestPane
	}
}
func (m *Model) syncViewport() {
	rows := len(m.visibleRows())
	capacity := m.explorerCapacity()
	if m.treeIndex < m.treeOffset {
		m.treeOffset = m.treeIndex
	}
	if m.treeIndex >= m.treeOffset+capacity {
		m.treeOffset = m.treeIndex - capacity + 1
	}
	maxOffset := rows - capacity
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.treeOffset < 0 {
		m.treeOffset = 0
	}
	if m.treeOffset > maxOffset {
		m.treeOffset = maxOffset
	}
}
func (m Model) explorerCapacity() int {
	height := m.height
	if height <= 0 {
		height = 24
	}
	capacity := height - 8
	if capacity < 1 {
		return 1
	}
	return capacity
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
	var rows []treeRow
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
		rows = append(rows, treeRow{group.ID, groupRow})
		if m.expanded[group.ID] {
			for _, request := range m.groupRequests(group.ID) {
				rows = append(rows, treeRow{request.ID, requestRow})
			}
		}
	}
	for _, id := range m.view.Tree.RequestIDs {
		request := m.view.Tree.Requests[id]
		if len(request.Groups) == 0 {
			rows = append(rows, treeRow{id, requestRow})
		}
	}
	invalidIDs := make([]string, 0, len(m.view.Tree.Invalid))
	for id := range m.view.Tree.Invalid {
		invalidIDs = append(invalidIDs, id)
	}
	sort.Strings(invalidIDs)
	for _, id := range invalidIDs {
		if m.ancestorsExpanded(id) {
			rows = append(rows, treeRow{id, invalidRow})
		}
	}
	return rows
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

func (m *Model) beginEdit(id string) {
	node, ok := m.view.Tree.Requests[id]
	if !ok {
		m.message = fmt.Sprintf("request %q not found", id)
		return
	}
	m.selectedID = id
	m.editor = newRequestEditor(m.service, app.Selection{Collection: m.collection, Environment: m.view.Environment, RequestID: id}, node.Request)
	m.mode, m.editorField, m.replaceField = requestEditMode, 0, true
	m.fieldDraft = m.editor.FieldText(0)
	m.fieldDraftDirty = false
	m.duplicateFlow, m.duplicateTarget = false, ""
	m.message = ""
}

func (m *Model) requestNavigation(direction int) {
	ids := m.view.Tree.RequestIDs
	if len(ids) == 0 || m.editor == nil {
		return
	}
	index := 0
	for i, id := range ids {
		if id == m.editor.Selection().RequestID {
			index = i
			break
		}
	}
	target := wrap(index+direction, len(ids))
	if m.editor.Dirty() {
		m.prompt = &confirmation{kind: confirmDirtyNavigation, targetID: ids[target]}
		return
	}
	m.beginEdit(ids[target])
}

func (m *Model) discardEdits() {
	m.editor, m.prompt = nil, nil
	m.duplicateFlow, m.duplicateTarget = false, ""
	m.mode = browseMode
	m.message = ""
}

func (m *Model) Save() tea.Cmd {
	if m.editor == nil || m.saving {
		return nil
	}
	m.commitEditorField()
	if m.duplicateFlow && (m.duplicateTarget == "" || m.requestIDExists(m.duplicateTarget)) {
		m.message = "duplicate destination already exists or is empty"
		return nil
	}
	cmd := m.editor.Save()
	m.message = m.editor.Validation()
	if cmd != nil {
		m.saving = true
	}
	return cmd
}

func (m *Model) commitEditorField() {
	if m.editor == nil || !m.fieldDraftDirty {
		return
	}
	if m.duplicateFlow && m.editorField == 7 {
		m.duplicateTarget = m.fieldDraft
		selection := m.editor.Selection()
		selection.RequestID = m.duplicateTarget
		selection.CreateOnly = true
		m.editor.SetSelection(selection)
	} else {
		m.editor.SetFieldText(m.editorField, m.fieldDraft)
	}
	m.fieldDraftDirty = false
}

func (m *Model) handleEditorKey(key tea.KeyMsg) tea.Cmd {
	if m.editor == nil {
		m.mode = browseMode
		return nil
	}
	if m.saving {
		return nil
	}
	switch key.Type {
	case tea.KeyEsc:
		if m.editor.Dirty() || m.fieldDraftDirty {
			m.prompt = &confirmation{kind: confirmDirtyNavigation, targetID: m.editor.Selection().RequestID}
			return nil
		}
		m.discardEdits()
	case tea.KeyTab, tea.KeyEnter:
		m.commitEditorField()
		fieldCount := 7
		if m.duplicateFlow {
			fieldCount = 8
		}
		m.editorField = (m.editorField + 1) % fieldCount
		m.fieldDraft = m.editorFieldText()
		m.fieldDraftDirty = false
		m.replaceField = true
	case tea.KeyUp, tea.KeyDown:
		m.commitEditorField()
		if key.Type == tea.KeyUp {
			m.requestNavigation(-1)
		} else {
			m.requestNavigation(1)
		}
	case tea.KeyCtrlS:
		return m.Save()
	case tea.KeyCtrlZ:
		m.commitEditorField()
		m.editor.Undo()
		m.fieldDraft = m.editor.FieldText(m.editorField)
	case tea.KeyCtrlY:
		m.commitEditorField()
		m.editor.Redo()
		m.fieldDraft = m.editor.FieldText(m.editorField)
	case tea.KeyCtrlB:
		m.commitEditorField()
		mode := BodyModeRaw
		if m.editor.mode == BodyModeRaw {
			mode = BodyModeJSON
		}
		m.editor.SwitchBodyMode(mode)
		m.fieldDraft = m.editor.FieldText(m.editorField)
		m.message = m.editor.Validation()
	case tea.KeyCtrlP:
		m.commitEditorField()
		m.prompt = &confirmation{kind: commandPalette}
	case tea.KeyCtrlU:
		m.fieldDraft = ""
		m.fieldDraftDirty = true
		m.replaceField = false
	case tea.KeyBackspace:
		value := []rune(m.editorFieldText())
		if len(value) > 0 {
			m.fieldDraft = string(value[:len(value)-1])
			m.fieldDraftDirty = true
		}
		m.replaceField = false
	case tea.KeyRunes:
		text := string(key.Runes)
		value := m.editorFieldText()
		if m.replaceField {
			value = ""
		}
		m.fieldDraft = value + text
		m.fieldDraftDirty = true
		m.replaceField = false
	}
	return nil
}

func (m Model) currentRequestIndex() int {
	if m.editor == nil {
		return m.treeIndex
	}
	for i, id := range m.view.Tree.RequestIDs {
		if id == m.editor.Selection().RequestID {
			return i
		}
	}
	return 0
}

func (m *Model) handleRequestSaved(msg requestSavedMsg) tea.Cmd {
	m.saving = false
	if msg.err != nil {
		m.message = msg.err.Error()
		return nil
	}
	view, err := m.service.OpenCollection(context.Background(), msg.selection.Collection)
	if err != nil {
		m.message = err.Error()
		return nil
	}
	m.view = view
	m.selectedID = msg.selection.RequestID
	wasDuplicate := m.duplicateFlow
	if m.editor == nil {
		m.beginEdit(msg.selection.RequestID)
	} else {
		m.editor.Load(msg.request)
	}
	savedSelection := msg.selection
	savedSelection.CreateOnly = false
	m.editor.SetSelection(savedSelection)
	if wasDuplicate {
		m.editorField = 0
	}
	m.fieldDraft = m.editor.FieldText(m.editorField)
	m.fieldDraftDirty = false
	m.replaceField = true
	m.message = "Saved " + msg.selection.RequestID
	if m.duplicateFlow {
		m.duplicateFlow, m.duplicateTarget = false, ""
	}
	if m.prompt != nil && m.prompt.kind == confirmDirtyNavigation && m.prompt.saving {
		target := m.prompt.targetID
		m.prompt = nil
		m.beginEdit(target)
	}
	return nil
}

func (m *Model) handleDeleteFinished(msg deleteFinishedMsg) {
	m.saving = false
	if msg.err != nil {
		m.message = msg.err.Error()
		return
	}
	view, err := m.service.OpenCollection(context.Background(), msg.target.Collection)
	if err != nil {
		m.message = err.Error()
		return
	}
	m.view = view
	m.editor = nil
	m.mode = browseMode
	m.selectedID = ""
	m.message = "Deleted " + msg.target.RequestID
	m.clampTreeIndex()
}

func (m *Model) editorFieldText() string { return m.fieldDraft }

func (m *Model) setEditorFieldText(value string) {
	m.fieldDraft = value
	m.fieldDraftDirty = true
}
