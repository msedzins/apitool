# Priority 1 TUI screen designs

Status: draft for review. Date: 2026-09-30.

Review decision: retain the accepted MVP behavior. Draft proposals for sending unsaved drafts, external malformed-YAML repair, and dirty quit/environment guards remain flagged differences, not accepted requirements. See the [scenario ambiguities](../../acceptance-tests/2026-09-20-apitool-mvp.md#ambiguities-that-block-a-trustworthy-case).

## Intent and scope

Design the seven priority-1 areas identified in the API tool audit: request editor, body editor, authentication, send/response lifecycle, diagnostics/repair, unsaved changes, and dangerous-send confirmation. The user requested a separate worktree and confirmed screen designs first. These are proposed layouts and interaction rules, not an implemented TUI or approved snapshot baselines.

The current three-pane collection layout, Git-readable definitions, explicit Save, single-request execution and process-environment secret references remain the product constraints. Success is a developer being able to edit a request, understand validation, execute exactly the intended draft, inspect its result, and navigate safely using the keyboard.

The screen mockups below compare 21 states at 100×30 and 80×24. `screens/` contains 42 plain-text mockups; `states.json` records their explanations. No approved `testdata/` files or application code are changed.

## Approach

Recommended: retain the explorer and use request tabs, the lower result pane, and centered confirmation dialogs. This preserves context and the approved shell while providing enough detail for editing and execution.

Alternatives considered: full-screen editor/result modes provide more space but interrupt context; separate wizard screens simplify initial setup but add steps to repeated editing. The proposed layout reserves the full request editor for focused editing when needed: a multiline text editor can expand into the available request/response area and restore its previous split on close. The small viewport retains the three-pane shell; long lines scroll horizontally in editing rather than being silently truncated. Mockup ellipses show overflow only.

## Shared behavior

- Exactly one pane has the non-color `▶` focus marker. Inside a pane, `>` marks selected rows and brackets mark the active tab. Dialogs capture focus; `*` in these mockups marks the default dialog action.
- Tab cycles panes when browsing. In an active field or dialog it cycles its controls; Escape exits a field, then its dialog, restoring the prior focus. Arrow keys navigate rows and tabs when not editing; typing edits only inside an active field. Global help and search must not consume literal `?` or `/` in text fields; help is also available through a labelled control.
- Ctrl+S explicitly saves the draft. Ctrl+Enter starts a send. Ctrl+Z/Ctrl+Y undo/redo within the current request session. Undo history resets after changing requests. Save acknowledgement changes the dirty indicator without changing previous response data.
- Explorer row selection previews that same request. Moving off a dirty request invokes the unsaved dialog before replacing its draft. There is no fallback to a sibling's editor when an invalid row is selected.
- Dirty is computed from the serialized draft versus its last saved definition. Successful Save resets the baseline. Undo back to the baseline clears dirty.
- At 80×24, tab strips and editors scroll to keep the active item visible. Dialog content wraps and scrolls if necessary; Cancel remains accessible. At 100×30, preserve the wider layout. Focus, warnings, status and disabled actions must remain intelligible without color.

## Editing contract

Name, method, URL, params, headers, body and request auth are editable. Request path is displayed read only; rename/move is outside this pass. Params and Headers are key/value tables. Insert adds a row; Delete on a selected row removes it as an undoable draft change. Enter starts and commits cell editing; Escape restores the previous value. Empty values are allowed; empty keys and duplicate keys are diagnosed. Header duplicates compare case-insensitively.

Structural validation runs after committing an edit and before Save. Invalid drafts remain editable and block Save and Send. Resolution failures such as missing process variables block Send, but do not prevent saving a structurally valid reusable definition. Save failures retain the draft and dirty flag.

Settings shows effective timeout and TLS verification with collection/environment provenance. It does not create unsupported request-local HTTP settings. Collection/group/environment editing belongs to a subsequent design pass.

JSON uses one typed value with Structured and Text presentations. Structured rows expose name, type, value and nested expansion; objects/arrays support insertion/removal. Text mode has line numbers, parser location and Pretty-print. Invalid text stays in the draft; switching to Structured is blocked until parsing succeeds. Formatting failures leave the text unchanged. A valid JSON-to-Raw type change produces formatted text; Raw-to-JSON requires successful parsing. Raw saves exact text. Payload type and presentation mode must be distinct controls in implementation.

## Authentication contract

Inherit means omit request auth; None explicitly disables it; OAuth2 creates a request override supporting only client_credentials. Inherited fields show source and are read only. Switching away from an override is undoable within the draft session.

OAuth fields show endpoint, client ID reference, client secret reference and scopes. The secret input accepts `${PROCESS_VARIABLE}` references and rejects literal secrets. Resolved credential values never appear in field previews, diagnostics, cache/history or logs.

Token state is not acquired, acquiring, valid or expired/unavailable. The example valid token is masked until Show token. Reveal is a view-local action; Hide token, leaving Auth, request/environment/config changes, token replacement and session exit remask it. Synthetic token text in the gallery is not a real credential. No clipboard action is implied. Token acquisition occurs on Send, not opening Auth.

## Sending and results

Proposed decision: Send uses the current in-memory draft without writing it to disk. The saved file remains unchanged until explicit Save. This requires a future application-service change: the current Send API consumes the loaded definition.

Validate and resolve the draft, freeze the request/environment snapshot, then apply optional dangerous-method confirmation. POST/PUT/PATCH/DELETE require the dialog only when --confirm-dangerous is enabled; an environment name alone never triggers it. The dialog shows method, active environment, resolved redacted URL and whether the draft is unsaved. Cancel has initial focus. Enter activates the focused action; confirming queues exactly one request. No request or token acquisition occurs on cancellation.

While sending, show elapsed time and Cancel request, disable repeat sends and draft mutations, and keep result scrolling available. Cancel aborts the client context but cannot guarantee the server did not execute the request. Timeout and cancellation of mutations must say the server outcome may be unknown. Do not retry automatically.

Responses show HTTP status, duration, byte size, environment, received time and Live/Cached source. Body supports Pretty JSON and Raw; invalid JSON uses raw fallback with a notice. Headers are redacted. 204/empty body is an explicit empty state. Completed 3xx/4xx/5xx are responses rather than transport errors.

After edits, previous results are labelled as earlier executions and may differ from the draft. Request/environment cache keys never mix. Changing environment loads that environment's cached response or shows No response yet. A new failure does not overwrite a previous successful cache entry or present that entry as the result of the failed send. Response arrival belongs to the frozen execution identity.

## Diagnostics and repair

The lower pane uses Response / Diagnostics / Request Log tabs. Diagnostics list stage, category, source file/field, safe explanation and a recovery action. Show all independent structural errors. Go to field focuses parsed editable fields. Resolution errors identify the missing reference without its secret value. OAuth diagnostics show safe error code and endpoint host; transport diagnostics distinguish DNS, TLS, timeout and cancellation. Storage failures after a completed HTTP response preserve the response and add a local persistence warning.

Proposed exception to the original spec's universal TUI-repair promise: malformed YAML that cannot be represented as a typed request shows file and parser location, disables request controls, and provides external-repair instructions plus Reload. Validly parsed field errors can be repaired in the typed editor. Reload replaces an invalid definition only after it can be loaded; otherwise the safe diagnostics remain. This is a proposed scope decision for review, not a silent specification change.

Request Log shows timestamped safe stage metadata. Do not log token/credential values, cookies, authorization headers, raw request bodies or unredacted sensitive URL values.

## Unsaved navigation

Intercept changing requests, collections, environments, and quit while dirty. The dialog identifies the draft and destination and offers Save and continue, Discard changes, Cancel. Cancel is initially focused and restores exact selection, pane focus, tab and scroll. Discard changes only affects the in-memory draft. Save and continue validates and writes first; navigate only after success. Failed validation returns to the relevant editable field; filesystem failure retains the dialog and pending destination with Retry save / Return to editing. Ctrl+C uses the same quit guard once editing exists; document this proposed change from today's immediate exit behavior.

Acceptance scenarios and requirement/task mappings are maintained in the [acceptance scenarios](../../acceptance-tests/2026-09-20-apitool-mvp.md#test-case-index).

## Decisions for review

1. Execute the current draft without autosaving; preserve explicit Save.
2. Use typed repair for parsed errors and external repair plus Reload for malformed YAML.
3. Apply the dirty guard to environment switching and Ctrl+C quit as well as request/collection navigation.
4. Retain the three-pane shell; expand multiline editing when needed rather than introduce separate wizard screens.

No implementation starts from this draft. After design approval, reconcile the original specification and acceptance cases, then write the implementation plan before changing the application.

## Screen index

- [Request editor: query parameters](#01-params)

- [Request editor: headers](#02-headers)

- [Body: structured JSON](#03-json)

- [Body: invalid JSON text](#04-json-invalid)

- [Body: raw payload](#05-raw)

- [Request identity and effective settings](#06-settings)

- [Auth: inherited OAuth](#07-auth-inherited)

- [Auth: request override](#08-auth-override)

- [Auth: explicit token reveal](#09-auth-reveal)

- [Execution: sending current draft](#10-sending)

- [Response: completed JSON exchange](#11-response)

- [Response: completed HTTP error](#12-http-error)

- [Response: cached result](#13-cached)

- [Response: no content](#14-empty-body)

- [Diagnostics: missing process variable](#15-validation)

- [Diagnostics: transport timeout](#16-transport)

- [Diagnostics: redacted request log](#17-log)

- [Diagnostics: invalid selected definition](#18-invalid-definition)

- [Unsaved changes: navigation](#19-dirty-navigation)

- [Unsaved changes: save failed](#20-save-failure)

- [Dangerous send: explicit confirmation](#21-confirm-send)

<a id="01-params"></a>

## Request editor: query parameters

Typed table editing; display parameter values as text. Commit a cell with Enter; Esc reverts that cell.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │[Params] | Headers | Auth | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Name: Create payment *                                                  │
│  > POST create          │Query parameters (Enter edit; Insert add; Delete remove)                │
│    ! broken             │  Key           Value                                                   │
│                         │> dry_run       false                                                   │
│▸ users                  │  api_version   {{api_version}}                                         │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │No response yet.                                                        │
│                         │Send executes the draft; Save writes the definition.                    │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │[Params] | Headers | Auth | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Name: Create payment *                                   │
│  > POST create     │Query parameters (Enter edit; Insert add; Delete remove) │
│    ! broken        │  Key           Value                                    │
│                    │> dry_run       false                                    │
│▸ users             │  api_version   {{api_version}}                          │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │No response yet.                                         │
│                    │Send executes the draft; Save writes the definition.     │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="02-headers"></a>

## Request editor: headers

Case-insensitive duplicate header names are validation errors. Mask sensitive resolved values; reference expressions remain editable.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | [Headers] | Auth | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Name: Create payment *                                                  │
│  > POST create          │Headers (Enter edit; Insert add; Delete remove)                         │
│    ! broken             │  Key                Value                                              │
│                         │> Accept             application/json                                   │
│▸ users                  │  Content-Type       application/json                                   │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │No response yet.                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | [Headers] | Auth | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Name: Create payment *                                   │
│  > POST create     │Headers (Enter edit; Insert add; Delete remove)          │
│    ! broken        │  Key                Value                               │
│                    │> Accept             application/json                    │
│▸ users             │  Content-Type       application/json                    │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │No response yet.                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="03-json"></a>

## Body: structured JSON

Structured JSON supports nested objects and arrays, typed values and field rename. Expand with arrows; edit a leaf with Enter. Text view edits the same JSON value.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | Text                               │
│  > POST create          │> amount     number   1200                                              │
│    ! broken             │  currency   string   PLN                                               │
│                         │  metadata   object   {...}                                             │
│▸ users                  │Enter edit value; Insert add field; Delete remove                       │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │No response yet.                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | Text                │
│  > POST create     │> amount     number   1200                               │
│    ! broken        │  currency   string   PLN                                │
│                    │  metadata   object   {...}                              │
│▸ users             │Enter edit value; Insert add field; Delete remove        │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │No response yet.                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="04-json-invalid"></a>

## Body: invalid JSON text

Preserve invalid text in the draft. Block Structured view until parseable. Pretty-print parses first and preserves text on error.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send disabled]  [Save] * Unsaved                                       │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | [Text]                             │
│  > POST create          │1 {                                                                     │
│    ! broken             │2   "amount": 1200,                                                     │
│                         │3   "currency": "PLN",                                                  │
│▸ users                  │4 }                                                                     │
│                         │! Line 4, column 1: trailing comma                                      │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │Validation blocked: JSON body is invalid.                               │
│                         │Fix the text before Save or Send.                                       │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send disabled]  [Save] * Unsaved                        │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | [Text]              │
│  > POST create     │1 {                                                      │
│    ! broken        │2   "amount": 1200,                                      │
│                    │3   "currency": "PLN",                                   │
│▸ users             │4 }                                                      │
│                    │! Line 4, column 1: trailing comma                       │
│                    ├─────────────────────────────────────────────────────────┤
│                    │Validation blocked: JSON body is invalid.                │
│                    │Fix the text before Save or Send.                        │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="05-raw"></a>

## Body: raw payload

JSON Structured/Text changes presentation only. JSON/Raw changes payload type: valid JSON converts to formatted text; Raw to JSON requires successful parsing and leaves the original intact on failure.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: Raw    Content-Type: application/xml                         │
│  > POST create          │1 <payment>                                                             │
│    ! broken             │2   <amount>1200</amount>                                               │
│                         │3   <currency>PLN</currency>                                            │
│▸ users                  │4 </payment>                                                            │
│                         │Text is preserved exactly when saved.                                   │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │No response yet.                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: Raw    Content-Type: application/xml          │
│  > POST create     │1 <payment>                                              │
│    ! broken        │2   <amount>1200</amount>                                │
│                    │3   <currency>PLN</currency>                             │
│▸ users             │4 </payment>                                             │
│                    │Text is preserved exactly when saved.                    │
│                    ├─────────────────────────────────────────────────────────┤
│                    │No response yet.                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="06-settings"></a>

## Request identity and effective settings

Settings shows effective collection/environment HTTP values without inventing request-local HTTP overrides. Name is editable; request path is not renamed here.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  Saved                                                           │
│▾ payments               │Params | Headers | Auth | Body | [Settings]                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Name: Create payment                                                    │
│  > POST create          │File: payments/create.yaml (read only)                                  │
│    ! broken             │Timeout: 30s (environment: test)                                        │
│                         │TLS certificate verification: enabled (collection)                      │
│▸ users                  │Request HTTP overrides: unavailable in MVP                              │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │Saved. Definition updated; response remains unchanged.                  │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  Saved                                            │
│▾ payments          │Params | Headers | Auth | Body | [Settings]              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Name: Create payment                                     │
│  > POST create     │File: payments/create.yaml (read only)                   │
│    ! broken        │Timeout: 30s (environment: test)                         │
│                    │TLS certificate verification: enabled (collection)       │
│▸ users             │Request HTTP overrides: unavailable in MVP               │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │Saved. Definition updated; response remains unchanged.   │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="07-auth-inherited"></a>

## Auth: inherited OAuth

Inherited fields are read only. Choosing OAuth2 creates a request override. None disables inherited auth. Token metadata belongs to the current effective configuration/environment.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | [Auth] | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Auth: [Inherit] | None | OAuth2                                         │
│  > POST create          │Effective source: collection Payments API                               │
│    ! broken             │Grant: client_credentials                                               │
│                         │Endpoint: {{oauth_token_url}}                                           │
│▸ users                  │Scopes: payments.write                                                  │
│                         │Token: ********  expires in 24m  [Show token]                           │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │No response yet.                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | [Auth] | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Auth: [Inherit] | None | OAuth2                          │
│  > POST create     │Effective source: collection Payments API                │
│    ! broken        │Grant: client_credentials                                │
│                    │Endpoint: {{oauth_token_url}}                            │
│▸ users             │Scopes: payments.write                                   │
│                    │Token: ********  expires in 24m  [Show token]            │
│                    ├─────────────────────────────────────────────────────────┤
│                    │No response yet.                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="08-auth-override"></a>

## Auth: request override

Credentials accept process environment references. Literal client secrets are rejected. Switching to Inherit removes the request override; undo can restore it.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | [Auth] | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Auth: Inherit | None | [OAuth2]                                         │
│  > POST create          │Grant: client_credentials                                               │
│    ! broken             │Endpoint: {{oauth_token_url}}                                           │
│                         │Client ID: ${PAYMENTS_CLIENT_ID}                                        │
│▸ users                  │Client secret: ${PAYMENTS_CLIENT_SECRET}                                │
│                         │Scopes: payments.write                                                  │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │Token: not acquired. Acquired on Send.                                  │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | [Auth] | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Auth: Inherit | None | [OAuth2]                          │
│  > POST create     │Grant: client_credentials                                │
│    ! broken        │Endpoint: {{oauth_token_url}}                            │
│                    │Client ID: ${PAYMENTS_CLIENT_ID}                         │
│▸ users             │Client secret: ${PAYMENTS_CLIENT_SECRET}                 │
│                    │Scopes: payments.write                                   │
│                    ├─────────────────────────────────────────────────────────┤
│                    │Token: not acquired. Acquired on Send.                   │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="09-auth-reveal"></a>

## Auth: explicit token reveal

Synthetic example only. Remask on leaving Auth, changing request/environment/effective auth, token replacement, or closing the session. No automatic clipboard action.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | [Auth] | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Auth: [Inherit] | None | OAuth2                                         │
│  > POST create          │Token visibility: revealed for this view                                │
│    ! broken             │Token: demo-opaque-token-not-a-real-credential                          │
│                         │[Hide token]                                                            │
│▸ users                  │Expires: 2026-09-30 12:30 UTC                                           │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │Token display never changes cache, logs or history.                     │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | [Auth] | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Auth: [Inherit] | None | OAuth2                          │
│  > POST create     │Token visibility: revealed for this view                 │
│    ! broken        │Token: demo-opaque-token-not-a-real-credential           │
│                    │[Hide token]                                             │
│▸ users             │Expires: 2026-09-30 12:30 UTC                            │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │Token display never changes cache, logs or history.      │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="10-sending"></a>

## Execution: sending current draft

Freeze a draft and environment snapshot for this send. Disable Send and mutating controls until completion; keep response scrolling and cancellation available.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Sending... / Cancel]  [Save] * Unsaved                                 │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | Text                               │
│  > POST create          │  amount     number   1200                                              │
│    ! broken             │  currency   string   PLN                                               │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │Sending draft... elapsed 1.2s                                           │
│                         │POST https://test.example.com/payments                                  │
│                         │[Cancel request]                                                        │
│                         │Any previous response is retained as cached data.                       │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Sending... / Cancel]  [Save] * Unsaved                  │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | Text                │
│  > POST create     │  amount     number   1200                               │
│    ! broken        │  currency   string   PLN                                │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │Sending draft... elapsed 1.2s                            │
│                    │POST https://test.example.com/payments                   │
│                    │[Cancel request]                                         │
│                    │Any previous response is retained as cached data.        │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="11-response"></a>

## Response: completed JSON exchange

The result is linked to the sent draft snapshot. Raw view shows response text; headers are redacted. Invalid JSON falls back to raw text with a parsing notice.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | Text                               │
│  > POST create          │  amount     number   1200                                              │
│    ! broken             │  currency   string   PLN                                               │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │201 Created | 184ms | 48 B | test | Live                                │
│                         │Received: 2026-09-30 12:06:00 UTC                                       │
│                         │[Body] Headers | Pretty JSON / Raw                                      │
│                         │{ "id": "pay_123", "status": "created" }                                │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | Text                │
│  > POST create     │  amount     number   1200                               │
│    ! broken        │  currency   string   PLN                                │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │201 Created | 184ms | 48 B | test | Live                 │
│                    │Received: 2026-09-30 12:06:00 UTC                        │
│                    │[Body] Headers | Pretty JSON / Raw                       │
│                    │{ "id": "pay_123", "status": "created" }                 │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="12-http-error"></a>

## Response: completed HTTP error

3xx/4xx/5xx stay responses. No automatic retries of mutation requests.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  Saved                                                           │
│▾ payments               │Params | [Headers] | Auth | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Name: Create payment                                                    │
│  > POST create          │Accept: application/json                                                │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │401 Unauthorized | 92ms | 36 B | test | Live                            │
│                         │[Body] Headers | Pretty JSON / Raw                                      │
│                         │{ "error": "insufficient_scope" }                                       │
│                         │HTTP exchange completed. Inspect body and auth.                         │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  Saved                                            │
│▾ payments          │Params | [Headers] | Auth | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Name: Create payment                                     │
│  > POST create     │Accept: application/json                                 │
│    ! broken        │                                                         │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │401 Unauthorized | 92ms | 36 B | test | Live             │
│                    │[Body] Headers | Pretty JSON / Raw                       │
│                    │{ "error": "insufficient_scope" }                        │
│                    │HTTP exchange completed. Inspect body and auth.          │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="13-cached"></a>

## Response: cached result

On environment change load only that environment cache; show No response if absent. On edit label any previous result as a previous execution, never a result of the current draft.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │[Params] | Headers | Auth | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Name: Create payment *                                                  │
│  > POST create          │Query: dry_run=false                                                    │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │201 Created | 184ms | 48 B | test | Cached                              │
│                         │Received: 2026-09-30 12:06:00 UTC                                       │
│                         │Previous execution; current draft may differ.                           │
│                         │{ "id": "pay_123", "status": "created" }                                │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │[Params] | Headers | Auth | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Name: Create payment *                                   │
│  > POST create     │Query: dry_run=false                                     │
│    ! broken        │                                                         │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │201 Created | 184ms | 48 B | test | Cached               │
│                    │Received: 2026-09-30 12:06:00 UTC                        │
│                    │Previous execution; current draft may differ.            │
│                    │{ "id": "pay_123", "status": "created" }                 │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="14-empty-body"></a>

## Response: no content

Empty payload is an explicit result state, not a viewer error.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  Saved                                                           │
│▾ payments               │Params | [Headers] | Auth | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Name: Create payment                                                    │
│  > POST create          │Accept: application/json                                                │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │204 No Content | 73ms | 0 B | test | Live                               │
│                         │[Body] Headers                                                          │
│                         │This response has no body.                                              │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  Saved                                            │
│▾ payments          │Params | [Headers] | Auth | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Name: Create payment                                     │
│  > POST create     │Accept: application/json                                 │
│    ! broken        │                                                         │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │204 No Content | 73ms | 0 B | test | Live                │
│                    │[Body] Headers                                           │
│                    │This response has no body.                               │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="15-validation"></a>

## Diagnostics: missing process variable

Diagnostics list all independent safe errors, offer Go to field where editable, and never print the secret value. Structural errors block Save; resolution errors block Send.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | [Auth] | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Auth: Inherit | None | [OAuth2]                                         │
│  > POST create          │Client secret: ${PAYMENTS_CLIENT_SECRET}                                │
│    ! broken             │! Process variable is unavailable                                       │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │[Diagnostics] Response Request Log                                      │
│                         │Stage: resolution | missing_process_variable                            │
│                         │payments/create.yaml > auth.client_secret                               │
│                         │Set PAYMENTS_CLIENT_SECRET before starting apitool.                     │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | [Auth] | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Auth: Inherit | None | [OAuth2]                          │
│  > POST create     │Client secret: ${PAYMENTS_CLIENT_SECRET}                 │
│    ! broken        │! Process variable is unavailable                        │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │[Diagnostics] Response Request Log                       │
│                    │Stage: resolution | missing_process_variable             │
│                    │payments/create.yaml > auth.client_secret                │
│                    │Set PAYMENTS_CLIENT_SECRET before starting apitool.      │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="16-transport"></a>

## Diagnostics: transport timeout

Timeout/cancellation cannot guarantee server rollback. Retry is an explicit Send action; keep the cached response separately labelled.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | Text                               │
│  > POST create          │  amount     number   1200                                              │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │[Diagnostics] Response Request Log                                      │
│                         │Stage: transport | timeout | elapsed 30s                                │
│                         │Host: test.example.com                                                  │
│                         │Check reachability and environment timeout.                             │
│                         │Outcome unknown: the server may have processed it.                      │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | Text                │
│  > POST create     │  amount     number   1200                               │
│    ! broken        │                                                         │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │[Diagnostics] Response Request Log                       │
│                    │Stage: transport | timeout | elapsed 30s                 │
│                    │Host: test.example.com                                   │
│                    │Check reachability and environment timeout.              │
│                    │Outcome unknown: the server may have processed it.       │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="17-log"></a>

## Diagnostics: redacted request log

OAuth failure points to auth configuration without exposing returned secret content. Log safe stage metadata; do not include raw request bodies.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | [Auth] | Body | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Auth: [Inherit] | None | OAuth2                                         │
│  > POST create          │Token: ********  [Show token]                                           │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │Response Diagnostics [Request Log]                                      │
│                         │12:06:00 resolution: complete                                           │
│                         │12:06:00 oauth: invalid_client                                          │
│                         │Host: auth.example.com | grant: client_credentials                      │
│                         │Client secret: [redacted] | token: [redacted]                           │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | [Auth] | Body | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Auth: [Inherit] | None | OAuth2                          │
│  > POST create     │Token: ********  [Show token]                            │
│    ! broken        │                                                         │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │Response Diagnostics [Request Log]                       │
│                    │12:06:00 resolution: complete                            │
│                    │12:06:00 oauth: invalid_client                           │
│                    │Host: auth.example.com | grant: client_credentials       │
│                    │Client secret: [redacted] | token: [redacted]            │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="18-invalid-definition"></a>

## Diagnostics: invalid selected definition

Never show a sibling request in the editor while an invalid item is selected. Parsed field errors can use typed repair; malformed YAML requires external repair in this proposed MVP recovery route.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ Invalid: payments/broken.yaml                                         │
│                         │[Send disabled]  Saved                                                  │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Selected: payments/broken.yaml                                          │
│    POST create          │! Definition cannot be loaded                                           │
│  > ! broken             │Request controls unavailable                                            │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         ├────────────────────────────────────────────────────────────────────────┤
│                         │[Diagnostics] Response Request Log                                      │
│                         │Stage: load | invalid_yaml                                              │
│                         │payments/broken.yaml: line 4                                            │
│                         │Repair this file externally, then [Reload].                             │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ Invalid: payments/broken.yaml                          │
│                    │[Send disabled]  Saved                                   │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Selected: payments/broken.yaml                           │
│    POST create     │! Definition cannot be loaded                            │
│  > ! broken        │Request controls unavailable                             │
│                    │                                                         │
│▸ users             │                                                         │
│                    │                                                         │
│                    ├─────────────────────────────────────────────────────────┤
│                    │[Diagnostics] Response Request Log                       │
│                    │Stage: load | invalid_yaml                               │
│                    │payments/broken.yaml: line 4                             │
│                    │Repair this file externally, then [Reload].              │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="19-dirty-navigation"></a>

## Unsaved changes: navigation

Also applies to collection/environment switching and quit. Cancel restores the exact selection, focus and scroll. Save validates then continues only on success.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | Text                               │
│  > POST create          │  amount     number   1200                                              │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                 ┌──────────────────────────────────────────────────────────────┐                 │
│                 │ Unsaved changes                                              │─────────────────┤
│                 │ payments/create.yaml                                         │                 │
│                 │ Continue to payments/list?                                   │                 │
│                 │ [Save and continue] [Discard] [Cancel*]                      │                 │
│                 └──────────────────────────────────────────────────────────────┘                 │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | Text                │
│  > POST create     │  amount     number   1200                               │
│    ! broken        │                                                         │
│                    │                                                         │
│▸ users┌──────────────────────────────────────────────────────────────┐       │
│       │ Unsaved changes                                              │       │
│       │ payments/create.yaml                                         │───────┤
│       │ Continue to payments/list?                                   │       │
│       │ [Save and continue] [Discard] [Cancel*]                      │       │
│       └──────────────────────────────────────────────────────────────┘       │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="20-save-failure"></a>

## Unsaved changes: save failed

Retain the draft and pending destination. Never navigate after a failed save. Return to editing dismisses the dialog without changing the file.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | Text                               │
│  > POST create          │  amount     number   1200                                              │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                 ┌──────────────────────────────────────────────────────────────┐                 │
│                 │ Save failed                                                  │─────────────────┤
│                 │ payments/create.yaml                                         │                 │
│                 │ Permission denied. Your draft is retained.                   │                 │
│                 │ [Retry save] [Return to editing*]                            │                 │
│                 └──────────────────────────────────────────────────────────────┘                 │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | Text                │
│  > POST create     │  amount     number   1200                               │
│    ! broken        │                                                         │
│                    │                                                         │
│▸ users┌──────────────────────────────────────────────────────────────┐       │
│       │ Save failed                                                  │       │
│       │ payments/create.yaml                                         │───────┤
│       │ Permission denied. Your draft is retained.                   │       │
│       │ [Retry save] [Return to editing*]                            │       │
│       └──────────────────────────────────────────────────────────────┘       │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

<a id="21-confirm-send"></a>

## Dangerous send: explicit confirmation

Only when --confirm-dangerous is enabled and method is POST/PUT/PATCH/DELETE. Resolve and validate before opening. Confirm sends the exact frozen draft once; Cancel sends nothing.

### 100 columns × 30 rows

```text
┌─────────────────────────┬────────────────────────────────────────────────────────────────────────┐
│Collections / tree       │▶ POST | {{base_url}}/payments                                          │
│                         │[Send]  [Save] * Unsaved                                                │
│▾ payments               │Params | Headers | Auth | [Body] | Settings                             │
│  ▾ payments             ├────────────────────────────────────────────────────────────────────────┤
│    GET list             │Body type: JSON   View: Structured | Text                               │
│  > POST create          │  amount     number   1200                                              │
│    ! broken             │                                                                        │
│                         │                                                                        │
│▸ users                  │                                                                        │
│                         │                                                                        │
│                 ┌──────────────────────────────────────────────────────────────┐                 │
│                 │ Confirm request                                              │                 │
│                 │ Environment: test | POST                                     │─────────────────┤
│                 │ https://test.example.com/payments                            │                 │
│                 │ Unsaved draft; sending does not save it.                     │                 │
│                 │ [Cancel*] [Send request]                                     │                 │
│                 └──────────────────────────────────────────────────────────────┘                 │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
│                         │                                                                        │
├─────────────────────────┼────────────────────────────────────────────────────────────────────────┤
│Collection: payments     │Environment: test | Git: design preview                                 │
│? Help                   │Tab panes | Ctrl+S save | Ctrl+Enter send                               │
└─────────────────────────┴────────────────────────────────────────────────────────────────────────┘
```

### 80 columns × 24 rows

```text
┌────────────────────┬─────────────────────────────────────────────────────────┐
│Collections / tree  │▶ POST | {{base_url}}/payments                           │
│                    │[Send]  [Save] * Unsaved                                 │
│▾ payments          │Params | Headers | Auth | [Body] | Settings              │
│  ▾ payments        ├─────────────────────────────────────────────────────────┤
│    GET list        │Body type: JSON   View: Structured | Text                │
│  > POST create     │  amount     number   1200                               │
│    ! broken        │                                                         │
│       ┌──────────────────────────────────────────────────────────────┐       │
│▸ users│ Confirm request                                              │       │
│       │ Environment: test | POST                                     │       │
│       │ https://test.example.com/payments                            │───────┤
│       │ Unsaved draft; sending does not save it.                     │       │
│       │ [Cancel*] [Send request]                                     │       │
│       └──────────────────────────────────────────────────────────────┘       │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
│                    │                                                         │
├────────────────────┼─────────────────────────────────────────────────────────┤
│Collection: payments│Environment: test | Git: design preview                  │
│? Help              │Tab panes | Ctrl+S save | Ctrl+Enter send                │
└────────────────────┴─────────────────────────────────────────────────────────┘
```

Dialog actions marked with `*` have initial focus. All times and token examples are synthetic.
