---
title: apitool MVP acceptance tests
requirements:
  - R-001 Workspace discovery and collection boundaries
  - R-002 Graceful invalid-definition loading
  - R-003 YAML definition editing and structural validation
  - R-004 Stable request tree and inherited group structure
  - R-005 Environment selection and session persistence
  - R-006 Deterministic variable and HTTP configuration resolution
  - R-007 Authentication inheritance and explicit disablement
  - R-008 OAuth2 client-credentials lifecycle and secrecy
  - R-009 Single-request execution and response classification
  - R-010 Local runtime isolation, redaction, and cache/history safety
  - R-011 TUI navigation and accessible status presentation
  - R-012 Explicit editing, save, undo/redo, duplicate, and delete workflow
  - R-013 Send confirmation and response/diagnostic/auth presentation
  - R-014 History and thin local-Git integration
  - R-015 Reproducible multi-collection MVP workflow
  - R-016 Continuous-integration quality gate
design_document: docs/superpowers/specs/2026-09-20-apitool-design.md
implementation_plan: docs/superpowers/plans/2026-09-20-apitool-mvp.md
---

# apitool MVP acceptance tests

## Test case index

| Test case | Category | Observable surface | Requirements | Tasks | Status |
|---|---|---|---|---|---|
| [UI-001 — Select a discovered collection](#ui-001) | UI | TUI | R-001 | T-002, T-008 | planned |
| [UI-002 — Keep invalid definitions visible](#ui-002) | UI | TUI | R-002 | T-001, T-002, T-008 | planned |
| [UI-003 — Render nested request groups](#ui-003) | UI | TUI | R-004 | T-002, T-008 | planned |
| [UI-004 — Navigate without color](#ui-004) | UI | TUI | R-011 | T-008 | planned |
| [UI-015 — Discover keyboard shortcuts and pane focus](#ui-015) | UI | TUI | R-011 | T-008 | planned |
| [CFG-001 — Save a valid request edit](#cfg-001) | CFG | TUI, Filesystem/Git | R-003 | T-001, T-009 | planned |
| [CFG-002 — Block invalid configuration](#cfg-002) | CFG | TUI, Filesystem/Git | R-003 | T-001, T-009 | planned |
| [CFG-003 — Block literal client secrets](#cfg-003) | CFG | TUI, Filesystem/Git | R-003 | T-001, T-009 | planned |
| [CFG-004 — Restore a collection environment](#cfg-004) | CFG | TUI, Filesystem/Git | R-005 | T-005, T-006, T-008 | planned |
| [CFG-005 — Override startup environment](#cfg-005) | CFG | CLI, TUI, Filesystem/Git | R-005 | T-005, T-006, T-008 | planned |
| [CFG-006 — Reject an unknown environment](#cfg-006) | CFG | CLI, TUI | R-005 | T-006, T-008 | planned |
| [API-001 — Send resolved environment values](#api-001) | API | API/Protocol (HTTP) | R-006 | T-003, T-004, T-006 | planned |
| [API-002 — Diagnose a missing variable safely](#api-002) | API | TUI | R-006 | T-003, T-006 | planned |
| [API-003 — Use nearest inherited authentication](#api-003) | API | API/Protocol (HTTP) | R-007 | T-003, T-004, T-006 | planned |
| [API-004 — Disable inherited authentication](#api-004) | API | API/Protocol (HTTP) | R-007 | T-003, T-004, T-006 | planned |
| [API-005 — Request OAuth scopes](#api-005) | API | API/Protocol (OAuth over HTTP) | R-008 | T-004 | planned |
| [API-006 — Reuse and refresh OAuth tokens](#api-006) | API | API/Protocol (OAuth over HTTP) | R-008 | T-004 | planned |
| [API-007 — Report OAuth failures safely](#api-007) | API | TUI | R-008 | T-004, T-005 | planned |
| [UI-005 — Resolve dirty navigation](#ui-005) | UI | TUI, Filesystem/Git | R-012 | T-009 | planned |
| [UI-006 — Limit undo and redo to one request](#ui-006) | UI | TUI | R-012 | T-009 | planned |
| [UI-007 — Switch between JSON and raw body modes](#ui-007) | UI | TUI, Filesystem/Git | R-012 | T-009 | planned |
| [UI-008 — Duplicate or delete definitions safely](#ui-008) | UI | TUI, Filesystem/Git | R-012 | T-009 | planned |
| [UI-009 — Confirm a dangerous send](#ui-009) | UI | TUI, API/Protocol (HTTP) | R-013 | T-006, T-010 | planned |
| [UI-010 — Render completed HTTP error responses](#ui-010) | UI | TUI | R-009 | T-004, T-006, T-010 | planned |
| [UI-011 — Render execution diagnostics](#ui-011) | UI | TUI | R-009 | T-004, T-006, T-010 | planned |
| [UI-012 — Reveal a token only for the session](#ui-012) | UI | TUI | R-013 | T-010 | planned |
| [DATA-001 — Isolate cached responses by environment](#data-001) | DATA | TUI, Filesystem/Git | R-010 | T-005, T-012 | planned |
| [DATA-002 — Redact runtime records](#data-002) | DATA | TUI, Filesystem/Git | R-010 | T-005, T-012 | planned |
| [DATA-003 — Keep runtime state out of Git](#data-003) | DATA | Filesystem/Git | R-010 | T-005, T-012 | planned |
| [UI-013 — Reopen a request from history](#ui-013) | UI | TUI | R-014 | T-005, T-011 | planned |
| [UI-014 — Run safe Git actions](#ui-014) | UI | TUI, Filesystem/Git | R-014 | T-007, T-011 | planned |
| [WF-001 — Complete the local multi-collection workflow](#wf-001) | WF | TUI, API/Protocol (HTTP), Filesystem/Git | R-015 | T-012 | planned |
| [OPS-001 — Enforce the CI quality gate](#ops-001) | OPS | CI | R-016 | T-013 | planned |

## Workspace navigation

<a id="ui-001"></a>
## UI-001 — Select a discovered collection

**Category:** UI<br>
**Observable surface:** TUI<br>
**Requirements:** R-001<br>
**Tasks:** T-002, T-008<br>
**Status:** planned

### Given

A Git workspace contains two directories with `.api/` children.

### When

The user opens the collection picker and selects either collection.

### Then

Both collections are listed, and selecting one opens its request tree.

### Approved screen workflow

**Visual approval:** approved<br>
**Viewport:** 100 columns × 30 rows<br>
**Interaction:** keyboard; `↑`/`↓` moves the selection and `Enter` opens it.<br>
**Snapshot candidate:** true

The workflow needs two screens: the picker proves discovery and selection; the
opened collection proves that the selection changed the active request tree.
The canonical approved snapshots are [Screen 1 — collection picker](../../testdata/ui-001/collection-picker.txt)
and [Screen 2 — selected collection request tree](../../testdata/ui-001/payments-tree.txt).

### Notes

No workspace manifest or Git remote is required.

<a id="ui-002"></a>
## UI-002 — Keep invalid definitions visible

**Category:** UI<br>
**Observable surface:** TUI<br>
**Requirements:** R-002<br>
**Tasks:** T-001, T-002, T-008<br>
**Status:** planned

### Given

A collection contains one malformed definition and one valid sibling request.

### When

The user opens the collection tree and selects the malformed entry.

### Then

The entry remains visible with a warning and precise file/field diagnostic, while the valid sibling remains selectable.

### Notes

Malformed collection metadata is diagnosed rather than silently hidden.

<a id="ui-003"></a>
## UI-003 — Render nested request groups

**Category:** UI<br>
**Observable surface:** TUI<br>
**Requirements:** R-004<br>
**Tasks:** T-002, T-008<br>
**Status:** planned

### Given

A collection has requests under nested group directories and optional `_group.yaml` files.

### When

The user expands the request tree.

### Then

The tree shows the nested groups and their requests, and does not present group metadata as a request.

### Notes

The underlying stable request ID is the collection-relative request path.

<a id="ui-004"></a>
## UI-004 — Navigate without color

**Category:** UI<br>
**Observable surface:** TUI<br>
**Requirements:** R-011<br>
**Tasks:** T-008<br>
**Status:** planned

### Given

The TUI runs without color support and displays an HTTP server-error status.

### When

The user navigates with keyboard controls to a request and its response.

### Then

The request is reachable without a mouse, and text identifies the server-error status without relying on color.

### Notes

Vim bindings are optional.

<a id="ui-015"></a>
## UI-015 — Discover keyboard shortcuts and pane focus

**Category:** UI<br>
**Observable surface:** TUI<br>
**Requirements:** R-011<br>
**Tasks:** T-008<br>
**Status:** planned

### Given

A collection is open in a 100 columns × 30 rows TUI without color support.

### When

The user presses `?`, closes keyboard help with `Esc` or `?`, and presses `Tab` through each primary pane.

### Then

Keyboard help lists navigation, workspace, search, layout, help, and exit controls. Closing it preserves the prior mode, selection, query, environment, and pane focus. Each Tab state shows exactly one textual `▶` focus marker, cycling collection, request, response, and collection; no result relies on color.

### Approved screen workflow

**Visual approval:** approved<br>
**Viewport:** 100 columns × 30 rows<br>
**Interaction:** keyboard; `?` opens help, `Esc` or `?` closes it, and `Tab` changes focus.<br>
**Snapshot candidate:** true

The canonical future snapshots are the four UI-015 states defined in the design specification: collection focus, keyboard help, request focus, and response focus. They are not added until Task 8 implementation begins.

### Notes

`Ctrl+C` exits whether or not help is open. Small terminals use a compact one-column help layout.

## Configuration lifecycle

<a id="cfg-001"></a>
## CFG-001 — Save a valid request edit

**Category:** CFG<br>
**Observable surface:** TUI, Filesystem/Git<br>
**Requirements:** R-003<br>
**Tasks:** T-001, T-009<br>
**Status:** planned

### Given

A user has opened an existing request definition.

### When

They edit its method, URL, headers, and structured JSON body, then explicitly save.

### Then

The saved YAML definition reflects the edit and is visible as a definition change.

### Notes

Methods are normalized to uppercase.

<a id="cfg-002"></a>
## CFG-002 — Block invalid configuration

**Category:** CFG<br>
**Observable surface:** TUI, Filesystem/Git<br>
**Requirements:** R-003<br>
**Tasks:** T-001, T-009<br>
**Status:** planned

### Given

A user is editing a request with a missing required field, unsupported auth/body type, invalid timeout, or invalid JSON body.

### When

They attempt to save or send it.

### Then

The action is blocked with actionable validation feedback and the persisted definition is unchanged.

### Notes

Validation details must not expose secrets.

<a id="cfg-003"></a>
## CFG-003 — Block literal client secrets

**Category:** CFG<br>
**Observable surface:** TUI, Filesystem/Git<br>
**Requirements:** R-003<br>
**Tasks:** T-001, T-009<br>
**Status:** planned

### Given

A user enters a literal value in a client-secret field.

### When

They attempt to save the definition.

### Then

Saving is blocked and the literal secret is absent from the YAML definition and Git-visible change.

### Notes

Process-environment references remain the permitted representation.

<a id="cfg-004"></a>
## CFG-004 — Restore a collection environment

**Category:** CFG<br>
**Observable surface:** TUI, Filesystem/Git<br>
**Requirements:** R-005<br>
**Tasks:** T-005, T-006, T-008<br>
**Status:** planned

### Given

The user previously selected a valid environment for a collection.

### When

They reopen the workspace and select that collection.

### Then

The active-environment display shows the collection's remembered environment.

### Notes

The preference is local session state only.

<a id="cfg-005"></a>
## CFG-005 — Override startup environment

**Category:** CFG<br>
**Observable surface:** CLI, TUI, Filesystem/Git<br>
**Requirements:** R-005<br>
**Tasks:** T-005, T-006, T-008<br>
**Status:** planned

### Given

A collection has a remembered environment different from a valid `--env` value.

### When

The user starts `apitool` with `--env`.

### Then

The opened collection displays the specified environment and repository definitions remain unchanged.

### Notes

The override is not persisted to Git.

<a id="cfg-006"></a>
## CFG-006 — Reject an unknown environment

**Category:** CFG<br>
**Observable surface:** CLI, TUI<br>
**Requirements:** R-005<br>
**Tasks:** T-006, T-008<br>
**Status:** planned

### Given

The user specifies an environment that the collection does not contain.

### When

They start or select the collection with that environment.

### Then

The application reports the unknown environment before opening a request view.

### Notes

No fallback environment is selected.

## Request and OAuth contracts

<a id="api-001"></a>
## API-001 — Send resolved environment values

**Category:** API<br>
**Observable surface:** API/Protocol (HTTP)<br>
**Requirements:** R-006<br>
**Tasks:** T-003, T-004, T-006<br>
**Status:** planned

### Given

An active environment and process environment supply values used by a request targeting a controlled HTTP endpoint.

### When

The user sends the request.

### Then

The endpoint receives the resolved URL, parameters, headers, and body values.

### Notes

Only the active environment and process environment participate in substitution.

<a id="api-002"></a>
## API-002 — Diagnose a missing variable safely

**Category:** API<br>
**Observable surface:** TUI<br>
**Requirements:** R-006<br>
**Tasks:** T-003, T-006<br>
**Status:** planned

### Given

A request contains an unresolved variable, including in a nested JSON value.

### When

The user sends the request.

### Then

Sending is blocked and the diagnostic identifies the variable and field location without displaying a secret value.

### Notes

No request-local, response-derived, implicit-fallback, or recursive cross-environment lookup is used.

<a id="api-003"></a>
## API-003 — Use nearest inherited authentication

**Category:** API<br>
**Observable surface:** API/Protocol (HTTP)<br>
**Requirements:** R-007<br>
**Tasks:** T-003, T-004, T-006<br>
**Status:** planned

### Given

Collection and nested groups define different authentication settings for a request sent to a controlled HTTP endpoint.

### When

The user sends the request.

### Then

The endpoint receives authorization derived from the nearest defined authentication setting.

### Notes

Auth inheritance follows collection, ancestor groups, nearest group, then request.

<a id="api-004"></a>
## API-004 — Disable inherited authentication

**Category:** API<br>
**Observable surface:** API/Protocol (HTTP)<br>
**Requirements:** R-007<br>
**Tasks:** T-003, T-004, T-006<br>
**Status:** planned

### Given

A collection defines OAuth authentication and a request explicitly sets `auth: none`.

### When

The user sends the request to a controlled HTTP endpoint.

### Then

The endpoint receives no Authorization header.

### Notes

Omitting auth has different inheritance behavior.

<a id="api-005"></a>
## API-005 — Request OAuth scopes

**Category:** API<br>
**Observable surface:** API/Protocol (OAuth over HTTP)<br>
**Requirements:** R-008<br>
**Tasks:** T-004<br>
**Status:** planned

### Given

OAuth client-credentials settings specify scopes and use a controlled token endpoint.

### When

The user sends an authenticated request.

### Then

The token endpoint receives a client-credentials request containing the requested scopes.

### Notes

Credentials are supplied through process-environment references.

<a id="api-006"></a>
## API-006 — Reuse and refresh OAuth tokens

**Category:** API<br>
**Observable surface:** API/Protocol (OAuth over HTTP)<br>
**Requirements:** R-008<br>
**Tasks:** T-004<br>
**Status:** planned

### Given

A controlled token endpoint returns a token with a finite expiry.

### When

The user sends requests before and after that token expires.

### Then

The endpoint receives one token request while the token is valid and a new token request after expiry.

### Notes

Tokens remain in process memory only.

<a id="api-007"></a>
## API-007 — Report OAuth failures safely

**Category:** API<br>
**Observable surface:** TUI<br>
**Requirements:** R-008<br>
**Tasks:** T-004, T-005<br>
**Status:** planned

### Given

A controlled token endpoint returns `invalid_client`.

### When

The user sends the authenticated request.

### Then

The diagnostic shows the OAuth error code without showing the client secret, access token, or raw Authorization value.

### Notes

Opaque tokens remain supported.

## Editing and execution

<a id="ui-005"></a>
## UI-005 — Resolve dirty navigation

**Category:** UI<br>
**Observable surface:** TUI, Filesystem/Git<br>
**Requirements:** R-012<br>
**Tasks:** T-009<br>
**Status:** planned

### Given

The current request has unsaved edits.

### When

The user attempts to navigate to another request.

### Then

The TUI offers Save and continue, Discard changes, and Cancel without autosaving.

### Notes

Discard and Cancel leave the persisted definition unchanged.

<a id="ui-006"></a>
## UI-006 — Limit undo and redo to one request

**Category:** UI<br>
**Observable surface:** TUI<br>
**Requirements:** R-012<br>
**Tasks:** T-009<br>
**Status:** planned

### Given

The user has edited one request and then opens another request.

### When

They invoke undo or redo in the second request.

### Then

Only edits made during the second request's editing session can be undone or redone.

### Notes

Undo history resets when another request is opened.

<a id="ui-007"></a>
## UI-007 — Switch between JSON and raw body modes

**Category:** UI<br>
**Observable surface:** TUI, Filesystem/Git<br>
**Requirements:** R-012<br>
**Tasks:** T-009<br>
**Status:** planned

### Given

A request body is edited in JSON mode and in raw mode.

### When

The user saves each body mode.

### Then

JSON is validated and pretty-printed, while raw body text is preserved exactly.

### Notes

Substitution occurs before a raw body is sent.

<a id="ui-008"></a>
## UI-008 — Duplicate or delete definitions safely

**Category:** UI<br>
**Observable surface:** TUI, Filesystem/Git<br>
**Requirements:** R-012<br>
**Tasks:** T-009<br>
**Status:** planned

### Given

The user has selected a request or group.

### When

They duplicate the request or begin deletion.

### Then

Duplicate creates a distinct definition for editing, and deletion lists affected paths and changes files only after confirmation.

### Notes

Deleting a group includes its contained request files.

<a id="ui-009"></a>
## UI-009 — Confirm a dangerous send

**Category:** UI<br>
**Observable surface:** TUI, API/Protocol (HTTP)<br>
**Requirements:** R-013<br>
**Tasks:** T-006, T-010<br>
**Status:** planned

### Given

`--confirm-dangerous` is active for a DELETE request targeting a controlled HTTP endpoint.

### When

The user accepts or rejects the confirmation.

### Then

The confirmation shows environment, method, and safe URL; acceptance sends once and rejection sends none.

### Notes

POST, PUT, and PATCH use the same confirmation rule; GET sends immediately.

<a id="ui-010"></a>
## UI-010 — Render completed HTTP error responses

**Category:** UI<br>
**Observable surface:** TUI<br>
**Requirements:** R-009<br>
**Tasks:** T-004, T-006, T-010<br>
**Status:** planned

### Given

A controlled HTTP endpoint returns a completed 3xx, 4xx, or 5xx response.

### When

The user sends the request.

### Then

The lower pane renders the completed exchange as a response with status, duration, headers, and body rather than as a request failure.

### Notes

JSON bodies may be formatted with raw-text fallback.

<a id="ui-011"></a>
## UI-011 — Render execution diagnostics

**Category:** UI<br>
**Observable surface:** TUI<br>
**Requirements:** R-009<br>
**Tasks:** T-004, T-006, T-010<br>
**Status:** planned

### Given

Sending fails during resolution, OAuth, transport, TLS, timeout, or cancellation.

### When

The user views the result.

### Then

The lower pane renders a diagnostic with the failure stage, category, safe message, and redacted request log.

### Notes

The failure is not represented as an HTTP response.

<a id="ui-012"></a>
## UI-012 — Reveal a token only for the session

**Category:** UI<br>
**Observable surface:** TUI<br>
**Requirements:** R-013<br>
**Tasks:** T-010<br>
**Status:** planned

### Given

The Auth view has an active OAuth token.

### When

The user opens the view and then selects Show token.

### Then

The token starts masked and is fully shown only after that explicit action for the current session.

### Notes

Revealing does not copy or persist the token.

## Runtime safety

<a id="data-001"></a>
## DATA-001 — Isolate cached responses by environment

**Category:** DATA<br>
**Observable surface:** TUI, Filesystem/Git<br>
**Requirements:** R-010<br>
**Tasks:** T-005, T-012<br>
**Status:** planned

### Given

The same request has completed under `prod` and `test` with distinct responses.

### When

The user opens that request under `test`.

### Then

The response view and cache identify and show only the `test` response.

### Notes

Collection path, environment, and request ID prevent cache collisions.

<a id="data-002"></a>
## DATA-002 — Redact runtime records

**Category:** DATA<br>
**Observable surface:** TUI, Filesystem/Git<br>
**Requirements:** R-010<br>
**Tasks:** T-005, T-012<br>
**Status:** planned

### Given

An execution produces authorization headers, cookies, OAuth values, and a raw request body.

### When

The user inspects history, logs, and cached metadata.

### Then

Those records contain none of the sensitive values or raw request body.

### Notes

Safe metadata such as request identity, status, and duration may remain.

<a id="data-003"></a>
## DATA-003 — Keep runtime state out of Git

**Category:** DATA<br>
**Observable surface:** Filesystem/Git<br>
**Requirements:** R-010<br>
**Tasks:** T-005, T-012<br>
**Status:** planned

### Given

The user has executed requests and changed local session preferences.

### When

They inspect runtime files and Git status/diff.

### Then

Runtime artifacts exist only below `.apitool/`, are absent from Git status/diff, and state contains only non-secret preferences.

### Notes

Cached runtime records use owner-only permissions where supported.

## History and Git

<a id="ui-013"></a>
## UI-013 — Reopen a request from history

**Category:** UI<br>
**Observable surface:** TUI<br>
**Requirements:** R-014<br>
**Tasks:** T-005, T-011<br>
**Status:** planned

### Given

Local history contains a prior request result.

### When

The user searches history and opens the entry.

### Then

The TUI reopens the current referenced definition rather than a saved request-body snapshot.

### Notes

History is ordered newest first.

<a id="ui-014"></a>
## UI-014 — Run safe Git actions

**Category:** UI<br>
**Observable surface:** TUI, Filesystem/Git<br>
**Requirements:** R-014<br>
**Tasks:** T-007, T-011<br>
**Status:** planned

### Given

The workspace is a local Git repository with a staged definition change.

### When

The user invokes Status, Diff, Commit, Pull, or Push from the command palette.

### Then

The TUI displays safe command output or failure details, does not stage files, and rejects a blank commit message.

### Notes

The actions run at the workspace root and require no provider-specific remote.

## End-to-end and operations

<a id="wf-001"></a>
## WF-001 — Complete the local multi-collection workflow

**Category:** WF<br>
**Observable surface:** TUI, API/Protocol (HTTP), Filesystem/Git<br>
**Requirements:** R-015<br>
**Tasks:** T-012<br>
**Status:** planned

### Given

A local fixture repository has two valid collections, an invalid sibling definition, process-environment credentials, and a local HTTP server.

### When

The user opens the workspace, selects `test`, opens a valid request, executes it, and inspects history and Git status.

### Then

The valid request completes once, the invalid sibling remains visibly marked, local history/cache are available, and runtime data is absent from Git status.

### Notes

The workflow makes no external network call and must not expose fixture credentials.

<a id="ops-001"></a>
## OPS-001 — Enforce the CI quality gate

**Category:** OPS<br>
**Observable surface:** CI<br>
**Requirements:** R-016<br>
**Tasks:** T-013<br>
**Status:** planned

### Given

The repository has its declared Go version and the CI workflow.

### When

CI runs for a push or pull request.

### Then

The workflow uses the Go version from `go.mod`, requests only `contents: read`, and fails when formatting, tests, vet, or build fails.

### Notes

It uses no secrets, artifacts, releases, or version matrix.

## Traceability report

| Test case | Category | Observable surface | Requirement | Task | Status |
|---|---|---|---|---|---|
| UI-001 | UI | TUI | R-001 | T-002, T-008 | planned |
| UI-002 | UI | TUI | R-002 | T-001, T-002, T-008 | planned |
| UI-003 | UI | TUI | R-004 | T-002, T-008 | planned |
| UI-004 | UI | TUI | R-011 | T-008 | planned |
| UI-015 | UI | TUI | R-011 | T-008 | planned |
| CFG-001 | CFG | TUI, Filesystem/Git | R-003 | T-001, T-009 | planned |
| CFG-002 | CFG | TUI, Filesystem/Git | R-003 | T-001, T-009 | planned |
| CFG-003 | CFG | TUI, Filesystem/Git | R-003 | T-001, T-009 | planned |
| CFG-004 | CFG | TUI, Filesystem/Git | R-005 | T-005, T-006, T-008 | planned |
| CFG-005 | CFG | CLI, TUI, Filesystem/Git | R-005 | T-005, T-006, T-008 | planned |
| CFG-006 | CFG | CLI, TUI | R-005 | T-006, T-008 | planned |
| API-001 | API | API/Protocol (HTTP) | R-006 | T-003, T-004, T-006 | planned |
| API-002 | API | TUI | R-006 | T-003, T-006 | planned |
| API-003 | API | API/Protocol (HTTP) | R-007 | T-003, T-004, T-006 | planned |
| API-004 | API | API/Protocol (HTTP) | R-007 | T-003, T-004, T-006 | planned |
| API-005 | API | API/Protocol (OAuth over HTTP) | R-008 | T-004 | planned |
| API-006 | API | API/Protocol (OAuth over HTTP) | R-008 | T-004 | planned |
| API-007 | API | TUI | R-008 | T-004, T-005 | planned |
| UI-005 | UI | TUI, Filesystem/Git | R-012 | T-009 | planned |
| UI-006 | UI | TUI | R-012 | T-009 | planned |
| UI-007 | UI | TUI, Filesystem/Git | R-012 | T-009 | planned |
| UI-008 | UI | TUI, Filesystem/Git | R-012 | T-009 | planned |
| UI-009 | UI | TUI, API/Protocol (HTTP) | R-013 | T-006, T-010 | planned |
| UI-010 | UI | TUI | R-009 | T-004, T-006, T-010 | planned |
| UI-011 | UI | TUI | R-009 | T-004, T-006, T-010 | planned |
| UI-012 | UI | TUI | R-013 | T-010 | planned |
| DATA-001 | DATA | TUI, Filesystem/Git | R-010 | T-005, T-012 | planned |
| DATA-002 | DATA | TUI, Filesystem/Git | R-010 | T-005, T-012 | planned |
| DATA-003 | DATA | Filesystem/Git | R-010 | T-005, T-012 | planned |
| UI-013 | UI | TUI | R-014 | T-005, T-011 | planned |
| UI-014 | UI | TUI, Filesystem/Git | R-014 | T-007, T-011 | planned |
| WF-001 | WF | TUI, API/Protocol (HTTP), Filesystem/Git | R-015 | T-012 | planned |
| OPS-001 | OPS | CI | R-016 | T-013 | planned |

### Requirements without acceptance coverage

None identified in the MVP design scope.

### Tasks without direct acceptance coverage

None identified. Future private implementation tasks may be intentionally covered only by unit or integration tests; no acceptance case is required unless the task contributes to a named observable contract.

### Ambiguities that block a trustworthy case

None identified. apitool has no database requirement, so no scenario uses the `Database` surface.
