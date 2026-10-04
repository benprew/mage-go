# RON semantic inventory API

```go
func InspectFS(files fs.FS, pattern string) (*Inventory, error)
```

Use `go run ./cmd/loadcards -inventory -json` to inspect the collection. The
command emits JSON with schema version 1. Inventory mode always emits JSON;
`-json` remains useful with the existing strict loader mode. The CLI also runs
`LoadFS` and returns nonzero when the collection fails strict loading. Inspection
never installs a registry, substitutes Go cards, or changes compilation rules.

`InspectFS` returns an error for invalid glob patterns and missing collections.
Read failures, parse failures, empty files, compiler errors, and conflicting
compiled reprints appear in `issues`. A parse failure rejects the whole file;
expressions from a partially parsed file are not included. Positions use the
existing `Position` JSON fields: `File`, `Line`, and `Column`.

The report counts parsed records, distinct nonempty card names, and compiled
records. `engine_validated` and `behavior_verified` are `null`: inspection does
not measure these stages. File entries have paths, SHA-256 hashes, and parsed
record counts. The first compiler error remains a record diagnostic. The AST
walk continues through every descendant independently of that error.

Each signature groups `(role, name, kind, arity, shape, context)`. Shapes include
immediate argument kinds and constructor names, or sorted struct field names.
Literal values are not part of the shape. Context records the constructor and
argument chain without list indices; expression paths retain all indices and
named fields, for example:

```text
Card.Rules[0].args[1].args[0][1].args[0]
```

Entries include all occurrence positions and paths, sorted distinct card names,
occurrence counts, and occurrences in compiled records. All entries have
`support: "unverified"`. A compiled record occurrence is evidence about that
record, not proof that the expression has the correct rules behavior in all
contexts. Cards attached to a signature can include both accepted and rejected
records; do not use this list to count compiled unique cards.

The walk covers symbols, literals, lists, tuples, and structs. Known argument
roles come from the parent's role and shape. In particular, `TapPermanent` in
an activation cost and in an action list has two distinct roles. Lists keep the
role of their elements. Unknown parent contracts give their children `unknown`
roles; known spellings below those parents do not imply bindings or support.
These unknown expressions remain visible for later compiler slices.

Current role adapters cover metadata; `SpellActions`, `Activated`, `TriggerA`,
`EnchantPermanent`, and the two static layer rule envelopes; `Targeted` and
`ActionList`; the three single-target declarations; tap/sacrifice/mana costs;
single-permanent actions, player action envelopes, source damage, draw/life
values; conjunction filters; and the three existing event envelopes. Other
arguments remain unknown. This role table is inspection metadata, not a second
compiler or a complete type checker. See `inventoryArgumentRole` for exact
name, role, and arity cases.

Traversal sorts files and named fields, preserves argument order, and sorts
signature keys and distinct card names. Compiler unknown-field diagnostics
also use sorted field names. The same unchanged collection therefore produces
the same report. Structural reprint comparisons use the loader's position-free
AST key. Conflicts retain the first compiled definition's position.
