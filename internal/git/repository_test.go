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

func TestGitDiffAndStatusExcludeRuntimeDirectoryAndCommitRejectsIt(t *testing.T) {
	root := initGitRepo(t)
	runtimePath := filepath.Join(root, ".apitool", "history.jsonl")
	if err := os.MkdirAll(filepath.Dir(runtimePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimePath, []byte("runtime data\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stage := exec.Command("git", "add", ".apitool")
	stage.Dir = root
	if output, err := stage.CombinedOutput(); err != nil {
		t.Fatalf("git add runtime data: %v\n%s", err, output)
	}

	repo := git.New(root, exec.CommandContext)
	status, err := repo.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(status, ".apitool") {
		t.Fatalf("status = %q, must hide runtime directory", status)
	}
	diff, err := repo.Diff(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(diff, ".apitool") || strings.Contains(diff, "runtime data") {
		t.Fatalf("diff = %q, must hide runtime directory", diff)
	}
	if _, err := repo.Commit(context.Background(), "runtime commit"); err == nil || !strings.Contains(err.Error(), "runtime data") {
		t.Fatalf("Commit error = %v, want runtime data rejection", err)
	}
}

func TestDiffRedactsSensitiveDefinitionValues(t *testing.T) {
	root := initGitRepo(t)
	path := filepath.Join(root, "request.yaml")
	content := "client_secret: super-secret\nAuthorization: Bearer bearer-secret\nCookie: session=cookie-secret\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	stage := exec.Command("git", "add", filepath.Base(path))
	stage.Dir = root
	if output, err := stage.CombinedOutput(); err != nil {
		t.Fatalf("git add sensitive definition: %v\n%s", err, output)
	}
	repo := git.New(root, exec.CommandContext)
	diff, err := repo.Diff(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"super-secret", "bearer-secret", "cookie-secret"} {
		if strings.Contains(diff, secret) {
			t.Fatalf("diff contains secret %q: %s", secret, diff)
		}
	}
	if strings.Count(diff, "[REDACTED]") != 3 {
		t.Fatalf("diff = %q, want three redacted values", diff)
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

func TestDiffRedactsMultilineYAMLCredentials(t *testing.T) {
	root := initGitRepo(t)
	path := filepath.Join(root, "request.yaml")
	partialPath := filepath.Join(root, "partial.yaml")
	before := `name: Old request
headers:
  Authorization: |-
    OLD_AUTHORIZATION_LINE_ONE
    OLD_AUTHORIZATION_LINE_TWO
  Cookie: >-
    OLD_COOKIE_LINE_ONE
    OLD_COOKIE_LINE_TWO
auth:
  access_token: >-
    OLD_TOKEN_LINE_ONE
    OLD_TOKEN_LINE_TWO
  client_secret: "OLD_SECRET_LINE_ONE
    OLD_SECRET_LINE_TWO"
`
	after := `name: Updated request
headers:
  Authorization: |-
    NEW_AUTHORIZATION_LINE_ONE
    NEW_AUTHORIZATION_LINE_TWO
  Cookie: >-
    NEW_COOKIE_LINE_ONE
    NEW_COOKIE_LINE_TWO
auth:
  access_token: >-
    NEW_TOKEN_LINE_ONE
    NEW_TOKEN_LINE_TWO
  client_secret: "NEW_SECRET_LINE_ONE
    NEW_SECRET_LINE_TWO"
`
	partialBefore := "name: Partial\nauth:\n  client_secret: |-\n    OLD_PARTIAL_SECRET_ONE\n    OLD_PARTIAL_SECRET_TWO\nmetadata:\n  description: Original text\n"
	partialAfter := "name: Partial\nauth:\n  client_secret: |-\n    OLD_PARTIAL_SECRET_ONE\n    OLD_PARTIAL_SECRET_TWO\nmetadata:\n  description: Updated text\n"
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(partialPath, []byte(partialBefore), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "request.yaml", "partial.yaml"}, {"commit", "-qm", "add requests"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	if err := os.WriteFile(path, []byte(after), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(partialPath, []byte(partialAfter), 0o644); err != nil {
		t.Fatal(err)
	}

	diff, err := git.New(root, exec.CommandContext).Diff(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, credential := range []string{
		"OLD_AUTHORIZATION_LINE_ONE", "OLD_AUTHORIZATION_LINE_TWO", "NEW_AUTHORIZATION_LINE_ONE", "NEW_AUTHORIZATION_LINE_TWO",
		"OLD_COOKIE_LINE_ONE", "OLD_COOKIE_LINE_TWO", "NEW_COOKIE_LINE_ONE", "NEW_COOKIE_LINE_TWO",
		"OLD_TOKEN_LINE_ONE", "OLD_TOKEN_LINE_TWO", "NEW_TOKEN_LINE_ONE", "NEW_TOKEN_LINE_TWO",
		"OLD_SECRET_LINE_ONE", "OLD_SECRET_LINE_TWO", "NEW_SECRET_LINE_ONE", "NEW_SECRET_LINE_TWO",
		"OLD_PARTIAL_SECRET_ONE", "OLD_PARTIAL_SECRET_TWO",
	} {
		if strings.Contains(diff, credential) {
			t.Fatalf("diff contains credential content %q: %s", credential, diff)
		}
	}
	if !strings.Contains(diff, "Old request") || !strings.Contains(diff, "Updated request") || !strings.Contains(diff, "Original text") || !strings.Contains(diff, "Updated text") {
		t.Fatalf("diff = %q, want unrelated definition changes preserved", diff)
	}
}
