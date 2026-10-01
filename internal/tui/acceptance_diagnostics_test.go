package tui

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"apitool/internal/app"
	"apitool/internal/model"
	"apitool/internal/runtime"
)

func TestAcceptanceUI011ExecutionFailuresRenderSafeDiagnostics(t *testing.T) {
	t.Run("resolution stage is identified by an unresolved request variable", func(t *testing.T) {
		m, _ := requestScreen(t, http.MethodGet, false)
		workspace, err := m.service.Workspace()
		if err != nil {
			t.Fatal(err)
		}
		requestPath := filepath.Join(workspace.Root, "demo", ".api", "requests", "item.yaml")
		const marker = "resolution-log-secret-marker"
		if err := os.WriteFile(requestPath, []byte("name: Item\nmethod: GET\nrequest:\n  url: '{{missing_host}}/items?access_token="+marker+"'\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := m.service.ReloadWorkspace(context.Background()); err != nil {
			t.Fatal(err)
		}
		selection, ok := m.executionSelection()
		if !ok {
			t.Fatal("request selection is unavailable")
		}
		result := m.service.Send(context.Background(), selection)
		if result.Response != nil || result.ExecutionError == nil || result.ExecutionError.Stage != model.StageResolution || result.ExecutionError.Category != model.CategoryResolution {
			t.Fatalf("unresolved variable result = %#v", result)
		}
		if len(result.Logs) != 1 || result.Logs[0].ErrorCategory != model.CategoryResolution || result.Logs[0].Path != "" {
			t.Fatalf("resolution failure log = %#v; expected one safe log entry without unresolved URL data", result.Logs)
		}
		persistedLog, err := os.ReadFile(filepath.Join(workspace.Root, ".apitool", "logs", "executions.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`"timestamp":`, `"method":"GET"`, `"error_category":"resolution"`} {
			if !strings.Contains(string(persistedLog), want) {
				t.Fatalf("persisted resolution log missing %q: %s", want, persistedLog)
			}
		}
		if strings.Contains(string(persistedLog), marker) {
			t.Fatalf("persisted resolution log leaked an unresolved URL secret: %s", persistedLog)
		}
		finished := finishSend(m, result).(Model)
		view := strings.Join(finished.responseLines(), "\n")
		for _, want := range []string{"Stage: resolution", "Category: resolution", "missing_host", "request.url", "Request Log", "Category: resolution"} {
			if !strings.Contains(view, want) {
				t.Fatalf("resolution diagnostic missing %q: %s", want, view)
			}
		}
		if strings.Contains(view, marker) {
			t.Fatalf("resolution failure log leaked an unresolved URL secret: %s", view)
		}
	})

	for _, failure := range []model.ExecutionError{
		{Stage: model.StageRequestBuild, Category: model.CategoryRequestBuild, SafeMessage: "Request definition is invalid"},
		{Stage: model.StageOAuth, Category: model.CategoryOAuth, SafeMessage: "OAuth token request failed"},
		{Stage: model.StageTransport, Category: model.CategoryTLS, SafeMessage: "TLS connection failed"},
		{Stage: model.StageTransport, Category: model.CategoryTimeout, SafeMessage: "Request timed out"},
		{Stage: model.StageTransport, Category: model.CategoryCanceled, SafeMessage: "Request canceled"},
	} {
		t.Run(string(failure.Stage)+"/"+string(failure.Category), func(t *testing.T) {
			m, _ := requestScreen(t, "GET", false)
			result := app.SendResult{
				ExecutionError: &failure,
				Logs:           []runtime.LogEntry{{Method: "GET", Path: "/items?access_token=[REDACTED]", Data: map[string]any{"access_token": "do-not-render-this-secret"}}},
			}
			next := finishSend(m, result)
			view := next.View()
			for _, want := range []string{"Request failed", "Stage: " + string(failure.Stage), "Category: " + string(failure.Category), failure.SafeMessage, "Request Log"} {
				if !strings.Contains(view, want) {
					t.Fatalf("view missing %q: %s", want, view)
				}
			}
			if strings.Contains(view, "200 OK") || strings.Contains(view, "do-not-render-this-secret") {
				t.Fatalf("execution failure rendered as HTTP response or exposed secret: %s", view)
			}
		})
	}
}

func TestAcceptanceUI021CacheWriteFailureKeepsHTTPResponseInspectable(t *testing.T) {
	m, _ := requestScreen(t, http.MethodGet, false)
	workspace, err := m.service.Workspace()
	if err != nil {
		t.Fatal(err)
	}
	responsesPath := filepath.Join(workspace.Root, ".apitool", "responses")
	if err := os.RemoveAll(responsesPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(responsesPath, []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}

	selection, ok := m.executionSelection()
	if !ok {
		t.Fatal("request selection is unavailable")
	}
	result := m.service.Send(context.Background(), selection)
	if result.Response == nil || result.Response.StatusCode != http.StatusUnauthorized || result.ExecutionError != nil {
		t.Fatalf("cache failure changed completed HTTP result: %#v", result)
	}
	finished := finishSend(m, result).(Model)
	view := strings.Join(finished.responseLines(), "\n")
	for _, want := range []string{"401 Unauthorized", "runtime_cache_write", ".apitool/responses", "could not save response cache", "Body (pretty"} {
		if !strings.Contains(view, want) {
			t.Fatalf("response view missing %q: %s", want, view)
		}
	}
	if strings.Contains(view, "Request failed") {
		t.Fatalf("storage failure was presented as an HTTP execution failure: %s", view)
	}
}
