# CLAUDE.md

## What This Is

Magic: The Gathering rules engine in Go (2-player, XMage-inspired).

## Commands

```bash
make test                                  # all tests
go test ./cards/...                        # card tests only
go test ./cards/arabian/ -run TestFooBar   # single test
make build                                 # build check
make lint                                  # lint (run after making changes and before commit)
```

## Code Style

- Avoid inline comments; comments explain "why" not "what"
- Function header comments for public functions are good
- TDD: write test first, implement, confirm tests pass

## Cardinal Rules

1. **Follow Oracle text exactly.** Include full Oracle text as comment above `Register()`. Check every word — "nontoken," "each," "nonbasic," etc. No paraphrasing, no shortcuts.

2. **Never simplify.** Every condition, restriction, and edge case matters. A simplified implementation is a **wrong** implementation. If the engine can't support something, mark with `// XXX:` and ask — never silently simplify.

3. **TDD.** Write a failing test first, then implement.

## Card Implementation Guide

**Card definitions**: Read cards from `cards/mtgish/*.ron`. These files use Rust Object Notation (RON). Use their card records to inspect card definitions and derive card requirements instead of reading `cards/<set>/*.go`.

**Card rulings**: Before you implement or change card behavior, read the complete RON record and all local rulings for the card. Use `data/rulings/index.json` to map the exact card name to `data/rulings/<oracle-id>.json`. Reprints share one file. For an engine feature, read the rulings for each card used to define or test the feature. Use the Comprehensive Rules to establish the shared engine contract. Preserve the distinction between `wotc` rulings and `scryfall` notes. If a card has no local entry, run `go run ./cmd/fetchset -card "Card Name"` before proceeding. An explicit empty `rulings` array means the card was checked and has no published rulings; a missing file does not. Report a fetch failure instead of assuming there are no rulings. Refresh the full RON collection with `go run ./cmd/fetchset -ron cards/mtgish`. See `data/rulings/README.md` for the storage format and commands.

**Read `pkg/mage/doc.go` first** — comprehensive reference for the engine API: card constructors, effects, targets, filters, costs, abilities, continuous effects, layers, attrs, replacement effects, and `*Game` methods.

Before card or engine implementation, run `go run ./cmd/fetchset -check -card "Card Name"` for each affected card. If the check fails, use the download command in the error and repeat the check. The normal test suite also checks rulings coverage for the full RON collection.

**Comprehensive Rules**: `docs/comprehensive-rules.md` (~9200 lines). Use `docs/comprehensive-rules-index.md` to find sections by line number, then read just that section.

**Replacement effects**: See "Replacement Effect System" in `doc.go`. Most cards use `*Game` proxy methods (`AddPreventionShield`, `AddRegenerationShield`, etc.).

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
cards/mtgish/          # card definitions in Rust Object Notation (*.ron)
cards/{limited,arabian,antiquities,legends,custom}/  # card sets
cmd/{tui,server,fetchset,genset}/               # binaries
data/                  # Scryfall JSON per set
docs/                  # comprehensive rules, tutorials
```

Cards register via `Register(name, factory)` in `init()`. Each set's `test.go` has blank-identifier imports to ensure registration.

## Test Harness DSL

Tests use `gametest.TestGame` with `PlayerA`/`PlayerB`. Card tests go in the card package.
