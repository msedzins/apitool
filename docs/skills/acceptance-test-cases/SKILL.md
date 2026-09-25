---
name: acceptance-test-cases
description: Generate BDD-style, human-readable acceptance test cases from requirements, architecture, and implementation plans, with requirement-to-task traceability. Do not derive expected behavior from code.
metadata:
  short-description: Design traceable BDD acceptance cases
---

# Acceptance Test Cases

Use this skill to define or review behavior-level acceptance test cases for a
product or feature.

## Source-of-truth rule

Use these authoritative documents in this order:

1. Product requirements or specification: required user-visible behavior.
2. Design document: intended user flows, system boundaries, decisions,
   constraints, error handling, and non-functional guarantees.
3. Implementation plan: task identifiers used only for traceability.

Do not inspect implementation code or existing automated tests to decide what
the product should do. They can be incomplete, defective, or accidentally
correct. If the authoritative documents conflict or leave behavior ambiguous,
state the ambiguity and request a decision instead of inferring intent.

## Required design document

Before generating cases, locate and read the design document for the feature.
In a Superpowers project this is normally the relevant file under
`docs/superpowers/specs/`.

The requirements define **what** must be true. The design document defines the
intended flows and boundaries that make the requirement testable. The plan
identifies **which task** owns the work.

If a design document is missing, ask for it or ask the user to explicitly
authorize deriving cases from requirements alone. Do not substitute code,
existing tests, or an implementation plan for the design document.

## Outcome

Create a small, risk-focused acceptance suite that maps:

```text
requirement -> implementation task -> acceptance test case
```

The test cases are a behavioral contract. They are not unit-test outlines,
implementation instructions, or a test runner specification.

## Workflow

1. Extract discrete requirements from the specification. Assign stable IDs if
   they do not already exist: `R-001`, `R-002`, and so on.
2. Read the design document and identify observable flows, state transitions,
   boundaries, safety guarantees, failure behavior, and UI/TUI states.
3. Read the implementation plan and map each relevant requirement and design
   behavior to its task
   IDs: `T-001`, `T-002`, and so on. A task is evidence of planned ownership,
   not a source of expected behavior.
4. Generate only the cases needed to make the requirements verifiable. Prefer
   one meaningful case per behavior over many near-duplicate cases.
5. Report requirements with no case and tasks with no mapped case. Do not fill
   gaps by reading code.

## Case selection

Include cases for:

- primary user or operator outcomes;
- stated security, privacy, validation, isolation, or confirmation guarantees;
- important error handling and graceful-degradation behavior;
- interaction boundaries between architectural components;
- UI/TUI states explicitly required by the specification.

Do not create cases for private helper behavior, function names, package
boundaries, incidental formatting, or every combination of input values.

## Test-case format

Use one Markdown file per feature or design document. Classify each scenario
with one behavior category and one or more observable surfaces. Use category
IDs: `UI-xxx`, `API-xxx`, `CFG-xxx`, `DATA-xxx`, `WF-xxx`, or `OPS-xxx`.
Start each suite by copying
[`assets/acceptance-test-suite.md`](assets/acceptance-test-suite.md), then
fill only fields supported by the authoritative documents.

At the top, include a test-case index with one row per case: clickable name,
category, observable surface, requirement IDs, task IDs, and status. Link the
name to the case's explicit HTML anchor in the same file. Keep each case
readable without source code.

Use `planned` until the mapped task is delivered; use `implemented` only when
the implementation is known to be complete from the plan/status provided by
the user. Do not infer status by inspecting code.

## Observable-contract ontology

Choose the category that describes the behavior, not its implementation:

| Category | Meaning |
|---|---|
| `UI` | A user-visible interaction, navigation flow, dialog, or rendered state. |
| `API` | Behavior exposed through a supported callable or protocol contract. |
| `CFG` | User-owned configuration creation, validation, persistence, or selection. |
| `DATA` | Runtime or persistent-data lifecycle, isolation, redaction, retention, permissions, and database-backed behavior. |
| `WF` | An end-to-end user or operator journey spanning multiple categories. |
| `OPS` | CI, deployment, monitoring, or other operational behavior. |

Name the observable surface used to prove the behavior:

| Surface | Meaning |
|---|---|
| `TUI` | A terminal user can see or do it in the rendered interactive application. |
| `CLI` | A command user can observe it through flags, output, exit status, or startup behavior. |
| `API/Protocol` | A supported service boundary can observe requests, responses, messages, callbacks, or protocol exchanges. Name the concrete protocol in the scenario when useful. |
| `Filesystem/Git` | A user can inspect documented files, repository status, diffs, or committed state. |
| `Database` | A supported database consumer can observe persisted records, query results, or transaction outcomes. |
| `CI` | An operator can observe workflow configuration, job status, logs, and quality-gate results. |
| `Supported internal API` | A documented stable cross-component interface can be observed by its supported consumer; ordinary package APIs and helpers do not qualify. |

HTTP, OAuth, gRPC, GraphQL, webhooks, and queues are `API/Protocol`
qualifiers, not universal surfaces. Use the `Database` surface only when the
requirements define an observable database contract; do not add it merely
because an implementation uses a database.

## BDD scenario design

Each scenario proves one primary behavior. `Given` describes externally
meaningful state, `When` contains one user, operator, or API-consumer action,
and `Then` proves one outcome through the named observable surface. Split
unrelated outcomes into separate scenarios. Private implementation behavior
belongs in unit or integration tests unless it is a supported internal API.

## UI and TUI cases

For a required visual or rendered state, make the expected state explicit in
`Then`. Mark it as a snapshot candidate when stable rendered output would be
an appropriate approval artifact:

```yaml
snapshot_candidate: true
```

This skill defines the case and expected snapshot intent. It does not generate,
approve, compare, or update snapshots.

## Traceability report

Return a compact table after creating or updating cases:

| Test case | Category | Observable surface | Requirement | Task | Status |
|---|---|---|---|---|---|
| UI-001 | UI | TUI | R-001 | T-003 | planned |

Also list, separately:

- requirements without acceptance coverage;
- tasks without mapped acceptance coverage;
- ambiguities that block a trustworthy case.

## Quality check

Before finishing, confirm each case:

- asserts behavior derivable from authoritative documents only;
- has a clear Given / When / Then flow;
- has one primary behavior, category, and at least one observable surface;
- has at least one requirement and one task mapping, unless the plan has no
  relevant task yet;
- proves its `Then` through the named user, operator, protocol, persistence,
  database, CI, or supported-internal-API surface;
- does not prescribe implementation details;
- does not invent behavior to make a case complete.
