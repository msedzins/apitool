---
title: Feature acceptance tests
requirements: [R-EDIT-001]
design_document: ../../../superpowers/specs/2026-09-20-apitool-design.md
implementation_plan: ../../../superpowers/plans/2026-09-20-apitool-mvp.md
---

# Feature acceptance tests

## Test case index

| Test case | Category | Observable surface | Requirements | Tasks | Status |
|---|---|---|---|---|---|
| [UI-009 — Dirty navigation requires a choice](#ui-009) | UI | TUI | R-EDIT-001 | T-009 | automated |
| [UI-010 — JSON and raw body modes preserve their data](#ui-010) | UI | TUI | R-EDIT-001 | T-009 | automated |
| [UI-011 — Invalid JSON is not saved](#ui-011) | Safety | TUI and YAML | R-EDIT-001 | T-009 | automated |
| [UI-012 — Undo is scoped to the open request](#ui-012) | UI | TUI | R-EDIT-001 | T-009 | automated |
| [UI-013 — Group deletion shows exact paths before confirmation](#ui-013) | Safety | TUI and filesystem | R-EDIT-001 | T-009 | automated |
| [UI-014 — Discard does not change YAML](#ui-014) | Safety | TUI and YAML | R-EDIT-001 | T-009 | automated |
| [UI-015 — Duplicate opens an unsaved copy flow](#ui-015) | UI | TUI and YAML | R-EDIT-001 | T-009 | automated |

<a id="ui-009"></a>
## UI-009 — Dirty navigation requires a choice

**Category:** UI<br>
**Observable surface:** TUI<br>
**Requirements:** R-EDIT-001<br>
**Tasks:** T-009<br>
**Status:** automated

### Given

A request has unsaved edits and another request is available in the collection.

### When

The user navigates to the other request.

### Then

The editor offers Save and continue, Discard changes, and Cancel before changing the active request.

<a id="ui-010"></a>
## UI-010 — JSON and raw body modes preserve their data

**Category:** UI<br>
**Observable surface:** TUI<br>
**Requirements:** R-EDIT-001<br>
**Tasks:** T-009<br>
**Status:** automated

### Given

A request has a JSON body containing `{"a":1}`.

### When

The user views the body in JSON mode, switches to raw mode, and enters `<a>1</a>`.

### Then

JSON mode displays indented JSON and raw mode preserves `<a>1</a>` exactly.

<a id="ui-011"></a>
## UI-011 — Invalid JSON is not saved

**Category:** Safety<br>
**Observable surface:** TUI and YAML<br>
**Requirements:** R-EDIT-001<br>
**Tasks:** T-009<br>
**Status:** automated

### Given

A request has a JSON body.

### When

The user enters malformed JSON and saves.

### Then

The TUI reports validation feedback and the request file remains unchanged.

<a id="ui-012"></a>
## UI-012 — Undo is scoped to the open request

**Category:** UI<br>
**Observable surface:** TUI<br>
**Requirements:** R-EDIT-001<br>
**Tasks:** T-009<br>
**Status:** automated

### Given

The user edits the URL of one request, then opens a different request.

### When

The user invokes undo.

### Then

The second request remains unchanged and no edit from the first request is restored.

<a id="ui-013"></a>
## UI-013 — Group deletion shows exact paths before confirmation

**Category:** Safety<br>
**Observable surface:** TUI and filesystem<br>
**Requirements:** R-EDIT-001<br>
**Tasks:** T-009<br>
**Status:** automated

### Given

A request group contains two YAML request files.

### When

The user invokes delete for that group.

### Then

The confirmation lists both exact paths, and no file is removed before affirmative confirmation.

<a id="ui-014"></a>
## UI-014 — Discard does not change YAML

**Category:** Safety<br>
**Observable surface:** TUI and YAML<br>
**Requirements:** R-EDIT-001<br>
**Tasks:** T-009<br>
**Status:** automated

### Given

A request file is loaded and edited in memory.

### When

The user discards the edit.

### Then

The YAML file remains byte-for-byte unchanged.

<a id="ui-015"></a>
## UI-015 — Duplicate opens an unsaved copy flow

**Category:** UI<br>
**Observable surface:** TUI and YAML<br>
**Requirements:** R-EDIT-001<br>
**Tasks:** T-009<br>
**Status:** automated

### Given

A request is open in the editor.

### When

The user chooses Duplicate request, edits its destination path, and saves.

### Then

The editor shows a new unsaved name and path flow before saving, then creates the copy at the chosen path without changing the source request.
