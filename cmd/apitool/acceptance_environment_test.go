package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"apitool/internal/runtime"
)

func TestAcceptanceCFG005CLIEnvironmentOverridesRememberedChoice(t *testing.T) {
	root := t.TempDir()
	for path, contents := range map[string]string{
		".git/placeholder":                     "",
		"test-api/.api/collection.yaml":        "name: Test API\n",
		"test-api/.api/environments/test.yaml": "name: test\n",
		"test-api/.api/environments/prod.yaml": "name: prod\nvariables:\n  host: https://prod.example.test\n",
		"test-api/.api/requests/one.yaml":      "name: One\nmethod: GET\nrequest:\n  url: https://example.test\n",
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := runtime.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(runtime.State{LastActiveEnvironment: map[string]string{"test-api": "test"}}); err != nil {
		t.Fatal(err)
	}
	definitions := acceptanceCFG005DefinitionSnapshot(t, root)
	t.Chdir(root)

	options, err := parseStartupOptions([]string{"--env", "prod", "test-api"})
	if err != nil {
		t.Fatal(err)
	}
	model, err := newApplication(options)
	if err != nil {
		t.Fatal(err)
	}
	view := model.View()
	if !strings.Contains(view, "Environment: prod") {
		t.Fatalf("CLI startup did not display the requested environment: %s", view)
	}
	if got := acceptanceCFG005DefinitionSnapshot(t, root); got != definitions {
		t.Fatal("--env override changed repository definition files")
	}
}

func acceptanceCFG005DefinitionSnapshot(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	for _, path := range []string{
		"test-api/.api/collection.yaml",
		"test-api/.api/environments/test.yaml",
		"test-api/.api/environments/prod.yaml",
		"test-api/.api/requests/one.yaml",
	} {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		b.WriteString(path)
		b.Write(data)
	}
	return b.String()
}
