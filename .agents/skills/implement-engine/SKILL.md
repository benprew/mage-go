---
name: implement-engine
description: Implement a reusable mage-go engine capability under pkg/mage or pkg/mage/core. Use for rules behavior that card code cannot express; do not wire or modify cards.
---

# Implement an engine capability

Build reusable engine behavior under `pkg/mage/` or `pkg/mage/core/`. Do not modify cards.

## Input

When card work found the gap, read and preserve the fields defined by [../implement-card/references/engine-gap.schema.json](../implement-card/references/engine-gap.schema.json). For a direct request, first derive the same observable requirements without creating an artifact unless it helps the work.

## Workflow

1. Read `pkg/mage/doc.go`, `docs/engine-rules-contracts.md`, and `docs/dsl-reference.md`. Read the complete `cards/mtgish/*.ron` record and all local rulings for each card used to define or test the feature. Find rulings through `data/rulings/index.json`, then read the referenced Oracle file. If an entry or file is missing, run `go run ./cmd/fetchset -card "Card Name"`. An empty `rulings` array records a successful check; missing data does not. Keep `wotc` rulings distinct from `scryfall` notes. Use only the Comprehensive Rules sections needed to establish the shared engine contract. See `data/rulings/README.md` for refresh commands and format. Report a fetch failure instead of assuming there are no rulings.
2. Find related engine behavior and follow its ownership, state, event, cloning, and test patterns.
3. Define observable requirements and interactions that distinguish complete behavior from the example card's happy path.
4. Write exhaustive engine contract tests (Tier 1 of the testing pyramid) and confirm that they fail for the expected reason. Cover combinatorial edge cases: leave-and-return object identity, phasing status, clone isolation, APNAP priority, replacement rechecks, and rollback. Use `gametest` and synthetic/test-only cards so card authors can trust the primitive without writing duplicate card-level tests.
5. Implement the smallest coherent reusable change. Add every required engine integration point and expose a concise card-facing API.
6. Update `docs/engine-rules-contracts.md` and/or `docs/dsl-reference.md` for new or changed behavior.

Do not put card-specific policy in the engine. Preserve compatibility when practical and report a necessary breaking API change.

## Verification

Run these levels in order:

1. Focused: the new tests with `-count=1`.
2. Engine: `go test ./pkg/mage/... -short -count=1`.
3. Final: `go test ./... -short -count=1`, `go build ./...`, `go vet ./...`, and `make lint`.

Run long simulations only when the change affects AI behavior or the user asks for the complete long-running suite.

Return the supported behavior, edge cases, card-facing API, verification, files changed, and remaining card wiring. If the input was an engine-gap record, return its ID with status `resolved` and resolution evidence.
