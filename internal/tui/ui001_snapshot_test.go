package tui_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"apitool/internal/app"
	"apitool/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUI001CollectionPickerMatchesApprovedScreen(t *testing.T) {
	service := ui001WorkspaceService(t)
	model := tui.New(service, tui.Options{Color: false})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	want, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ui-001", "collection-picker.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got := model.View(); got != string(want) {
		t.Fatalf("UI-001 collection picker mismatch:\n%s", lineDiff(string(want), got))
	}
}

func ui001WorkspaceService(t *testing.T) *app.Service {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, data string) {
		t.Helper()
		path = filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("payments/.api/collection.yaml", "name: Payments API\n")
	write("users/.api/collection.yaml", "name: Users API\n")

	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}
	return service
}

func lineDiff(want, got string) string {
	wantLines, gotLines := strings.Split(want, "\n"), strings.Split(got, "\n")
	var diff strings.Builder
	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		var expected, actual string
		if i < len(wantLines) {
			expected = wantLines[i]
		}
		if i < len(gotLines) {
			actual = gotLines[i]
		}
		if expected != actual {
			fmt.Fprintf(&diff, "line %d (-want +got):\n-%q\n+%q\n", i+1, expected, actual)
		}
	}
	if len(wantLines) != len(gotLines) {
		fmt.Fprintf(&diff, "line count differs: want=%d got=%d\n", len(wantLines), len(gotLines))
	}
	if diff.Len() == 0 && want != got {
		fmt.Fprintf(&diff, "final newline differs: want=%t got=%t\n", strings.HasSuffix(want, "\n"), strings.HasSuffix(got, "\n"))
	}
	return diff.String()
}
