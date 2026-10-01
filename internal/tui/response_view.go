package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"apitool/internal/auth"
	"apitool/internal/runtime"
)

func prettyBody(body []byte) string {
	var formatted bytes.Buffer
	if json.Indent(&formatted, body, "", "  ") == nil {
		return formatted.String()
	}
	return string(body)
}
func (m Model) responseLines() []string {
	if m.sending {
		return []string{"Sending… (you can still navigate panes)"}
	}
	if m.gitAction != "" {
		lines := []string{"Git: " + m.gitAction}
		if strings.TrimSpace(m.gitOutput) == "" {
			return append(lines, "No output.")
		}
		return append(lines, strings.Split(strings.TrimSuffix(m.gitOutput, "\n"), "\n")...)
	}
	if m.responseTab == 2 {
		return m.logLines()
	}
	if m.result.ExecutionError != nil {
		e := m.result.ExecutionError
		lines := []string{"Request failed", "Stage: " + string(e.Stage), "Category: " + string(e.Category), e.SafeMessage}
		for _, d := range m.result.Diagnostics {
			lines = append(lines, d.Path+": "+d.Message)
		}
		return append(lines, m.logLines()...)
	}
	if m.responseTab == 1 {
		return []string{"No execution diagnostic."}
	}
	if m.result.Response == nil {
		return nil
	}
	r := m.result.Response
	lines := []string{fmt.Sprintf("%d %s • %s • %d bytes", r.StatusCode, http.StatusText(r.StatusCode), r.Duration, len(r.Body)), "Headers:"}
	headers := runtime.RedactHeaders(r.Headers)
	names := make([]string, 0, len(headers))
	for k := range headers {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		lines = append(lines, k+": "+strings.Join(headers[k], ", "))
	}
	body := prettyBody(r.Body)
	title := "Body (pretty; b raw)"
	if m.rawResponse {
		body = string(r.Body)
		title = "Body (raw; b pretty)"
	}
	lines = append(lines, title)
	return append(lines, strings.Split(body, "\n")...)
}
func (m Model) logLines() []string {
	lines := []string{"Request Log (redacted)"}
	for _, entry := range m.result.Logs {
		lines = append(lines, fmt.Sprintf("%s %s • status %d • %s", entry.Method, safeURL(entry.Path), entry.StatusCode, entry.Duration))
		if entry.ErrorCategory != "" {
			lines = append(lines, "Category: "+string(entry.ErrorCategory))
		}
		if entry.Data != nil {
			encoded, _ := json.Marshal(runtime.RedactData(entry.Data))
			lines = append(lines, string(encoded))
		}
	}
	if len(m.result.Logs) == 0 {
		lines = append(lines, "No request log entries.")
	}
	return lines
}
func (m Model) executionConfirmationView() string {
	cancel, send := "[ Cancel ]", "[ Send ]"
	if m.sendPrompt.accept {
		send = "▶ " + send
	} else {
		cancel = "▶ " + cancel
	}
	return fmt.Sprintf("Confirmation required\nEnvironment: %s\nMethod: %s\nURL: %s\n\n%s   %s\nTab/←/→ choose • Enter activate • s Send • Esc cancel\n", m.sendPrompt.selection.Environment, m.sendPrompt.method, m.sendPrompt.url, cancel, send)
}
func (m Model) authView() string {
	lines := []string{"Auth (session only)", "Esc close • ↑/↓ scroll"}
	if m.authLoading {
		return strings.Join(append(lines, "Loading token…"), "\n") + "\n"
	}
	action := "Show token (s)"
	token := auth.Mask(m.authToken.AccessToken)
	if m.tokenRevealed {
		action = "Hide token (s)"
		token = m.authToken.AccessToken
	}
	lines = append(lines, "", action)
	if m.authFailure != nil {
		lines = append(lines, "Stage: "+string(m.authFailure.Stage), "Category: "+string(m.authFailure.Category), m.authFailure.SafeMessage)
	} else if m.authConfig == nil || m.authConfig.None || m.authConfig.Type == "none" {
		lines = append(lines, "No OAuth authentication.")
	} else {
		lines = append(lines, "Grant: "+m.authConfig.Grant, "Endpoint: "+safeURL(m.authConfig.TokenURL), "Scopes: "+strings.Join(m.authConfig.Scopes, ", "), "Expiry: "+m.authToken.Expiry.Format(time.RFC3339), "Token: "+token)
	}
	// Scroll within the session view for long token values and narrow terminals.
	width := m.width
	if width <= 0 {
		width = 80
	}
	lines = wrapLines(lines, width)
	offset := min(m.authOffset, max(0, len(lines)-1))
	lines = lines[offset:]
	if m.height > 0 {
		lines = lines[:min(len(lines), m.height)]
	}
	return strings.Join(lines, "\n") + "\n"
}
func wrapLines(lines []string, width int) []string {
	width = max(1, width)
	var out []string
	for _, line := range lines {
		runes := []rune(line)
		for len(runes) > width {
			out = append(out, string(runes[:width]))
			runes = runes[width:]
		}
		out = append(out, string(runes))
	}
	return out
}
