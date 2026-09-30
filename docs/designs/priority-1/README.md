# Priority 1 TUI screen designs

Status: draft for review. Date: 2026-09-30.

## Intent and scope

Design the seven priority-1 areas identified in the API tool audit: request editor, body editor, authentication, send/response lifecycle, diagnostics/repair, unsaved changes, and dangerous-send confirmation. The user requested a separate worktree and confirmed screen designs first. These are proposed layouts and interaction rules, not an implemented TUI or approved snapshot baselines.

The current three-pane collection layout, Git-readable definitions, explicit Save, single-request execution and process-environment secret references remain the product constraints. Success is a developer being able to edit a request, understand validation, execute exactly the intended draft, inspect its result, and navigate safely using the keyboard.

Open [gallery.md](gallery.md) to compare 21 states at 100×30 and 80×24. `screens/` contains 42 plain-text mockups; `states.json` records their explanations. No approved `testdata/` files or application code are changed.

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

## Review and acceptance mapping

| Area | Mockup states | Existing acceptance coverage |
|---|---|---|
| Request editor | 01, 02, 06 | CFG-001/002, UI-006 |
| Body editor | 03–05 | UI-007 |
| Authentication | 07–09 | CFG-003, API-003/004, UI-012 |
| Sending and responses | 10–14 | UI-004/010, DATA-001 |
| Diagnostics and repair | 15–18 | UI-002/011, API-002/007, DATA-002 |
| Unsaved changes | 19–20 | UI-005 |
| Dangerous-send confirmation | 21 | UI-009 |

New acceptance cases should cover sending a draft without saving, suppressing duplicate send, cancellation with unknown server outcome, successful response with cache-write failure, selected-invalid identity, malformed-YAML external repair/reload, failed save before navigation, dirty environment/quit guards, and remasking a token. Cases remain proposed until design review; existing planned cases are not marked implemented.

## Decisions for review

1. Execute the current draft without autosaving; preserve explicit Save.
2. Use typed repair for parsed errors and external repair plus Reload for malformed YAML.
3. Apply the dirty guard to environment switching and Ctrl+C quit as well as request/collection navigation.
4. Retain the three-pane shell; expand multiline editing when needed rather than introduce separate wizard screens.

No implementation starts from this draft. After design approval, reconcile the original specification and acceptance cases, then write the implementation plan before changing the application.
