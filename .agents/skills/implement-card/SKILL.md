---
name: implement-card
description: "Implement one Magic: The Gathering card or a related card batch under cards/. Use for Oracle-accurate card wiring and card tests; stop at reusable engine gaps."
---

# Implement cards

Implement the requested cards under `cards/`. Do not add engine behavior under `pkg/mage/` or `pkg/mage/core/`.

## Workflow

1. Read each complete Oracle comment, `docs/dsl-reference.md`, and the closest implemented cards and tests.
2. Use `docs/comprehensive-rules-index.md` to locate rules needed for uncertain behavior. Oracle text and the comprehensive rules are authoritative.
3. Determine testing requirement using the **High-Leverage Testing Model**:
   - **Skip game-simulation tests** for purely declarative cards: vanilla creatures, French vanilla creatures with standard keywords (e.g. Flying, Trample, Vigilance), and simple single-verb spells/abilities that directly map to proven engine primitives. These rely on exhaustive engine contract tests and automated registration validation (`TestRegisteredDefinitions`).
   - **Write focused tests first (TDD)** for non-trivial cards: complex multi-clause rules, unusual conditions, intervening-if clauses, custom triggers, or mechanics that exercise engine boundaries. Confirm that the test fails before implementing.
4. Make the smallest complete card-level change that satisfies Oracle requirements. Use existing composable DSL APIs (`pkg/mage/dsl`).
5. Run verification in order:
   - focused: newly added card tests with `-count=1` (if applicable);
   - package: `go test ./cards/<package>/ -count=1`;
   - card regression & registration validation: `go test ./cards/... -short -count=1` (validates all card definitions through `TestRegisteredDefinitions`);
   - final: `go build ./...`, `go vet ./cards/<package>/`, and `make lint`.

Keep the full Oracle text immediately above `Register()` and remove the genset TODO only when implementation is complete. Implement every condition, choice, target restriction, zone, duration, and rules-relevant word.

## Terminal states

Each requested card must end as one of:

- `implemented`;
- `blocked_engine_gap`;
- `accepted_unsupported`, only after the user confirms that the behavior is outside engine scope.

Never return a silent approximation or an ambiguous partial implementation.

For an engine gap, create a record that follows [references/engine-gap.schema.json](references/engine-gap.schema.json). Include the Oracle requirement, rules references, required scenarios, and likely integration points. Validate a saved record with:

```bash
python3 .agents/skills/scripts/validate_artifact.py .agents/skills/implement-card/references/engine-gap.schema.json <artifact.json>
```

Use `// XXX:` only for a user-approved deferred gap. Use `// UNIMPLEMENTABLE:` and `UNSUPPORTED.md` only for a confirmed scope exclusion.

Report card states, tests, files changed, and engine-gap IDs.
