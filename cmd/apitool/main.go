package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if _, err := newApplication(); err != nil {
		fmt.Fprintf(os.Stderr, "apitool: initialize application: %v\n", err)
		os.Exit(1)
	}
}

func newApplication() (tea.Model, error) {
	return appModel{}, nil
}

type appModel struct{}

func (appModel) Init() tea.Cmd { return nil }

func (model appModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	return model, nil
}

func (appModel) View() string { return "" }
