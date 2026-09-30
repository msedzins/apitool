package collection_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"apitool/internal/collection"
	"apitool/internal/model"
)

func TestLoadRequestAndSaveRoundTrip(t *testing.T) {
	path := writeFile(t, `name: Create user
method: POST
request:
  url: "{{base_url}}/users"
  body:
    type: json
    content: {email: jane@example.com}`)

	got, err := collection.LoadRequest(path)
	if err != nil {
		t.Fatalf("LoadRequest() error = %v", err)
	}
	if got.Method != "POST" {
		t.Fatalf("LoadRequest().Method = %q, want %q", got.Method, "POST")
	}
	if err := collection.SaveRequest(path, got); err != nil {
		t.Fatalf("SaveRequest() error = %v", err)
	}
	reloaded, err := collection.LoadRequest(path)
	if err != nil {
		t.Fatalf("LoadRequest() after save error = %v", err)
	}
	if reloaded.Request.Body == nil || reloaded.Request.Body.Type != "json" {
		t.Fatalf("LoadRequest() after save body = %#v, want JSON body", reloaded.Request.Body)
	}
}

func TestLoadRequestDecodesAuthNoneAndNormalizesMethod(t *testing.T) {
	path := writeFile(t, `name: List users
method: get
request:
  url: https://api.example.test/users
auth: none`)

	got, err := collection.LoadRequest(path)
	if err != nil {
		t.Fatalf("LoadRequest() error = %v", err)
	}
	if got.Method != "GET" {
		t.Fatalf("LoadRequest().Method = %q, want GET", got.Method)
	}
	if got.Auth == nil || !got.Auth.None {
		t.Fatalf("LoadRequest().Auth = %#v, want auth none", got.Auth)
	}
}

func TestSaveRequestRejectsLiteralClientSecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), "request.yaml")
	err := collection.SaveRequest(path, model.Request{
		Name:    "List users",
		Method:  "GET",
		Request: model.RequestConfig{URL: "https://api.example.test/users"},
		Auth: &model.Auth{
			Type:         "oauth2",
			Grant:        "client_credentials",
			ClientSecret: "plaintext-secret",
		},
	})
	if err == nil {
		t.Fatal("SaveRequest() error = nil, want literal client secret rejection")
	}
}

func TestLoadEnvironmentRejectsNonPositiveTimeout(t *testing.T) {
	path := writeFile(t, `name: test
http:
  timeout: -1s`)
	if _, err := collection.LoadEnvironment(path); err == nil {
		t.Fatal("LoadEnvironment() error = nil, want invalid timeout rejection")
	}
}

func TestLoadCollectionMetaRequiresName(t *testing.T) {
	path := writeFile(t, "description: Missing name\n")
	if _, err := collection.LoadCollectionMeta(path); err == nil {
		t.Fatal("LoadCollectionMeta() error = nil, want missing name rejection")
	}
}

func TestLoadEnvironmentRejectsNameDifferentFromFilename(t *testing.T) {
	path := writeNamedFile(t, "test.yaml", "name: prod\n")
	if _, err := collection.LoadEnvironment(path); err == nil {
		t.Fatal("LoadEnvironment() error = nil, want mismatched environment name rejection")
	}
}

func TestLoadRequestDecodesScalarAndSequenceScopes(t *testing.T) {
	for _, test := range []struct {
		name   string
		scopes string
		want   []string
	}{
		{name: "scalar", scopes: "users.read users.write", want: []string{"users.read", "users.write"}},
		{name: "sequence", scopes: "[users.read, users.write]", want: []string{"users.read", "users.write"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := writeFile(t, "name: List users\nmethod: GET\nrequest:\n  url: https://api.example.test/users\nauth:\n  type: oauth2\n  grant: client_credentials\n  scopes: "+test.scopes+"\n")
			got, err := collection.LoadRequest(path)
			if err != nil {
				t.Fatalf("LoadRequest() error = %v", err)
			}
			if got.Auth == nil || len(got.Auth.Scopes) != len(test.want) {
				t.Fatalf("LoadRequest().Auth.Scopes = %#v, want %#v", got.Auth, test.want)
			}
			for index, want := range test.want {
				if got.Auth.Scopes[index] != want {
					t.Fatalf("LoadRequest().Auth.Scopes[%d] = %q, want %q", index, got.Auth.Scopes[index], want)
				}
			}
		})
	}
}

func TestSecretReferencesAreAcceptedAndLiteralSecretsAreRejectedOnLoad(t *testing.T) {
	t.Run("collection reference", func(t *testing.T) {
		path := writeFile(t, "name: Users\nauth:\n  type: oauth2\n  grant: client_credentials\n  client_secret: ${USERS_CLIENT_SECRET}\n")
		if _, err := collection.LoadCollectionMeta(path); err != nil {
			t.Fatalf("LoadCollectionMeta() error = %v", err)
		}
	})
	t.Run("collection literal", func(t *testing.T) {
		path := writeFile(t, "name: Users\nauth:\n  client_secret: plaintext-secret\n")
		if _, err := collection.LoadCollectionMeta(path); err == nil {
			t.Fatal("LoadCollectionMeta() error = nil, want literal secret rejection")
		}
	})
	t.Run("request reference", func(t *testing.T) {
		path := writeFile(t, "name: List users\nmethod: GET\nrequest:\n  url: https://api.example.test/users\nauth:\n  client_secret: ${USERS_CLIENT_SECRET}\n")
		if _, err := collection.LoadRequest(path); err != nil {
			t.Fatalf("LoadRequest() error = %v", err)
		}
	})
	t.Run("request literal", func(t *testing.T) {
		path := writeFile(t, "name: List users\nmethod: GET\nrequest:\n  url: https://api.example.test/users\nauth:\n  client_secret: plaintext-secret\n")
		if _, err := collection.LoadRequest(path); err == nil {
			t.Fatal("LoadRequest() error = nil, want literal secret rejection")
		}
	})
	t.Run("save reference", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "request.yaml")
		err := collection.SaveRequest(path, model.Request{Auth: &model.Auth{ClientSecret: "${USERS_CLIENT_SECRET}"}})
		if err != nil {
			t.Fatalf("SaveRequest() error = %v", err)
		}
	})
}

func TestSaveRequestSerializesAuthNone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "request.yaml")
	want := model.Request{
		Name: "List users", Method: "GET",
		Request: model.RequestConfig{URL: "https://api.example.test/users"},
		Auth:    &model.Auth{None: true},
	}
	if err := collection.SaveRequest(path, want); err != nil {
		t.Fatalf("SaveRequest() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "auth: none") {
		t.Fatalf("SaveRequest() YAML = %q, want auth: none", data)
	}
	got, err := collection.LoadRequest(path)
	if err != nil {
		t.Fatalf("LoadRequest() error = %v", err)
	}
	if got.Auth == nil || !got.Auth.None {
		t.Fatalf("LoadRequest().Auth = %#v, want auth none", got.Auth)
	}
}

func writeFile(t *testing.T, content string) string {
	t.Helper()
	return writeNamedFile(t, "request.yaml", content)
}

func writeNamedFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadRequestAndSavePreservesExactJSONNumbers(t *testing.T) {
	path := writeFile(t, `name: Numbers
method: POST
request:
  url: https://example.test/numbers
  body:
    type: json
    content:
      large: 9007199254740993
      precise: 0.123456789123456789`)
	request, err := collection.LoadRequest(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := collection.SaveRequest(path, request); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, number := range []string{"9007199254740993", "0.123456789123456789"} {
		if !strings.Contains(string(data), number) {
			t.Fatalf("saved YAML lost numeric token %q: %s", number, data)
		}
	}
}

func TestLoadRequestPreservesYAMLScalarSemanticsInJSONBody(t *testing.T) {
	path := writeFile(t, `name: Scalars
method: POST
request:
  url: https://example.test
  body:
    type: json
    content:
      upper: TRUE
      title: True
      octal: 077
      signed_hex: -0x10
      leading_zero_float: 01.20
      positive_float: +2.5
      short_float: .25`)
	request, err := collection.LoadRequest(path)
	if err != nil {
		t.Fatal(err)
	}
	content := request.Request.Body.Content.(map[string]any)
	if content["upper"] != true || content["title"] != true {
		t.Fatalf("booleans = %#v", content)
	}
	if err := collection.SaveRequest(path, request); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"upper: true", "title: true", "octal: 63", "signed_hex: -16", "leading_zero_float: 1.20", "positive_float: 2.5", "short_float: 0.25"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("saved YAML missing %q: %s", want, data)
		}
	}
}

func TestLoadRequestRejectsDuplicateBodyKeysAndResolvesAliases(t *testing.T) {
	t.Run("duplicate", func(t *testing.T) {
		path := writeFile(t, "name: Duplicate\nmethod: POST\nrequest:\n  url: https://example.test\n  body:\n    type: json\n    content: {a: 1, a: 2}\n")
		if _, err := collection.LoadRequest(path); err == nil {
			t.Fatal("LoadRequest() accepted duplicate JSON body keys")
		}
	})
	t.Run("alias", func(t *testing.T) {
		path := writeFile(t, "defaults: &payload {a: 1}\nname: Alias\nmethod: POST\nrequest:\n  url: https://example.test\n  body:\n    type: json\n    content: *payload\n")
		request, err := collection.LoadRequest(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := request.Request.Body.Content.(map[string]any)["a"]; got != 1 {
			t.Fatalf("alias value = %#v", got)
		}
	})
}

func TestLoadRequestPreservesYAMLTimestampAndBinarySemantics(t *testing.T) {
	path := writeFile(t, `name: YAML types
method: POST
request:
  url: https://example.test
  body:
    type: json
    content:
      instant: 2026-09-30T12:30:00Z
      encoded: !!binary SGVsbG8=`)
	request, err := collection.LoadRequest(path)
	if err != nil {
		t.Fatal(err)
	}
	content := request.Request.Body.Content.(map[string]any)
	if got, ok := content["instant"].(time.Time); !ok || !got.Equal(time.Date(2026, 9, 30, 12, 30, 0, 0, time.UTC)) {
		t.Fatalf("timestamp = %#v", content["instant"])
	}
	if got, ok := content["encoded"].(string); !ok || got != "Hello" {
		t.Fatalf("binary = %#v", content["encoded"])
	}
}

func TestLoadRequestExpandsYAMLMergeKeysInJSONBody(t *testing.T) {
	path := writeFile(t, "defaults: &base {shared: inherited, extra: 1}\nname: Merge\nmethod: POST\nrequest:\n  url: https://example.test\n  body:\n    type: json\n    content:\n      merged:\n        <<: *base\n        shared: explicit\n")
	request, err := collection.LoadRequest(path)
	if err != nil {
		t.Fatal(err)
	}
	merged := request.Request.Body.Content.(map[string]any)["merged"].(map[string]any)
	if merged["shared"] != "explicit" || merged["extra"] != 1 {
		t.Fatalf("merged content = %#v", merged)
	}
}
