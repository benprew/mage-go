---
name: implement-card
description: "Implement one Magic: The Gathering card or a related card batch under cards/. Use for Oracle-accurate card wiring and card tests; stop at reusable engine gaps."
---

# Implement cards

Implement the requested cards under `cards/`. Do not add engine behavior under `pkg/mage/` or `pkg/mage/core/`.

## Workflow

1. Read each complete Oracle comment, `pkg/mage/doc.go`, and the closest implemented cards and tests.
2. Use `docs/comprehensive-rules-index.md` to locate rules needed for uncertain behavior. Oracle text and the comprehensive rules are authoritative.
3. Add a test for each distinct behavior or restriction and confirm that it fails for the expected reason. A purely declarative use of an already-tested primitive may not need another test.
4. Make the smallest complete card-level change that passes the tests. Use existing engine APIs.
5. Run verification in order:
   - focused: the new tests with `-count=1`;
   - package: `go test ./cards/<package>/ -count=1`;
   - card regression: `go test ./cards/... -short -count=1`;
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
