package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"apitool/internal/app"
	"apitool/internal/model"
	"apitool/internal/validate"

	tea "github.com/charmbracelet/bubbletea"
)

// BodyMode controls whether the request body is edited as structured JSON or exact text.
type BodyMode uint8

const (
	BodyModeJSON BodyMode = iota
	BodyModeRaw
)

// RequestEditor owns one request editing session and its isolated undo history.
type RequestEditor struct {
	service    *app.Service
	selection  app.Selection
	original   model.Request
	request    model.Request
	mode       BodyMode
	bodyText   string
	initial    string
	undo       []editorSnapshot
	redo       []editorSnapshot
	validation string
}

type editorSnapshot struct {
	request  model.Request
	bodyText string
	mode     BodyMode
}

func newRequestEditor(service *app.Service, selection app.Selection, request model.Request) *RequestEditor {
	e := &RequestEditor{service: service, selection: selection}
	e.Load(request)
	return e
}

// Load starts a fresh editing session and clears undo/redo history.
func (e *RequestEditor) Load(request model.Request) {
	e.request = cloneRequest(request)
	e.original = cloneRequest(request)
	e.undo, e.redo = nil, nil
	e.validation = ""
	e.mode = BodyModeJSON
	e.bodyText = ""
	if request.Request.Body != nil {
		if request.Request.Body.Type == "raw" {
			e.mode = BodyModeRaw
			if text, ok := request.Request.Body.Content.(string); ok {
				e.bodyText = text
			}
		} else {
			data, err := json.MarshalIndent(request.Request.Body.Content, "", "  ")
			if err == nil {
				e.bodyText = string(data)
			}
		}
	}
	e.initial = e.bodyText
}

// Request returns a copy of the current typed request.
func (e *RequestEditor) Request() model.Request { return cloneRequest(e.request) }

// Dirty reports whether typed values or body text differ from the loaded request.
func (e *RequestEditor) Dirty() bool {
	return !reflect.DeepEqual(e.request, e.original) || e.bodyText != e.initial || e.mode != modeFor(e.original)
}

// Undo restores the previous typed request state.
func (e *RequestEditor) Undo() {
	if len(e.undo) == 0 {
		return
	}
	e.redo = append(e.redo, e.snapshot())
	last := e.undo[len(e.undo)-1]
	e.undo = e.undo[:len(e.undo)-1]
	e.restore(last)
}

// Redo reapplies the next typed request state.
func (e *RequestEditor) Redo() {
	if len(e.redo) == 0 {
		return
	}
	e.undo = append(e.undo, e.snapshot())
	last := e.redo[len(e.redo)-1]
	e.redo = e.redo[:len(e.redo)-1]
	e.restore(last)
}

// SwitchBodyMode changes the body editor while preserving the represented bytes/data.
func (e *RequestEditor) SwitchBodyMode(mode BodyMode) {
	if e.mode == mode {
		return
	}
	if mode == BodyModeRaw && e.request.Request.Body == nil && strings.TrimSpace(e.bodyText) == "" {
		e.pushUndo()
		e.mode = BodyModeRaw
		e.setBody("raw", "")
		e.validation = ""
		return
	}
	value, err := decodeJSONValue(e.bodyText)
	if err != nil {
		e.validation = "invalid JSON body: " + err.Error()
		return
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		e.validation = "JSON body cannot be represented as text"
		return
	}
	e.pushUndo()
	e.mode = mode
	e.bodyText = string(data)
	if mode == BodyModeRaw {
		e.setBody("raw", e.bodyText)
	} else {
		e.setBody("json", value)
	}
	e.validation = ""
}

// SetBodyText changes the active body's text without normalizing raw content.
func (e *RequestEditor) SetBodyText(text string) {
	if e.bodyText == text {
		return
	}
	e.pushUndo()
	e.bodyText = text
	e.validation = ""
	if e.mode == BodyModeRaw {
		e.setBody("raw", text)
		return
	}
	value, err := decodeJSONValue(text)
	if err == nil {
		e.setBody("json", value)
	}
}

// SetURL edits the typed request URL.
func (e *RequestEditor) SetURL(value string) { e.mutate(func() { e.request.Request.URL = value }) }

// SetName edits the typed request name.
func (e *RequestEditor) SetName(value string) { e.mutate(func() { e.request.Name = value }) }

// SetMethod edits the typed HTTP method.
func (e *RequestEditor) SetMethod(value string) {
	e.mutate(func() { e.request.Method = strings.ToUpper(value) })
}

func (e *RequestEditor) SetParams(value map[string]string) {
	e.mutate(func() { e.request.Request.Params = cloneMap(value) })
}
func (e *RequestEditor) SetHeaders(value map[string]string) {
	e.mutate(func() { e.request.Request.Headers = cloneMap(value) })
}
func (e *RequestEditor) SetAuth(value *model.Auth) {
	e.mutate(func() { e.request.Auth = cloneAuth(value) })
}

func (e *RequestEditor) SetSelection(selection app.Selection) { e.selection = selection }
func (e *RequestEditor) Selection() app.Selection             { return e.selection }
func (e *RequestEditor) Validation() string                   { return e.validation }

// FieldText returns the editable text for the typed editor field at index.
func (e *RequestEditor) FieldText(index int) string {
	switch index {
	case 0:
		return e.request.Name
	case 1:
		return e.request.Method
	case 2:
		return e.request.Request.URL
	case 3:
		return formatStringMap(e.request.Request.Params)
	case 4:
		return formatStringMap(e.request.Request.Headers)
	case 5:
		return formatAuth(e.request.Auth)
	case 6:
		return e.bodyText
	default:
		return ""
	}
}

// SetFieldText parses the selected typed field, keeping YAML serialization outside the TUI.
func (e *RequestEditor) SetFieldText(index int, value string) {
	switch index {
	case 0:
		e.SetName(value)
	case 1:
		e.SetMethod(value)
	case 2:
		e.SetURL(value)
	case 3:
		e.SetParams(parseStringMap(value))
	case 4:
		e.SetHeaders(parseStringMap(value))
	case 5:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || trimmed == "inherit" {
			e.SetAuth(nil)
		} else if trimmed == "none" {
			e.SetAuth(&model.Auth{None: true})
		} else {
			var value struct {
				Type         string   `json:"type"`
				Grant        string   `json:"grant"`
				TokenURL     string   `json:"token_url"`
				ClientID     string   `json:"client_id"`
				ClientSecret string   `json:"client_secret"`
				Scopes       []string `json:"scopes"`
			}
			if err := json.Unmarshal([]byte(trimmed), &value); err != nil {
				e.validation = "auth must be inherit, none, or a JSON OAuth object"
			} else {
				e.SetAuth(&model.Auth{Type: value.Type, Grant: value.Grant, TokenURL: value.TokenURL, ClientID: value.ClientID, ClientSecret: value.ClientSecret, Scopes: value.Scopes})
			}
		}
	case 6:
		e.SetBodyText(value)
	}
}

// Save validates and persists the request through the application service.
func (e *RequestEditor) Save() tea.Cmd {
	request, err := e.validatedRequest()
	if err != nil {
		e.validation = err.Error()
		return nil
	}
	if e.service == nil {
		e.validation = "request service is unavailable"
		return nil
	}
	selection := e.selection
	return func() tea.Msg {
		err := e.service.SaveRequest(context.Background(), selection, request)
		return requestSavedMsg{selection: selection, request: request, err: err}
	}
}

func (e *RequestEditor) validatedRequest() (model.Request, error) {
	request := cloneRequest(e.request)
	if request.Request.Body != nil || e.bodyText != "" {
		if request.Request.Body == nil {
			request.Request.Body = &model.Body{}
		}
		if e.mode == BodyModeRaw {
			request.Request.Body.Type = "raw"
			request.Request.Body.Content = e.bodyText
		} else {
			value, err := decodeJSONValue(e.bodyText)
			if err != nil {
				return model.Request{}, fmt.Errorf("invalid JSON body: %w", err)
			}
			request.Request.Body.Type = "json"
			request.Request.Body.Content = value
		}
	}
	if diagnostics := validate.Definition(request); len(diagnostics) > 0 {
		return model.Request{}, fmt.Errorf("request validation failed: %s", diagnostics[0].Message)
	}
	return request, nil
}

func (e *RequestEditor) mutate(change func()) {
	before := e.snapshot()
	change()
	if !reflect.DeepEqual(before.request, e.request) {
		e.undo = append(e.undo, before)
		e.redo = nil
	}
	e.validation = ""
}

func (e *RequestEditor) pushUndo() {
	e.undo = append(e.undo, e.snapshot())
	e.redo = nil
}
func (e *RequestEditor) snapshot() editorSnapshot {
	return editorSnapshot{request: cloneRequest(e.request), bodyText: e.bodyText, mode: e.mode}
}
func (e *RequestEditor) restore(s editorSnapshot) {
	e.request, e.bodyText, e.mode = cloneRequest(s.request), s.bodyText, s.mode
	e.validation = ""
}
func (e *RequestEditor) requestBodyContent() any {
	if e.request.Request.Body == nil {
		return nil
	}
	return e.request.Request.Body.Content
}
func (e *RequestEditor) requestBody() *model.Body {
	if e.request.Request.Body == nil {
		e.request.Request.Body = &model.Body{Type: "json"}
	}
	return e.request.Request.Body
}
func (e *RequestEditor) setBody(kind string, content any) {
	body := e.requestBody()
	body.Type, body.Content = kind, content
}
func modeFor(request model.Request) BodyMode {
	if request.Request.Body != nil && request.Request.Body.Type == "raw" {
		return BodyModeRaw
	}
	return BodyModeJSON
}
func cloneRequest(request model.Request) model.Request {
	copy := request
	copy.Request.Params = cloneMap(request.Request.Params)
	copy.Request.Headers = cloneMap(request.Request.Headers)
	copy.Auth = cloneAuth(request.Auth)
	if request.Request.Body != nil {
		body := *request.Request.Body
		body.Content = cloneJSONValue(body.Content)
		copy.Request.Body = &body
	}
	return copy
}
func decodeJSONValue(text string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("unexpected trailing JSON value")
		}
		return nil, err
	}
	return preserveJSONNumbers(value), nil
}
func preserveJSONNumbers(value any) any {
	switch value := value.(type) {
	case json.Number:
		return model.JSONNumber(value.String())
	case []any:
		for i := range value {
			value[i] = preserveJSONNumbers(value[i])
		}
		return value
	case map[string]any:
		for key, item := range value {
			value[key] = preserveJSONNumbers(item)
		}
		return value
	default:
		return value
	}
}

func cloneJSONValue(value any) any {
	switch value := value.(type) {
	case map[string]any:
		copy := make(map[string]any, len(value))
		for key, item := range value {
			copy[key] = cloneJSONValue(item)
		}
		return copy
	case []any:
		copy := make([]any, len(value))
		for index, item := range value {
			copy[index] = cloneJSONValue(item)
		}
		return copy
	default:
		return value
	}
}
func formatStringMap(values map[string]string) string {
	if values == nil {
		return "{}"
	}
	data, _ := json.Marshal(values)
	return string(data)
}
func parseStringMap(value string) map[string]string {
	var result map[string]string
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if json.Unmarshal([]byte(value), &result) != nil {
		return nil
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func cloneMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	copy := make(map[string]string, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}
func cloneAuth(source *model.Auth) *model.Auth {
	if source == nil {
		return nil
	}
	copy := *source
	copy.Scopes = append([]string(nil), source.Scopes...)
	return &copy
}

func (m Model) editorDisplay() string {
	if m.editor == nil {
		return "Request editor unavailable\n"
	}
	request := m.editor.Request()
	mode := "JSON"
	if m.editor.mode == BodyModeRaw {
		mode = "Raw"
	}
	labels := []string{"Name", "Method", "URL", "Params", "Headers", "Auth", "Body (" + mode + ")"}
	values := []string{request.Name, request.Method, request.Request.URL, formatStringMap(request.Request.Params), formatStringMap(request.Request.Headers), formatAuth(request.Auth), strings.ReplaceAll(m.editor.bodyText, "\n", "\\n")}
	lines := []string{"Request editor — Ctrl+S save • Ctrl+Z undo • Ctrl+Y redo • Esc close"}
	for index, label := range labels {
		marker := "  "
		if m.editorField == index {
			marker = "▶ "
		}
		value := values[index]
		if m.editorField == index {
			value = m.editorFieldText()
		}
		lines = append(lines, marker+label+": "+value)
	}
	lines = append(lines, "Edit fields with Tab/Enter and type; Ctrl+B switches body mode; Ctrl+P opens actions.")
	if m.duplicateFlow {
		lines = append(lines, "Save as: "+m.duplicateTarget)
		lines = append(lines, "Duplicate path is the final editor field; Ctrl+S saves the copy.")
	}
	if m.editor.validation != "" {
		lines = append(lines, "Validation: "+m.editor.validation)
	}
	if m.message != "" {
		lines = append(lines, "Status: "+m.message)
	}
	return strings.Join(lines, "\n") + "\n"
}
func formatPairs(values map[string]string) string {
	var lines []string
	for key, value := range values {
		lines = append(lines, key+"="+value)
	}
	return strings.Join(lines, ", ")
}
func formatAuth(auth *model.Auth) string {
	if auth == nil {
		return "inherit"
	}
	if auth.None {
		return "none"
	}
	value := struct {
		Type         string   `json:"type"`
		Grant        string   `json:"grant,omitempty"`
		TokenURL     string   `json:"token_url,omitempty"`
		ClientID     string   `json:"client_id,omitempty"`
		ClientSecret string   `json:"client_secret,omitempty"`
		Scopes       []string `json:"scopes,omitempty"`
	}{auth.Type, auth.Grant, auth.TokenURL, auth.ClientID, auth.ClientSecret, auth.Scopes}
	data, _ := json.Marshal(value)
	return string(data)
}

func parsePairs(value string) map[string]string {
	result := map[string]string{}
	for _, item := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '\n' }) {
		key, val, ok := strings.Cut(strings.TrimSpace(item), "=")
		if ok && strings.TrimSpace(key) != "" {
			result[strings.TrimSpace(key)] = strings.TrimSpace(val)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func (m Model) editorFieldTextFor(index int) string {
	if m.editor == nil {
		return ""
	}
	if index == m.editorField {
		return m.editorFieldText()
	}
	if m.duplicateFlow && index == 7 {
		return m.duplicateTarget
	}
	return m.editor.FieldText(index)
}
