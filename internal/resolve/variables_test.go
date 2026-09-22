package resolve_test

import (
	"testing"

	"apitool/internal/model"
	"apitool/internal/resolve"
)

func TestEffectiveResolvesNestedJSONStringsBeforeMarshaling(t *testing.T) {
	request := model.Request{
		Name: "Create", Method: "POST",
		Request: model.RequestConfig{URL: "https://api.example.test/items", Body: &model.Body{
			Type:    "json",
			Content: map[string]any{"items": []any{map[string]any{"name": "{{name}}"}}},
		}},
	}
	environment := testEnvironment()
	environment.Variables["name"] = "resolved item"

	got, diagnostics := resolve.Effective(model.Collection{Name: "API"}, environment, nil, request)
	if len(diagnostics) != 0 {
		t.Fatalf("Effective() diagnostics = %#v, want none", diagnostics)
	}
	if string(got.Body) != `{"items":[{"name":"resolved item"}]}` {
		t.Fatalf("Effective().Body = %s, want resolved JSON", got.Body)
	}
}

func TestEffectiveOrdersMapVariableDiagnosticsByFieldPath(t *testing.T) {
	request := model.Request{
		Name: "Create", Method: "POST",
		Request: model.RequestConfig{
			URL: "https://api.example.test/items",
			Params: map[string]string{
				"z-last":  "{{missing_z}}",
				"a-first": "{{missing_a}}",
			},
		},
	}

	_, diagnostics := resolve.Effective(model.Collection{Name: "API"}, testEnvironment(), nil, request)
	if len(diagnostics) != 2 {
		t.Fatalf("Effective() diagnostics = %#v, want two missing variables", diagnostics)
	}
	if diagnostics[0].Path != "request.params.a-first" || diagnostics[1].Path != "request.params.z-last" {
		t.Fatalf("Effective() diagnostic paths = %v, want field-path order", diagnosticPaths(diagnostics))
	}
}
