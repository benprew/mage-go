---
name: validate-set
description: Audit a mage-go card set for completeness, Oracle fidelity, test quality, unsupported behavior, and regressions. Use to assess an implemented or partially implemented set and produce actionable findings without changing card or engine code.
---

# Validate a set

Produce an evidence-based audit. Do not modify card or engine implementations.

## Establish authoritative inputs

Resolve the package and Scryfall set code through the arguments, `data/sets.txt`, and package metadata. Fetch missing canonical JSON when necessary:

```bash
FETCHSET_SKIP_TLS=1 go run ./cmd/fetchset -o data/<CODE>.json <CODE>
```

Use Scryfall JSON as the Oracle source of truth. Read `docs/comprehensive-rules-index.md` and consult only the rules sections needed to resolve specific questions.

## Audit

Inventory every registered card and its tests. For each card, determine:

- whether it is complete, partial, a stub, or explicitly unsupported;
- whether the source preserves the full Oracle text above `Register()`;
- whether the implementation matches every rules-relevant part of that text;
- whether tests adequately prove its distinct behavior and restrictions;
- whether any gap belongs in card code or requires reusable engine support.

Inspect actual behavior in code rather than inferring correctness from constructor names, comments, markers, or passing tests alone. Look for omitted choices, conditions, targets, zones, timing, durations, values, and interactions. Treat any approximation as a defect unless it follows from the engine's documented two-player scope.

Tests are appropriate when they can catch card-specific behavior or wiring. Do not demand redundant tests for purely declarative use of already-tested engine primitives. When tests exist, check that their assertions would fail for plausible incorrect implementations; include negative and boundary coverage where rules distinctions warrant it.

Verify every `UNIMPLEMENTABLE:` marker against `UNSUPPORTED.md`. Distinguish intentional architectural exclusions from capabilities that are merely not implemented yet.

## Run checks

Run the set package tests without cache and run vet for the package:

```bash
go test ./cards/<package>/ -count=1
go vet ./cards/<package>/
```

Run broader checks when needed to confirm an engine interaction or repository-wide regression. Report pre-existing failures separately from audit findings when the evidence permits.

## Report

Lead with a count-based summary, then list actionable findings by severity:

1. failing tests or build errors;
2. Oracle or rules mismatches;
3. incomplete, stubbed, or inconsistently unsupported cards;
4. missing or ineffective tests;
5. shared engine gaps.

For each correctness finding, identify the card, authoritative behavior, actual behavior, evidence location, and required outcome. Group findings that share a root cause. End with a prioritized next-work plan and explicitly state what was checked and any limits on confidence.
