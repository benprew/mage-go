---
name: implement-set
description: "Implement or resume an entire Magic: The Gathering set in mage-go. Use to obtain set data, generate missing stubs, inventory remaining work, coordinate shared engine features and card batches, and verify set completeness."
---

# Implement a set

Drive a set from its current state to a verified implementation. Reuse `implement-card`, `implement-engine`, and `validate-set` rather than duplicating their detailed procedures.

## Resolve the set

Accept a Scryfall set code, package name, or set name. Use `data/sets.txt` and existing package metadata to resolve omitted values when possible.

If canonical JSON is missing, fetch it with:

```bash
FETCHSET_SKIP_TLS=1 go run ./cmd/fetchset -o data/<CODE>.json <CODE>
```

If the card package is missing, generate it with:

```bash
go run ./cmd/genset "<Set Name>" data/<CODE>.json cards/<package>/
```

Do not regenerate over existing work.

## Assess the current state

Prefer an applicable user-maintained document under `active-design-docs/`. Otherwise use `validate-set` for an existing or partially implemented package.

Build an inventory of outcomes, not an exhaustive taxonomy of implementation techniques:

- complete and verified;
- card work possible with current engine APIs;
- blocked by a shared engine capability;
- intentionally unsupported, if the user has confirmed that scope decision;
- incorrect or insufficiently tested existing work.

Read `pkg/mage/doc.go` and only the rules sections needed to assess uncertain mechanics. Group remaining work by shared behavior, dependency order, and likely file overlap. Keep batches small enough to review and diagnose, but do not use arbitrary numeric limits.

Present a concise inventory and execution order. Ask for direction only when a material scope choice, unsupported designation, or priority cannot be inferred safely.

## Execute

1. Address correctness issues and shared engine blockers in dependency order.
2. Use `implement-engine` for reusable engine capabilities, then `implement-card` for card wiring and card tests.
3. Delegate independent batches when useful, but avoid concurrent edits to shared files or coupled engine behavior.
4. Re-run relevant tests after each batch so regressions are localized.
5. Reassess when a card reveals an unanticipated engine gap; do not approximate Oracle behavior to keep a batch moving.

Preserve unrelated work and do not commit unless the user asks.

## Complete the set

Run `validate-set`, then run the repository's full tests, build, vet, and lint checks. Inspect all remaining `TODO: implement`, `XXX:`, and `UNIMPLEMENTABLE:` markers.

Every in-scope card must be exactly implemented and tested as appropriate. Any accepted partial or unsupported card must be explicit, justified, and consistently recorded in `UNSUPPORTED.md` when applicable.

Report completed cards, unresolved cards and reasons, engine capabilities added, verification results, and files changed.
