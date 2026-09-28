package tui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"apitool/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUI002InvalidRequestMatchesApprovedScreen(t *testing.T) {
	model := tui.New(ui002FixtureService(t), tui.Options{StartingCollection: "users", Color: false})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	for range 30 {
		if strings.Contains(model.View(), "> ! broken (warning)") {
			break
		}
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if !strings.Contains(model.View(), "> ! broken (warning)") {
		t.Fatalf("UI-002 screen did not select broken request:\n%s", model.View())
	}
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := model.View()
	want, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ui-002", "invalid-request.txt"))
	if err != nil {
		t.Fatalf("read UI-002 screen fixture: %v\nactual screen:\n%s", err, got)
	}
	if got != string(want) {
		t.Fatalf("UI-002 screen mismatch:\n%s", lineDiff(string(want), got))
	}
}
