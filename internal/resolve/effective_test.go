package resolve_test

import (
	"strings"
	"testing"
	"time"

	"apitool/internal/model"
	"apitool/internal/resolve"
)

func TestEffectiveUsesNearestAuthAndAuthNoneDisablesInheritance(t *testing.T) {
	got, diagnostics := resolve.Effective(collectionWithOAuth(), testEnvironment(), []model.Group{adminGroup()}, requestWithAuthNone())
	if len(diagnostics) != 0 {
		t.Fatalf("Effective() diagnostics = %#v, want none", diagnostics)
	}
	if got.Auth == nil || !got.Auth.None {
		t.Fatalf("Effective().Auth = %#v, want auth none", got.Auth)
	}
}

func TestEffectiveRejectsUnresolvedVariableInNestedJSONWithoutLeakingValue(t *testing.T) {
	request := model.Request{
		Name: "Create credentials", Method: "POST",
		Request: model.RequestConfig{URL: "https://api.example.test/credentials", Body: &model.Body{
			Type: "json",
			Content: map[string]any{"credentials": map[string]any{
				"key": "{{missing_key}} plaintext-secret",
			}},
		}},
	}
	_, diagnostics := resolve.Effective(model.Collection{Name: "API"}, testEnvironment(), nil, request)
	if !hasDiagnosticCode(diagnostics, "variable_missing") {
		t.Fatalf("Effective() diagnostic codes = %v, want variable_missing", diagnosticCodes(diagnostics))
	}
	if messages := joinedMessages(diagnostics); strings.Contains(messages, "plaintext-secret") {
		t.Fatalf("Effective() diagnostics leaked secret: %q", messages)
	}
	if !hasDiagnosticPath(diagnostics, "request.body.content.credentials.key") {
		t.Fatalf("Effective() diagnostic paths = %v, want nested body path", diagnosticPaths(diagnostics))
	}
}

func TestEffectiveResolvesScalarRequestFieldsAndAuth(t *testing.T) {
	oldLookup := resolve.LookupEnv
	resolve.LookupEnv = func(name string) (string, bool) {
		switch name {
		case "API_CLIENT_ID":
			return "client-from-process", true
		case "API_HOST":
			return "gateway.example.test", true
		case "API_VERSION":
			return "v1", true
		}
		return "", false
	}
	t.Cleanup(func() { resolve.LookupEnv = oldLookup })

	request := model.Request{
		Name: "List", Method: "get",
		Request: model.RequestConfig{
			URL:     "{{base_url}}/${API_VERSION}/users",
			Params:  map[string]string{"page": "{{page}}"},
			Headers: map[string]string{"X-Tenant": "{{tenant}}-${API_VERSION}"},
		},
	}
	collection := collectionWithOAuth()
	collection.Auth.ClientID = "${API_CLIENT_ID}"
	collection.Auth.TokenURL = "{{token_url}}"
	environment := testEnvironment()
	environment.Variables["base_url"] = "https://{{host}}/${API_HOST}"
	environment.Variables["host"] = "not-recursed.example.test"
	environment.Variables["page"] = "2"
	environment.Variables["tenant"] = "payments"
	environment.Variables["token_url"] = "https://auth.example.test/token"

	got, diagnostics := resolve.Effective(collection, environment, nil, request)
	if len(diagnostics) != 0 {
		t.Fatalf("Effective() diagnostics = %#v, want none", diagnostics)
	}
	if got.Method != "GET" || got.URL != "https://{{host}}/gateway.example.test/v1/users" {
		t.Fatalf("Effective() method, URL = %q, %q", got.Method, got.URL)
	}
	if got.Params["page"] != "2" || got.Headers["X-Tenant"] != "payments-v1" {
		t.Fatalf("Effective() params, headers = %#v, %#v", got.Params, got.Headers)
	}
	if got.Auth == nil || got.Auth.ClientID != "client-from-process" || got.Auth.TokenURL != "https://auth.example.test/token" {
		t.Fatalf("Effective().Auth = %#v, want resolved auth", got.Auth)
	}
}

func TestEffectiveReportsMissingProcessVariableWithoutValue(t *testing.T) {
	oldLookup := resolve.LookupEnv
	resolve.LookupEnv = func(string) (string, bool) { return "", false }
	t.Cleanup(func() { resolve.LookupEnv = oldLookup })

	request := model.Request{Name: "List", Method: "GET", Request: model.RequestConfig{URL: "https://api.example.test/${MISSING_SECRET}"}}
	_, diagnostics := resolve.Effective(model.Collection{Name: "API"}, testEnvironment(), nil, request)
	if !hasDiagnosticCode(diagnostics, "variable_missing") {
		t.Fatalf("Effective() diagnostic codes = %v, want variable_missing", diagnosticCodes(diagnostics))
	}
	if messages := joinedMessages(diagnostics); strings.Contains(messages, "MISSING_SECRET") == false || strings.Contains(messages, "https://") {
		t.Fatalf("Effective() messages = %q, want only safe variable detail", messages)
	}
}

func TestEffectiveDoesNotUseRequestValuesAsVariables(t *testing.T) {
	request := model.Request{
		Name: "List", Method: "GET",
		Request: model.RequestConfig{
			URL:    "{{request_local}}/users",
			Params: map[string]string{"request_local": "https://would-be-local.example.test"},
		},
	}
	_, diagnostics := resolve.Effective(model.Collection{Name: "API"}, testEnvironment(), nil, request)
	if !hasDiagnosticCode(diagnostics, "variable_missing") {
		t.Fatalf("Effective() diagnostic codes = %v, want variable_missing", diagnosticCodes(diagnostics))
	}
}

func TestEffectiveUsesNearestDefinedGroupAuth(t *testing.T) {
	outer := model.Group{Auth: &model.Auth{Type: "oauth2", Grant: "client_credentials", Scopes: []string{"outer.read"}}}
	inner := model.Group{Auth: &model.Auth{Type: "oauth2", Grant: "client_credentials", Scopes: []string{"inner.write"}}}
	got, diagnostics := resolve.Effective(collectionWithOAuth(), testEnvironment(), []model.Group{outer, inner}, basicRequest())
	if len(diagnostics) != 0 {
		t.Fatalf("Effective() diagnostics = %#v, want none", diagnostics)
	}
	if got.Auth == nil || len(got.Auth.Scopes) != 1 || got.Auth.Scopes[0] != "inner.write" {
		t.Fatalf("Effective().Auth = %#v, want nearest group auth", got.Auth)
	}
}

func TestEffectiveMergesEnvironmentHTTPOverrides(t *testing.T) {
	for _, test := range []struct {
		name         string
		environment  *model.HTTPConfig
		wantTimeout  time.Duration
		wantInsecure bool
	}{
		{
			name:        "environment timeout and false TLS override collection defaults",
			environment: &model.HTTPConfig{Timeout: "5s", InsecureSkipVerify: boolPointer(false)},
			wantTimeout: 5 * time.Second,
		},
		{
			name:        "environment retains collection timeout when absent",
			environment: &model.HTTPConfig{InsecureSkipVerify: boolPointer(false)},
			wantTimeout: 45 * time.Second,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			collection := model.Collection{Name: "API", HTTP: &model.HTTPConfig{Timeout: "45s", InsecureSkipVerify: boolPointer(true)}}
			environment := testEnvironment()
			environment.HTTP = test.environment

			got, diagnostics := resolve.Effective(collection, environment, nil, basicRequest())
			if len(diagnostics) != 0 {
				t.Fatalf("Effective() diagnostics = %#v, want none", diagnostics)
			}
			if got.Timeout != test.wantTimeout {
				t.Fatalf("Effective().Timeout = %s, want %s", got.Timeout, test.wantTimeout)
			}
			if got.InsecureSkipVerify != test.wantInsecure {
				t.Fatalf("Effective().InsecureSkipVerify = %t, want %t", got.InsecureSkipVerify, test.wantInsecure)
			}
		})
	}
}

func TestEffectiveDefaultsToNoTimeoutAndVerifiedTLS(t *testing.T) {
	got, diagnostics := resolve.Effective(model.Collection{Name: "API"}, testEnvironment(), nil, basicRequest())
	if len(diagnostics) != 0 {
		t.Fatalf("Effective() diagnostics = %#v, want none", diagnostics)
	}
	if got.Timeout != 0 {
		t.Fatalf("Effective().Timeout = %s, want no timeout", got.Timeout)
	}
	if got.InsecureSkipVerify {
		t.Fatal("Effective().InsecureSkipVerify = true, want verified TLS by default")
	}
}

func collectionWithOAuth() model.Collection {
	return model.Collection{Name: "API", Auth: &model.Auth{
		Type: "oauth2", Grant: "client_credentials", TokenURL: "https://auth.example.test/token", Scopes: []string{"collection.read"},
	}}
}

func adminGroup() model.Group {
	return model.Group{Auth: &model.Auth{Type: "oauth2", Grant: "client_credentials", Scopes: []string{"admin.write"}}}
}

func requestWithAuthNone() model.Request {
	request := basicRequest()
	request.Auth = &model.Auth{None: true}
	return request
}

func basicRequest() model.Request {
	return model.Request{Name: "List", Method: "GET", Request: model.RequestConfig{URL: "https://api.example.test/users"}}
}

func testEnvironment() model.Environment {
	return model.Environment{Name: "test", Variables: map[string]string{"base_url": "https://api.example.test"}}
}

func boolPointer(value bool) *bool { return &value }

func diagnosticCodes(diagnostics []model.Diagnostic) []string {
	codes := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	return codes
}

func diagnosticPaths(diagnostics []model.Diagnostic) []string {
	paths := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		paths = append(paths, diagnostic.Path)
	}
	return paths
}

func hasDiagnosticCode(diagnostics []model.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func hasDiagnosticPath(diagnostics []model.Diagnostic, path string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Path == path {
			return true
		}
	}
	return false
}

func joinedMessages(diagnostics []model.Diagnostic) string {
	messages := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		messages = append(messages, diagnostic.Message)
	}
	return strings.Join(messages, "\n")
}
