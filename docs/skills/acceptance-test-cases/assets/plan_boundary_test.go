package boundary_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var acceptanceCaseID = regexp.MustCompile(`\b(UI|API|CFG|DATA|WF|OPS)-[0-9]{3}\b`)

func TestDocumentBoundaries(t *testing.T) {
	rules := []documentBoundaryRule{
		{
			name:         "implementation plans do not contain acceptance-case IDs",
			documentGlob: "docs/superpowers/plans/*.md",
			forbidden:    acceptanceCaseID,
		},
	}

	for _, rule := range rules {
		rule := rule
		t.Run(rule.name, func(t *testing.T) {
			violations, err := findDocumentBoundaryViolations(".", rule)
			if err != nil {
				t.Fatal(err)
			}
			if len(violations) == 0 {
				return
			}

			messages := make([]string, 0, len(violations))
			for _, violation := range violations {
				messages = append(messages, violation.path+" contains forbidden value \""+violation.match+"\"")
			}
			t.Fatalf("%s:\n%s", rule.name, strings.Join(messages, "\n"))
		})
	}
}

func TestFindDocumentBoundaryViolationsReportsEveryMatch(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "docs", "plans")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	files := []struct {
		path    string
		content string
	}{
		{path: "first.md", content: "references UI-001"},
		{path: "second.md", content: "references API-002"},
		{path: "clean.md", content: "no acceptance identifier"},
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(directory, file.path), []byte(file.content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	violations, err := findDocumentBoundaryViolations(root, documentBoundaryRule{
		documentGlob: "docs/plans/*.md",
		forbidden:    acceptanceCaseID,
	})
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, 0, len(violations))
	for _, violation := range violations {
		got = append(got, filepath.Base(violation.path)+":"+violation.match)
	}
	want := []string{"first.md:UI-001", "second.md:API-002"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("violations = %v, want %v", got, want)
	}
}

type documentBoundaryRule struct {
	name         string
	documentGlob string
	forbidden    *regexp.Regexp
}

type documentBoundaryViolation struct {
	path  string
	match string
}

func findDocumentBoundaryViolations(root string, rule documentBoundaryRule) ([]documentBoundaryViolation, error) {
	paths, err := filepath.Glob(filepath.Join(root, rule.documentGlob))
	if err != nil {
		return nil, err
	}

	var violations []documentBoundaryViolation
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		for _, match := range rule.forbidden.FindAllString(string(content), -1) {
			violations = append(violations, documentBoundaryViolation{path: path, match: match})
		}
	}
	return violations, nil
}
