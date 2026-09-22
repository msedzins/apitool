package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"apitool/internal/git"
)

func TestStatusAndDiffRunAtWorkspaceRoot(t *testing.T) {
	root := initGitRepo(t)
	writeWorkspaceChange(t, root)
	repo := git.New(root, exec.CommandContext)

	status, err := repo.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(status, " M ") && !strings.Contains(status, "M") {
		t.Fatalf("status = %q, want modified file", status)
	}

	diff, err := repo.Diff(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "changed") {
		t.Fatalf("diff = %q, want changed content", diff)
	}
}

func TestDiffIncludesStagedChanges(t *testing.T) {
	root := initGitRepo(t)
	path := writeWorkspaceChange(t, root)
	stage := exec.Command("git", "add", filepath.Base(path))
	stage.Dir = root
	if output, err := stage.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, output)
	}

	repo := git.New(root, exec.CommandContext)
	diff, err := repo.Diff(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "changed") {
		t.Fatalf("diff = %q, want staged changed content", diff)
	}
}

func TestCommitRejectsBlankMessageBeforeInvokingGit(t *testing.T) {
	repo := git.New(initGitRepo(t), exec.CommandContext)

	_, err := repo.Commit(context.Background(), "   ")
	if err == nil || !strings.Contains(err.Error(), "commit message") {
		t.Fatalf("error = %v, want commit message validation", err)
	}
}

func TestGitCommandsRejectNonRepositoryRoot(t *testing.T) {
	repo := git.New(t.TempDir(), exec.CommandContext)

	_, err := repo.Status(context.Background())
	if err == nil || !strings.Contains(err.Error(), "not a git repository") {
		t.Fatalf("error = %v, want clear repository error", err)
	}
}

func TestPullFailureRetainsGitOutput(t *testing.T) {
	repo := git.New(initGitRepo(t), exec.CommandContext)

	output, err := repo.Pull(context.Background())
	if err == nil {
		t.Fatal("Pull succeeded without a remote")
	}
	if strings.TrimSpace(output) == "" {
		t.Fatalf("Pull output is empty for error: %v", err)
	}
}

func TestDiffDoesNotWriteWorkspace(t *testing.T) {
	root := initGitRepo(t)
	path := writeWorkspaceChange(t, root)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	repo := git.New(root, exec.CommandContext)
	if _, err := repo.Diff(context.Background()); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("diff changed workspace file: before %q after %q", before, after)
	}
}

func TestCommitSucceedsOnlyForAlreadyStagedFile(t *testing.T) {
	root := initGitRepo(t)
	path := writeWorkspaceChange(t, root)
	repo := git.New(root, exec.CommandContext)

	if _, err := repo.Commit(context.Background(), "unstaged change"); err == nil {
		t.Fatal("Commit accepted an unstaged file")
	}
	stage := exec.Command("git", "add", filepath.Base(path))
	stage.Dir = root
	if output, err := stage.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, output)
	}
	if _, err := repo.Commit(context.Background(), "save change"); err != nil {
		t.Fatal(err)
	}
	status, err := repo.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(status) != "" {
		t.Fatalf("status after commit = %q, want clean", status)
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test User"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	path := filepath.Join(root, "tracked.txt")
	if err := os.WriteFile(path, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", "tracked.txt")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, output)
	}
	cmd = exec.Command("git", "commit", "-qm", "initial")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, output)
	}
	return root
}

func writeWorkspaceChange(t *testing.T, root string) string {
	t.Helper()
	path := filepath.Join(root, "tracked.txt")
	if err := os.WriteFile(path, []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
