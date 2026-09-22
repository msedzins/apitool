package main

import "testing"

import "github.com/muesli/termenv"

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
