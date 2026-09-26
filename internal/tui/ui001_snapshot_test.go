package tui_test

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"apitool/internal/app"
	"apitool/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUI001CollectionPickerMatchesApprovedScreen(t *testing.T) {
	service := ui001WorkspaceService(t)
	model := tui.New(service, tui.Options{Color: false, ApprovedShell: true})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	want, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ui-001", "collection-picker.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got := model.View(); got != string(want) {
		t.Fatalf("UI-001 collection picker mismatch:\n%s", lineDiff(string(want), got))
	}
}

func TestUI001SelectedCollectionMatchesApprovedScreen(t *testing.T) {
	service := ui001WorkspaceService(t)
	model := tui.New(service, tui.Options{Color: false, ApprovedShell: true})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	got := model.View()
	if !strings.Contains(got, "Collection: payments") {
		t.Fatalf("selected collection screen = %q, want payments active", got)
	}
	if !strings.Contains(got, "▸ users") {
		t.Fatalf("selected collection screen = %q, want users represented as collapsed", got)
	}
	want, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ui-001", "payments-tree.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("UI-001 selected collection mismatch:\n%s", lineDiff(string(want), got))
	}
}

func TestUI001ExampleWorkspaceDiscoversApprovedCollections(t *testing.T) {
	root := ui001ExampleWorkspaceRoot(t)
	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}

	collections := make([]string, 0, len(workspace.Collections))
	for collectionPath := range workspace.Collections {
		collections = append(collections, collectionPath)
	}
	sort.Strings(collections)
	if got, want := strings.Join(collections, ","), "payments,users"; got != want {
		t.Fatalf("discovered collections = %q, want %q", got, want)
	}
}

func TestUI001ApprovedShellKeepsBrowseStateObservable(t *testing.T) {
	t.Run("search filters the same-named root group requests", func(t *testing.T) {
		model := ui001SelectedCollectionModel(t)
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("list")})
		got := model.View()
		if !strings.Contains(got, "Search: list") || !strings.Contains(got, "payments/list") || strings.Contains(got, "payments/create") {
			t.Fatalf("search view = %q, want only the matching request", got)
		}
	})

	t.Run("collapse and expand change the same-named root group tree", func(t *testing.T) {
		model := ui001SelectedCollectionModel(t)
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
		if got := model.View(); strings.Contains(got, "GET list") || strings.Contains(got, "POST create") {
			t.Fatalf("collapsed tree = %q, want root requests hidden", got)
		}
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRight})
		if got := model.View(); !strings.Contains(got, "GET list") || !strings.Contains(got, "POST create") {
			t.Fatalf("expanded tree = %q, want root requests restored", got)
		}
	})

	t.Run("request selection changes visible request state", func(t *testing.T) {
		model := ui001SelectedCollectionModel(t)
		for range 2 {
			model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
		}
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if got := model.View(); !strings.Contains(got, "Request: payments/create") || !strings.Contains(got, "Focus: request") {
			t.Fatalf("selected request view = %q, want payments/create request state", got)
		}
	})

	t.Run("Ctrl+Right moves the splitter", func(t *testing.T) {
		model := ui001SelectedCollectionModel(t)
		before := strings.Index(model.View(), "┬")
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlRight})
		if after := strings.Index(model.View(), "┬"); after <= before {
			t.Fatalf("splitter = %d after Ctrl+Right, want greater than %d", after, before)
		}
	})
}

func ui001SelectedCollectionModel(t *testing.T) tea.Model {
	t.Helper()
	model := tui.New(ui001WorkspaceService(t), tui.Options{Color: false, ApprovedShell: true})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return model
}

func TestUI001PreviewSkipsRequestWithInvalidGroupDiagnostics(t *testing.T) {
	service := ui001InvalidGroupPreviewService(t)
	model := tui.New(service, tui.Options{Color: false, ApprovedShell: true})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	got := model.View()
	if !strings.Contains(got, "GET | {{base_url}}/valid") || !strings.Contains(got, "Request: valid") {
		t.Fatalf("preview = %q, want the later valid request", got)
	}
}

func ui001WorkspaceService(t *testing.T) *app.Service {
	t.Helper()
	root := ui001ExampleWorkspaceRoot(t)
	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SelectEnvironment(context.Background(), "payments", "test"); err != nil {
		t.Fatal(err)
	}
	return service
}

func ui001ExampleWorkspaceRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join("..", "..", "examples", "workspace")
	if err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relativePath == "." {
			return nil
		}
		destination := filepath.Join(root, relativePath)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, contents, 0o644)
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func ui001InvalidGroupPreviewService(t *testing.T) *app.Service {
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
	write("payments/.api/environments/test.yaml", "name: test\nvariables:\n  base_url: https://api.example.test\n")
	write("payments/.api/requests/payments/_group.yaml", "auth: invalid\n")
	write("payments/.api/requests/payments/affected.yaml", "name: Affected\nmethod: GET\nrequest:\n  url: \"{{base_url}}/affected\"\n")
	write("payments/.api/requests/valid.yaml", "name: Valid\nmethod: GET\nrequest:\n  url: \"{{base_url}}/valid\"\n")

	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SelectEnvironment(context.Background(), "payments", "test"); err != nil {
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
