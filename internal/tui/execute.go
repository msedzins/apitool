package tui

import (
	"context"
	"net/url"
	"strings"

	"apitool/internal/app"
	"apitool/internal/auth"
	"apitool/internal/model"
	"apitool/internal/resolve"
	"apitool/internal/runtime"
	tea "github.com/charmbracelet/bubbletea"
)

type sendFinishedMsg struct {
	selection app.Selection
	result    app.SendResult
}
type authFinishedMsg struct {
	selection app.Selection
	config    *model.Auth
	token     auth.Token
	failure   *model.ExecutionError
}
type sendConfirmation struct {
	selection   app.Selection
	method, url string
	accept      bool
}

func (m Model) executionSelection() (app.Selection, bool) {
	if m.service == nil {
		return app.Selection{}, false
	}
	if m.mode == requestEditMode && m.editor != nil {
		return m.editor.Selection(), true
	}
	rows := m.visibleRows()
	if m.treeIndex >= 0 && m.treeIndex < len(rows) && rows[m.treeIndex].kind == invalidRow {
		return app.Selection{}, false
	}
	node, ok := m.activeRequest()
	return app.Selection{Collection: m.collection, Environment: m.view.Environment, RequestID: node.ID}, ok
}
func sameSelection(a, b app.Selection) bool {
	return a.Collection == b.Collection && a.Environment == b.Environment && a.RequestID == b.RequestID
}
func (m Model) hasCurrentResult() bool {
	if m.resultSelection.RequestID == "" {
		return false
	}
	selection, ok := m.executionSelection()
	return ok && sameSelection(selection, m.resultSelection)
}
func (m Model) effectiveSelection(selection app.Selection) model.EffectiveRequest {
	node := m.view.Tree.Requests[selection.RequestID]
	groups := make([]model.Group, len(node.Groups))
	for i, g := range node.Groups {
		groups[i] = g.Group
	}
	e, _ := resolve.Effective(m.view.Collection, m.view.Environments[selection.Environment], groups, node.Request)
	return e
}
func (m *Model) beginSend() tea.Cmd {
	if m.sending || m.saving || m.authLoading {
		return nil
	}
	selection, ok := m.executionSelection()
	if !ok {
		return nil
	}
	if m.editor != nil && (m.editor.Dirty() || m.fieldDraftDirty) {
		m.message = "Save request changes with Ctrl+S before sending."
		return nil
	}
	e := m.effectiveSelection(selection)
	if m.confirmDangerous && dangerousMethod(e.Method) {
		m.sendPrompt = &sendConfirmation{selection: selection, method: e.Method, url: safeRequestURL(e)}
		return nil
	}
	return m.sendCommand(selection)
}
func (m *Model) sendCommand(selection app.Selection) tea.Cmd {
	m.sending = true
	m.sendSelection = selection
	m.clearResult()
	m.resultSelection = selection
	m.responseTab = 0
	m.message = ""
	service := m.service
	return func() tea.Msg {
		return sendFinishedMsg{selection: selection, result: service.Send(context.Background(), selection)}
	}
}

func (m *Model) clearResult() {
	m.result = app.SendResult{}
	m.resultSelection = app.Selection{}
	m.resultCached = false
	m.responseOffset = 0
	m.responseTab = 0
	m.rawResponse = false
}

func (m *Model) loadCachedResponse() {
	selection, ok := m.executionSelection()
	if !ok || selection.Environment == "" {
		return
	}
	response, err := m.service.CachedResponse(context.Background(), selection)
	if err == runtime.ErrNotFound {
		return
	}
	if err != nil {
		m.result = app.SendResult{Diagnostics: []model.Diagnostic{{
			Code:     "runtime_cache_read",
			Path:     ".apitool/responses",
			Message:  "could not read cached response",
			Severity: model.SeverityWarning,
		}}}
		m.resultSelection = selection
		m.resultCached = false
		return
	}
	m.result = app.SendResult{Response: &response}
	m.resultSelection = selection
	m.resultCached = true
	m.responseOffset = 0
	m.responseTab = 0
	m.rawResponse = false
}

func dangerousMethod(method string) bool {
	switch strings.ToUpper(method) {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	}
	return false
}
func safeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "[invalid URL]"
	}
	u.User = nil
	u.Fragment = ""
	q := u.Query()
	for k, values := range q {
		safe := runtime.RedactData(map[string]string{k: strings.Join(values, ",")}).(map[string]string)[k]
		if safe == "[REDACTED]" {
			q.Set(k, safe)
		}
	}
	u.RawQuery = q.Encode()
	return strings.NewReplacer("%7B", "{", "%7D", "}").Replace(u.String())
}
func safeRequestURL(e model.EffectiveRequest) string {
	u, err := url.Parse(e.URL)
	if err != nil {
		return "[invalid URL]"
	}
	q := u.Query()
	for k, v := range e.Params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return safeURL(u.String())
}
func (m *Model) handleSendConfirmation(k tea.KeyMsg) tea.Cmd {
	switch k.Type {
	case tea.KeyEsc:
		m.sendPrompt = nil
	case tea.KeyTab, tea.KeyLeft, tea.KeyRight:
		m.sendPrompt.accept = !m.sendPrompt.accept
	case tea.KeyEnter:
		prompt := m.sendPrompt
		m.sendPrompt = nil
		if prompt.accept {
			selection := prompt.selection
			selection.Confirmed = true
			return m.sendCommand(selection)
		}
	case tea.KeyRunes:
		if string(k.Runes) == "c" {
			m.sendPrompt = nil
		}
		if string(k.Runes) == "s" {
			selection := m.sendPrompt.selection
			selection.Confirmed = true
			m.sendPrompt = nil
			return m.sendCommand(selection)
		}
	}
	return nil
}
func (m *Model) openAuth() tea.Cmd {
	selection, ok := m.executionSelection()
	if !ok || m.authLoading || m.sending || m.saving {
		return nil
	}
	m.authOpen = true
	m.authLoading = true
	m.tokenRevealed = false
	m.authSelection = selection
	m.authToken = auth.Token{}
	m.authConfig = nil
	m.authFailure = nil
	m.authOffset = 0
	service := m.service
	return func() tea.Msg {
		config, token, failure := service.AuthToken(context.Background(), selection)
		return authFinishedMsg{selection: selection, config: config, token: token, failure: failure}
	}
}

// Execution actions are handled before the editor, so sending never inserts
// a newline or changes a field. Other editor keys retain their existing meaning.
func (m *Model) executionKey(k tea.KeyMsg) (bool, tea.Cmd) {
	if m.sendPrompt != nil {
		return true, m.handleSendConfirmation(k)
	}
	if m.mode != browseMode && m.mode != requestEditMode {
		return false, nil
	}
	if m.sending && (k.Type == tea.KeyCtrlE || (m.mode == requestEditMode && k.Type == tea.KeyCtrlS)) {
		return true, nil
	}
	if k.Type == tea.KeyCtrlJ && !(m.mode == requestEditMode && m.editor != nil && m.editorField == 6 && m.editor.mode == BodyModeJSON) {
		return true, m.beginSend()
	}
	if m.authOpen {
		switch k.Type {
		case tea.KeyEsc:
			m.authOpen = false
			m.tokenRevealed = false
			return true, nil
		case tea.KeyUp:
			m.authOffset = max(0, m.authOffset-1)
			return true, nil
		case tea.KeyDown:
			m.authOffset++
			return true, nil
		case tea.KeyRunes:
			if string(k.Runes) == "s" {
				m.tokenRevealed = !m.tokenRevealed
			}
			return true, nil
		}
		return true, nil
	}
	if k.Type == tea.KeyCtrlA || (m.mode == browseMode && m.focus == requestPane && k.Type == tea.KeyRunes && string(k.Runes) == "a") {
		return true, m.openAuth()
	}
	if m.mode == browseMode && m.focus == requestPane && k.Type == tea.KeyEnter {
		return true, m.beginSend()
	}
	if m.focus == responsePane {
		switch k.Type {
		case tea.KeyUp:
			m.responseOffset = max(0, m.responseOffset-1)
			return true, nil
		case tea.KeyDown:
			m.responseOffset++
			return true, nil
		case tea.KeyLeft:
			m.responseTab = wrap(m.responseTab-1, 3)
			m.responseOffset = 0
			return true, nil
		case tea.KeyRight:
			m.responseTab = wrap(m.responseTab+1, 3)
			m.responseOffset = 0
			return true, nil
		case tea.KeyRunes:
			if string(k.Runes) == "b" {
				m.rawResponse = !m.rawResponse
				m.responseOffset = 0
				return true, nil
			}
		}
	}
	return false, nil
}
func (m *Model) executionMouse(x tea.MouseMsg) (bool, tea.Cmd) {
	if x.Action != tea.MouseActionPress || x.Button != tea.MouseButtonLeft {
		return false, nil
	}
	if m.sendPrompt != nil {
		lines := strings.Split(m.executionConfirmationView(), "\n")
		if x.Y == 5 {
			line := lines[5]
			if pos := strings.Index(line, "[ Send ]"); pos >= 0 {
				column := len([]rune(line[:pos]))
				if x.X >= column && x.X < column+8 {
					selection := m.sendPrompt.selection
					selection.Confirmed = true
					m.sendPrompt = nil
					return true, m.sendCommand(selection)
				}
			}
			if pos := strings.Index(line, "[ Cancel ]"); pos >= 0 {
				column := len([]rune(line[:pos]))
				if x.X >= column && x.X < column+10 {
					m.sendPrompt = nil
				}
			}
		}
		return true, nil
	}
	if m.authOpen {
		lines := strings.Split(m.authView(), "\n")
		if x.Y >= 0 && x.Y < len(lines) {
			line := lines[x.Y]
			if (strings.HasPrefix(line, "Show token (s)") || strings.HasPrefix(line, "Hide token (s)")) && x.X >= 0 && x.X < 14 {
				m.tokenRevealed = !m.tokenRevealed
			}
		}
		return true, nil
	}
	if m.mode != browseMode && m.mode != requestEditMode {
		return false, nil
	}
	// Hit-test the rendered control, including headings that fit without truncation.
	lines := strings.Split(m.collectionView(), "\n")
	if x.Y == 1 && len(lines) > 1 {
		if pos := strings.Index(lines[1], "[ Send ]"); pos >= 0 {
			column := len([]rune(lines[1][:pos]))
			if x.X >= column && x.X < column+8 {
				return true, m.beginSend()
			}
		}
	}
	left := m.collectionExplorerWidth() + 2
	if m.mode == browseMode && x.Y == 3 && x.X >= left+20 && x.X < left+26 {
		return true, m.openAuth()
	}
	return false, nil
}
