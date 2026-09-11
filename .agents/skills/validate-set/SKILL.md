---
name: validate-set
description: Audit one mage-go card set for completeness, Oracle fidelity, test quality, unsupported behavior, and regressions. Use for evidence-based assessment without changing card or engine implementations.
---

# Validate a set

Audit actual implementation behavior. Do not edit card or engine code.

## Inputs

Resolve the package and Scryfall code with the request, `data/sets.txt`, and package metadata. Use Scryfall JSON as the Oracle source of truth.

If canonical JSON is missing, disclose that the audit will add `data/<CODE>.json`, or fetch it to a temporary path when persistence is unnecessary:

```bash
go run ./cmd/fetchset -o <output-path> <CODE>
```

Do not disable TLS verification as part of the normal workflow. Read only the comprehensive-rules sections needed to resolve specific questions.

## Audit

For every registered card, inspect code and tests and record:

- one implementation status;
- whether the complete Oracle text is above `Register()`;
- whether actual behavior matches every rules-relevant word;
- whether tests can fail for plausible incorrect behavior;
- any card-code defect or reusable engine gap.

Do not infer correctness from names, comments, markers, or passing tests alone. Do not demand redundant tests for declarative use of tested engine primitives. Reconcile every `UNIMPLEMENTABLE:` marker with `UNSUPPORTED.md`.

For a set audit, create an inventory that follows [references/set-inventory.schema.json](references/set-inventory.schema.json). Use exact counts and validate the saved artifact:

```bash
python3 .agents/skills/scripts/validate_artifact.py .agents/skills/validate-set/references/set-inventory.schema.json <inventory.json>
```

## Checks and report

Run:

```bash
go test ./cards/<package>/ -count=1
go vet ./cards/<package>/
```

Run broader checks only to confirm an engine interaction or repository regression. Separate pre-existing failures when evidence permits.

Write the result according to [references/validation-report.schema.json](references/validation-report.schema.json) when another skill or tool will consume it. Validate a saved report with the same script. Then give the user a concise human report led by exact counts and severity-ranked findings.

Each correctness finding must state the card, authoritative behavior, actual behavior, evidence, required outcome, and confidence. Group shared root causes and end with prioritized next work, checks run, and confidence limits.
