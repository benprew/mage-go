---
name: implement-engine
description: Implement a mage-go engine capability needed for exact card behavior. Use for changes under pkg/mage or pkg/mage/core, with engine tests, card-facing documentation, and regression verification.
---

# Implement an engine capability

Build the requested capability from required behavior, rather than assuming it fits a fixed category of engine change.

## Boundary

Edit only `pkg/mage/` and `pkg/mage/core/`. Do not implement or modify cards. Return enough of the card-facing API for card work to continue.

Preserve unrelated work and do not commit unless the user asks.

## Workflow

1. Read `pkg/mage/doc.go`, the request's Oracle text, and the relevant sections found through `docs/comprehensive-rules-index.md`.
2. Search the engine for related behavior and follow its established ownership, state, event, cloning, and testing patterns. Consult XMage only when it helps clarify edge cases or architecture.
3. Turn the request into observable requirements, including interactions and rules edge cases that distinguish a correct implementation from a partial one.
4. Write focused engine tests first and confirm they fail for the expected reason. Use the lowest test level that proves the behavior; use `gametest` and test-only cards when game integration matters.
5. Implement the smallest coherent engine change that satisfies those requirements. Integrate it wherever engine semantics require, and expose a concise card-facing API when callers need one. Do not force the design into a preselected list such as keyword, effect, trigger, or replacement effect.
6. Update `pkg/mage/doc.go` for new or changed card-facing APIs and behavior.
7. Run focused tests, all engine tests, the full repository suite, and the repository lint command. Fix regressions caused by the change.

## Constraints

- Follow the comprehensive rules exactly; do not implement only the example card's happy path.
- Keep card-specific policy out of the engine unless it represents reusable game behavior.
- Avoid unrelated refactors, but make all changes needed for a complete and internally consistent feature.
- Preserve compatibility where practical. If a breaking API change is necessary, state it clearly.

## Handoff

Report:

- the supported behavior and important edge cases;
- the card-facing API;
- tests and verification run;
- files changed;
- remaining limitations or card wiring still needed.
