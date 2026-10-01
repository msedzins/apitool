package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type gitActionFinishedMsg struct {
	action string
	output string
	err    error
}

func (m *Model) startGitAction(action, message string) tea.Cmd {
	if m.service == nil || m.gitBusy {
		return nil
	}
	if m.mode != gitCommitMode {
		m.paletteReturnMode = m.mode
	}
	m.gitBusy = true
	m.gitAction, m.gitOutput = "", ""
	m.message = "Running Git " + action + "…"
	service := m.service
	return func() tea.Msg {
		ctx := context.Background()
		var output string
		var err error
		switch action {
		case "Status":
			output, err = service.GitStatus(ctx)
		case "Diff":
			output, err = service.GitDiff(ctx)
		case "Pull":
			output, err = service.GitPull(ctx)
		case "Push":
			output, err = service.GitPush(ctx)
		case "Commit":
			output, err = service.GitCommit(ctx, message)
		}
		return gitActionFinishedMsg{action: action, output: output, err: err}
	}
}

func (m *Model) handleGitCommitKey(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyEsc:
		m.mode, m.gitCommitMessage, m.message = m.paletteReturnMode, "", ""
	case tea.KeyEnter:
		if strings.TrimSpace(m.gitCommitMessage) == "" {
			m.message = "Commit message must not be blank"
			return nil
		}
		message := strings.TrimSpace(m.gitCommitMessage)
		return m.startGitAction("Commit", message)
	case tea.KeyBackspace:
		runes := []rune(m.gitCommitMessage)
		if len(runes) > 0 {
			m.gitCommitMessage = string(runes[:len(runes)-1])
		}
	case tea.KeyCtrlU:
		m.gitCommitMessage = ""
	case tea.KeyRunes:
		m.gitCommitMessage += string(key.Runes)
	}
	return nil
}

func (m Model) gitCommitView() string {
	lines := []string{"Git: Commit", "Commit message: " + m.gitCommitMessage, "Enter commit • Esc cancel"}
	if m.message != "" {
		lines = append(lines, "", m.message)
	}
	return strings.Join(lines, "\n") + "\n"
}
