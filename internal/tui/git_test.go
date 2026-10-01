package tui_test

import (
	"strings"
	"testing"

	"apitool/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func TestGitCommitPromptsForNonBlankMessage(t *testing.T) {
	m := gitScreen(t)
	if got := m.View(); !strings.Contains(got, "Commit message") {
		t.Fatalf("Git commit prompt = %q, want message prompt", got)
	}
	m = updateTUI(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("   ")})
	m = updateTUI(m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.View(); !strings.Contains(got, "Commit message") {
		t.Fatalf("blank commit message left prompt: %q", got)
	}
}

func TestGitCommandErrorsRemainVisible(t *testing.T) {
	m := tui.New(fixtureService(t), tui.Options{StartingCollection: "payments"})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if cmd == nil {
		t.Fatalf("Git: Status returned no command: %q", m.View())
	}
	next, _ := m.Update(cmd())
	if got := next.View(); !strings.Contains(got, "Git:") || !strings.Contains(strings.ToLower(got), "fatal") {
		t.Fatalf("Git error view = %q, want visible diagnostic", got)
	}
}

func gitScreen(t *testing.T) tui.Model {
	t.Helper()
	m := tui.New(fixtureService(t), tui.Options{StartingCollection: "payments"})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	return m.(tui.Model)
}
