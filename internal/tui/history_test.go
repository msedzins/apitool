package tui_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"apitool/internal/model"
	"apitool/internal/runtime"
	"apitool/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHistorySearchReopensReferencedRequest(t *testing.T) {
	m := historyScreenWithEntry(t, "billing/nested")
	m = searchAndEnter(m, "nested")
	if got := m.View(); !containsAll(got, "GET", "billing/nested") {
		t.Fatalf("reopened history request = %q, want current GET definition", got)
	}
}

func TestHistoryShowsEmptyStateAndFiltersEntries(t *testing.T) {
	service := fixtureService(t)
	workspace, err := service.Workspace()
	if err != nil {
		t.Fatal(err)
	}
	store, err := runtime.Open(workspace.Root)
	if err != nil {
		t.Fatal(err)
	}
	model := tui.New(service, tui.Options{StartingCollection: "payments"})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if got := model.View(); !strings.Contains(got, "No request history") {
		t.Fatalf("empty history = %q, want empty state", got)
	}
	for _, id := range []string{"check", "second"} {
		if err := store.AppendHistory(runtime.Key{CollectionPath: "payments", Environment: "test", RequestID: id}, "GET", modelResponse(200), nil); err != nil {
			t.Fatal(err)
		}
	}
	model = searchAndEnter(model.(tui.Model), "check")
	if got := model.View(); !strings.Contains(got, "Request: check") {
		t.Fatalf("filtered history reopen = %q, want check request", got)
	}
}

func TestSplitResizingRespectsMinimumsAndPersists(t *testing.T) {
	service := fixtureService(t)
	m := tui.New(service, tui.Options{StartingCollection: "payments"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	before := responseSplitRow(m.View())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown, Alt: true})
	if after := responseSplitRow(m.View()); after <= before {
		t.Fatalf("request/response split row = %d after Ctrl+Up, want greater than %d", after, before)
	}
	for range 40 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown, Alt: true})
	}
	upper := responseSplitRow(m.View())
	for range 60 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp, Alt: true})
	}
	lower := responseSplitRow(m.View())
	if upper > 20 || lower < 8 {
		t.Fatalf("split rows upper=%d lower=%d, want bounded 8..20 at height 24", upper, lower)
	}
	workspace, err := service.Workspace()
	if err != nil {
		t.Fatal(err)
	}
	stateBytes, err := os.ReadFile(filepath.Join(workspace.Root, ".apitool", "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var state map[string]any
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		t.Fatal(err)
	}
	if state["request_response_split_percent"] == nil {
		t.Fatal("resized split percentage was not persisted")
	}
	restored := tui.New(service, tui.Options{StartingCollection: "payments"})
	restored, _ = restored.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	if got := responseSplitRow(restored.View()); got != lower {
		t.Fatalf("restored split row = %d, want %d", got, lower)
	}
}

func TestMouseDragResizesRequestResponseSplit(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{StartingCollection: "payments"})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	before := responseSplitRow(m.View())
	m, _ = m.Update(tea.MouseMsg{X: 60, Y: before, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m, _ = m.Update(tea.MouseMsg{X: 60, Y: before + 2, Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft})
	if after := responseSplitRow(m.View()); after <= before {
		t.Fatalf("request/response split row = %d after mouse drag, want greater than %d", after, before)
	}
}

func historyScreenWithEntry(t *testing.T, requestID string) tui.Model {
	t.Helper()
	service := fixtureService(t)
	workspace, err := service.Workspace()
	if err != nil {
		t.Fatal(err)
	}
	store, err := runtime.Open(workspace.Root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AppendHistory(runtime.Key{CollectionPath: "payments", Environment: "test", RequestID: requestID}, "GET", modelResponse(200), nil); err != nil {
		t.Fatal(err)
	}
	m := tui.New(service, tui.Options{StartingCollection: "payments"})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	return m.(tui.Model)
}

func searchAndEnter(m tui.Model, query string) tui.Model {
	for _, r := range query {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(string(r))})
		m = next.(tui.Model)
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(tui.Model)
}

func updateTUI(m tui.Model, message tea.Msg) tui.Model {
	next, _ := m.Update(message)
	return next.(tui.Model)
}

func modelResponse(status int) model.Response { return model.Response{StatusCode: status} }

func responseSplitRow(view string) int {
	lines := strings.Split(view, "\n")
	for row, line := range lines {
		if strings.Contains(line, "├") && row+1 < len(lines) && strings.Contains(lines[row+1], "Response / Diagnostics / Request Log") {
			return row
		}
	}
	return -1
}
