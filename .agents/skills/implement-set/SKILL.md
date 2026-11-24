---
name: implement-set
description: Bring a Magic set to verified playable coverage in mage-go. Use the expected set roster and mtgish definitions to prioritize shared compiler and engine work.
---

# Implement a set

Resolve the set name, code, or package using `data/sets.txt` and package metadata. Build the expected roster and initial coverage inventory with [validate-set](../validate-set/SKILL.md). Use the set's Scryfall JSON for the roster and Oracle text. If it is missing, fetch it with `go run ./cmd/fetchset -o data/<CODE>.json <CODE>`.

Use `../mtgish/data/mtgish.lines.json` for card definitions. Join matching records to the roster, including cards playable through an existing registration from another set. Use the checked-in compiler API to establish what compiles today. Check whether the game loads each compiled card through its normal entry point. Use `cmd/genset` when the task calls for a new legacy Go package. Check the output paths for conflicts with existing work first.

Read the local rulings for the cards in each batch to identify shared behavior and interactions specific to individual cards. Find the corresponding files by exact card name in `data/rulings/index.json`. See [Card rulings](../../../README.md#card-rulings).

## Execute

1. Group cards by the source, compiler, or engine work they share. Order batches by their dependencies.
2. Use [implement-engine](../implement-engine/SKILL.md) to complete each batch, including compiler or engine changes needed for the selected cards. Cover the shared rule with representative gameplay tests and add cases for distinct behavior.
3. Update the inventory after each batch and rerun affected tests without cache. Recheck other cards that use the constructs you changed.

For a multi-batch handoff, use the [inventory schema](../validate-set/references/set-inventory.schema.json). Delegate independent batches only when the user requests or permits delegation.

## Complete

Run [validate-set](../validate-set/SKILL.md) against the complete expected roster. Reconcile legacy TODOs and deferred or unsupported markers with the findings. Run the set and importer checks that apply. Run repository-wide tests and build checks for shared engine changes.

Report exact playable and mtgish coverage, unresolved cards, shared capabilities added, and verification.
