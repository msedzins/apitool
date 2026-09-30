// Package git provides the deliberately small local-Git surface used by the application.
package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var sensitiveDiffValue = regexp.MustCompile(`(?im)(["']?(?:client[_-]?secret|access[_-]?token|refresh[_-]?token|authorization|proxy[_-]?authorization|set[_-]?cookie|cookie|api[_-]?key|password|secret)["']?\s*[:=]\s*).*$`)
var diffHunk = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

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
	return r.run(ctx, "status", "--short", "--", ".", ":(exclude).apitool/**")
}

func (r Repository) Diff(ctx context.Context) (string, error) {
	// Comparing with HEAD includes both unstaged and staged changes. This lets
	// users review the exact content a subsequent commit would contain when
	// staging is performed outside apitool.
	args := []string{"diff", "HEAD", "--", ".", ":(exclude).apitool/**"}
	contextOutput, contextErr := r.run(ctx, "diff", "HEAD", "--unified=999999999", "--", ".", ":(exclude).apitool/**")
	if contextErr != nil {
		return "", contextErr
	}
	output, err := r.run(ctx, args...)
	return redactDiffWithContext(output, contextOutput), err
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
	staged, err := r.run(ctx, "diff", "--cached", "--name-only", "--", ".")
	if err != nil {
		return staged, err
	}
	for _, path := range strings.Split(staged, "\n") {
		path = strings.TrimPrefix(strings.TrimSpace(path), "./")
		if path == ".apitool" || strings.HasPrefix(path, ".apitool/") {
			return staged, errors.New("commit includes runtime data under .apitool")
		}
	}
	return r.run(ctx, "commit", "-m", message)
}

type sensitiveLines struct {
	old map[int]string
	new map[int]string
}

type redactionState struct {
	blockIndent int
	plainIndent int
	quote       byte
}

func newRedactionState() redactionState {
	return redactionState{blockIndent: -1, plainIndent: -1}
}

func redactDiffWithContext(text, fullContext string) string {
	lines := sensitiveLineMap(fullContext)
	var output strings.Builder
	currentOld, currentNew := "", ""
	oldLine, newLine := 0, 0
	for _, line := range strings.SplitAfter(text, "\n") {
		if line == "" {
			continue
		}
		content := strings.TrimSuffix(line, "\n")
		if strings.HasPrefix(content, "diff --git ") {
			currentOld, currentNew, _ = parseDiffHeader(content)
			output.WriteString(line)
			continue
		}
		if match := diffHunk.FindStringSubmatch(content); match != nil {
			oldLine, _ = strconv.Atoi(match[1])
			newLine, _ = strconv.Atoi(match[2])
			output.WriteString(line)
			continue
		}
		if strings.HasPrefix(content, "+++") || strings.HasPrefix(content, "---") || strings.HasPrefix(content, "\\") || content == "" {
			output.WriteString(line)
			continue
		}
		if len(content) == 0 {
			output.WriteString(line)
			continue
		}
		prefix := content[0]
		payload := content[1:]
		var replacement string
		var redact bool
		switch prefix {
		case '-':
			replacement, redact = lineReplacement(sensitiveSide(lines, currentOld, true), oldLine, payload)
			oldLine++
		case '+':
			replacement, redact = lineReplacement(sensitiveSide(lines, currentNew, false), newLine, payload)
			newLine++
		case ' ':
			replacement, redact = lineReplacement(sensitiveSide(lines, currentOld, true), oldLine, payload)
			if !redact {
				replacement, redact = lineReplacement(sensitiveSide(lines, currentNew, false), newLine, payload)
			}
			oldLine++
			newLine++
		default:
			output.WriteString(line)
			continue
		}
		if redact {
			output.WriteByte(prefix)
			output.WriteString(replacement)
			if strings.HasSuffix(line, "\n") {
				output.WriteByte('\n')
			}
		} else {
			output.WriteString(line)
		}
	}
	return output.String()
}

func sensitiveSide(lines map[string]*sensitiveLines, path string, old bool) map[int]string {
	entry := lines[path]
	if entry == nil {
		return nil
	}
	if old {
		return entry.old
	}
	return entry.new
}

func sensitiveLineMap(fullContext string) map[string]*sensitiveLines {
	result := map[string]*sensitiveLines{}
	currentOld, currentNew := "", ""
	oldLine, newLine := 0, 0
	oldState, newState := newRedactionState(), newRedactionState()
	for _, line := range strings.Split(fullContext, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			currentOld, currentNew, _ = parseDiffHeader(line)
			oldState, newState = newRedactionState(), newRedactionState()
			continue
		}
		if match := diffHunk.FindStringSubmatch(line); match != nil {
			oldLine, _ = strconv.Atoi(match[1])
			newLine, _ = strconv.Atoi(match[2])
			continue
		}
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "\\") || line == "" {
			continue
		}
		prefix, payload := line[0], line[1:]
		switch prefix {
		case '-':
			if safe, redact := oldState.redact(payload); redact {
				addSensitiveLine(result, currentOld, true, oldLine, safe)
			}
			oldLine++
		case '+':
			if safe, redact := newState.redact(payload); redact {
				addSensitiveLine(result, currentNew, false, newLine, safe)
			}
			newLine++
		case ' ':
			if safe, redact := oldState.redact(payload); redact {
				addSensitiveLine(result, currentOld, true, oldLine, safe)
			}
			if safe, redact := newState.redact(payload); redact {
				addSensitiveLine(result, currentNew, false, newLine, safe)
			}
			oldLine++
			newLine++
		}
	}
	return result
}

func addSensitiveLine(lines map[string]*sensitiveLines, path string, old bool, number int, safe string) {
	if path == "" || path == "/dev/null" {
		return
	}
	entry := lines[path]
	if entry == nil {
		entry = &sensitiveLines{old: map[int]string{}, new: map[int]string{}}
		lines[path] = entry
	}
	if old {
		entry.old[number] = safe
	} else {
		entry.new[number] = safe
	}
}

func lineReplacement(lines map[int]string, number int, payload string) (string, bool) {
	if safe, ok := lines[number]; ok {
		return safe, true
	}
	match := sensitiveDiffValue.FindStringSubmatchIndex(payload)
	if len(match) < 4 {
		return "", false
	}
	return payload[:match[3]] + "[REDACTED]", true
}

func (state *redactionState) redact(line string) (string, bool) {
	indent := len(line) - len(strings.TrimLeft(line, " "))
	if state.plainIndent >= 0 {
		if strings.TrimSpace(line) == "" {
			return line, false
		}
		if indent > state.plainIndent {
			return line[:indent] + "[REDACTED]", true
		}
		state.plainIndent = -1
	}
	if state.blockIndent >= 0 {
		if strings.TrimSpace(line) == "" {
			return line, false
		}
		if indent > state.blockIndent {
			return line[:indent] + "[REDACTED]", true
		}
		state.blockIndent = -1
	}
	if state.quote != 0 {
		closed := quoteClosedOnLine(line, state.quote)
		if closed {
			state.quote = 0
		}
		return line[:indent] + "[REDACTED]", true
	}
	match := sensitiveDiffValue.FindStringSubmatchIndex(line)
	if len(match) < 4 {
		return "", false
	}
	prefixEnd := match[3]
	value := strings.TrimSpace(line[prefixEnd:])
	safe := line[:prefixEnd] + "[REDACTED]"
	if isYAMLBlockMarker(value) {
		state.blockIndent = indent
		return safe, true
	}
	if len(value) > 0 && (value[0] == '"' || value[0] == '\'') && !quoteClosedOnLine(value, value[0]) {
		state.quote = value[0]
	} else {
		state.plainIndent = indent
	}
	return safe, true
}

func isYAMLBlockMarker(value string) bool {
	marker, _, _ := strings.Cut(value, "#")
	marker = strings.TrimSpace(marker)
	if marker == "" || (marker[0] != '|' && marker[0] != '>') {
		return false
	}
	for _, char := range marker[1:] {
		if char != '+' && char != '-' && (char < '1' || char > '9') {
			return false
		}
	}
	return true
}

func quoteClosedOnLine(value string, quote byte) bool {
	for index := 1; index < len(value); index++ {
		if quote == '"' && value[index] == '\\' {
			index++
			continue
		}
		if value[index] != quote {
			continue
		}
		if quote == '\'' && index+1 < len(value) && value[index+1] == quote {
			index++
			continue
		}
		return true
	}
	return false
}

func parseDiffHeader(line string) (oldPath, newPath string, ok bool) {
	rest := strings.TrimPrefix(line, "diff --git ")
	oldToken, rest, ok := parseGitPathToken(rest)
	if !ok {
		return "", "", false
	}
	newToken, _, ok := parseGitPathToken(strings.TrimLeft(rest, " "))
	if !ok {
		return "", "", false
	}
	return strings.TrimPrefix(oldToken, "a/"), strings.TrimPrefix(newToken, "b/"), true
}

func parseGitPathToken(value string) (token, rest string, ok bool) {
	value = strings.TrimLeft(value, " ")
	if value == "" {
		return "", "", false
	}
	if value[0] != '"' {
		index := strings.IndexByte(value, ' ')
		if index < 0 {
			return value, "", true
		}
		return value[:index], value[index+1:], true
	}
	for index := 1; index < len(value); index++ {
		if value[index] == '\\' {
			index++
			continue
		}
		if value[index] == '"' {
			decoded, err := strconv.Unquote(value[:index+1])
			if err != nil {
				return "", "", false
			}
			return decoded, value[index+1:], true
		}
	}
	return "", "", false
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
