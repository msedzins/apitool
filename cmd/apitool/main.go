package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"apitool/internal/app"
	"apitool/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "apitool: %v\n", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	options, err := parseStartupOptions(arguments)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	model, err := newApplication(options)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}
	if err := runProgram(model); err != nil {
		return fmt.Errorf("run application: %w", err)
	}
	return nil
}

var runProgram = func(model tea.Model) error {
	_, err := tea.NewProgram(model, tea.WithMouseCellMotion()).Run()
	return err
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
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{Collection: options.Collection, Environment: options.Environment, ConfirmDangerous: options.ConfirmDangerous}); err != nil {
		return nil, err
	}
	return tui.New(service, tui.Options{
		StartingCollection:  options.Collection,
		StartingEnvironment: options.Environment,
		ConfirmDangerous:    options.ConfirmDangerous,
		Color:               colorEnabled(termenv.EnvColorProfile()),
	}), nil
}

func colorEnabled(profile termenv.Profile) bool { return profile != termenv.Ascii }

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
