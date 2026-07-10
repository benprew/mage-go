---
name: implement-card
description: "Implement one Magic: The Gathering card or a related batch from genset stubs. Use for card-level work in cards/, including tests, Oracle-accurate behavior, and identifying engine gaps."
---

# Implement cards

Implement the requested cards with tests first and Oracle text as the specification.

## Boundary

Edit only `cards/`. Do not add engine behavior under `pkg/mage/` or `pkg/mage/core/`. If exact card behavior requires an engine change, report the gap for `implement-engine`.

Preserve unrelated work and do not commit unless the user asks.

## Workflow

1. Locate each card and read its complete Oracle comment. Read `pkg/mage/doc.go` and search existing cards for reusable APIs and patterns.
2. Use `docs/comprehensive-rules-index.md` to locate and read only rules relevant to behavior that is not already clear. Consult XMage when it provides useful implementation evidence, not as a substitute for Oracle text or the rules.
3. Add a test that fails for the missing behavior before changing the implementation. Cover each distinct behavior and important restriction; include negative or edge cases when they can catch an incorrect implementation. Purely declarative cards using already-tested constructors or keywords may not need new tests.
4. Run the focused test and confirm it fails for the expected reason.
5. Implement the smallest complete card-level change that makes the test pass. Prefer existing engine APIs over custom effects.
6. Verify the focused tests, the card package, and then the broader suite in proportion to the change. Run the repository lint command after changes.

## Correctness

- Keep the full Oracle text immediately above `Register()` and remove the genset TODO when implementation is complete.
- Implement every condition, choice, target restriction, zone, duration, and rules-relevant word. Never silently approximate unsupported behavior.
- Treat Scryfall/Oracle text and the comprehensive rules as authoritative; use other implementations only as references.
- Respect the two-player scope where the rules naturally collapse multiplayer wording to one opponent.

## Engine gaps

When the current engine cannot express the exact behavior, do not work around it with a weaker implementation. Return:

- the missing capability;
- the affected cards;
- the exact Oracle text and relevant rules;
- the tests or scenarios the engine feature must support;
- any useful observations about the likely integration points.

Use `// XXX:` only when the user explicitly accepts a deferred, fixable gap. Use `// UNIMPLEMENTABLE:` and `UNSUPPORTED.md` only after confirming that the behavior is intentionally outside the engine's scope, not merely absent today.

## Handoff

Report the cards implemented, tests added and run, any remaining gaps, and the files changed.
