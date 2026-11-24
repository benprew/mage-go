---
name: validate-set
description: Audit a mage-go set for playable coverage, mtgish imports, Oracle fidelity, and test quality. Use for read-only assessment of source, compiler, and engine behavior.
---

# Validate a set

Resolve the set code and any Go package with `data/sets.txt` and package metadata. Use Scryfall JSON for the expected roster and Oracle text. Fetch missing data to a temporary path with `go run ./cmd/fetchset -o <output-path> <CODE>`. Count distinct card definitions across printings and include every expected card in the audit.

## Audit coverage and behavior

- Use `../mtgish/data/mtgish.lines.json` or checked-in fixtures to inspect definitions. Match them to the expected cards. For an individual example, run `../mtgish/show_card "Sol Ring"` from the mage-go root.
- Check `mtgish.ParseMtgishJSON` and `mtgish.Compile` with the available records. Record the failing card and rule path when present. Identify the cause as a missing or incorrect record, missing compiler support, or missing engine behavior. A successful compile establishes compiler coverage.
- Check the game's normal loading path for each expected card. Include reprints provided by another package and cards with no registration. Record whether the active definition comes from mtgish or Go. Count cards as playable when the active definition implements the required behavior. Count mtgish cards as playable imports when the game loads the compiled definition.
- Read the local rulings for each audited card. Look up the exact name in `data/rulings/index.json` and read the referenced file. See [Card rulings](../../../README.md#card-rulings).
- Compare actual gameplay against Oracle text and the relevant rules. Inspect the tests for failures they can detect. For hand-written Go cards, check the full Oracle comment above `Register()` and reconcile any `UNIMPLEMENTABLE:` markers with the set's `UNSUPPORTED.md`.

## Checks and report

Run `go test ./pkg/mage/mtgish -count=1` for import checks. If the set has a Go package, run its package tests and `go vet` on that package. Use broader checks when needed to confirm a shared regression.

Report the expected count, playable count, mtgish compiler coverage, and how many playable cards load mtgish definitions. Give the card, expected behavior, actual behavior, and evidence for each finding. Disclose any portion of the roster or behavior you could not verify.

Use the [inventory schema](references/set-inventory.schema.json) and [report schema](references/validation-report.schema.json) when a saved handoff is needed. Record the active path in `registration_source` and the import result in `mtgish_status`. Use `missing` for an absent implementation and `null` for fields that lack a Go package, a definition path, or a Go Oracle comment check. Validate saved artifacts with [the artifact validator](../scripts/validate_artifact.py).
