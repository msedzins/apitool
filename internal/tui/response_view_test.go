package tui

import (
	"apitool/internal/app"
	"apitool/internal/auth"
	"apitool/internal/model"
	"apitool/internal/runtime"
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHTTP401RendersResponseNotFailure(t *testing.T) {
	m, _ := requestScreen(t, "GET", false)
	next, _ := m.Update(sendFinishedMsg{result: app.SendResult{Response: &model.Response{StatusCode: 401}}})
	if !strings.Contains(next.View(), "401 Unauthorized") || strings.Contains(next.View(), "Request failed") {
		t.Fatal(next.View())
	}
}
func TestResponseViewsAndDiagnostics(t *testing.T) {
	for _, code := range []int{302, 401, 500} {
		m, _ := requestScreen(t, "GET", false)
		m.width, m.height = 110, 32
		next, _ := m.Update(sendFinishedMsg{result: app.SendResult{Response: &model.Response{StatusCode: code, Duration: time.Millisecond, Headers: http.Header{"Set-Cookie": {"secret"}, "Content-Type": {"application/json"}}, Body: []byte(`{"ok":true}`)}}})
		view := next.View()
		for _, want := range []string{http.StatusText(code), "1ms", "11 bytes", "Content-Type", "[REDACTED]", "\"ok\": true"} {
			if !strings.Contains(view, want) {
				t.Fatalf("missing %s: %s", want, view)
			}
		}
		if strings.Contains(view, "secret") {
			t.Fatal(view)
		}
	}
	for _, x := range []*model.ExecutionError{{Stage: model.StageOAuth, Category: model.CategoryOAuth, SafeMessage: "OAuth failed"}, {Stage: model.StageTransport, Category: model.CategoryDNS, SafeMessage: "DNS failed"}} {
		m, _ := requestScreen(t, "GET", false)
		next, _ := m.Update(sendFinishedMsg{result: app.SendResult{ExecutionError: x, Logs: []runtime.LogEntry{{Method: "GET", Data: map[string]any{"access_token": "secret"}}}}})
		view := next.View()
		for _, want := range []string{string(x.Stage), string(x.Category), x.SafeMessage, "Request Log"} {
			if !strings.Contains(view, want) {
				t.Fatalf("missing %s: %s", want, view)
			}
		}
		if strings.Contains(view, "secret") {
			t.Fatal(view)
		}
	}
}
func TestPrettyBodyFallsBackToRaw(t *testing.T) {
	if got := prettyBody([]byte("not json")); got != "not json" {
		t.Fatal(got)
	}
	if got := prettyBody([]byte(`{"x":1}`)); got != "{\n  \"x\": 1\n}" {
		t.Fatal(got)
	}
}

type sessionTokenProvider struct{}

func (sessionTokenProvider) Token(_ context.Context, _ model.Auth) (auth.Token, error) {
	return auth.Token{AccessToken: "session-secret", TokenType: "Bearer", Expiry: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)}, nil
}
func TestAuthTokenMaskedAndRevealIsSessionOnly(t *testing.T) {
	m, _ := requestScreenWithAuth(t, "GET", false, sessionTokenProvider{})
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if cmd == nil {
		t.Fatalf("Auth must load asynchronously: %s", next.View())
	}
	next, _ = next.Update(cmd())
	view := next.View()
	for _, want := range []string{"client_credentials", "oauth.example.test/token", "read, write", "2030", "[REDACTED]", "Show token"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %s: %s", want, view)
		}
	}
	if strings.Contains(view, "session-secret") || strings.Contains(view, "hidden-client-secret") {
		t.Fatal(view)
	}
	next, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if !strings.Contains(next.View(), "session-secret") {
		t.Fatal("explicit reveal failed")
	}
	fresh := New(m.service, Options{StartingCollection: "demo"})
	fresh, cmd = fresh.Update(tea.KeyMsg{Type: tea.KeyTab})
	fresh, cmd = fresh.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if cmd == nil {
		t.Fatal("fresh Auth load missing")
	}
	fresh, _ = fresh.Update(cmd())
	if strings.Contains(fresh.View(), "session-secret") {
		t.Fatal("reveal leaked to new session")
	}
	workspace, _ := m.service.Workspace()
	err := filepath.WalkDir(filepath.Join(workspace.Root, ".apitool"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(data), "session-secret") {
				t.Errorf("persisted token in %s", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestResponsePaneScrollAndRawToggle(t *testing.T) {
	m, _ := requestScreen(t, "GET", false)
	m.width, m.height = 100, 24
	m.focus = responsePane
	next, _ := m.Update(sendFinishedMsg{result: app.SendResult{Response: &model.Response{StatusCode: 500, Body: []byte(`{"many":[1,2,3,4,5,6,7,8,9]}`)}}})
	if strings.Contains(next.View(), "9") {
		t.Fatal("body should exceed pane")
	}
	for i := 0; i < 20; i++ {
		next, _ = next.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if !strings.Contains(next.View(), "9") {
		t.Fatal("body cannot scroll")
	}
	next, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	if !strings.Contains(next.View(), `{"many":[1,2,3,4,5,6,7,8,9]}`) {
		t.Fatal("raw body lost")
	}
}

func TestCompactResponseScrollsThroughWrappedBody(t *testing.T) {
	m, _ := requestScreen(t, "GET", false)
	m.width, m.height = 60, 10
	m.focus = responsePane
	next, _ := m.Update(sendFinishedMsg{result: app.SendResult{Response: &model.Response{StatusCode: 200, Body: []byte("first\nsecond\nthird\nfourth\nfifth\nlast-visible-body-line")}}})
	before := next.View()
	for i := 0; i < 30; i++ {
		next, _ = next.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if next.View() == before || !strings.Contains(next.View(), "last-visible-body-line") {
		t.Fatalf("compact scroll did not reveal body tail: %s", next.View())
	}
}
