package tui_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"apitool/internal/app"
	"apitool/internal/runtime"
	"apitool/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCollectionPickerOpensCollectionAndRestoresItsEnvironment(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := next.View(); !containsAll(got, "Collection: payments", "Environment: test") {
		t.Fatalf("View() = %q, want payments and restored test environment", got)
	}
}

func TestStatusViewUsesTextWhenColorDisabled(t *testing.T) {
	view := tui.StatusView(tui.Status{Code: 500, Duration: time.Millisecond}, false)
	if !containsAll(view, "500", "error") {
		t.Fatalf("StatusView() = %q, want code and textual error category", view)
	}
}

func TestCollectionSwitchChangesTreeAndEnvironment(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.View(); !containsAll(got, "Collection: orders", "Environment: dev", "list") {
		t.Fatalf("View() = %q, want orders collection, environment, and tree", got)
	}
}

func TestInvalidRequestHasWarningWhileValidSiblingOpens(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.View(); !containsAll(got, "warning", "check") {
		t.Fatalf("tree View() = %q, want warning and valid sibling", got)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.View(); !containsAll(got, "Request: billing/nested") {
		t.Fatalf("opened View() = %q, want selected valid request", got)
	}
}

func TestCtrlEOpensEnvironmentPicker(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	if got := m.View(); !containsAll(got, "Environments", "test") {
		t.Fatalf("View() = %q, want environment picker", got)
	}
}

func TestOptionsStartingEnvironmentSelectsEnvironment(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{StartingCollection: "payments", StartingEnvironment: "prod"})
	if got := m.View(); !containsAll(got, "Collection: payments", "Environment: prod") {
		t.Fatalf("View() = %q, want options-selected environment", got)
	}
}

func TestTabCyclesPanes(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if got := m.View(); !containsAll(got, "Focus: request") {
		t.Fatalf("View() = %q, want request pane focus after Tab", got)
	}
}

func TestVimNavigationIsOptIn(t *testing.T) {
	withoutVim := tui.New(fixtureService(t), tui.Options{})
	withoutVim, _ = withoutVim.Update(tea.KeyMsg{Type: tea.KeyDown})
	withoutVim, _ = withoutVim.Update(tea.KeyMsg{Type: tea.KeyDown})
	withoutVim, _ = withoutVim.Update(tea.KeyMsg{Type: tea.KeyDown})
	withoutVim, _ = withoutVim.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if got := withoutVim.View(); !containsAll(got, "Request: billing/nested") {
		t.Fatalf("default View() = %q, want selection unchanged", got)
	}
	withVim := tui.New(fixtureService(t), tui.Options{VimMode: true})
	withVim, _ = withVim.Update(tea.KeyMsg{Type: tea.KeyDown})
	withVim, _ = withVim.Update(tea.KeyMsg{Type: tea.KeyDown})
	withVim, _ = withVim.Update(tea.KeyMsg{Type: tea.KeyDown})
	withVim, _ = withVim.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if got := withVim.View(); !containsAll(got, "Request: check") {
		t.Fatalf("vim View() = %q, want selection moved", got)
	}
}

func TestLeftAndRightToggleExpandedTreeGroup(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{})
	if got := m.View(); !containsAll(got, "billing", "nested") {
		t.Fatalf("expanded View() = %q, want group and request", got)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if got := m.View(); strings.Contains(got, "\n  billing/nested\n") {
		t.Fatalf("collapsed View() = %q, should hide grouped request", got)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := m.View(); !strings.Contains(got, "GET nested") {
		t.Fatalf("expanded View() = %q, should restore grouped request", got)
	}
}

func fixtureService(t *testing.T) *app.Service {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, data string) {
		t.Helper()
		path = filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("payments/.api/collection.yaml", "name: Payments\n")
	write("payments/.api/environments/test.yaml", "name: test\n")
	write("payments/.api/environments/prod.yaml", "name: prod\n")
	write("payments/.api/requests/check.yaml", "name: Check\nmethod: GET\nrequest:\n  url: https://example.test/check\n")
	write("payments/.api/requests/second.yaml", "name: Second\nmethod: GET\nrequest:\n  url: https://example.test/second\n")
	write("payments/.api/requests/broken.yaml", "name: Broken\nmethod: GET\n")
	write("payments/.api/requests/billing/nested.yaml", "name: Nested\nmethod: GET\nrequest:\n  url: https://example.test/nested\n")
	write("payments/.api/requests/billing/bad.yaml", "name: Bad\nmethod: GET\n")
	write("payments/.api/requests/billing/deep/item.yaml", "name: Item\nmethod: GET\nrequest:\n  url: https://example.test/item\n")
	write("payments/.api/requests/billing/deep/bad.yaml", "name: Deep Bad\nmethod: GET\n")
	write("orders/.api/collection.yaml", "name: Orders\n")
	write("orders/.api/environments/dev.yaml", "name: dev\n")
	write("orders/.api/requests/list.yaml", "name: List\nmethod: GET\nrequest:\n  url: https://example.test/orders\n")
	for i := range 10 {
		write(fmt.Sprintf("payments/.api/requests/z%d.yaml", i), fmt.Sprintf("name: Z%d\nmethod: GET\nrequest:\n  url: https://example.test/z%d\n", i, i))
	}
	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	store, err := runtime.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(runtime.State{LastActiveEnvironment: map[string]string{"payments": "test", "orders": "dev"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{Collection: "payments"}); err != nil {
		t.Fatal(err)
	}
	return service
}

func containsAll(value string, wants ...string) bool {
	for _, want := range wants {
		if !strings.Contains(value, want) {
			return false
		}
	}
	return true
}
