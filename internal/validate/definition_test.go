package validate_test

import (
	"math"
	"testing"

	"apitool/internal/model"
	"apitool/internal/validate"
)

func TestDefinitionRejectsLiteralClientSecret(t *testing.T) {
	got := validate.Definition(model.Request{Auth: &model.Auth{
		Type: "oauth2", Grant: "client_credentials",
		ClientSecret: "plaintext-secret",
	}})
	if !containsDiagnosticCode(got, "secret_literal") {
		t.Fatalf("Definition() codes = %v, want secret_literal", diagnosticCodes(got))
	}
}

func TestDefinitionRequiresRequestURL(t *testing.T) {
	got := validate.Definition(model.Request{Name: "List users", Method: "GET"})
	if !containsDiagnosticCode(got, "request_url_required") {
		t.Fatalf("Definition() codes = %v, want request_url_required", diagnosticCodes(got))
	}
}

func TestDefinitionRejectsNonJSONSerializableJSONBody(t *testing.T) {
	got := validate.Definition(model.Request{
		Name:   "Create user",
		Method: "POST",
		Request: model.RequestConfig{
			URL:  "https://api.example.test/users",
			Body: &model.Body{Type: "json", Content: math.Inf(1)},
		},
	})
	if !containsDiagnosticCode(got, "body_json_invalid") {
		t.Fatalf("Definition() codes = %v, want body_json_invalid", diagnosticCodes(got))
	}
}

func TestDefinitionRejectsUnsupportedOAuthGrant(t *testing.T) {
	got := validate.Definition(model.Request{
		Name:    "List users",
		Method:  "GET",
		Request: model.RequestConfig{URL: "https://api.example.test/users"},
		Auth:    &model.Auth{Type: "oauth2", Grant: "authorization_code"},
	})
	if !containsDiagnosticCode(got, "auth_grant_unsupported") {
		t.Fatalf("Definition() codes = %v, want auth_grant_unsupported", diagnosticCodes(got))
	}
}

func TestDefinitionRejectsUnsupportedBodyType(t *testing.T) {
	got := validate.Definition(model.Request{
		Name:   "Upload",
		Method: "POST",
		Request: model.RequestConfig{
			URL:  "https://api.example.test/files",
			Body: &model.Body{Type: "multipart"},
		},
	})
	if !containsDiagnosticCode(got, "body_type_unsupported") {
		t.Fatalf("Definition() codes = %v, want body_type_unsupported", diagnosticCodes(got))
	}
}

func TestDefinitionRequiresRawBodyContentToBeString(t *testing.T) {
	t.Run("mapping is rejected", func(t *testing.T) {
		got := validate.Definition(model.Request{
			Name: "Create user", Method: "POST",
			Request: model.RequestConfig{
				URL:  "https://api.example.test/users",
				Body: &model.Body{Type: "raw", Content: map[string]string{"email": "jane@example.com"}},
			},
		})
		if !containsDiagnosticCode(got, "body_raw_invalid") {
			t.Fatalf("Definition() codes = %v, want body_raw_invalid", diagnosticCodes(got))
		}
	})
	t.Run("string is accepted", func(t *testing.T) {
		got := validate.Definition(model.Request{
			Name: "Create user", Method: "POST",
			Request: model.RequestConfig{
				URL:  "https://api.example.test/users",
				Body: &model.Body{Type: "raw", Content: "<user/>"},
			},
		})
		if containsDiagnosticCode(got, "body_raw_invalid") {
			t.Fatalf("Definition() codes = %v, do not want body_raw_invalid", diagnosticCodes(got))
		}
	})
}

func TestDefinitionRejectsNonASCIIHTTPMethod(t *testing.T) {
	got := validate.Definition(model.Request{
		Name: "List users", Method: "GÉT",
		Request: model.RequestConfig{URL: "https://api.example.test/users"},
	})
	if !containsDiagnosticCode(got, "method_invalid") {
		t.Fatalf("Definition() codes = %v, want method_invalid", diagnosticCodes(got))
	}
}

func diagnosticCodes(diagnostics []model.Diagnostic) []string {
	codes := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	return codes
}

func containsDiagnosticCode(diagnostics []model.Diagnostic, want string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == want {
			return true
		}
	}
	return false
}
