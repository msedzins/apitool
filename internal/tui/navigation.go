package tui

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = message.Width, message.Height
	case tea.KeyMsg:
		m.handleKey(message)
	case tea.MouseMsg:
		m.handleMouse(message)
	}
	return m, nil
}

func (m *Model) handleKey(key tea.KeyMsg) {
	if m.mode == collectionPickerMode {
		m.handleCollectionPicker(key)
		return
	}
	if m.mode == environmentPickerMode {
		m.handleEnvironmentPicker(key)
		return
	}
	if m.mode == searchMode {
		if key.Type == tea.KeyEsc || key.Type == tea.KeyEnter {
			m.mode = browseMode
			return
		}
		if key.Type == tea.KeyRunes {
			m.query += string(key.Runes)
		}
		return
	}

	switch key.Type {
	case tea.KeyTab:
		m.focus = (m.focus + 1) % 3
	case tea.KeyCtrlP:
		m.mode = collectionPickerMode
		for index, collection := range m.collections {
			if collection == m.collection {
				m.selected = index
				break
			}
		}
	case tea.KeyCtrlE:
		if m.collection != "" {
			m.mode = environmentPickerMode
		}
	case tea.KeyEsc:
		m.message = ""
	case tea.KeyEnter:
		m.openSelected()
	case tea.KeyUp:
		m.move(-1)
	case tea.KeyDown:
		m.move(1)
	case tea.KeyLeft:
		m.collapseSelected()
	case tea.KeyRight:
		m.expandSelected()
	case tea.KeyRunes:
		m.handleRune(string(key.Runes))
	}
}

func (m *Model) handleRune(value string) {
	switch value {
	case "/":
		m.mode = searchMode
		m.query = ""
	case "j":
		if m.vim {
			m.move(1)
		}
	case "k":
		if m.vim {
			m.move(-1)
		}
	case "h":
		if m.vim {
			m.collapseSelected()
		}
	case "l":
		if m.vim {
			m.expandSelected()
		}
	}
}

func (m *Model) handleCollectionPicker(key tea.KeyMsg) {
	switch key.Type {
	case tea.KeyEsc:
		m.mode = browseMode
	case tea.KeyEnter:
		if len(m.collections) > 0 {
			m.openCollection(m.collections[m.selected%len(m.collections)])
		}
	case tea.KeyUp:
		m.moveCollection(-1)
	case tea.KeyDown:
		m.moveCollection(1)
	}
	if key.Type == tea.KeyRunes && m.vim {
		if string(key.Runes) == "j" {
			m.moveCollection(1)
		}
		if string(key.Runes) == "k" {
			m.moveCollection(-1)
		}
	}
}

func (m *Model) handleEnvironmentPicker(key tea.KeyMsg) {
	envs := m.environmentNames()
	switch key.Type {
	case tea.KeyEsc:
		m.mode = browseMode
	case tea.KeyEnter:
		if len(envs) > 0 {
			m.selectEnvironment(envs[m.selected%len(envs)])
		}
	case tea.KeyUp:
		m.moveEnvironment(-1, len(envs))
	case tea.KeyDown:
		m.moveEnvironment(1, len(envs))
	}
}

func (m *Model) environmentNames() []string {
	result := make([]string, 0, len(m.view.Environments))
	for name := range m.view.Environments {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}
func (m *Model) move(delta int) {
	if len(m.requestIDs) != 0 {
		m.selected = (m.selected + delta + len(m.requestIDs)) % len(m.requestIDs)
	}
}
func (m *Model) moveCollection(delta int) {
	if len(m.collections) != 0 {
		m.selected = (m.selected + delta + len(m.collections)) % len(m.collections)
	}
}
func (m *Model) moveEnvironment(delta, count int) {
	if count != 0 {
		m.selected = (m.selected + delta + count) % count
	}
}
func (m *Model) openSelected() {
	if m.focus == collectionPane && len(m.requestIDs) > 0 {
		m.focus = requestPane
		m.message = "Request: " + m.requestIDs[m.selected]
	}
}
func (m *Model) collapseSelected() {
	if len(m.view.Tree.Groups) > 0 {
		m.expanded[m.view.Tree.Groups[0].ID] = false
	}
}
func (m *Model) expandSelected() {
	if len(m.view.Tree.Groups) > 0 {
		m.expanded[m.view.Tree.Groups[0].ID] = true
	}
}
func (m *Model) handleMouse(mouse tea.MouseMsg) {
	if mouse.X < max(24, m.width/3) {
		m.focus = collectionPane
		m.openSelected()
	} else if mouse.Y < m.height/2 {
		m.focus = requestPane
	} else {
		m.focus = responsePane
	}
}
