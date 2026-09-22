// Package git provides the deliberately small local-Git surface used by the application.
package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// CommandFunc is injectable so callers can test command construction without
// changing the adapter's production behavior.
type CommandFunc func(context.Context, string, ...string) *exec.Cmd

type Repository struct {
	root    string
	command CommandFunc
}

func New(root string, command CommandFunc) Repository {
	if command == nil {
		command = exec.CommandContext
	}
	return Repository{root: root, command: command}
}

func (r Repository) Status(ctx context.Context) (string, error) {
	return r.run(ctx, "status", "--short")
}

func (r Repository) Diff(ctx context.Context) (string, error) {
	return r.run(ctx, "diff")
}

func (r Repository) Pull(ctx context.Context) (string, error) {
	return r.run(ctx, "pull")
}

func (r Repository) Push(ctx context.Context) (string, error) {
	return r.run(ctx, "push")
}

func (r Repository) Commit(ctx context.Context, message string) (string, error) {
	if strings.TrimSpace(message) == "" {
		return "", errors.New("commit message must not be blank")
	}
	return r.run(ctx, "commit", "-m", message)
}

func (r Repository) run(ctx context.Context, args ...string) (string, error) {
	cmd := r.command(ctx, "git", args...)
	cmd.Dir = r.root
	output, err := cmd.CombinedOutput()
	text := string(output)
	if err != nil {
		if strings.TrimSpace(text) != "" {
			return text, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(text))
		}
		return text, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return text, nil
}
