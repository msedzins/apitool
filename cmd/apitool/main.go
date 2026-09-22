package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"apitool/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	options, err := parseStartupOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "apitool: %v\n", err)
		os.Exit(2)
	}
	if _, err := newApplication(options); err != nil {
		fmt.Fprintf(os.Stderr, "apitool: initialize application: %v\n", err)
		os.Exit(1)
	}
}

func newApplication(options startupOptions) (tea.Model, error) {
	service, err := app.New(app.Dependencies{})
	if err != nil {
		return nil, err
	}
	root, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	opened, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{Collection: options.Collection, Environment: options.Environment, ConfirmDangerous: options.ConfirmDangerous})
	if err != nil {
		return nil, err
	}
	return appModel{options: options, service: service, workspace: opened, selectedCollection: options.Collection}, nil
}

type startupOptions struct {
	Collection, Environment string
	ConfirmDangerous        bool
}

func parseStartupOptions(arguments []string) (startupOptions, error) {
	flags := flag.NewFlagSet("apitool", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var options startupOptions
	flags.StringVar(&options.Environment, "env", "", "starting environment")
	flags.StringVar(&options.Environment, "e", "", "starting environment")
	flags.BoolVar(&options.ConfirmDangerous, "confirm-dangerous", false, "confirm POST, PUT, PATCH, and DELETE requests")
	flags.BoolVar(&options.ConfirmDangerous, "c", false, "confirm POST, PUT, PATCH, and DELETE requests")
	if err := flags.Parse(arguments); err != nil {
		return startupOptions{}, err
	}
	if flags.NArg() > 1 {
		return startupOptions{}, fmt.Errorf("expected at most one collection name")
	}
	if flags.NArg() == 1 {
		options.Collection = flags.Arg(0)
	}
	return options, nil
}

type appModel struct {
	options            startupOptions
	service            *app.Service
	workspace          app.Workspace
	selectedCollection string
}

func (appModel) Init() tea.Cmd { return nil }

func (model appModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	return model, nil
}

func (appModel) View() string { return "" }
