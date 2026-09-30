package tui

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"apitool/internal/app"
	"apitool/internal/collection"
	"apitool/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDirtyRequestPromptsBeforeNavigation(t *testing.T) {
	m := editorWithChangedURL(t)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if !strings.Contains(next.View(), "Save and continue") || !strings.Contains(next.View(), "Discard changes") {
		t.Fatalf("dirty navigation view = %q, want save/discard choices", next.View())
	}
}

func TestJSONModePrettyPrintsAndRawModePreservesRawBytes(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	if !strings.Contains(m.View(), `"a": 1`) {
		t.Fatalf("JSON editor view = %q, want pretty JSON", m.View())
	}
	m.editor.SwitchBodyMode(BodyModeRaw)
	m.editor.SetBodyText("<a>1</a>")
	if got := m.editor.Request().Request.Body.Content; got != "<a>1</a>" {
		t.Fatalf("raw body = %#v, want exact raw text", got)
	}
}

func TestInvalidJSONCannotSaveOrSend(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	path := filepath.Join(m.view.Root, ".api", "requests", "one.yaml")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m.editor.SetBodyText(`{"a":`)
	if cmd := m.Save(); cmd != nil {
		t.Fatal("Save() returned a command for invalid JSON")
	}
	if !strings.Contains(m.message, "JSON") {
		t.Fatalf("save error = %q, want JSON validation feedback", m.message)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("invalid JSON changed the request YAML")
	}
}

func TestUndoRedoRestoresURLAndBody(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	originalURL := m.editor.Request().Request.URL
	m.editor.SetURL(originalURL + "/changed")
	m.editor.SetBodyText(`{"a":2}`)
	m.editor.Undo()
	if got := m.editor.bodyText; got != "{\n  \"a\": 1\n}" {
		t.Fatalf("undo body text = %q", got)
	}
	m.editor.Undo()
	if got := m.editor.Request().Request.URL; got != originalURL {
		t.Fatalf("undo URL = %q, want %q", got, originalURL)
	}
	m.editor.Redo()
	if got := m.editor.Request().Request.URL; got != originalURL+"/changed" {
		t.Fatalf("redo URL = %q", got)
	}
	m.editor.Redo()
	if got := m.editor.bodyText; got != `{"a":2}` {
		t.Fatalf("redo body text = %q", got)
	}
}

func TestUndoHistoryResetsAfterOpeningAnotherRequest(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.editor.SetURL("https://changed.example.test")
	m.editor.Load(validRequest("Other", "https://other.example.test", nil))
	m.editor.Undo()
	if got := m.editor.Request().Request.URL; got != "https://other.example.test" {
		t.Fatalf("undo crossed request boundary: %q", got)
	}
}

func TestDuplicateOpensNewUnsavedRequestFlow(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.beginDuplicate()
	if !strings.Contains(m.View(), "Save as") || !m.editor.Dirty() {
		t.Fatalf("duplicate view = %q, want unsaved rename/path flow", m.View())
	}
}

func TestGroupDeleteListsDescendantsBeforeConfirmation(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.beginDelete("nested", true)
	view := m.View()
	if !strings.Contains(view, "nested/one.yaml") || !strings.Contains(view, "nested/two.yaml") || !strings.Contains(view, "confirm") {
		t.Fatalf("delete confirmation = %q, want exact descendant paths and confirmation", view)
	}
}

func TestDiscardLeavesYAMLFileUnchanged(t *testing.T) {
	m := editorWithChangedURL(t)
	before, err := os.ReadFile(filepath.Join(m.view.Root, ".api", "requests", "one.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	m.discardEdits()
	after, err := os.ReadFile(filepath.Join(m.view.Root, ".api", "requests", "one.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("discard wrote changes to the request YAML")
	}
}

func editorWithChangedURL(t *testing.T) Model {
	t.Helper()
	m := editorWithJSON(t, `{"a":1}`)
	m.editor.SetURL("https://changed.example.test")
	return m
}

func editorWithJSON(t *testing.T, body string) Model {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	collectionPath := "test-api"
	writeEditorFile(t, root, collectionPath+"/.api/collection.yaml", "name: Test\n")
	writeEditorFile(t, root, collectionPath+"/.api/environments/test.yaml", "name: test\n")
	writeEditorFile(t, root, collectionPath+"/.api/requests/one.yaml", "name: One\nmethod: POST\nrequest:\n  url: https://example.test/one\n  body:\n    type: json\n    content: "+body+"\n")
	writeEditorFile(t, root, collectionPath+"/.api/requests/nested/one.yaml", "name: Nested one\nmethod: GET\nrequest:\n  url: https://example.test/one\n")
	writeEditorFile(t, root, collectionPath+"/.api/requests/nested/two.yaml", "name: Nested two\nmethod: GET\nrequest:\n  url: https://example.test/two\n")
	service, err := app.New(app.Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{}); err != nil {
		t.Fatal(err)
	}
	m := ModelFrom(New(service, Options{StartingCollection: collectionPath}))
	m.view, err = service.OpenCollection(context.Background(), collectionPath)
	if err != nil {
		t.Fatal(err)
	}
	m.collection = collectionPath
	m.beginEdit("one")
	return m
}

func ModelFrom(m tea.Model) Model { return m.(Model) }

func writeEditorFile(t *testing.T, root, rel, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func validRequest(name, url string, body *model.Body) model.Request {
	return model.Request{Name: name, Method: "GET", Request: model.RequestConfig{URL: url, Body: body}}
}

func TestSavePersistsStructuredJSON(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.editor.SetBodyText(`{"b":2,"a":1}`)
	cmd := m.Save()
	if cmd == nil {
		t.Fatalf("Save() rejected valid JSON: %s", m.message)
	}
	next, _ := m.Update(cmd())
	m = next.(Model)
	request, err := collection.LoadRequest(filepath.Join(m.view.Root, ".api", "requests", "one.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := request.Request.Body.Content.(map[string]any)["b"]; got != 2 {
		t.Fatalf("saved JSON content = %#v", request.Request.Body.Content)
	}
}

func TestRawModeSavePreservesExactText(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.editor.SwitchBodyMode(BodyModeRaw)
	body := "  first line\r\nsecond line \n"
	m.editor.SetBodyText(body)
	cmd := m.Save()
	if cmd == nil {
		t.Fatalf("Save() rejected raw body: %s", m.message)
	}
	next, _ := m.Update(cmd())
	m = next.(Model)
	request, err := collection.LoadRequest(filepath.Join(m.view.Root, ".api", "requests", "one.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := request.Request.Body.Content; got != body {
		t.Fatalf("saved raw body = %q, want %q", got, body)
	}
}

func TestDuplicateDestinationCanBeEditedBeforeSave(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.beginDuplicate()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("copy")})
	m = next.(Model)
	if !strings.Contains(m.View(), "Save as: copy") {
		t.Fatalf("duplicate destination view = %q", m.View())
	}
	cmd := m.Save()
	if cmd == nil {
		t.Fatalf("saving duplicate failed: %s", m.message)
	}
	next, _ = m.Update(cmd())
	m = next.(Model)
	if _, ok := m.view.Tree.Requests["copy"]; !ok {
		t.Fatalf("duplicate was not saved at new path: %v", m.view.Tree.RequestIDs)
	}
}

func TestDeleteRequiresAffirmativeConfirmation(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	groupPath := filepath.Join(m.view.Root, ".api", "requests", "nested")
	m.beginDelete("nested", true)
	if _, err := os.Stat(groupPath); err != nil {
		t.Fatalf("group removed before confirmation: %v", err)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = next.(Model)
	if cmd == nil {
		t.Fatal("affirmative delete did not issue a delete command")
	}
	next, _ = m.Update(cmd())
	m = next.(Model)
	if _, err := os.Stat(groupPath); !os.IsNotExist(err) {
		t.Fatalf("group remains after confirmation: %v", err)
	}
}

func TestDiscardPromptLeavesYAMLUnchanged(t *testing.T) {
	m := editorWithChangedURL(t)
	path := filepath.Join(m.view.Root, ".api", "requests", "one.yaml")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyDown})
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("discard choice changed request YAML")
	}
}

func updateModel(m Model, msg tea.Msg) (Model, tea.Cmd) {
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

func TestEditorKeyboardEditsTypedURLAndCtrlSSaves(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.editorField, m.replaceField = 0, true
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyTab})
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyTab})
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("https://edited.example.test")})
	m, cmd := updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatalf("Ctrl+S did not save: %s", m.message)
	}
	m, _ = updateModel(m, cmd())
	request, err := collection.LoadRequest(filepath.Join(m.view.Root, ".api", "requests", "one.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if request.Request.URL != "https://edited.example.test" {
		t.Fatalf("saved URL = %q", request.Request.URL)
	}
}

func TestDirtyNavigationSaveAndContinueOpensTarget(t *testing.T) {
	m := editorWithChangedURL(t)
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyDown})
	if !strings.Contains(m.View(), "Save and continue") {
		t.Fatalf("prompt = %q", m.View())
	}
	m, cmd := updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if cmd == nil {
		t.Fatalf("Save and continue did not save: %s", m.message)
	}
	m, _ = updateModel(m, cmd())
	if m.editor.Selection().RequestID == "one" {
		t.Fatal("navigation did not continue to another request")
	}
	if m.editor.Dirty() {
		t.Fatal("new request unexpectedly has dirty editor state")
	}
}

func TestCommandPaletteOpensDuplicateFlow(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyCtrlP})
	if !strings.Contains(m.View(), "Duplicate request") {
		t.Fatalf("palette = %q", m.View())
	}
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if !strings.Contains(m.View(), "Save as:") || !m.editor.Dirty() {
		t.Fatalf("duplicate view = %q", m.View())
	}
}

func TestSwitchingToRawUsesCurrentStructuredJSON(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.editor.SetBodyText(`{"a":2}`)
	m.editor.SwitchBodyMode(BodyModeRaw)
	if m.editor.Validation() != "" {
		t.Fatalf("body mode switch failed: %s", m.editor.Validation())
	}
	if got := m.editor.Request().Request.Body.Content; got != "{\n  \"a\": 2\n}" {
		t.Fatalf("raw mode body = %#v", got)
	}
}

func TestInvalidJSONCannotSwitchToRawMode(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.editor.SetBodyText(`{"a":`)
	m.editor.SwitchBodyMode(BodyModeRaw)
	if m.editor.mode != BodyModeJSON {
		t.Fatal("invalid JSON switched to raw mode")
	}
	if m.editor.Validation() == "" {
		t.Fatal("invalid JSON switch lacked validation feedback")
	}
}

func TestSaveValidationMessageVisibleInEditor(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.editor.SetURL("")
	if cmd := m.Save(); cmd != nil {
		t.Fatal("invalid request unexpectedly saved")
	}
	if !strings.Contains(m.View(), "Status:") || !strings.Contains(m.View(), "required") {
		t.Fatalf("editor view lacks validation status: %q", m.View())
	}
}

func TestEOpensTypedRequestEditor(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.editor = nil
	m.mode = browseMode
	m.treeIndex = indexRow(m.visibleRows(), "one")
	m.handleRune("e")
	if m.mode != requestEditMode || m.editor == nil {
		t.Fatal("e did not open the selected request editor")
	}
}

func TestBodylessRequestCanStartInRawMode(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.editor.Load(validRequest("Empty body", "https://example.test/empty", nil))
	m.editor.SwitchBodyMode(BodyModeRaw)
	if m.editor.Validation() != "" || m.editor.mode != BodyModeRaw {
		t.Fatalf("raw mode unavailable for bodyless request: %s", m.editor.Validation())
	}
	m.editor.SetBodyText("<xml/>")
	request, err := m.editor.validatedRequest()
	if err != nil {
		t.Fatal(err)
	}
	if request.Request.Body == nil || request.Request.Body.Content != "<xml/>" {
		t.Fatalf("raw body = %#v", request.Request.Body)
	}
}

func TestUnrelatedEditPreservesLargeJSONInteger(t *testing.T) {
	m := editorWithJSON(t, `{"large":9007199254740993}`)
	m.editor.SetURL("https://example.test/changed")
	cmd := m.Save()
	if cmd == nil {
		t.Fatalf("Save() failed: %s", m.message)
	}
	next, _ := m.Update(cmd())
	m = next.(Model)
	data, err := os.ReadFile(filepath.Join(m.view.Root, ".api", "requests", "one.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "9007199254740993") {
		t.Fatalf("large integer changed during unrelated save: %s", data)
	}
}

func TestKeyboardCompositionPreservesHeadersAndParams(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	typeText := func(field int, text string) {
		m.editorField, m.replaceField = field, true
		for _, r := range text {
			m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
		m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyTab})
	}
	typeText(4, `{"Accept":"application/json, text/plain "}`)
	typeText(3, `{"q":" a,b "}`)
	request := m.editor.Request()
	if got := request.Request.Headers["Accept"]; got != "application/json, text/plain " {
		t.Fatalf("Accept header = %q", got)
	}
	if got := request.Request.Params["q"]; got != " a,b " {
		t.Fatalf("query parameter = %q", got)
	}
}

func TestDiscardNavigationOpensRequestedNestedRequest(t *testing.T) {
	m := editorWithChangedURL(t)
	ids := m.view.Tree.RequestIDs
	current := m.editor.Selection().RequestID
	index := 0
	for i, id := range ids {
		if id == current {
			index = i
			break
		}
	}
	target := ids[(index+1)%len(ids)]
	m.requestNavigation(1)
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if m.editor == nil || m.editor.Selection().RequestID != target {
		t.Fatalf("discard opened %#v, want %q", m.editor, target)
	}
}

func TestEditorLocksInputUntilSaveCompletes(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.editorField, m.replaceField = 2, false
	cmd := m.Save()
	if cmd == nil {
		t.Fatal("Save() returned no command")
	}
	before := m.editor.Request().Request.URL
	m, _ = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if got := m.editor.Request().Request.URL; got != before {
		t.Fatalf("editor accepted input while saving: %q", got)
	}
	m, _ = updateModel(m, cmd())
	if got := m.editor.Request().Request.URL; got != before {
		t.Fatalf("save completion changed URL to %q", got)
	}
}

func TestDuplicateCannotOverwriteNormalizedSourcePath(t *testing.T) {
	for _, target := range []string{"./one", "nested/../one"} {
		t.Run(target, func(t *testing.T) {
			m := editorWithJSON(t, `{"a":1}`)
			path := filepath.Join(m.view.Root, ".api", "requests", "one.yaml")
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			m.beginDuplicate()
			m.setEditorFieldText(target)
			cmd := m.Save()
			if cmd == nil {
				t.Fatalf("duplicate command was not issued: %s", m.message)
			}
			m, _ = updateModel(m, cmd())
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != string(before) {
				t.Fatalf("duplicate overwrote source via %q", target)
			}
		})
	}
}

func TestDuplicateDoesNotOverwriteDestinationCreatedAfterFlowStarts(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.beginDuplicate()
	path := filepath.Join(m.view.Root, ".api", "requests", "one-copy.yaml")
	if err := os.WriteFile(path, []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := m.Save()
	if cmd == nil {
		t.Fatal("duplicate save command was not issued")
	}
	m, _ = updateModel(m, cmd())
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep me\n" {
		t.Fatalf("late destination was overwritten: %s", data)
	}
}

func TestEditorMarksActiveFieldAndPreservesThreePaneShell(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.editorField = 2
	view := m.View()
	for _, want := range []string{"Collections / tree", "Request editor", "Response / Diagnostics", "▶ URL:"} {
		if !strings.Contains(view, want) {
			t.Fatalf("editor view lacks %q: %s", want, view)
		}
	}
}

func TestSavedDuplicateCanBeEditedAgain(t *testing.T) {
	m := editorWithJSON(t, `{"a":1}`)
	m.beginDuplicate()
	if cmd := m.Save(); cmd == nil {
		t.Fatalf("saving duplicate: %s", m.message)
	} else {
		m, _ = updateModel(m, cmd())
	}
	m.editor.SetURL("https://example.test/updated-copy")
	if cmd := m.Save(); cmd == nil {
		t.Fatalf("saving edited copy: %s", m.message)
	} else {
		m, _ = updateModel(m, cmd())
	}
	if m.message != "Saved one-copy" {
		t.Fatalf("second save message = %q", m.message)
	}
}

func TestAuthFieldTextPreservesTypedOAuthConfiguration(t *testing.T) {
	request := validRequest("OAuth", "https://example.test", nil)
	request.Auth = &model.Auth{Type: "oauth2", Grant: "client_credentials", TokenURL: "https://auth.example.test/token", ClientID: "client", ClientSecret: "${CLIENT_SECRET}", Scopes: []string{"read", "write"}}
	editor := newRequestEditor(nil, app.Selection{}, request)
	text := editor.FieldText(5)
	editor.SetFieldText(5, text)
	if got := editor.Request().Auth; !reflect.DeepEqual(got, request.Auth) {
		t.Fatalf("OAuth config after field round trip = %#v, want %#v", got, request.Auth)
	}
}
