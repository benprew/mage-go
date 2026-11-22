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
- Function header comments for public functions are good
- TDD: write test first, implement, confirm tests pass

## Cardinal Rules

1. **Follow Oracle text exactly.** Check each mtgish definition against Oracle text and the relevant rules. For hand-written Go cards, keep the full Oracle text above `Register()`. Preserve every condition, choice, timing restriction, and duration.

2. **Reject incomplete behavior.** Fix reusable compiler and engine gaps when they are in scope. Reject an mtgish record if a required rule cannot compile. If a material gap requires deferral, report it and ask. Mark a deferred Go definition with `// XXX:`.


## Card Implementation Guide

Inspect the mtgish definition from the mage-go root with `../mtgish/show_card "Sol Ring"`. The source is `../mtgish/data/mtgish.lines.json`. Read `pkg/mage/doc.go` for the engine API before changing game behavior.

**Comprehensive Rules**: `docs/comprehensive-rules.md` (~9200 lines). Use `docs/comprehensive-rules-index.md` to find sections by line number, then read just that section.

**Replacement effects**: See "Replacement Effect System" in `doc.go`. Most cards use `*Game` proxy methods (`AddPreventionShield`, `AddRegenerationShield`, etc.).

## Set Implementation

Use Scryfall JSON for the expected roster and Oracle text. Fetch missing set data with `go run ./cmd/fetchset -o data/DRK.json DRK` for The Dark. Use mtgish for the card definitions. Use `cmd/genset` for a requested legacy Go package.

- [implement-engine](.agents/skills/implement-engine/SKILL.md) covers named cards, mtgish compiler support, and reusable game rules.
- [implement-set](.agents/skills/implement-set/SKILL.md) coordinates coverage and shared work across a set.
- [validate-set](.agents/skills/validate-set/SKILL.md) audits coverage and Oracle fidelity.

## Package Layout

```
pkg/mage/              # engine
pkg/mage/core/         # enums, value types
pkg/mage/mtgish/        # tagged JSON parser and compiler
pkg/mage/gametest/     # test harness DSL
pkg/mage/interactive/  # TUI/AI player layer
cards/{limited,arabian,antiquities,legends,custom}/  # card sets
cmd/{tui,server,wasm,fetchset,genset}/               # binaries
data/                  # Scryfall JSON per set
web/                   # browser UI
docs/                  # comprehensive rules, tutorials
```

Legacy Go cards register via `Register(name, factory)`. The mtgish compiler produces immutable `CardDefinition` values that can be registered with `RegisterDefinition`.

## Test Harness DSL

Tests use `gametest.TestGame` with `PlayerA`/`PlayerB`. Keep legacy Go tests in the card package. Test compiled cards from checked-in mtgish JSON fixtures.
