package collection_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
