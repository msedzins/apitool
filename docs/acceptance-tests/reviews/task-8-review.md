# Task 8 acceptance review

Independent read-only review of Task 8 implementation and acceptance evidence.
Reviewed 2026-09-28 against the implementation plan, design, current tests, and
approved snapshots.

Verification performed:

- `GOCACHE=/private/tmp/apitool-task8-gocache go test ./internal/tui -count=1` — PASS
- `GOCACHE=/private/tmp/apitool-task8-gocache go test ./...` — PASS
- `git diff --check` — PASS

| Case | Status | Evidence | Remaining gap |
|---|---|---|---|
| UI-001 | Implemented | `TestUI001CollectionPickerMatchesApprovedScreen`, `TestUI001SelectedCollectionMatchesApprovedScreen`; approved picker and tree snapshots | None |
| UI-002 | Implemented | `TestInvalidRequestSelectionShowsPreciseDiagnostic`, `TestInvalidRequestHasWarningWhileValidSiblingOpens`, `TestMalformedCollectionMetadataRemainsVisibleAndDiagnosed` | Nested invalid rows use leaf names plus depth indentation; same-depth duplicate leaf names could be hard to distinguish |
| UI-003 | Implemented | `TestLeftAndRightToggleExpandedTreeGroup`, `TestCollapsedAncestorHidesNestedGroupAndInvalidDescendant`, approved tree snapshot | None |
| UI-004 | Planned | `StatusView` categorizes statuses and `TestStatusViewCategoriesRemainTextualWithoutColor` checks the text | No TUI response path renders an HTTP status; add T-010 ownership |
| UI-015 | Implemented | `TestUI015HelpRestoresEveryTUIState`, `TestUI015KeyboardHelpMatchesApprovedScreenAndCompacts`; picker, environment picker, search, and collection view covered | None |
| UI-016 | Implemented | Same state-restoration test compares full views before and after closing help with either key | None |
| UI-017 | Implemented | `TestUI015FocusMarkersMatchApprovedScreens` checks snapshots across the complete Tab cycle and asserts one marker at each state | None |
| UI-018 | Implemented | `TestUI015KeyboardHelpMatchesApprovedScreenAndCompacts` checks shortcut groups and 80×24 bounds | None |
| UI-019 | Implemented | `TestUI015CtrlCQuitsHelp` verifies `tea.QuitMsg` | None |
| CFG-004 | Implemented | `TestCollectionPickerOpensCollectionAndRestoresItsEnvironment` uses persisted environment state | None |
| CFG-005 | Planned | Startup environment option is covered by TUI tests | No CLI startup/definition immutability evidence |
| CFG-006 | Implemented | `TestInvalidPendingStartingEnvironmentBlocksRequestViewAndFallback` checks the error and absence of fallback/request view | Exercises TUI startup option; no dedicated CLI test |

The nested invalid-row display is a known usability limitation: depth is shown
through indentation, but duplicate leaf names at the same depth do not show a
full path. This does not block the reviewed acceptance cases.
