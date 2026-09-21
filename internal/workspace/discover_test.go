package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"apitool/internal/collection"
	"apitool/internal/workspace"
)

func TestDiscoverFindsOnlyDirectoriesContainingDotAPI(t *testing.T) {
	root := fixtureWorkspace(t)
	writeWorkspaceFile(t, root, "broken/.api/collection.yaml", "description: missing required name\n")
	writeWorkspaceFile(t, root, "missing/.api/requests/.keep", "")
	writeWorkspaceFile(t, root, ".apitool/hidden/.api/collection.yaml", "name: Hidden\n")
	writeWorkspaceFile(t, root, ".git/internal/.api/collection.yaml", "name: Git internals\n")
	writeWorkspaceFile(t, root, "payments/.api/internal/.api/collection.yaml", "name: Not a collection\n")

	collections := workspace.Discover(root)
	if got, want := collectionPaths(collections), []string{"broken", "missing", "payments"}; !equalStrings(got, want) {
		t.Fatalf("Discover() collection paths = %#v, want %#v", got, want)
	}
	if len(collections[0].Diagnostics) == 0 {
		t.Fatal("Discover() did not retain diagnostics for invalid collection metadata")
	}
	if len(collections[1].Diagnostics) == 0 {
		t.Fatal("Discover() did not retain diagnostics for missing collection metadata")
	}
}

func TestDiscoverDoesNotFollowSymlinkedDirectories(t *testing.T) {
	root := fixtureWorkspace(t)
	external := t.TempDir()
	writeWorkspaceFile(t, external, ".api/collection.yaml", "name: Outside workspace\n")
	if err := os.Symlink(external, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}

	if got, want := collectionPaths(workspace.Discover(root)), []string{"payments"}; !equalStrings(got, want) {
		t.Fatalf("Discover() collection paths = %#v, want %#v", got, want)
	}
}

func TestFindRootSupportsDirectoryAndLinkedWorktreeGitFile(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFile(t, root, ".git/HEAD", "ref: refs/heads/main\n")
	start := filepath.Join(root, "nested", "directory")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := workspace.FindRoot(start); err != nil || got != root {
		t.Fatalf("FindRoot(directory) = %q, %v; want %q, nil", got, err, root)
	}

	linked := t.TempDir()
	writeWorkspaceFile(t, linked, ".git", "gitdir: /tmp/example-worktree.git\n")
	writeWorkspaceFile(t, linked, "child/.keep", "")
	if got, err := workspace.FindRoot(filepath.Join(linked, "child")); err != nil || got != linked {
		t.Fatalf("FindRoot(linked worktree) = %q, %v; want %q, nil", got, err, linked)
	}
}

func fixtureWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeWorkspaceFile(t, root, "payments/.api/collection.yaml", "name: Payments API\n")
	return root
}

func writeWorkspaceFile(t *testing.T, root, relativePath, content string) {
	t.Helper()
	path := filepath.Join(root, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func collectionPaths(collections []collection.LocatedCollection) []string {
	paths := make([]string, 0, len(collections))
	for _, located := range collections {
		paths = append(paths, located.Path)
	}
	return paths
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
