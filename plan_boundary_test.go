package apitool_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var acceptanceCaseID = regexp.MustCompile(`\b(UI|API|CFG|DATA|WF|OPS)-[0-9]{3}\b`)

func TestImplementationPlansDoNotContainAcceptanceCaseIDs(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("docs", "superpowers", "plans", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if match := acceptanceCaseID.FindString(string(content)); match != "" {
			t.Fatalf("%s contains acceptance-case ID %q", path, match)
		}
	}
}
