package tui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"apitool/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUI015FocusMarkersMatchApprovedScreens(t *testing.T) {
	model := ui001SelectedCollectionModel(t)
	assertSnapshot(t, model.View(), filepath.Join("..", "..", "testdata", "ui-001", "payments-tree.txt"))

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	assertSnapshot(t, model.View(), filepath.Join("..", "..", "testdata", "ui-015", "request-focus.txt"))

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	assertSnapshot(t, model.View(), filepath.Join("..", "..", "testdata", "ui-015", "response-focus.txt"))
}

func TestUI015HelpRestoresEveryTUIState(t *testing.T) {
	tests := []struct {
		name  string
		model tea.Model
	}{
		{"collection view", ui001SelectedCollectionModel(t)},
		{"collection picker", tui.New(fixtureService(t), tui.Options{})},
		{"environment picker", environmentPickerModel(t)},
		{"search", searchModel(t)},
	}

	for _, closeKey := range []tea.KeyMsg{{Type: tea.KeyEsc}, {Type: tea.KeyRunes, Runes: []rune("?")}} {
		for _, test := range tests {
			t.Run(test.name+"/close-"+closeKey.String(), func(t *testing.T) {
				before := test.model.View()
				model, _ := test.model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
				if got := model.View(); !strings.Contains(got, "Keyboard shortcuts") {
					t.Fatalf("help view = %q, want keyboard help", got)
				}
				model, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
				if got := model.View(); !strings.Contains(got, "Keyboard shortcuts") {
					t.Fatalf("Tab dismissed help: %q", got)
				}
				model, _ = model.Update(closeKey)
				if got := model.View(); got != before {
					t.Fatalf("restored view differs:\n%s", lineDiff(before, got))
				}
			})
		}
	}
}

func TestUI015KeyboardHelpMatchesApprovedScreenAndCompacts(t *testing.T) {
	model := ui001SelectedCollectionModel(t)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	assertSnapshot(t, model.View(), filepath.Join("..", "..", "testdata", "ui-015", "keyboard-help.txt"))

	compact := tui.New(fixtureService(t), tui.Options{})
	compact, _ = compact.Update(tea.KeyMsg{Type: tea.KeyEnter})
	compact, _ = compact.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	compact, _ = compact.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	for _, want := range []string{"Keyboard shortcuts", "Navigation", "Workspace", "Layout", "Help", "Exit"} {
		if got := compact.View(); !strings.Contains(got, want) {
			t.Fatalf("compact help = %q, want %q", got, want)
		}
	}
}

func TestUI015CtrlCQuitsHelp(t *testing.T) {
	model := ui001SelectedCollectionModel(t)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	_, command := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if command == nil {
		t.Fatal("Ctrl+C returned no quit command")
	}
	if _, ok := command().(tea.QuitMsg); !ok {
		t.Fatalf("Ctrl+C command = %T, want QuitMsg", command())
	}
}

func TestUI015HelpBlocksMouseInput(t *testing.T) {
	model := ui001SelectedCollectionModel(t)
	before := model.View()
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	model, _ = model.Update(tea.MouseMsg{X: 80, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if got := model.View(); got != before {
		t.Fatalf("mouse input changed the obscured view:\n%s", lineDiff(before, got))
	}
}

func environmentPickerModel(t *testing.T) tea.Model {
	t.Helper()
	model := tui.New(fixtureService(t), tui.Options{})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	return model
}

func searchModel(t *testing.T) tea.Model {
	t.Helper()
	model := ui001SelectedCollectionModel(t)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("list")})
	return model
}

func assertSnapshot(t *testing.T, got, path string) {
	t.Helper()
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("snapshot mismatch:\n%s", lineDiff(string(want), got))
	}
}
