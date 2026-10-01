package tui

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"apitool/internal/app"
	"apitool/internal/auth"
	"apitool/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

// These acceptance tests exercise resolved definitions through the TUI and a
// controlled HTTP endpoint.
func TestAcceptanceAPI001SendsResolvedEnvironmentAndProcessValues(t *testing.T) {
	var gotURL, gotHeader, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotURL, gotHeader, gotBody = r.URL.String(), r.Header.Get("X-Process"), string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	t.Setenv("ACCEPTANCE_PROTOCOL_HEADER", "process-header-value")
	root := acceptanceProtocolWorkspace(t, map[string]string{
		"demo/.api/environments/test.yaml": fmt.Sprintf("name: test\nvariables:\n  host: %s\n  path: env-path\n  env_header: '${ACCEPTANCE_PROTOCOL_HEADER}'\n  payload: environment-body\n", server.URL),
		"demo/.api/requests/item.yaml":     "name: Item\nmethod: POST\nrequest:\n  url: '{{host}}/{{path}}?q={{payload}}'\n  params:\n    from_env: '{{path}}'\n  headers:\n    X-Process: '{{env_header}}'\n  body:\n    type: json\n    content:\n      nested:\n        value: '{{payload}}'\n",
	})
	_, result := acceptanceProtocolSend(t, acceptanceProtocolModelWithDeps(t, root, "item", false, app.Dependencies{}))
	if result.Response == nil || result.Response.StatusCode != http.StatusNoContent {
		t.Fatalf("send result=%#v; expected controlled endpoint response", result)
	}
	if gotURL != "/env-path?from_env=env-path&q=environment-body" || gotHeader != "process-header-value" || gotBody != `{"nested":{"value":"environment-body"}}` {
		t.Fatalf("controlled endpoint received URL=%q header=%q body=%q", gotURL, gotHeader, gotBody)
	}
}

func TestAcceptanceAPI002MissingNestedVariableIsSafeAndBlocksSend(t *testing.T) {
	calls := 0
	deps := app.Dependencies{Execute: func(context.Context, model.EffectiveRequest, auth.TokenProvider) (model.Response, *model.ExecutionError) {
		calls++
		return model.Response{StatusCode: 204}, nil
	}}
	root := acceptanceProtocolWorkspace(t, map[string]string{
		"demo/.api/environments/test.yaml": "name: test\nvariables:\n  host: https://api.example.test\n  api_key: never-fallback-secret\n",
		"demo/.api/requests/item.yaml":     "name: Item\nmethod: POST\nrequest:\n  url: '{{host}}/items'\n  body:\n    type: json\n    content:\n      nested:\n        credential: '{{missing_value}}'\n",
	})
	m := acceptanceProtocolModelWithDeps(t, root, "item", false, deps)
	completed, result := acceptanceProtocolSend(t, m)
	view := completed.View()
	if calls != 0 || result.ExecutionError == nil || !strings.Contains(view, "missing_value") || !strings.Contains(view, "request.body.content.nested.credential") || strings.Contains(view, "never-fallback-secret") {
		t.Fatalf("calls=%d result=%#v diagnostic=%s", calls, result, view)
	}
}

func TestAcceptanceAPI003UsesNearestInheritedAuthentication(t *testing.T) {
	var gotAuthorization string
	var tokenClientID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			tokenClientID, _, _ = r.BasicAuth()
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"access_token":%q,"token_type":"Bearer","expires_in":3600}`, tokenClientID+"-token")
			return
		}
		gotAuthorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	t.Setenv("ACCEPTANCE_PROTOCOL_SECRET", "test-secret")
	root := acceptanceProtocolWorkspace(t, map[string]string{
		"demo/.api/collection.yaml":                   fmt.Sprintf("name: Demo\nauth:\n  type: oauth2\n  grant: client_credentials\n  token_url: %s/token\n  client_id: collection\n  client_secret: ${ACCEPTANCE_PROTOCOL_SECRET}\n", server.URL),
		"demo/.api/requests/group/_group.yaml":        fmt.Sprintf("name: Parent\nauth:\n  type: oauth2\n  grant: client_credentials\n  token_url: %s/token\n  client_id: parent\n  client_secret: ${ACCEPTANCE_PROTOCOL_SECRET}\n", server.URL),
		"demo/.api/requests/group/nested/_group.yaml": fmt.Sprintf("name: Nearest\nauth:\n  type: oauth2\n  grant: client_credentials\n  token_url: %s/token\n  client_id: nearest\n  client_secret: ${ACCEPTANCE_PROTOCOL_SECRET}\n", server.URL),
		"demo/.api/requests/group/nested/item.yaml":   fmt.Sprintf("name: Item\nmethod: GET\nrequest:\n  url: %s/api\n", server.URL),
	})
	_, result := acceptanceProtocolSend(t, acceptanceProtocolModelWithDeps(t, root, "group/nested/item", false, app.Dependencies{}))
	if result.Response == nil || gotAuthorization != "Bearer nearest-token" || tokenClientID != "nearest" {
		t.Fatalf("response=%#v token endpoint client=%q API Authorization=%q, want nearest group credentials", result.Response, tokenClientID, gotAuthorization)
	}
}

func TestAcceptanceAPI004ExplicitNoneDisablesInheritedAuthentication(t *testing.T) {
	var gotAuthorization string
	tokenRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			tokenRequests++
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"access_token":"unexpected-token","token_type":"Bearer","expires_in":3600}`)
			return
		}
		gotAuthorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	t.Setenv("ACCEPTANCE_PROTOCOL_SECRET", "test-secret")
	root := acceptanceProtocolWorkspace(t, map[string]string{
		"demo/.api/collection.yaml":    fmt.Sprintf("name: Demo\nauth:\n  type: oauth2\n  grant: client_credentials\n  token_url: %s/token\n  client_id: client\n  client_secret: ${ACCEPTANCE_PROTOCOL_SECRET}\n", server.URL),
		"demo/.api/requests/item.yaml": fmt.Sprintf("name: Item\nmethod: GET\nrequest:\n  url: %s/api\nauth: none\n", server.URL),
	})
	_, result := acceptanceProtocolSend(t, acceptanceProtocolModelWithDeps(t, root, "item", false, app.Dependencies{}))
	if result.Response == nil || gotAuthorization != "" || tokenRequests != 0 {
		t.Fatalf("result=%#v Authorization=%q token requests=%d; expected no inherited auth", result.Response, gotAuthorization, tokenRequests)
	}
}

func TestAcceptanceAPI007OAuthFailureShowsCodeWithoutSecrets(t *testing.T) {
	const secret = "acceptance-client-secret-marker"
	t.Setenv("ACCEPTANCE_PROTOCOL_SECRET", secret)
	deps := app.Dependencies{TokenProvider: acceptanceProtocolFailingTokenProvider{err: &auth.OAuthError{Code: "invalid_client"}}}
	root := acceptanceProtocolWorkspace(t, map[string]string{
		"demo/.api/collection.yaml":    "name: Demo\nauth:\n  type: oauth2\n  grant: client_credentials\n  token_url: https://auth.example.test/token\n  client_id: client\n  client_secret: ${ACCEPTANCE_PROTOCOL_SECRET}\n",
		"demo/.api/requests/item.yaml": "name: Item\nmethod: GET\nrequest:\n  url: https://api.example.test/api\n",
	})
	m := acceptanceProtocolModelWithDeps(t, root, "item", false, deps)
	completed, result := acceptanceProtocolSend(t, m)
	view := strings.Join(completed.responseLines(), "\n")
	if result.ExecutionError == nil || !strings.Contains(view, "invalid_client") || strings.Contains(view, secret) || strings.Contains(view, "Authorization: Basic") {
		t.Fatalf("OAuth failure result=%#v view=%s", result, view)
	}
}

func TestAcceptanceUI009DangerousSendConfirmationShowsSafeContextAndControlsRequest(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	root := acceptanceProtocolWorkspace(t, map[string]string{
		"demo/.api/environments/test.yaml": fmt.Sprintf("name: test\nvariables:\n  host: %s\n  secret_path: safe-item?access_token=acceptance-secret-marker\n", server.URL),
		"demo/.api/requests/item.yaml":     "name: Item\nmethod: DELETE\nrequest:\n  url: '{{host}}/{{secret_path}}'\n",
	})
	m := acceptanceProtocolModelWithDeps(t, root, "item", true, app.Dependencies{})
	next, cmd := m.Update(sendKey())
	view := next.View()
	if cmd != nil || calls != 0 || !strings.Contains(view, "Confirmation required") || !strings.Contains(view, "Environment: test") || !strings.Contains(view, "Method: DELETE") || !strings.Contains(view, server.URL+"/safe-item") || strings.Contains(view, "acceptance-secret-marker") {
		t.Fatalf("confirmation cmd=%v calls=%d view=%s", cmd, calls, view)
	}
	next, _ = next.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if calls != 0 {
		t.Fatalf("rejection sent %d requests", calls)
	}
	next, _ = next.Update(sendKey())
	next, _ = next.Update(tea.KeyMsg{Type: tea.KeyTab})
	next, cmd = next.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("accepted confirmation did not start request")
	}
	next, _ = next.Update(cmd())
	if calls != 1 {
		t.Fatalf("acceptance sent %d requests, want one", calls)
	}
}

func TestAcceptanceUI020CancellationResultRendersDiagnostic(t *testing.T) {
	m := acceptanceProtocolModelWithDeps(t, acceptanceProtocolWorkspace(t, nil), "item", false, app.Dependencies{})
	selection, ok := m.executionSelection()
	if !ok {
		t.Fatal("request selection is unavailable")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := m.service.Send(ctx, selection)
	if result.Response != nil || result.ExecutionError == nil || result.ExecutionError.Category != model.CategoryCanceled {
		t.Fatalf("pre-canceled HTTP execution result = %#v", result)
	}
	m = finishSend(m, result).(Model)
	lines := strings.Join(m.responseLines(), "\n")
	for _, expected := range []string{"Stage: transport", "Category: canceled", "Request canceled"} {
		if !strings.Contains(lines, expected) {
			t.Fatalf("diagnostic missing %q: %s", expected, lines)
		}
	}
	if strings.Contains(lines, "HTTP response") || strings.Contains(lines, "200 OK") {
		t.Fatalf("cancellation shown as response: %s", lines)
	}
}

func acceptanceProtocolWorkspace(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	for path, contents := range files {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for path, contents := range map[string]string{
		"demo/.api/collection.yaml":        "name: Demo\n",
		"demo/.api/environments/test.yaml": "name: test\n",
		"demo/.api/requests/item.yaml":     "name: Item\nmethod: GET\nrequest:\n  url: https://example.test\n",
	} {
		if _, exists := files[path]; exists {
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func acceptanceProtocolModelWithDeps(t *testing.T, root, requestID string, confirm bool, deps app.Dependencies) Model {
	t.Helper()
	service, err := app.New(deps)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.OpenWorkspace(context.Background(), root, app.OpenOptions{Collection: "demo", Environment: "test", ConfirmDangerous: confirm}); err != nil {
		t.Fatal(err)
	}
	m := New(service, Options{StartingCollection: "demo", ConfirmDangerous: confirm}).(Model)
	for i := 0; i < len(m.visibleRows()); i++ {
		rows := m.visibleRows()
		if m.treeIndex < len(rows) && rows[m.treeIndex].kind == requestRow && rows[m.treeIndex].id == requestID {
			return m
		}
		m, _ = workflowUpdate(m, tea.KeyMsg{Type: tea.KeyDown})
	}
	t.Fatalf("request %q not found in TUI tree: %s", requestID, m.View())
	return Model{}
}

func acceptanceProtocolSend(t *testing.T, m Model) (Model, app.SendResult) {
	t.Helper()
	next, cmd := m.Update(sendKey())
	if cmd == nil {
		t.Fatalf("send command missing: %s", next.View())
	}
	updated, _ := next.Update(cmd())
	model := updated.(Model)
	return model, model.result
}

type acceptanceProtocolFailingTokenProvider struct{ err error }

func (p acceptanceProtocolFailingTokenProvider) Token(context.Context, model.Auth) (auth.Token, error) {
	return auth.Token{}, p.err
}
