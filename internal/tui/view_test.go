package tui_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"apitool/internal/app"
	"apitool/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestPickerMarksActualSelectionAndEnvironmentStartsActive(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{Color: false})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	if got := m.View(); !strings.Contains(got, "│ > payments") || !strings.Contains(got, "No collection selected") {
		t.Fatalf("collection picker = %q, want persistent frame with active collection marked", got)
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
	m, _ = m.Update(tea.MouseMsg{X: 2, Y: 6, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got := m.View(); !strings.Contains(got, "Request: check") {
		t.Fatalf("mouse selection = %q, want clicked check request", got)
	}
}

func TestWindowResizeClampsPersistedSplitter(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	for range 8 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlRight})
	}
	m, _ = m.Update(tea.WindowSizeMsg{Width: 30, Height: 12})
	if divider := strings.Index(m.View(), "│"); divider >= 30 || divider < 1 {
		t.Fatalf("divider = %d after narrow resize, want within terminal width", divider)
	}
}

func TestSearchEnterOpensMatchingRequestEvenWhenItsGroupIsCollapsed(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("nested")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.View(); !strings.Contains(got, "Request: billing/nested") || !strings.Contains(got, "Focus: request") {
		t.Fatalf("search enter = %q, want matching request opened", got)
	}
}

func TestCollapsedAncestorHidesNestedGroupAndInvalidDescendant(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if got := m.View(); strings.Contains(got, "billing/nested") || strings.Contains(got, "billing/bad") || strings.Contains(got, "billing/deep") {
		t.Fatalf("collapsed tree = %q, should hide nested descendants", got)
	}
}

func TestStatusViewCategoriesRemainTextualWithoutColor(t *testing.T) {
	for _, tc := range []struct {
		code     int
		category string
	}{{200, "success"}, {302, "redirect"}, {404, "client error"}, {500, "server error"}, {0, "warning"}} {
		t.Run(tc.category, func(t *testing.T) {
			plain := tui.StatusView(tui.Status{Code: tc.code}, false)
			colored := tui.StatusView(tui.Status{Code: tc.code}, true)
			if !strings.Contains(plain, tc.category) || !strings.Contains(colored, tc.category) {
				t.Fatalf("status = %q / %q, want %q", plain, colored, tc.category)
			}
		})
	}
}

func TestStatusViewAddsStyleOnlyWhenColorIsEnabled(t *testing.T) {
	original := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(original) })

	plain := tui.StatusView(tui.Status{Code: 500}, false)
	colored := tui.StatusView(tui.Status{Code: 500}, true)
	if strings.Contains(plain, "\x1b[") || !strings.Contains(colored, "\x1b[") || colored == plain {
		t.Fatalf("plain = %q, colored = %q; want only colored status styled", plain, colored)
	}
}

func TestNilServiceHandlesResizeAndSplitterEvents(t *testing.T) {
	m := tui.New(nil, tui.Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 30, Height: 12})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlRight})
	if got := m.View(); !strings.Contains(got, "No workspace") {
		t.Fatalf("nil service view = %q", got)
	}
}

func TestCtrlCQuitsFromPickerAndSearch(t *testing.T) {
	for _, messages := range [][]tea.Msg{{tea.KeyMsg{Type: tea.KeyCtrlC}}, {tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}, tea.KeyMsg{Type: tea.KeyCtrlC}}} {
		m := tui.New(fixtureService(t), tui.Options{})
		var command tea.Cmd
		for _, message := range messages {
			m, command = m.Update(message)
		}
		if command == nil {
			t.Fatal("Ctrl+C returned no quit command")
		}
		if _, ok := command().(tea.QuitMsg); !ok {
			t.Fatalf("Ctrl+C command = %T, want QuitMsg", command())
		}
	}
}

func TestExplorerViewportFollowsKeyboardSelectionAndMouseUsesOffset(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 6})
	for range 6 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if got := m.View(); !strings.Contains(got, "> z0") {
		t.Fatalf("viewport = %q, want selected z0 visible", got)
	}
	m, _ = m.Update(tea.MouseMsg{X: 2, Y: 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got := m.View(); !strings.Contains(got, "Request: billing/nested") {
		t.Fatalf("offset mouse = %q, want visible top request selected", got)
	}
}

func TestPendingStartingEnvironmentAppliesAfterColdPickerSelection(t *testing.T) {
	m := tui.New(coldPickerService(t), tui.Options{StartingEnvironment: "prod"})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.View(); !strings.Contains(got, "Environment: prod") {
		t.Fatalf("cold picker = %q, want pending prod environment", got)
	}
}

func TestInvalidPendingStartingEnvironmentSurfacesErrorAfterColdPickerSelection(t *testing.T) {
	m := tui.New(coldPickerService(t), tui.Options{StartingEnvironment: "missing"})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.View(); !strings.Contains(got, `has no environment "missing"`) {
		t.Fatalf("cold picker = %q, want invalid environment error", got)
	}
}

func coldPickerService(t *testing.T) *app.Service {
	t.Helper()
	root := t.TempDir()
	write := func(name, data string) {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	write("payments/.api/collection.yaml", "name: Payments\n")
	write("payments/.api/environments/test.yaml", "name: test\n")
	write("payments/.api/environments/prod.yaml", "name: prod\n")
	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}
	return service
}
