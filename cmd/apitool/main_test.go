package main

import "testing"

func TestRunTreatsHelpAsSuccessful(t *testing.T) {
	if err := run([]string{"--help"}); err != nil {
		t.Fatalf("run(--help) error = %v, want nil", err)
	}
}
