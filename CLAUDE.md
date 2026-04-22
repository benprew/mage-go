# CLAUDE.md

## What This Is

MTG rules engine in Go (2-player, XMage-inspired). Only external dep: `github.com/google/uuid`.

XMage reference: `~/mage/Mage.Sets/src/mage/cards/` and `~/mage/Mage/src/main/java/mage/`.

## Commands

```bash
go test ./...                              # all tests
go test ./cards/...                        # card tests only
go test ./cards/arabian/ -run TestFooBar   # single test
go build ./...                             # build check
go vet ./...                               # vet
make wasm                                  # WASM build
golangci-lint run --fix                    # lint (run after changes)
```

## Code Style

- Avoid inline comments; comments explain "why" not "what"
- Function header comments for public functions are good
- TDD: write test first, implement, confirm tests pass

## Cardinal Rules

1. **Follow Oracle text exactly.** Include full Oracle text as comment above `Register()`. Check every word — "nontoken," "each," "nonbasic," etc. No paraphrasing, no shortcuts.

2. **Never simplify.** Every condition, restriction, and edge case matters. A simplified implementation is a **wrong** implementation. If the engine can't support something, mark with `// XXX:` and ask — never silently simplify.

3. **TDD.** Write a failing test first, then implement.

4. **Consult XMage** (`~/mage`) when Oracle text is ambiguous or mechanics are complex.

## Card Implementation Guide

**Read `pkg/mage/doc.go` first** — comprehensive reference for the engine API: card constructors, effects, targets, filters, costs, abilities, continuous effects, layers, attrs, replacement effects, and `*Game` methods.

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
cards/{limited,arabian,antiquities,legends,custom}/  # card sets
cmd/{tui,server,wasm,fetchset,genset}/               # binaries
data/                  # Scryfall JSON per set
web/                   # browser UI
docs/                  # comprehensive rules, tutorials
```

Cards register via `Register(name, factory)` in `init()`. Each set's `test.go` has blank-identifier imports to ensure registration.

## Test Harness DSL

Tests use `gametest.TestGame` with `PlayerA`/`PlayerB`. Card tests go in the card package.

```go
g := gametest.NewTestGame(t)
g.AddCard(ZoneBattlefield, gametest.PlayerA, "Card Name")
g.StopAt(1, EndStep)
g.Execute()
g.AssertLife(gametest.PlayerB, 20)
```

**Setup**: `NewTestGame(t)`, `AddCard(zone, player, name, count...)`, `SetLife(player, n)`, `AddCounters(turn, step, player, card, counterType, n)`

**Actions** (turn, step, player, args...): `CastSpell`, `CastSpellWithX`, `CastInResponseTo`, `CastInResponseToWithX`, `ActivateAbility`, `ActivateAbilityWithX`, `ActivateInResponseTo`, `Attack(turn, player, attackers...)`, `Block(turn, player, blocker, attacker)`, `FormBand(turn, player, creatures...)`

**Choices**: `ChoosePermanent`, `ChooseDiscard(player, cards...)`, `ChooseManaColor`, `ChooseFromLibrary`, `ChooseMode`, `ChooseBandingDistribution(player, map[string]int)`

**Control**: `StopAt(turn, step)`, `Execute()`, `PlayToEnd(maxTurns...)`

**Assertions**: `AssertLife`, `AssertPoisonCounters`, `AssertPermanentCount`, `AssertGraveyardCount`, `AssertExileCount(name, n)` (global), `AssertHandCount`, `AssertLibraryCount`, `AssertPowerToughness`, `AssertCounterCount`, `AssertTapped`, `AssertHasAbility(player, name, keyword, has)`, `AssertAttachedTo`, `AssertLibraryTop(player, names...)`, `AssertGraveyardOrder`, `AssertAnteCount`, `AssertBanded(p1, c1, p2, c2, want)`, `AssertWinner`, `AssertGameOver(want)`, `AssertTotalTurns(n)`
