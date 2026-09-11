---
name: implement-set
description: "Drive one Magic: The Gathering set from stubs or partial work to a verified mage-go implementation. Use to coordinate inventory, shared engine gaps, card batches, and final set validation."
---

# Implement a set

Coordinate this state transition:

```text
resolve → inventory → order dependencies → execute batches → validate
```

## Resolve and bootstrap

Resolve a set code, package, or name with `data/sets.txt` and package metadata.

If canonical JSON is missing, fetch it with TLS verification enabled:

```bash
go run ./cmd/fetchset -o data/<CODE>.json <CODE>
```

If the package is missing, generate it with `cmd/genset`. Never generate over existing work.

## Inventory

Read `../validate-set/SKILL.md` and use its inventory workflow. For more than a few cards, maintain a record that follows `../validate-set/references/set-inventory.schema.json`. Do not start implementation from approximate counts or an unverified design note.

Order remaining work by shared engine dependency, then by independent card batches. A card must have one explicit inventory status; a reusable engine gap must have a stable gap ID.

## Execute

1. Read `../implement-engine/SKILL.md` before resolving a shared engine gap.
2. Read `../implement-card/SKILL.md` before wiring cards that current engine APIs support.
3. Update the inventory after each batch and rerun affected tests without cache.
4. Reorder work when a new engine gap appears. Do not approximate Oracle behavior to keep a batch moving.

Delegate only when the user requests or permits delegation and the batches have disjoint files with no unresolved shared engine dependency.

Ask for direction only for a material scope choice, an unsupported designation, or a priority that cannot be inferred safely.

## Complete

Read and run `../validate-set/SKILL.md` again. Reconcile every card and every `TODO: implement`, `XXX:`, and `UNIMPLEMENTABLE:` marker with the final inventory.

Run:

```bash
go test ./cards/<package>/ -count=1
go test ./... -short -count=1
go build ./...
go vet ./...
make lint
```

Report exact status counts, completed and unresolved cards, engine capabilities, verification, and changed files.
