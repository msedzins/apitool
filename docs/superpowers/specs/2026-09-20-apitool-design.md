# apitool — design spec (MVP)

**Status:** accepted design; ready for review before implementation planning  
**Date:** 2026-09-20  
**Technology:** Go and Bubble Tea

## 1. Purpose and product boundary

`apitool` is an open-source, terminal-first API client: a deliberately smaller, Git-friendly alternative to Postman. It lets a team keep API request definitions in a normal Git repository, edit and execute them from a TUI, and share the definitions through ordinary Git workflows.

The repository is the source of truth for collection definitions. GitHub is an optional Git remote, not an application backend or dependency. The product must work with a local Git repository and any compatible remote (GitHub, GitLab, self-hosted Git, or none).

MVP success means a developer can open a repository containing several API collections, select a collection and environment, safely edit a YAML request, execute it with OAuth2 client credentials when configured, inspect its response or failure, and commit/push the resulting definition changes.

## 2. Scope

### MVP

- Go executable with Bubble Tea TUI.
- One Git repository as a workspace containing many discoverable collections.
- YAML files for collections, environments, groups, and requests.
- Request methods supported by Go's HTTP client (with common `GET`, `POST`, `PUT`, `PATCH`, and `DELETE` exposed naturally in the editor).
- Query parameters, headers, URL, structured JSON body, raw body, and OAuth configuration editing.
- Per-collection environments, `{{variable}}` substitution, and `${ENV_VAR}` process-environment substitution only.
- OAuth2 `client_credentials`, including scopes and in-memory token reuse.
- HTTP responses, execution diagnostics, local response cache, and local request history.
- Full TUI editing, hybrid save, undo/redo for the current request, duplicate and confirmed delete.
- Basic Git status, diff, pull, push, and commit integration.
- Keyboard, mouse, and optional Vim-style navigation.

### Explicit non-goals for MVP

- A hosted backend, user accounts, collaboration service, GitHub API integration, or database for definitions.
- Secret manager, encrypted secrets in Git, request-local variables, scripting, plugins, or a test runner.
- OpenAPI import/export, request chaining, batch execution, parallel execution, mTLS, client certificates, and OAuth authorization-code login.
- A full Git client: branches, checkout, merge/rebase, conflict resolution, staging UI, and remote-provider features remain outside scope.

## 3. Workspace and filesystem model

The user starts `apitool` inside a Git repository. That repository is a workspace. A collection is any directory below the workspace that contains an `.api/` directory; no workspace manifest exists.

`collection.yaml` is required for a collection to be usable, but discovery is based on `.api/` so a missing or malformed metadata file is surfaced as a collection diagnostic rather than silently hidden.

```text
api-workspace/                         # Git repository / workspace
├── .git/
├── payments/                          # collection root
│   └── .api/
│       ├── collection.yaml
│       ├── environments/
│       │   ├── local.yaml
│       │   ├── test.yaml
│       │   └── prod.yaml
│       └── requests/
│           ├── payments/
│           │   ├── _group.yaml
│           │   ├── list.yaml
│           │   └── create.yaml
│           └── admin/
│               ├── _group.yaml
│               └── refund.yaml
└── users/                             # another collection root
    └── .api/
        ├── collection.yaml
        ├── environments/
        └── requests/
```

Groups are physical directories below `.api/requests`. `_group.yaml` is optional group metadata/configuration for that directory. Nested group directories are supported; effective group configuration is inherited from outermost ancestor to innermost ancestor, then overridden by the request. Request IDs are stable, collection-relative file paths without the `.yaml` suffix (for example, `payments/list`). They are not UUIDs.

The `.api` tree contains only reviewable definitions. Runtime data is kept in one explicit workspace-local directory, `<workspace-root>/.apitool/`, and is not written into Git. A workspace using `apitool` must ignore `.apitool/` in its root `.gitignore`.

## 4. YAML data model

YAML is intentionally direct: it describes HTTP, rather than introducing a separate request language. The TUI edits an in-memory typed model and serializes it back to YAML; it does not expose YAML as its primary editing surface.

### Collection: `.api/collection.yaml`

```yaml
name: Payments API
description: Payment service requests

auth:
  type: oauth2
  grant: client_credentials
  token_url: "{{oauth_token_url}}"
  client_id: ${PAYMENTS_CLIENT_ID}
  client_secret: ${PAYMENTS_CLIENT_SECRET}
  scopes:
    - payments.read

http:
  timeout: 30s
  insecure_skip_verify: false
```

`name` is required. `description`, `auth`, and `http` are optional. The collection-level `http` object supplies defaults; active-environment values override it where defined. `timeout` must be a positive Go duration. `insecure_skip_verify` defaults to `false` and can only be enabled explicitly.

### Environment: `.api/environments/<name>.yaml`

```yaml
name: test
variables:
  base_url: https://test-api.example.com
  oauth_token_url: https://auth.example.com/oauth/token
  api_version: v1
http:
  timeout: 30s
  insecure_skip_verify: false
```

The file name supplies the selectable environment key; if `name` is present it must match that key. Environment variables are strings (or values serialized to strings) and may contain `${PROCESS_ENV_VAR}` references. Secrets must be represented only by process environment references, never as literal values committed to YAML.

### Group: `.api/requests/<group>/_group.yaml`

```yaml
name: Admin payments
auth:
  type: oauth2
  grant: client_credentials
  token_url: "{{oauth_token_url}}"
  client_id: ${ADMIN_CLIENT_ID}
  client_secret: ${ADMIN_CLIENT_SECRET}
  scopes: [payments.admin]
```

### Request: `.api/requests/<group>/<request>.yaml`

```yaml
name: Create payment
method: POST
request:
  url: "{{base_url}}/{{api_version}}/payments"
  params:
    dry_run: "false"
  headers:
    Accept: application/json
    Content-Type: application/json
  body:
    type: json
    content:
      amount: 1200
      currency: PLN

# Omit to inherit. Use `none` to disable inherited authentication.
auth: none
```

`name`, `method`, and `request.url` are required. `method` is normalized to uppercase and must be a valid HTTP token. `params` and `headers` are string maps. Header names are case-insensitive at execution time. Values in URL, params, headers, auth fields, and body strings undergo variable resolution.

Body is explicitly typed:

```yaml
body:
  type: json
  content:
    email: john@example.com
```

or:

```yaml
body:
  type: raw
  content: |
    <user><email>john@example.com</email></user>
```

`json` content is YAML data serialized as JSON and validated before sending. `raw` content is an exact text payload (after substitution). Other body types, forms, multipart, and files are future work.

### Auth resolution

Authentication is inherited as:

```text
collection auth → ancestor group auth → nearest group auth → request auth
```

The nearest defined setting wins. Omitting `auth` means inherit; `auth: none` explicitly disables it. MVP supports exactly this concrete OAuth configuration:

```yaml
auth:
  type: oauth2
  grant: client_credentials
  token_url: "{{oauth_token_url}}"
  client_id: ${API_CLIENT_ID}
  client_secret: ${API_CLIENT_SECRET}
  scopes:
    - users.read
    - users.write
```

`scopes` accepts a YAML sequence or a space-delimited string and is normalized internally to `[]string`. Unsupported grant/type values are validation errors. Access tokens remain in process memory only, keyed by effective OAuth settings and scope; they are refreshed or reacquired when unavailable or expired.

## 5. Resolution and execution flow

The active environment is a collection-level session choice. The application remembers the last selected environment locally per collection; switching collections restores that collection's remembered environment when valid. `--env` / `-e <name>` overrides this at startup for the opened collection and is not written to Git.

Resolution order is deterministic:

1. Load and validate collection, active environment, group chain, and request.
2. Merge HTTP configuration (collection defaults, then environment overrides).
3. Resolve environment placeholders `{{name}}` using only the active collection environment.
4. Resolve `${NAME}` from the process environment, including references obtained from environment values.
5. Resolve effective auth using the inheritance rule.
6. Validate resolved URL, method, JSON body, required OAuth values, duration, and TLS configuration.
7. Obtain an OAuth token when effective auth requires it, then build the HTTP request.
8. Execute exactly one request and record either an HTTP response or an execution error.

No request-local variables, response-derived variables, implicit environment fallbacks, recursive cross-environment lookup, or automatic execution of other requests exist in MVP. Missing variables and missing process values fail validation with the variable name and source location, without printing a secret value.

An HTTP status, including 3xx/4xx/5xx, is a valid HTTP response and appears in the response viewer. DNS, connection, timeout, TLS, cancellation, YAML-resolution, and OAuth acquisition failures are execution errors and appear in the diagnostic view.

`--confirm-dangerous` / `-c` enables an optional confirmation mode for the current run. In this mode every `POST`, `PUT`, `PATCH`, or `DELETE` must be explicitly confirmed immediately before sending. It does not infer that an environment called `prod` is dangerous, and the option is neither persisted nor stored in the repository.

## 6. Runtime data, logging, and security

All runtime state is stored under one visible, workspace-local directory:

```text
<workspace-root>/.apitool/
├── responses/<collection-path>/<environment>/<request-id>/latest.json
├── history.jsonl
├── state.json
└── logs/
```

`.apitool/` is mandatory in the workspace root `.gitignore`; it is never added, staged, committed, displayed in application Git diffs, or shared by Git. `state.json` contains only non-secret UI/session preferences, including the last active environment per collection. The collection path, environment, and request ID in the cache layout prevent collisions between requests with the same relative name.

### Response cache

The response cache stores the latest successful HTTP response per request/environment locally: status, received time, duration, headers after redaction, and body. It is an operational convenience, not a definition, is never committed, and must use owner-only filesystem permissions where supported. Cached records carry their request/environment identity so the TUI never presents one environment's response as another's.

### Request history

History is separate from the response cache and is local only. It stores timestamp, workspace/collection, request ID, method, result status (or error category), and duration. It supports search/filter and reopening the referenced request. It must not store secrets, authorization headers, access tokens, client secrets, or raw request bodies.

### Redaction and token inspection

Secrets must never be serialized into collection files by the TUI, written to cache/history/logs, shown in diagnostics, or included in Git diff output generated by the app. At minimum redact values for `Authorization`, `Proxy-Authorization`, `Cookie`, `Set-Cookie`, client secrets, and token values. Logs record safe metadata: request identity, method, host/path with sensitive values redacted, status, duration, OAuth endpoint host, grant, scopes, expiry metadata, and returned OAuth error code.

The Auth view may inspect the current OAuth token, but it starts masked. Revealing the complete token requires an explicit `Show token` action and only displays it in memory for that session; it does not change logging, cache, history, or clipboard behavior. JWT decoding may be added later; opaque tokens remain supported and must not be assumed to be JWTs.

## 7. TUI and interaction design

The primary screen is Postman-like rather than a terminal menu: a persistent left explorer, a request editor above, and a response/diagnostic area below. The request/response split is resizable. The active pane always has a leading `▶` marker in its heading; this marker is required even when color is available, so keyboard focus is visible without relying on color.

```text
┌──────────────────────┬────────────────────────────────────────────────┐
│ ▶ Collections / tree │ GET | {{base_url}}/users              [ Send ]  │
│                      ├────────────────────────────────────────────────┤
│ ▾ payments           │ Params | Headers | Auth | Body | Settings       │
│   ▾ payments         │ request editor                                 │
│     GET list         ├────────────────────────────────────────────────┤
│     POST create      │ Response / Diagnostics / Request Log            │
│   ▸ admin            │ status, headers, formatted JSON, raw body       │
└──────────────────────┴────────────────────────────────────────────────┘
```

The top bar displays active collection, environment, and compact Git status. The explorer reflects physical folder/group/request structure and marks invalid definitions. Collection switching is a first-class action available from the collection selector and command palette; it replaces the tree and restores the per-collection environment/session context.

The editor covers method, URL, params, headers, auth, body, and request name. JSON bodies offer a structured JSON editor by default plus a raw/text mode; the user can switch between modes, pretty-print JSON, and receives validation feedback before send. Non-JSON bodies use raw editing.

Core interaction supports keyboard and mouse. Standard controls include arrows, `Tab`, `Enter`, `Esc`, `Ctrl+S` to save, `Ctrl+Enter` to send, `Ctrl+P` for the command palette, `/` for search, `Ctrl+E` for environment selection, and `?` for keyboard help. Vim bindings (`j/k`, `h/l`) are optional, not required. Status colors distinguish 2xx/3xx/4xx/5xx, warnings, and errors, with text/symbol fallback when color is unavailable.

`?` opens a modal keyboard-help overlay from every TUI mode: collection picker, environment picker, search, and the three-pane collection view. The overlay blocks all input except `?`, `Esc`, and `Ctrl+C`; `?` or `Esc` closes it and restores the exact underlying mode, selection, query, and active pane. `Ctrl+C` always exits. At the approved 100×30 viewport, the overlay is centered above the underlying screen and lists navigation, workspace, search, layout, help, and exit controls. On smaller terminals it uses a compact one-column layout rather than clipping.

The following approved 100×30 visual states define UI-015. Their future automated snapshot names are `collection-focus.txt`, `keyboard-help.txt`, `request-focus.txt`, and `response-focus.txt`.

### Collection pane active

```text
┌───────────────────────────────┬──────────────────────────────────────────────────────────────────┐
│ ▶ Collections / tree          │ GET | https://api.example.test/payments                 [ Send ] │
│ ▾ payments                    ├──────────────────────────────────────────────────────────────────┤
│   ▾ payments                  │ Params | Headers | Auth | Body | Settings                        │
│     GET list                  │                                                                  │
│     POST create               │                                                                  │
│   ▸ admin                     │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               ├──────────────────────────────────────────────────────────────────┤
│                               │ Response / Diagnostics / Request Log                             │
│                               │ No response yet.                                                 │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
├───────────────────────────────┴──────────────────────────────────────────────────────────────────┤
│ payments • test • Tab focus • ? help • Ctrl+C exit                                               │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Keyboard help overlay

```text
┌───────────────────────────────┬──────────────────────────────────────────────────────────────────┐
│ ▶ Collections / tree          │ GET | https://api.example.test/payments                 [ Send ] │
│ ▾ payments                    ├──────────────────────────────────────────────────────────────────┤
│   ▾ payments                  │ Params | Headers | Auth | Body | Settings                        │
│     GET list                  │        ┌──────────── Keyboard shortcuts ────────────┐            │
│     POST create               │        │ Navigate   ↑/↓ select • Enter open           │          │
│   ▸ admin                     │        │ Focus      Tab • ←/→ expand or collapse      │          │
│                               │        │ Workspace  Ctrl+P collections • Ctrl+E env   │          │
│                               │        │ Search     / search                           │         │
│                               │        │ Layout     Ctrl+←/→ resize split              │         │
│                               │        │ Help       ? or Esc close                     │         │
│                               ├────────│ Exit       Ctrl+C                            │──────────┤
│                               │        └─────────────────────────────────────────────┘           │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
├───────────────────────────────┴──────────────────────────────────────────────────────────────────┤
│ payments • test • Tab focus • ? help • Ctrl+C exit                                               │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Request pane active

```text
┌───────────────────────────────┬──────────────────────────────────────────────────────────────────┐
│ Collections / tree            │ ▶ GET | https://api.example.test/payments               [ Send ] │
│ ▾ payments                    ├──────────────────────────────────────────────────────────────────┤
│   ▾ payments                  │ Params | Headers | Auth | Body | Settings                        │
│     GET list                  │                                                                  │
│     POST create               │                                                                  │
│   ▸ admin                     │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               ├──────────────────────────────────────────────────────────────────┤
│                               │ Response / Diagnostics / Request Log                             │
│                               │ No response yet.                                                 │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
├───────────────────────────────┴──────────────────────────────────────────────────────────────────┤
│ payments • test • Tab focus • ? help • Ctrl+C exit                                               │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Response pane active

```text
┌───────────────────────────────┬──────────────────────────────────────────────────────────────────┐
│ Collections / tree            │ GET | https://api.example.test/payments                 [ Send ] │
│ ▾ payments                    ├──────────────────────────────────────────────────────────────────┤
│   ▾ payments                  │ Params | Headers | Auth | Body | Settings                        │
│     GET list                  │                                                                  │
│     POST create               │                                                                  │
│   ▸ admin                     │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               ├──────────────────────────────────────────────────────────────────┤
│                               │ ▶ Response / Diagnostics / Request Log                           │
│                               │ No response yet.                                                 │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
│                               │                                                                  │
├───────────────────────────────┴──────────────────────────────────────────────────────────────────┤
│ payments • test • Tab focus • ? help • Ctrl+C exit                                               │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

Edits are in memory until explicitly saved. On navigation away from a dirty request, offer **Save and continue**, **Discard changes**, or **Cancel**. There is no autosave. Undo/redo applies to the current request editing session. Duplicate creates a new request definition for the user to rename/edit/save. Delete request and delete group perform physical deletion only after a clear confirmation; a group delete includes its contained request files.

The command palette exposes collection/environment switching, request actions, history, diagnostics/log, and Git actions. It is the discoverable home for commands without requiring every feature to have a permanent shortcut.

## 8. Validation, diagnostics, and failure handling

Loading is graceful: one invalid request or group must not stop collection discovery or prevent opening other valid requests. Invalid entries receive a visible warning marker in the tree. Selecting one presents precise diagnostics with file and field path; it can be repaired in the TUI, revalidated on change/save, but cannot be sent until valid.

Validation includes YAML decoding, required fields, HTTP method and URL, no unresolved placeholders, body-type/content compatibility, JSON well-formedness, auth schema, positive timeout, and explicit TLS configuration. The validator reports all independent errors it can safely identify rather than only the first.

Each send creates a timestamped, redacted request log. The diagnostic viewer distinguishes the stage that failed—load, resolution, auth, OAuth token request, HTTP transport, or response parsing/storage—and includes safe facts such as host, status, duration, OAuth error code, and actionable cause hints. HTTP responses are not recast as failures solely because status is non-2xx.

## 9. Thin Git integration

Git is a small adapter over the local system `git` executable. The TUI owns user interaction; the adapter owns command invocation and normalized results. It must not require GitHub or a particular remote.

MVP commands are `Git: Status`, `Git: Diff`, `Git: Pull`, `Git: Push`, and `Git: Commit` (with user-provided commit message). Status is shown compactly in the TUI and detailed output is available on demand. Commands execute from the workspace repository root. Git output is captured for a readable result panel, and failures retain enough safe output to explain the problem. Git operations never stage secrets or runtime cache because those data are outside the workspace; users remain responsible for avoiding literal secrets in YAML.

Branch management, checkout, conflict resolution, staging selection, and provider-specific pull-request actions are deliberately excluded.

## 10. Go component boundaries

Implementation should preserve these boundaries:

```text
cmd/apitool          process setup, CLI flags, workspace selection
internal/workspace   Git-root detection and .api collection discovery
internal/collection  YAML load/save, tree construction, stable request IDs
internal/model       typed collection/environment/group/request/auth models
internal/resolve     variables, inheritance, effective configuration
internal/validate    structural and resolved-definition validation
internal/auth        OAuth2 client-credentials token lifecycle and masking
internal/transport   HTTP request construction, timeout, TLS, execution
internal/runtime     local response cache, history, redacted execution logs
internal/git         thin system-git adapter
internal/tui         Bubble Tea presentation/state/commands only
```

The TUI orchestrates use cases but does not parse YAML, implement OAuth, invoke Git directly, or make HTTP calls itself. Definition loading/saving, resolution, validation, authentication, transport, runtime persistence, and Git should be independently testable through narrow interfaces.

## 11. Verification strategy for implementation planning

- Unit-test YAML decode/encode, discovery, request-ID derivation, inheritance, variable resolution, redaction, validation, and dangerous-operation classification.
- Use `httptest` for HTTP request construction, timeout behavior, response capture, and transport errors.
- Use a fake OAuth server for client-credentials form payload, scopes, token reuse, expiry, and safe OAuth diagnostics.
- Use temporary Git repositories for status/diff/commit/pull/push adapter behavior; do not depend on GitHub.
- Test cache/history isolation by workspace, collection, request, and environment and assert secret/token exclusion.
- Use Bubble Tea model tests for navigation, dirty-save prompts, send states, collection switching, and invalid-definition presentation.
- Add end-to-end smoke tests around a fixture workspace with multiple collections and invalid/valid request definitions.

## 12. Post-MVP roadmap

- OpenAPI/Swagger import and OpenAPI export.
- Request chaining and response-derived variables.
- Batch and parallel request execution.
- mTLS and client certificate/key support.
- Additional authentication methods, including OAuth authorization code where appropriate.
- Richer token inspection (for example, decoded JWT claims) without assuming all tokens are JWTs.

These additions must not weaken the MVP guarantees: Git-readable definitions, no stored secrets/tokens, clear scope boundaries, and a responsive single-request TUI workflow.

## 13. Continuous integration

The repository includes a basic GitHub Actions workflow at `.github/workflows/ci.yml`. Once the local repository is connected to GitHub, it runs on every `push` and `pull_request` with read-only `contents` permission. It uses the Go version declared in `go.mod` and runs formatting verification, `go test ./...`, `go vet ./...`, and `go build ./cmd/apitool` in that order.

CI does not use secrets, publish artifacts, create releases, or test a Go-version matrix in MVP. Its only responsibility is to prevent an unformatted, unbuildable, or failing change from being accepted.
