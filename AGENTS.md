# CLAUDE.md

## What This Is

Magic: The Gathering rules engine in Go (2-player, XMage-inspired).

## Commands

```bash
make test                                  # all tests
go test ./cards/...                        # card tests only
go test ./cards/arabian/ -run TestFooBar   # single test
make build                                 # build check
make wasm                                  # WASM build
make lint                                  # lint (run after making changes and before commit)
```

## Code Style

- Avoid inline comments; comments explain "why" not "what"
- High-leverage testing: exhaustive engine contract tests; risk-based card tests (TDD for complex cards; skip simulation tests for declarative cards)

## Cardinal Rules

1. **Follow Oracle text exactly.** Include full Oracle text as comment above `Register()`. Check every word — "nontoken," "each," "nonbasic," etc. No paraphrasing, no shortcuts.

2. **Never simplify.** Every condition, restriction, and edge case matters. A simplified implementation is a **wrong** implementation. If the engine can't support something, mark with `// XXX:` and ask — never silently simplify.

3. **High-leverage testing.** Exhaustive engine tests prove the primitives; automated registration validation checks 100% of cards. Use TDD for non-trivial cards (multi-step rules, custom triggers, unusual edge cases). Skip boilerplate simulation tests for purely declarative cards (vanilla, French vanilla, standard single-verb spells).

## Card Implementation Guide

- **Card DSL Reference**: `docs/dsl-reference.md` — comprehensive reference for the card author DSL: constructors, subjects, selectors, targets, actions, triggers, spells, activations, costs, continuous effects, and durations.
- **Rules & Engine Contracts**: `docs/engine-rules-contracts.md` — technical reference for Continuous Layers 1–7, LKI, state triggers, replacement pipelines, life payment boundaries, and cleanup.
- **Subsystem Architecture**: `pkg/mage/doc.go` — high-level package overview and subsystem ownership (`Game`, `ZoneSystem`, `TurnSystem`, etc.).
- **Comprehensive Rules**: `docs/comprehensive-rules.md` (~9200 lines). Use `docs/comprehensive-rules-index.md` to find sections by line number, then read just that section.
- **Replacement effects**: See `docs/engine-rules-contracts.md#replacement-and-prevention-pipeline-cr-614616`. Most cards use `*Game` proxy methods (`AddPreventionShield`, `AddRegenerationShield`, etc.).

## Set Implementation

```bash
go run ./cmd/fetchset -o data/DRK.json DRK                       # fetch from Scryfall
go run ./cmd/genset "The Dark" data/DRK.json cards/thedark/       # generate stubs
# then use /implement-set, /implement-card, /validate-set skills
```

- **`cmd/genset`** creates `register.go`, `test.go`, `creatures.go`, `artifacts.go`, `enchantments.go`, `spells.go`, `lands.go` with Oracle text comments and `// TODO: implement` markers. Skips reprints from other sets.
- **`/implement-card <names>`** — TDD workflow for one card or a batch
- **`/implement-set <set-code> [set-name] [pkg]`** — full set implementation pipeline
- **`/validate-set <pkg> [set-code]`** — audit completeness and Oracle fidelity

## Package Layout

```
pkg/mage/              # engine
pkg/mage/core/         # enums, value types
pkg/mage/gametest/     # test harness DSL
pkg/mage/interactive/  # TUI/AI player layer
cards/{limited,arabian,antiquities,legends,custom}/  # card sets
cmd/{tui,server,wasm,fetchset,genset}/               # binaries
data/                  # Scryfall JSON per set
web/                   # browser UI
docs/                  # comprehensive rules, tutorials
```

Cards register via `Register(name, factory)` in `init()`. Each set's `test.go` has blank-identifier imports to ensure registration.

## Test Harness DSL

Tests use `gametest.TestGame` with `PlayerA`/`PlayerB`. Card tests go in the card package.
