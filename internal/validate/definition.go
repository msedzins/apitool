// Package validate provides structural validation for API definitions.
package validate

import (
	"encoding/json"
	"strings"
	"unicode"

	"apitool/internal/model"
)

// Definition reports the independent structural problems in request.
func Definition(request model.Request) []model.Diagnostic {
	var diagnostics []model.Diagnostic

	if strings.TrimSpace(request.Name) == "" {
		diagnostics = append(diagnostics, errorDiagnostic("name_required", "name", "request name is required"))
	}
	if strings.TrimSpace(request.Method) == "" {
		diagnostics = append(diagnostics, errorDiagnostic("method_required", "method", "request method is required"))
	} else if !isHTTPToken(request.Method) {
		diagnostics = append(diagnostics, errorDiagnostic("method_invalid", "method", "request method must be a valid HTTP token"))
	}
	if strings.TrimSpace(request.Request.URL) == "" {
		diagnostics = append(diagnostics, errorDiagnostic("request_url_required", "request.url", "request URL is required"))
	}

	if body := request.Request.Body; body != nil {
		switch body.Type {
		case "json":
			if _, err := json.Marshal(body.Content); err != nil {
				diagnostics = append(diagnostics, errorDiagnostic("body_json_invalid", "request.body.content", "JSON body content must be JSON-serializable"))
			}
		case "raw":
			if _, ok := body.Content.(string); !ok {
				diagnostics = append(diagnostics, errorDiagnostic("body_raw_invalid", "request.body.content", "raw body content must be a string"))
			}
		default:
			diagnostics = append(diagnostics, errorDiagnostic("body_type_unsupported", "request.body.type", "body type must be json or raw"))
		}
	}

	if auth := request.Auth; auth != nil && !auth.None {
		if auth.Type != "oauth2" {
			diagnostics = append(diagnostics, errorDiagnostic("auth_type_unsupported", "auth.type", "auth type must be oauth2"))
		}
		if auth.Grant != "client_credentials" {
			diagnostics = append(diagnostics, errorDiagnostic("auth_grant_unsupported", "auth.grant", "OAuth grant must be client_credentials"))
		}
		if auth.ClientSecret != "" && !isEnvironmentReference(auth.ClientSecret) {
			diagnostics = append(diagnostics, errorDiagnostic("secret_literal", "auth.client_secret", "client_secret must be a single process environment reference"))
		}
	}

	return diagnostics
}

func errorDiagnostic(code, path, message string) model.Diagnostic {
	return model.Diagnostic{Code: code, Path: path, Message: message, Severity: model.SeverityError}
}

func isHTTPToken(value string) bool {
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		switch r {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
			continue
		default:
			return false
		}
	}
	return true
}

func isEnvironmentReference(value string) bool {
	if len(value) < 4 || !strings.HasPrefix(value, "${") || !strings.HasSuffix(value, "}") {
		return false
	}
	for index, r := range value[2 : len(value)-1] {
		if r == '_' || unicode.IsLetter(r) || (index > 0 && unicode.IsDigit(r)) {
			continue
		}
		return false
	}
	return true
}
