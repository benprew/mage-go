---
name: implement-engine
description: Implement mage-go rules and mtgish compiler support. Use for a named Magic card, a related card batch, or a reusable engine rule or API change.
---

# Implement rules and card support

## Establish the required behavior

For a named card, inspect the mtgish definition from the mage-go root.

```bash
../mtgish/show_card "Sol Ring"
```

If the helper is unavailable, look up the name in `../mtgish/data/mtgish.lines.json`. Use a matching [checked-in JSON fixture](../../../pkg/mage/testdata/mtgish/README.md) if the sibling checkout is unavailable. Check Scryfall Oracle text and the relevant [comprehensive rules](../../../docs/comprehensive-rules-index.md). For custom cards, use the requested rules text. Read the [engine API guide](../../../pkg/mage/doc.go) before changing runtime behavior.

Read local rulings for the cards you implement and any published cards used to define or test a shared engine or compiler change. Find the corresponding files by exact card name in `data/rulings/index.json`. See [Card rulings](../../../README.md#card-rulings).

## Change the layer responsible

- **Definition.** Fix a missing or incorrect mtgish record or grammar when the sibling repository is in scope. Identify the exact source issue if it remains open.
- **Compiler.** Translate valid tagged JSON into immutable `mage.CardDefinition` values in `pkg/mage/mtgish`. Reuse engine primitives. Reject records with any required rule that cannot compile and include the card name and rule path in the diagnostic.
- **Engine.** Put reusable rules in `pkg/mage` and shared value types in `pkg/mage/core`. Follow existing state ownership, event flow, and cloning patterns. Keep per-game and per-object state on runtime objects. Update `pkg/mage/doc.go` when card-facing APIs or behavior change.

If the definition already compiles and the engine supports it, use the data through the normal loading path. For fixes to active legacy Go definitions or custom Go cards, work in `cards/`. Keep the full Oracle text above `Register()` for hand-written Magic cards. Remove a genset TODO after the complete behavior passes.

## Prove the behavior

Write a regression test for missing or incorrect behavior and confirm it fails for the expected reason. For mtgish work, compile a fixture from the actual tagged JSON and exercise the compiled card with `gametest`. Record the upstream revision for new fixtures. Instantiate the compiled definition directly or use an isolated registry so a legacy registration cannot satisfy the test. For a migration, also verify the normal loader uses the imported definition.

Run the focused tests first. For compiler changes, run `go test ./pkg/mage/mtgish -count=1`. For runtime changes, run `go test ./pkg/mage/... -short -count=1`. For legacy Go changes, run the affected card package tests. Run repository-wide tests and build checks when shared interfaces or gameplay paths change.

Report the active definition for card work, API changes for engine work, verification, and any remaining blockers. For a structured handoff, use the [engine gap schema](references/engine-gap.schema.json) and [artifact validator](../scripts/validate_artifact.py). Cite the gap ID and verification when resolving a tracked gap.
