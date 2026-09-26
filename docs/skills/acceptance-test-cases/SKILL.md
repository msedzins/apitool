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
identifies **which task** owns the work. Acceptance cases and visual baselines
refine verification detail; they do not supersede requirements or introduce
unrelated product behavior.

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

## Required two-step flow

### Step 1: Design or update the scenarios

1. Extract discrete requirements from the specification. Assign stable IDs if
   they do not already exist: `R-001`, `R-002`, and so on.
2. Read the design document and identify observable flows, state transitions,
   boundaries, safety guarantees, failure behavior, and UI/TUI states that are
   supported by the requirements.
3. Read the implementation plan and map each relevant requirement and design
   behavior to its task IDs: `T-001`, `T-002`, and so on. A task is evidence of
   planned ownership, not a source of expected behavior.
4. Generate only the cases needed to make the requirements verifiable. Prefer
   one meaningful case per behavior over many near-duplicate cases.
5. Report requirements with no case and tasks with no mapped case. Do not fill
   gaps by reading code.

### Step 2: Independent scenario-status review

After implementation, run a fresh review subagent before changing any case to
`implemented`. Give it the acceptance suite, authoritative documents, plan,
implementation diff or branch, relevant test commands, and any required
screen-baseline locations.

The reviewer may inspect code, tests, test output, and snapshots only to
verify delivery and status; it must not use them to redefine expected behavior.
For every mapped test case, the reviewer must record:

| Test case | Verified status | Evidence | Missing or blocked work |
|---|---|---|---|
| UI-001 | planned or implemented | exact test, snapshot, or manual verification result | task, fixture, behavior, or none |

The reviewer verifies that the delivered behavior proves the scenario's
`Then`, the mapped task actually covers it, required fixtures or baselines
exist and are linked, and verification results are current. The scenario
author updates statuses only from this report. If review evidence is absent,
incomplete, or failing, keep the case `planned` and report the gap.

## Case selection

Include cases for:

- primary user or operator outcomes;
- stated security, privacy, validation, isolation, or confirmation guarantees;
- important error handling and graceful-degradation behavior;
- interaction boundaries between architectural components;
- UI/TUI states explicitly required by the specification, plus approved
  requirement-derived fallback screens described below.

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

New and changed cases begin as `planned`. Use `implemented` only when the
independent Step 2 review records current evidence that the delivered behavior
proves the case. A task name, a claimed task status, or a passing unrelated
test is not enough.

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
`Then`. First reuse an existing approved screen baseline when it proves the
same requirement-derived scenario. Mark a stable rendered state as a snapshot
candidate when a new approval artifact is appropriate:

```yaml
snapshot_candidate: true
```

### Approved visual baselines and fallback screens

When a requirement needs a TUI screen but the specification has no suitable
approved baseline, an acceptance case may propose a fallback screen. Its
rendered state must follow directly from a named requirement: it may make that
required outcome observable, but must not add user actions, data, permissions,
or semantics. Record the requirement ID and the derivation in the case, mark
the screen `proposed`, and obtain user approval before marking it `approved`.
If the screen cannot be derived directly, report an ambiguity and request a
decision instead of inventing the behavior.

In a feature PR, add proposed fallback screens early. Store each approved
screen once in the canonical snapshot location and link it from the case.
Finish the UI implementation later in the same PR, even if CI is temporarily
red; require relevant CI checks to pass before completion or merge. A baseline
records the approved target, not proof that the app renders or matches it.
Keep cases `planned` until implementation is complete, and never claim a
passing visual comparison without verification.

## Traceability report

Return a compact table after creating or updating cases:

| Test case | Category | Observable surface | Requirement | Task | Status |
|---|---|---|---|---|---|
| UI-001 | UI | TUI | R-001 | T-003 | planned |

Also list, separately:

- requirements without acceptance coverage;
- tasks without mapped acceptance coverage;
- ambiguities that block a trustworthy case.

For Step 2, include the independent review table and retain its exact evidence
with the PR or review report.

## Keep acceptance cases and implementation plans distinct

The acceptance-test document owns stable case IDs, BDD scenarios, and the
requirement-to-task-to-case mapping. The implementation plan owns the work to
build and verify the behavior. Describe that work by behavior and implementation
verification, without copying acceptance-case IDs or restating their scenarios.
Implementation plans must not contain acceptance-case IDs (`UI-xxx`,
`API-xxx`, `CFG-xxx`, `DATA-xxx`, `WF-xxx`, or `OPS-xxx`), including when
identifying fixture ownership or describing verification. They may reference
requirements, task IDs, behavior, concrete test commands, and fixture paths.
Keep case-ID traceability in the acceptance-test index and report; refer to the
design behavior in the implementation plan when context is needed.

For example, write “test opening help from each TUI mode and restoring the
previous view” in the implementation plan, while the acceptance document maps
those behaviors to their `UI-xxx` cases.

The independent Step 2 review report may use acceptance-case IDs because it
verifies acceptance status; it is not an implementation plan.

## Reusable plan-boundary guard

For Go repositories, copy
[`assets/plan_boundary_test.go`](assets/plan_boundary_test.go) to the
repository root as `plan_boundary_test.go`, and set its package declaration to
the repository's root test package. The template is the canonical source for a
table-driven guard: add a rule for each document boundary, with a document
glob, forbidden expression, and policy name. Its default rule keeps
implementation plans free of acceptance-case IDs. The root copy is required so
`go test ./...` enforces the policy.

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
- has an independent Step 2 review result before it is marked `implemented`.
- has no acceptance-case identifiers in the implementation plan. Verify with
  `go test . -run TestDocumentBoundaries`.
