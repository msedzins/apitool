# apitool MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first usable `apitool` binary: a Git-workspace-aware TUI that edits, validates, and executes one YAML-defined API request at a time.

**Architecture:** Keep definition I/O, resolution, validation, OAuth, HTTP transport, runtime persistence, Git, application use cases, and Bubble Tea presentation as separate packages. YAML definitions live under `.api`; all mutable operational data stays in a Git-ignored `.apitool` directory at the workspace root. The TUI calls an application service and never implements YAML parsing, transport, OAuth, or Git itself.

**Tech Stack:** Go, Bubble Tea, YAML decoder/encoder, Go `net/http`, Go `oauth2` client-credentials support, and the local `git` executable.

**Spec:** [../specs/2026-09-20-apitool-design.md](../specs/2026-09-20-apitool-design.md)

## Global Constraints

- The workspace is a Git repository and may contain multiple collections discovered solely from a descendant `.api/` directory.
- `.api` contains reviewable YAML definitions only; all runtime state is in `<workspace>/.apitool/`, which is ignored by Git.
- A collection is unusable without valid `.api/collection.yaml`, but malformed definitions must not hide unrelated valid collections or requests.
- Only `{{environment_variable}}` and `${PROCESS_ENVIRONMENT_VARIABLE}` interpolation is supported; there are no request-local or response-derived variables.
- Authentication inheritance is collection → ancestor groups → nearest group → request; omission inherits and `auth: none` disables it.
- MVP OAuth is only OAuth2 `client_credentials`; client secrets and tokens must never be stored in YAML, runtime files, Git output, history, cache, or logs.
- `insecure_skip_verify` defaults to false and can be enabled only explicitly; `--confirm-dangerous/-c` confirms POST, PUT, PATCH, and DELETE immediately before send.
- Execute exactly one request at a time. A 3xx/4xx/5xx result is an HTTP response, not a transport failure.
- Edits use explicit/hybrid save, with dirty-navigation confirmation and undo/redo limited to the current request.
- Git integration is only status, diff, pull, push, and commit through local Git.
- The repository includes basic GitHub Actions CI on `push` and `pull_request`; it has only `contents: read` permission, reads its Go version from `go.mod`, and runs format verification, tests, `vet`, and build without secrets, artifacts, releases, or a version matrix.

## Review Focus

- A literal secret accidentally written into YAML must be rejected before it can enter a Git diff; Task 1 owns this validation test.
- An unresolved variable in a nested JSON string must prevent sending and report its location without echoing a secret; Task 3 owns this test.
- A request inheriting `auth: none` must send without an Authorization header even when its collection has OAuth; Task 3 owns this test.
- An OAuth token endpoint returning `invalid_client` must generate a redacted diagnostic, not expose the client secret; Task 4 owns this test.
- Response/history data from `prod` must never appear while viewing the same request under `test`; Task 5 owns this test.

---

## File structure and dependency order

```text
cmd/apitool/main.go                         CLI parsing, process construction, TUI launch
internal/model/                             Typed YAML and runtime domain objects
internal/collection/                        YAML load/save and collection tree construction
internal/workspace/                         Git-root lookup and `.api` discovery
internal/validate/                          Definition and resolved-model validation
internal/resolve/                           variable interpolation and effective inherited settings
internal/auth/                              OAuth2 client-credentials token acquisition/masking
internal/transport/                         HTTP request creation, TLS, execution classification
internal/runtime/                           `.apitool` cache, history, session state, redacted log
internal/git/                               narrow adapter around the `git` binary
internal/app/                               use cases joining the non-UI packages
internal/tui/                               Bubble Tea models, views, commands, input handling
.github/workflows/ci.yml                    GitHub-hosted formatting, test, vet, and build gate
testdata/workspace/                         committed safe fixture workspace
```

Tasks 1–7 establish a headless, testable application. Tasks 8–11 layer the TUI onto those use cases. Task 12 verifies the assembled binary against a multi-collection fixture.

### Task 1: Bootstrap, typed definitions, YAML persistence, and structural validation

**Files:**
- Create: `go.mod`
- Create: `cmd/apitool/main.go`
- Create: `internal/model/definition.go`
- Create: `internal/model/result.go`
- Create: `internal/collection/yaml_store.go`
- Create: `internal/validate/definition.go`
- Create: `internal/collection/yaml_store_test.go`
- Create: `internal/validate/definition_test.go`
- Create: `.gitignore`

**Interfaces:**
- Produces `model.Collection`, `model.Environment`, `model.Group`, `model.Request`, `model.Auth`, `model.HTTPConfig`, `model.Body`, and `model.Diagnostic`.
- Produces `collection.LoadCollectionMeta(path string)`, `collection.LoadEnvironment(path string)`, `collection.LoadRequest(path string)`, and `collection.SaveRequest(path string, request model.Request)`.
- Produces `validate.Definition(request model.Request) []model.Diagnostic`.

- [ ] **Step 1: Initialize the module and ignore operational data**

Create `go.mod` with module `apitool`; add Bubble Tea, YAML, and OAuth dependencies with `go get` in the implementation step that first imports them. Create `.gitignore` containing:

```gitignore
.apitool/
```

Create a minimal `main.go` that exits with a clear error if the application constructor fails. Do not add product behavior in `main` yet.

- [ ] **Step 2: Write failing YAML and validation tests**

```go
func TestLoadRequestAndSaveRoundTrip(t *testing.T) {
    path := writeFile(t, `name: Create user
method: POST
request:
  url: "{{base_url}}/users"
  body:
    type: json
    content: {email: jane@example.com}`)
    got, err := collection.LoadRequest(path)
    require.NoError(t, err)
    require.Equal(t, "POST", got.Method)
    require.NoError(t, collection.SaveRequest(path, got))
}

func TestDefinitionRejectsLiteralClientSecret(t *testing.T) {
    got := validate.Definition(model.Request{Auth: model.Auth{
        Type: "oauth2", Grant: "client_credentials",
        ClientSecret: "plaintext-secret",
    }})
    require.Contains(t, diagnosticCodes(got), "secret_literal")
}
```

- [ ] **Step 3: Run the focused tests and confirm failure**

Run: `go test ./internal/collection ./internal/validate -run 'Test(LoadRequestAndSaveRoundTrip|DefinitionRejectsLiteralClientSecret)'`

Expected: FAIL because the packages and functions do not exist.

- [ ] **Step 4: Implement the domain model and YAML store**

Use these exact core shapes so later tasks share one vocabulary:

```go
type Request struct {
    Name string `yaml:"name"`
    Method string `yaml:"method"`
    Request RequestConfig `yaml:"request"`
    Auth *Auth `yaml:"auth,omitempty"`
}
type RequestConfig struct {
    URL string `yaml:"url"`
    Params map[string]string `yaml:"params,omitempty"`
    Headers map[string]string `yaml:"headers,omitempty"`
    Body *Body `yaml:"body,omitempty"`
}
type Body struct { Type string `yaml:"type"`; Content any `yaml:"content"` }
type Auth struct {
    None bool `yaml:"-"`
    Type string `yaml:"type,omitempty"`; Grant string `yaml:"grant,omitempty"`
    TokenURL string `yaml:"token_url,omitempty"`; ClientID string `yaml:"client_id,omitempty"`
    ClientSecret string `yaml:"client_secret,omitempty"`; Scopes []string `yaml:"scopes,omitempty"`
}
type HTTPConfig struct { Timeout string `yaml:"timeout,omitempty"`; InsecureSkipVerify *bool `yaml:"insecure_skip_verify,omitempty"` }
type Diagnostic struct { Code, Path, Message string; Severity Severity }
```

Decode `auth: none` with a small YAML intermediary, normalize methods to uppercase, accept scope sequence or space-delimited scalar, serialize JSON-body content as ordinary YAML data, and write atomically through a temporary file in the target directory. Reject literal values in fields named `client_secret` unless the complete value is a single `${NAME}` reference. Validate required `name`, `method`, `request.url`, allowed body types, supported OAuth grant/type, positive parsable timeout, and JSON-serializability of `body.type: json`.

- [ ] **Step 5: Run focused and package tests**

Run: `go test ./internal/collection ./internal/validate`

Expected: PASS, including missing URL, invalid JSON content, unsupported auth grant, negative timeout, and literal-secret tests.

- [ ] **Step 6: Commit the bootstrap**

```bash
git add .gitignore go.mod go.sum cmd/apitool internal/model internal/collection internal/validate
git commit -m "feat: add YAML definition model and validation"
```

### Task 2: Discover workspace collections and construct request trees

**Files:**
- Create: `internal/workspace/discover.go`
- Create: `internal/workspace/discover_test.go`
- Create: `internal/collection/tree.go`
- Create: `internal/collection/tree_test.go`
- Create: `testdata/workspace/.gitkeep`
- Create: `testdata/workspace/payments/.api/collection.yaml`
- Create: `testdata/workspace/payments/.api/requests/payments/_group.yaml`
- Create: `testdata/workspace/payments/.api/requests/payments/list.yaml`

**Interfaces:**
- Consumes `collection.LoadCollectionMeta`, `collection.LoadRequest`, and `model.Diagnostic` from Task 1.
- Produces `workspace.FindRoot(start string) (string, error)` and `workspace.Discover(root string) []collection.LocatedCollection`.
- Produces `collection.BuildTree(collectionRoot string) (collection.Tree, []model.Diagnostic)` where `Tree.Requests` uses stable relative IDs.

- [ ] **Step 1: Write failing discovery/tree tests**

```go
func TestDiscoverFindsOnlyDirectoriesContainingDotAPI(t *testing.T) {
    root := fixtureWorkspace(t)
    collections := workspace.Discover(root)
    require.Equal(t, []string{"payments"}, collectionNames(collections))
}

func TestBuildTreeUsesRequestPathAsStableIDAndKeepsInvalidSibling(t *testing.T) {
    tree, diagnostics := collection.BuildTree(paymentCollection(t))
    require.Contains(t, tree.Requests, "payments/list")
    require.Contains(t, tree.Invalid, "payments/broken")
    require.NotEmpty(t, diagnostics)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/workspace ./internal/collection -run 'Test(DiscoverFindsOnlyDirectoriesContainingDotAPI|BuildTreeUsesRequestPathAsStableIDAndKeepsInvalidSibling)'`

Expected: FAIL because discovery and tree APIs do not exist.

- [ ] **Step 3: Implement discovery and graceful tree loading**

Walk only below the Git root, skip `.git` and `.apitool`, and record a collection for every directory whose direct child is `.api`. Sort collection names and tree nodes lexically for deterministic UI/tests. Derive a request ID by trimming `.api/requests/` and `.yaml`; reject `_group.yaml` as a request. For every group directory, collect ancestors in root-to-leaf order. Retain malformed collection/request entries as nodes with diagnostics so a bad file does not abort discovery.

- [ ] **Step 4: Expand tests for nested groups and malformed collection metadata**

Add a nested path `admin/refunds/list.yaml`; assert ID `admin/refunds/list` and group chain `admin`, `admin/refunds`. Add a collection with `.api` but invalid `collection.yaml`; assert it is discovered with a collection diagnostic rather than omitted.

- [ ] **Step 5: Run all package tests**

Run: `go test ./internal/workspace ./internal/collection`

Expected: PASS.

- [ ] **Step 6: Commit discovery**

```bash
git add internal/workspace internal/collection testdata/workspace
git commit -m "feat: discover collections and request trees"
```

### Task 3: Resolve environments, inheritance, and effective configuration

**Files:**
- Create: `internal/resolve/effective.go`
- Create: `internal/resolve/variables.go`
- Create: `internal/resolve/effective_test.go`
- Create: `internal/resolve/variables_test.go`
- Modify: `internal/model/definition.go`

**Interfaces:**
- Consumes `model.Collection`, `model.Environment`, ordered `[]model.Group`, `model.Request`, and `os.LookupEnv`.
- Produces `resolve.Effective(collection model.Collection, env model.Environment, groups []model.Group, request model.Request) (model.EffectiveRequest, []model.Diagnostic)`.
- `model.EffectiveRequest` contains resolved method, URL, params, headers, body bytes, effective auth, parsed timeout, and explicit TLS flag.

- [ ] **Step 1: Write failing resolution tests**

```go
func TestEffectiveUsesNearestAuthAndAuthNoneDisablesInheritance(t *testing.T) {
    got, diagnostics := resolve.Effective(collectionWithOAuth(), testEnv(), []model.Group{adminGroup()}, requestWithAuthNone())
    require.Empty(t, diagnostics)
    require.True(t, got.Auth.None)
}

func TestEffectiveRejectsUnresolvedVariableInNestedJSONWithoutLeakingValue(t *testing.T) {
    request := requestWithJSON(map[string]any{"credentials": map[string]any{"key": "{{missing_key}}"}})
    _, diagnostics := resolve.Effective(collectionWithoutAuth(), testEnv(), nil, request)
    require.Contains(t, diagnosticCodes(diagnostics), "variable_missing")
    require.NotContains(t, joinedMessages(diagnostics), "plaintext-secret")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/resolve -run 'TestEffective(UsesNearestAuthAndAuthNoneDisablesInheritance|RejectsUnresolvedVariableInNestedJSONWithoutLeakingValue)'`

Expected: FAIL because `Effective` does not exist.

- [ ] **Step 3: Implement interpolation and merge rules**

First replace every `{{name}}` from `Environment.Variables`; then replace every `${NAME}` through an injected `LookupEnv func(string) (string, bool)`. Traverse JSON body structures recursively before marshaling. A missing value produces diagnostic code `variable_missing`, field path, and variable name only. Do not recursively resolve a value after one environment and one process pass; this prevents surprising expansion cycles.

Merge HTTP as collection defaults followed by environment fields that are present. Select auth from collection, then each group in order, then request. A `None` auth stops inheritance. Parse the effective timeout with `time.ParseDuration`; use `http.DefaultClient`-compatible behavior only when no timeout is set. Always set TLS verification on unless the effective pointer equals `true`.

- [ ] **Step 4: Add table tests for resolution boundaries**

Cover: scalar URL/header/param expansion; `${API_CLIENT_ID}` read from injected lookup; absent process variable; no request-local scope; group scopes overriding collection scopes; environment timeout override; and `insecure_skip_verify: false` overriding a collection-level true value.

- [ ] **Step 5: Run all resolver tests**

Run: `go test ./internal/resolve`

Expected: PASS.

- [ ] **Step 6: Commit configuration resolution**

```bash
git add internal/model internal/resolve
git commit -m "feat: resolve environments and inherited configuration"
```

### Task 4: OAuth client credentials and HTTP transport

**Files:**
- Create: `internal/auth/client_credentials.go`
- Create: `internal/auth/client_credentials_test.go`
- Create: `internal/transport/client.go`
- Create: `internal/transport/client_test.go`
- Create: `internal/transport/error.go`

**Interfaces:**
- Consumes `model.EffectiveRequest` from Task 3.
- Produces `auth.TokenProvider` with `Token(ctx context.Context, config model.Auth) (auth.Token, error)` and `auth.Mask(value string) string`.
- Produces `transport.Execute(ctx context.Context, effective model.EffectiveRequest, tokenProvider auth.TokenProvider) (model.Response, *model.ExecutionError)`.

- [ ] **Step 1: Write failing OAuth and transport tests with `httptest`**

```go
func TestClientCredentialsCachesUnexpiredTokenAndSendsScopes(t *testing.T) {
    tokenServer, calls := oauthFixture(t, `{"access_token":"token-123","token_type":"Bearer","expires_in":3600}`)
    provider := auth.NewClientCredentials(tokenServer.Client())
    _, err := provider.Token(context.Background(), oauthConfig(tokenServer.URL, []string{"users.read"}))
    require.NoError(t, err)
    _, err = provider.Token(context.Background(), oauthConfig(tokenServer.URL, []string{"users.read"}))
    require.NoError(t, err)
    require.Equal(t, 1, calls())
}

func TestOAuthInvalidClientDiagnosticIsRedacted(t *testing.T) {
    _, err := auth.NewClientCredentials(http.DefaultClient).Token(context.Background(), failingOAuthConfig("very-secret"))
    require.NotContains(t, err.Error(), "very-secret")
    require.Contains(t, err.Error(), "invalid_client")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/auth ./internal/transport -run 'Test(ClientCredentialsCachesUnexpiredTokenAndSendsScopes|OAuthInvalidClientDiagnosticIsRedacted)'`

Expected: FAIL because the packages do not exist.

- [ ] **Step 3: Implement token and transport behavior**

POST client credentials to `token_url` with requested scopes, cache tokens in the provider memory only, and refresh before expiry. Never return an error string containing secret, token, or raw Authorization value. Build an HTTP request with resolved URL, query parameters, headers, and body. Add `Authorization: Bearer <token>` only when effective auth is OAuth and not `None`. Use a transport configured with the effective timeout and `tls.Config.InsecureSkipVerify` only when explicitly true.

Classify only request-build, OAuth, DNS, connection, TLS, context, and timeout failures as `ExecutionError{Stage, Category, SafeMessage}`. Return every completed `http.Response`, regardless of status code, as `model.Response` with status, headers, body, duration, and received time.

- [ ] **Step 4: Add transport boundary tests**

Test: a 401 response is returned as a response; a canceled context is `CategoryCanceled`; an expired cached token triggers reacquisition; `auth: none` sends no Authorization header; raw body bytes are unchanged; and a self-signed test server fails until `InsecureSkipVerify` is explicitly true.

- [ ] **Step 5: Run package tests**

Run: `go test ./internal/auth ./internal/transport`

Expected: PASS.

- [ ] **Step 6: Commit authenticated transport**

```bash
git add internal/auth internal/transport
git commit -m "feat: execute OAuth client-credentials requests"
```

### Task 5: Local runtime state, cache, history, and redacted execution log

**Files:**
- Create: `internal/runtime/store.go`
- Create: `internal/runtime/history.go`
- Create: `internal/runtime/redact.go`
- Create: `internal/runtime/store_test.go`
- Create: `internal/runtime/history_test.go`
- Create: `internal/runtime/redact_test.go`

**Interfaces:**
- Consumes `model.Response`, `model.ExecutionError`, request identity, and collection/environment identity.
- Produces `runtime.Open(workspaceRoot string) (*runtime.Store, error)` and methods `SaveResponse`, `LatestResponse`, `AppendHistory`, `SearchHistory`, `AppendLog`, `LoadState`, `SaveState`.
- `runtime.Key` is `{CollectionPath, Environment, RequestID string}`.

- [ ] **Step 1: Write failing runtime-isolation and redaction tests**

```go
func TestResponseCacheIsIsolatedByEnvironment(t *testing.T) {
    store := runtime.Open(tempWorkspace(t))
    key := runtime.Key{CollectionPath: "payments", RequestID: "users/list", Environment: "prod"}
    require.NoError(t, store.SaveResponse(key, model.Response{StatusCode: 200, Body: []byte(`{"source":"prod"}`)}))
    _, err := store.LatestResponse(runtime.Key{CollectionPath: "payments", RequestID: "users/list", Environment: "test"})
    require.ErrorIs(t, err, runtime.ErrNotFound)
}

func TestRedactionNeverWritesAuthorizationOrToken(t *testing.T) {
    record := runtime.RedactHeaders(http.Header{"Authorization": {"Bearer token-123"}, "X-Trace": {"ok"}})
    require.Equal(t, "[REDACTED]", record.Get("Authorization"))
    require.Equal(t, "ok", record.Get("X-Trace"))
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/runtime -run 'Test(ResponseCacheIsIsolatedByEnvironment|RedactionNeverWritesAuthorizationOrToken)'`

Expected: FAIL because `runtime.Store` is absent.

- [ ] **Step 3: Implement the visible `.apitool` layout**

Create only this workspace-local structure:

```text
.apitool/
├── responses/<collection>/<environment>/<request-id>/latest.json
├── history.jsonl
├── state.json
└── logs/
```

Encode cache records with redacted headers, never request headers/body/auth configuration, and owner-only permissions where supported. Keep complete response bodies local in cache. Append history entries containing timestamp, collection, request ID, method, result status or safe error category, and duration—never token, secret, raw request body, or Authorization header. Store one JSON object per history line and filter by case-insensitive collection/request/method/status text. `state.json` contains only last active environment per collection and panel preferences.

- [ ] **Step 4: Add redaction and persistence tests**

Test redaction for `Authorization`, `Proxy-Authorization`, `Cookie`, `Set-Cookie`, `client_secret`, and `access_token`; assert cache path includes collection/environment/request ID; assert `history.jsonl` contains no secret after an OAuth failure; assert `LoadState` on no file returns empty state.

- [ ] **Step 5: Run runtime tests**

Run: `go test ./internal/runtime`

Expected: PASS.

- [ ] **Step 6: Commit runtime persistence**

```bash
git add internal/runtime
git commit -m "feat: add local cache history and redacted logs"
```

### Task 6: Application service and CLI startup modes

**Files:**
- Create: `internal/app/service.go`
- Create: `internal/app/service_test.go`
- Modify: `cmd/apitool/main.go`

**Interfaces:**
- Consumes Tasks 1–5.
- Produces `app.New(deps app.Dependencies) (*app.Service, error)`, `OpenWorkspace`, `OpenCollection`, `SelectEnvironment`, `SaveRequest`, `Send`, `DuplicateRequest`, and `Delete`.
- `Send(ctx, app.Selection) app.SendResult` contains exactly one of `Response` or `ExecutionError`, plus redacted log entries.

- [ ] **Step 1: Write failing service tests**

```go
func TestOpenWorkspaceRestoresCollectionEnvironmentUnlessCLIOverride(t *testing.T) {
    service := fixtureService(t, stateWith("payments", "test"))
    opened, err := service.OpenWorkspace(context.Background(), fixtureRoot(t), app.OpenOptions{})
    require.NoError(t, err)
    require.Equal(t, "test", opened.Collections["payments"].Environment)
    opened, err = service.OpenWorkspace(context.Background(), fixtureRoot(t), app.OpenOptions{Environment: "prod"})
    require.NoError(t, err)
    require.Equal(t, "prod", opened.Collections["payments"].Environment)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/app -run TestOpenWorkspaceRestoresCollectionEnvironmentUnlessCLIOverride`

Expected: FAIL because the service does not exist.

- [ ] **Step 3: Implement use cases and CLI flags**

Add `-e, --env <name>` and `-c, --confirm-dangerous` in `main`. With no collection positional argument, launch on the collection picker; one positional collection name opens it directly. Reject unknown environments before rendering the request. Save validates before writing. Duplicate copies a request to a caller-provided, collision-free path; delete returns a confirmation-ready deletion target and performs file/folder removal only after the caller confirms. `Send` refuses invalid definitions and returns a confirmation requirement for unsafe methods when confirm-dangerous mode is on.

- [ ] **Step 4: Add app behavior tests**

Test: invalid request cannot send; `-c` requires confirmation for DELETE but not GET; selecting environment writes only `.apitool/state.json`; duplicate has a distinct file/ID; deleting a group lists descendants before deletion; an HTTP 500 produces `Response`, while a connection error produces `ExecutionError` and a history entry.

- [ ] **Step 5: Run service tests and compile binary**

Run: `go test ./internal/app && go build ./cmd/apitool`

Expected: PASS and an `apitool` binary is built.

- [ ] **Step 6: Commit application use cases**

```bash
git add cmd/apitool internal/app
git commit -m "feat: add application service and CLI options"
```

### Task 7: Thin local-Git adapter

**Files:**
- Create: `internal/git/repository.go`
- Create: `internal/git/repository_test.go`
- Modify: `internal/app/service.go`

**Interfaces:**
- Produces `git.Repository` methods `Status(ctx)`, `Diff(ctx)`, `Pull(ctx)`, `Push(ctx)`, and `Commit(ctx, message string)`.
- App exposes corresponding commands as safe text results; it never stages files.

- [ ] **Step 1: Write failing adapter tests using temporary repositories**

```go
func TestStatusAndDiffRunAtWorkspaceRoot(t *testing.T) {
    root := initGitRepo(t)
    writeWorkspaceChange(t, root)
    repo := git.New(root, exec.CommandContext)
    status, err := repo.Status(context.Background())
    require.NoError(t, err)
    require.Contains(t, status, "M")
}

func TestCommitRejectsBlankMessageBeforeInvokingGit(t *testing.T) {
    repo := git.New(initGitRepo(t), exec.CommandContext)
    err := repo.Commit(context.Background(), "   ")
    require.ErrorContains(t, err, "commit message")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/git -run 'Test(StatusAndDiffRunAtWorkspaceRoot|CommitRejectsBlankMessageBeforeInvokingGit)'`

Expected: FAIL because the adapter is absent.

- [ ] **Step 3: Implement bounded Git commands**

Invoke `git` with its working directory pinned to workspace root. Return captured stdout/stderr as a displayable safe result, preserving exit failure. `Commit` runs only `git commit -m <message>` and never runs `git add`. `Pull` and `Push` pass no invented branch/remote. Ensure `.apitool` remains absent from app Git status/diff by relying on the root ignore file created in Task 1.

- [ ] **Step 4: Add negative tests**

Test non-repository root gives a clear error; failing pull retains command output; diff does not cause a write; and commit succeeds only when the fixture has already staged a file.

- [ ] **Step 5: Run adapter and app tests**

Run: `go test ./internal/git ./internal/app`

Expected: PASS.

- [ ] **Step 6: Commit Git integration**

```bash
git add internal/git internal/app
git commit -m "feat: add thin Git workspace integration"
```

### Task 8: Bubble Tea shell, collection navigation, and accessible status presentation

**Files:**
- Create: `internal/tui/model.go`
- Create: `internal/tui/navigation.go`
- Create: `internal/tui/view.go`
- Create: `internal/tui/model_test.go`
- Create: `internal/tui/view_test.go`
- Modify: `cmd/apitool/main.go`

**Interfaces:**
- Consumes `app.Service` and workspace/collection view models from Task 6.
- Produces `tui.New(service *app.Service, options tui.Options) tea.Model`.
- `tui.Options` contains starting collection, starting environment, and confirm-dangerous mode.

- [ ] **Step 1: Write failing model tests**

```go
func TestCollectionPickerOpensCollectionAndRestoresItsEnvironment(t *testing.T) {
    m := tui.New(fixtureService(t), tui.Options{})
    next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
    require.Contains(t, next.View(), "Collection: payments")
    require.Contains(t, next.View(), "Environment: test")
}

func TestStatusViewUsesTextWhenColorDisabled(t *testing.T) {
    view := tui.StatusView(tui.Status{Code: 500, Duration: time.Millisecond}, false)
    require.Contains(t, view, "500")
    require.Contains(t, view, "error")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/tui -run 'Test(CollectionPickerOpensCollectionAndRestoresItsEnvironment|StatusViewUsesTextWhenColorDisabled)'`

Expected: FAIL because the TUI package does not exist.

- [ ] **Step 3: Implement the shell and explorer**

Build a three-region view: persistent left collection/request tree, upper request area, lower response/diagnostic area. Implement collection picker, direct opening, environment picker, tree expansion, `Tab`, arrows, `Enter`, `Esc`, `/`, and `Ctrl+P`. Add mouse click routing for focus/selection and optional `j/k/h/l` aliases. Render status color when capable and explicit textual category (`success`, `redirect`, `client error`, `server error`, `warning`) when not. Store splitter and active collection/environment preferences through Task 5 state.

- [ ] **Step 4: Add navigation tests**

Test collection switching changes tree and environment; invalid request gets a warning marker but valid siblings remain selectable; `Ctrl+E` opens environment picker; `Tab` cycles panes; mouse-independent keyboard flow opens a request; and `j/k` behavior is disabled unless Vim option is enabled.

- [ ] **Step 5: Run TUI tests and manual smoke test**

Run: `go test ./internal/tui && go run ./cmd/apitool --help`

Expected: PASS; help lists `--env` and `--confirm-dangerous`.

- [ ] **Step 6: Commit TUI shell**

```bash
git add internal/tui cmd/apitool
git commit -m "feat: add collection navigation TUI"
```

### Task 9: Request editor, structured/raw body modes, save prompts, undo/redo, duplicate, and delete

**Files:**
- Create: `internal/tui/editor.go`
- Create: `internal/tui/editor_test.go`
- Create: `internal/tui/confirm.go`
- Create: `internal/tui/confirm_test.go`
- Modify: `internal/tui/model.go`
- Modify: `internal/app/service.go`

**Interfaces:**
- Consumes `app.SaveRequest`, `app.DuplicateRequest`, and delete confirmation target from Task 6.
- Produces editor methods `Load(model.Request)`, `Dirty() bool`, `Undo()`, `Redo()`, `Save() tea.Cmd`, and `SwitchBodyMode(mode BodyMode)`.

- [ ] **Step 1: Write failing editor behavior tests**

```go
func TestDirtyRequestPromptsBeforeNavigation(t *testing.T) {
    m := editorWithChangedURL(t)
    next, _ := m.Update(navigateToOtherRequest())
    require.Contains(t, next.View(), "Save and continue")
    require.Contains(t, next.View(), "Discard changes")
}

func TestJSONModePrettyPrintsAndRawModePreservesRawBytes(t *testing.T) {
    m := editorWithJSON(t, `{"a":1}`)
    require.Contains(t, m.View(), "\"a\": 1")
    m = switchToRaw(t, m, "<a>1</a>")
    require.Equal(t, "<a>1</a>", savedRawBody(t, m))
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/tui -run 'Test(DirtyRequestPromptsBeforeNavigation|JSONModePrettyPrintsAndRawModePreservesRawBytes)'`

Expected: FAIL because the editor is absent.

- [ ] **Step 3: Implement typed editor state**

Edit typed request fields—method, URL, params, headers, auth, body, and name—rather than YAML text. JSON mode uses decoded structured data and validates/pretty-prints before it is saved or sent; raw mode stores exact text. Maintain an undo and redo stack only for the currently open request. `Ctrl+S` validates through Task 1 and saves via Task 6. On dirty navigation show Save and continue / Discard changes / Cancel; never autosave. Add command-palette actions for duplicate and delete. Delete request/group shows exact affected path(s) and requires affirmative confirmation.

- [ ] **Step 4: Add editor safety tests**

Test invalid JSON cannot save or send; undo/redo restores URL/body; undo history resets after opening another request; duplicate immediately opens a new unsaved name/path flow; group delete lists descendants; and discard leaves the YAML file unchanged.

- [ ] **Step 5: Run TUI suite**

Run: `go test ./internal/tui`

Expected: PASS.

- [ ] **Step 6: Commit editing workflow**

```bash
git add internal/tui internal/app
git commit -m "feat: add request editing and save workflow"
```

### Task 10: Send workflow, response/diagnostic/auth/log panels, and dangerous-operation confirmation

**Files:**
- Create: `internal/tui/execute.go`
- Create: `internal/tui/execute_test.go`
- Create: `internal/tui/response_view.go`
- Create: `internal/tui/response_view_test.go`
- Modify: `internal/tui/model.go`

**Interfaces:**
- Consumes `app.Send`, `runtime.Store`, and `auth.Mask`.
- Produces send command messages with either `model.Response` or `model.ExecutionError` and redacted log entries.

- [ ] **Step 1: Write failing send/view tests**

```go
func TestConfirmDangerousRequiresExplicitSendForDelete(t *testing.T) {
    m := requestScreen(t, http.MethodDelete, true)
    next, _ := m.Update(sendKey())
    require.Contains(t, next.View(), "Confirmation required")
    require.NotContains(t, next.View(), "Sending")
}

func TestHTTP401RendersResponseNotFailure(t *testing.T) {
    m := responseScreen(t, model.Response{StatusCode: 401, Status: "401 Unauthorized"})
    require.Contains(t, m.View(), "401 Unauthorized")
    require.NotContains(t, m.View(), "Request failed")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/tui -run 'Test(ConfirmDangerousRequiresExplicitSendForDelete|HTTP401RendersResponseNotFailure)'`

Expected: FAIL because send and response views do not exist.

- [ ] **Step 3: Implement execution UI**

Bind `Ctrl+Enter` and Send button to exactly one app send. In confirmation mode, POST/PUT/PATCH/DELETE render a modal with environment, method, resolved safe URL, cancel, and send actions; GET sends immediately. During send show a non-blocking busy state. Render response status, duration, size, headers, pretty JSON body, and raw body in the lower pane. Render execution errors in a diagnostic card with stage/category/safe message and the redacted request log. Add an Auth tab showing grant, endpoint, scopes, expiry, and a masked token; full token is revealed only after `Show token` and never copied to log/history/cache.

- [ ] **Step 4: Add send/view tests**

Test POST confirmation accepted invokes one send; rejected confirmation invokes none; 500 is a response; DNS/OAuth errors show diagnostic stage; JSON response pretty-print falls back to raw text; sensitive headers/log values are masked; token starts masked and reveal is session-only.

- [ ] **Step 5: Run TUI suite**

Run: `go test ./internal/tui`

Expected: PASS.

- [ ] **Step 6: Commit execution UI**

```bash
git add internal/tui
git commit -m "feat: add request execution and response diagnostics"
```

### Task 11: TUI history, Git commands, layout resizing, and final user-facing documentation

**Files:**
- Create: `internal/tui/history.go`
- Create: `internal/tui/git.go`
- Create: `internal/tui/history_test.go`
- Create: `internal/tui/git_test.go`
- Create: `README.md`
- Modify: `internal/tui/model.go`

**Interfaces:**
- Consumes Task 5 history/state and Task 7 Git adapter.
- Produces palette actions `History`, `Git: Status`, `Git: Diff`, `Git: Pull`, `Git: Push`, `Git: Commit` and persisted panel-split preferences.

- [ ] **Step 1: Write failing history/Git UI tests**

```go
func TestHistorySearchReopensReferencedRequest(t *testing.T) {
    m := historyScreenWithEntry(t, "payments/users/list")
    next, _ := m.Update(searchAndEnter("users/list"))
    require.Contains(t, next.View(), "GET")
    require.Contains(t, next.View(), "users/list")
}

func TestGitCommitPromptsForNonBlankMessage(t *testing.T) {
    m := gitScreen(t)
    next, _ := m.Update(selectPalette("Git: Commit"))
    require.Contains(t, next.View(), "Commit message")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/tui -run 'Test(HistorySearchReopensReferencedRequest|GitCommitPromptsForNonBlankMessage)'`

Expected: FAIL because history and Git views do not exist.

- [ ] **Step 3: Implement history, Git, and resizing**

Create searchable local history ordered newest first; `Enter` reopens the referenced current definition, not a saved request-body snapshot. Add palette Git actions that show adapter result output and never add/stage files. Commit asks for a nonblank message. Implement keyboard and mouse resizing for request/response split with a bounded minimum height for both panes, then persist the ratio in `.apitool/state.json`.

- [ ] **Step 4: Write README usage and safety instructions**

Document workspace layout, `.apitool/` and `.gitignore`, safe environment YAML examples, process-env secrets, `apitool -e test`, `apitool -e prod -c`, hybrid save, Git command limitations, and the explicit non-MVP feature list. Do not include a literal secret or access token in any example.

- [ ] **Step 5: Add final UI tests and run all tests**

Test empty history state; history filter; Git command errors visible without panic; split resizing respects minima; state persistence restores split; and README command examples match CLI flags. Run: `go test ./...`

Expected: PASS.

- [ ] **Step 6: Commit final MVP surfaces**

```bash
git add internal/tui README.md
git commit -m "feat: complete history Git controls and documentation"
```

### Task 12: End-to-end fixture verification and release-readiness checks

**Files:**
- Create: `internal/app/e2e_test.go`
- Create: `testdata/workspace/users/.api/collection.yaml`
- Create: `testdata/workspace/users/.api/environments/test.yaml`
- Create: `testdata/workspace/users/.api/requests/users/list.yaml`
- Modify: `README.md`

**Interfaces:**
- Consumes the complete binary/application stack from Tasks 1–11.
- Produces a reproducible fixture with two valid collections and intentionally invalid sibling definitions.

- [ ] **Step 1: Write the failing end-to-end scenario**

```go
func TestFixtureWorkspaceCanDiscoverEditExecuteCacheAndRecall(t *testing.T) {
    server := newAPIServer(t, http.StatusOK, `{"ok":true}`)
    service := app.New(fixtureDependencies(t, server))
    opened, err := service.OpenWorkspace(context.Background(), fixtureRoot(t), app.OpenOptions{Environment: "test"})
    require.NoError(t, err)
    require.Len(t, opened.Collections, 2)
    result := service.Send(context.Background(), app.Selection{Collection: "users", Environment: "test", RequestID: "users/list"})
    require.Equal(t, 200, result.Response.StatusCode)
    require.FileExists(t, filepath.Join(fixtureRoot(t), ".apitool", "history.jsonl"))
}
```

- [ ] **Step 2: Run the scenario and verify it fails before fixture completion**

Run: `go test ./internal/app -run TestFixtureWorkspaceCanDiscoverEditExecuteCacheAndRecall`

Expected: FAIL until the two-collection fixture and test dependencies are complete.

- [ ] **Step 3: Complete the fixture and verify all MVP boundaries**

Use a local test HTTP server and process-environment injection; never call an external API. Assert two collections are discoverable, one malformed request is marked invalid, a valid request executes once, response/history are written below `.apitool`, and no runtime path appears in `git status --short` after initializing the fixture repo. Add explicit assertions that cache/history/log contents do not include fixture secret or access token values.

- [ ] **Step 4: Run quality gates**

Run: `gofmt -w cmd internal && go test ./... && go vet ./... && go build ./cmd/apitool`

Expected: all commands exit 0.

- [ ] **Step 5: Manually exercise the terminal workflow**

Run: `go run ./cmd/apitool -e test`

Expected: collection picker appears; opening a collection shows request tree, editor, and response pane; no secret is visible by default.

- [ ] **Step 6: Commit verification fixture**

```bash
git add internal/app testdata README.md
git commit -m "test: verify end-to-end MVP workflow"
```

### Task 13: GitHub Actions continuous-integration gate

**Files:**
- Create: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes `go.mod` for the declared Go version and the repository test/build entry points produced by Tasks 1–12.
- Produces one GitHub Actions workflow named `CI`, triggered by `push` and `pull_request`.

- [ ] **Step 1: Write the workflow acceptance checks**

The workflow must fail when `gofmt -l .` returns any path, when `go test ./...` fails, when `go vet ./...` fails, or when `go build ./cmd/apitool` fails. It must use no GitHub secrets and request only `contents: read` permission.

- [ ] **Step 2: Create the workflow**

```yaml
name: CI

on:
  push:
  pull_request:

permissions:
  contents: read

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true
      - name: Check formatting
        run: test -z "$(gofmt -l .)"
      - name: Test
        run: go test ./...
      - name: Vet
        run: go vet ./...
      - name: Build
        run: go build ./cmd/apitool
```

- [ ] **Step 3: Verify the equivalent local commands**

Run: `test -z "$(gofmt -l .)" && go test ./... && go vet ./... && go build ./cmd/apitool`

Expected: exit code 0 after Tasks 1–12 have created the Go module and binary.

- [ ] **Step 4: Commit CI configuration**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add GitHub Actions quality gate"
```

## Plan self-review

- **Spec coverage:** Tasks 1–3 cover YAML, workspace, groups, inheritance, variables, validation, and TLS configuration. Task 4 covers OAuth/HTTP semantics. Task 5 covers `.apitool`, cache/history/log security. Tasks 6–11 cover CLI, complete TUI UX, dangerous confirmation, thin Git, and documentation. Task 12 verifies the assembled multi-collection workflow. Task 13 provides the GitHub Actions quality gate. Post-MVP features intentionally have no implementation task.
- **Placeholder scan:** No implementation step defers behavior; each task names files, interfaces, concrete tests, commands, and a commit boundary.
- **Type consistency:** `model.*` is introduced in Task 1; effective definitions in Task 3; auth/transport in Task 4; runtime key/store in Task 5; service in Task 6; UI in Tasks 8–11.
- **Review focus coverage:** Literal secret validation: Task 1. Nested missing variable: Task 3. `auth: none`: Tasks 3–4. OAuth redaction: Task 4. Environment cache isolation: Task 5.
