package main

import (
	"os"
	"strings"
	"testing"

	"github.com/muesli/termenv"
)

func TestRunTreatsHelpAsSuccessful(t *testing.T) {
	if err := run([]string{"--help"}); err != nil {
		t.Fatalf("run(--help) error = %v, want nil", err)
	}
}

func TestColorEnabledRejectsASCIIProfile(t *testing.T) {
	if colorEnabled(termenv.Ascii) {
		t.Fatal("ASCII terminal must use textual status categories")
	}
	if !colorEnabled(termenv.ANSI) {
		t.Fatal("ANSI terminal should enable color")
	}
}

func TestREADMEStartupExamplesMatchCLIFlags(t *testing.T) {
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	examples := []struct {
		text        string
		environment string
		confirm     bool
	}{
		{text: "apitool -e test", environment: "test"},
		{text: "apitool -e prod -c", environment: "prod", confirm: true},
	}
	for _, example := range examples {
		t.Run(example.text, func(t *testing.T) {
			if !strings.Contains(string(readme), example.text) {
				t.Fatalf("README does not include %q", example.text)
			}
			options, err := parseStartupOptions(strings.Fields(strings.TrimPrefix(example.text, "apitool ")))
			if err != nil {
				t.Fatalf("parse documented command: %v", err)
			}
			if options.Environment != example.environment || options.ConfirmDangerous != example.confirm {
				t.Fatalf("parsed options = %#v, want environment %q and confirmation %t", options, example.environment, example.confirm)
			}
		})
	}
}
