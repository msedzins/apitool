package tui_test

import (
	"strings"
	"testing"

	"apitool/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPickerMarksActualSelectionAndEnvironmentStartsActive(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	if got := m.View(); !strings.Contains(got, "> payments") {
		t.Fatalf("collection picker = %q, want active collection marked", got)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	if got := m.View(); !strings.Contains(got, "> test") {
		t.Fatalf("environment picker = %q, want active environment marked", got)
	}
}

func TestPickerNavigationThenEnterDoesNotCorruptRequestSelection(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.View(); !strings.Contains(got, "Request: list") {
		t.Fatalf("picker flow = %q, want a valid request without panic", got)
	}
}

func TestSearchFiltersRequestsAndEscRestoresTree(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("second")})
	if got := m.View(); !strings.Contains(got, "Search: second") || strings.Contains(got, "\n  check\n") {
		t.Fatalf("search view = %q, want only matching request", got)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if got := m.View(); !strings.Contains(got, "check") {
		t.Fatalf("restored view = %q, want full tree", got)
	}
}

func TestViewUsesWindowDimensionsForThreeSpatialRegions(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	if got := m.View(); !strings.Contains(got, "Explorer") || !strings.Contains(got, "Request") || !strings.Contains(got, "Response / diagnostics") || !strings.Contains(got, "│") {
		t.Fatalf("layout = %q, want spatial explorer/request/response regions", got)
	}
}

func TestCtrlArrowsResizeExplorerSplitter(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	before := strings.Index(m.View(), "│")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlRight})
	if after := strings.Index(m.View(), "│"); after <= before {
		t.Fatalf("splitter = %d after Ctrl+Right, want greater than %d", after, before)
	}
}

func TestMousePressSelectsClickedTreeRow(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	m, _ = m.Update(tea.MouseMsg{X: 2, Y: 4, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got := m.View(); !strings.Contains(got, "Request: check") {
		t.Fatalf("mouse selection = %q, want clicked check request", got)
	}
}
