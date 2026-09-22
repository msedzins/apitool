// Package tui provides the keyboard-first interactive apitool interface.
package tui

import (
	"context"
	"sort"

	"apitool/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// Options configures the initial TUI selection. VimMode enables j/k/h/l aliases.
type Options struct {
	StartingCollection  string
	StartingEnvironment string
	ConfirmDangerous    bool
	VimMode             bool
	Color               bool
}

type pane int

const (
	collectionPane pane = iota
	requestPane
	responsePane
)

type mode int

const (
	browseMode mode = iota
	collectionPickerMode
	environmentPickerMode
	searchMode
)

// Model is the application shell. It deliberately exposes only Bubble Tea's
// Model interface so request editing and execution can be added independently.
type Model struct {
	service *app.Service
	view    app.CollectionView

	collections                                  []string
	collection                                   string
	collectionIndex, environmentIndex, treeIndex int
	focus                                        pane
	mode                                         mode
	vim                                          bool
	color                                        bool
	query                                        string
	expanded                                     map[string]bool
	width                                        int
	height                                       int
	explorer                                     int
	message                                      string
}

// New creates a shell around a workspace already opened by service.
func New(service *app.Service, options Options) tea.Model {
	m := Model{service: service, vim: options.VimMode, color: options.Color, expanded: map[string]bool{}}
	if service == nil {
		m.message = "No workspace is open"
		return m
	}
	// The application boundary restores per-collection environments during its
	// OpenWorkspace call; OpenCollection is the only workspace data we consume.
	// Start with the requested collection when available, otherwise show picker.
	m.mode = collectionPickerMode
	if workspace, err := service.Workspace(); err == nil {
		for path := range workspace.Collections {
			m.collections = append(m.collections, path)
		}
		sort.Strings(m.collections)
		if options.StartingCollection == "" && workspace.ActiveCollection != "" {
			options.StartingCollection = workspace.ActiveCollection
		}
	}
	if preferences, err := service.UIPreferences(); err == nil {
		if options.StartingCollection == "" && preferences.ActiveCollection != "" {
			options.StartingCollection = preferences.ActiveCollection
		}
		m.explorer = preferences.ExplorerWidth
	}
	if options.StartingCollection != "" {
		m.openCollection(options.StartingCollection)
		if options.StartingEnvironment != "" && m.collection != "" {
			m.selectEnvironment(options.StartingEnvironment)
		}
	}
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m *Model) openCollection(path string) {
	view, err := m.service.OpenCollection(context.Background(), path)
	if err != nil {
		m.message = err.Error()
		return
	}
	m.collection, m.view, m.treeIndex, m.focus, m.mode = path, view, 0, collectionPane, browseMode
	for _, group := range view.Tree.Groups {
		m.expanded[group.ID] = true
	}
	_ = m.service.SaveUIPreferences(path, m.explorer)
}

func (m *Model) selectEnvironment(environment string) {
	view, err := m.service.SelectEnvironment(context.Background(), m.collection, environment)
	if err != nil {
		m.message = err.Error()
		return
	}
	m.view, m.mode, m.message = view, browseMode, ""
}
