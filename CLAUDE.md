# CLAUDE.md

## What This Is

A Magic: The Gathering rules engine in Go, inspired by XMage's architecture. 2-player only. The only external dependency is `github.com/google/uuid`.

Reference XMage source lives at `~/mage`. Consult `~/mage/Mage.Sets/src/mage/cards/` and `~/mage/Mage/src/main/java/mage/` for rules behavior and card implementations.

## Commands

```bash
go test ./...                              # all tests
go test ./cards/...                        # card tests only
go test ./cards/arabian/ -run TestFooBar   # single test
go build ./...                             # build check
go vet ./...                               # vet
```

## Cardinal Rules

1. **Follow Oracle text exactly.** Every card implementation must match its Oracle text word-for-word. Include the full Oracle text as a comment above each `Register()` call. If Oracle says "nontoken," check for nontoken. If it says "each," hit each. No paraphrasing, no shortcuts.

2. **TDD — write the test first.** Before implementing a card, write a failing test that exercises its key behavior. Then implement the card to make the test pass. Tests catch rules bugs that code review misses.

3. **Consult XMage.** When the Oracle text is ambiguous or a mechanic is complex, check how `~/mage` implements it. XMage has 15+ years of rules-correctness work.

## Card Implementation Guide

**Read `pkg/mage/doc.go` first.** It is the comprehensive reference for every engine subsystem: card constructors, effects (50+), targets, filters, costs, activated abilities, triggered abilities (20+ convenience constructors), continuous effects, the layer system, attrs, and the GameMutator API. It includes complete card examples.

## Set Implementation Toolkit

The project has a pipeline for implementing entire card sets, from fetching data through validated completion.

### Tools

- **`cmd/fetchset`** — Fetches card data from the Scryfall API and writes JSON to `data/`.
  ```bash
  go run ./cmd/fetchset -o data/DRK.json DRK
  ```

- **`cmd/genset`** — Generates Go stub files from the JSON. Creates `register.go`, `test.go`, `creatures.go`, `artifacts.go`, `enchantments.go`, `spells.go`, and `lands.go` with full Oracle text comments and `// TODO: implement` markers. Automatically skips reprints already registered in other sets.
  ```bash
  go run ./cmd/genset "The Dark" data/DRK.json cards/thedark/
  ```

### Claude Code Skills

Three skills automate the card implementation workflow:

- **`/implement-card <card names>`** — TDD workflow for a single card or batch of similar cards. Reads the stub, writes failing tests, implements the card, verifies all tests pass, and commits. Consults `doc.go` for the engine API and XMage for complex mechanics. Uses `AskUserQuestion` before cutting any scope.

- **`/implement-set <set-code> [set-name] [package-name]`** — End-to-end set implementation. Runs `fetchset` and `genset`, then analyzes every card into tiers (vanilla → standard effects → complex → engine work required → out of scope). Presents the plan for approval, then works through batches via `/implement-card`, parallelizing independent work. Commits after each batch.

- **`/validate-set <package-name>`** — Audits a set for completeness and correctness. Checks every card's implementation against Scryfall Oracle text for fidelity — catches missing abilities, simplified effects (e.g., "nontoken" not checked), wrong values, and missing conditions. Audits test coverage, runs the test suite, and produces a structured report with prioritized next steps.

### Typical Workflow

```
1. go run ./cmd/fetchset -o data/DRK.json DRK       # fetch from Scryfall
2. go run ./cmd/genset "The Dark" data/DRK.json cards/thedark/  # generate stubs
3. /implement-set DRK "The Dark" thedark             # implement all cards
4. /validate-set thedark                             # audit completeness
```

## Package Layout

```
pkg/mage/              # engine (package mage)
pkg/mage/core/         # enums, value types (package core)
pkg/mage/gametest/     # test harness DSL (package gametest)
pkg/mage/interactive/  # TUI/AI player layer
cards/limited/         # Alpha cards
cards/arabian/         # Arabian Nights cards
cards/antiquities/     # Antiquities cards
cards/legends/         # Legends cards
cards/custom/          # custom/test cards
cmd/tui/               # terminal UI
cmd/server/            # SSH multiplayer server
cmd/fetchset/          # set data fetcher
cmd/genset/            # stub generator from Scryfall JSON
data/                  # Scryfall JSON card data per set
```

Cards register via `Register(name, factory)` in `init()`. Each card file has a registration function called from `init()`. The `test.go` file in each card package holds blank-identifier references to ensure registration runs.

## Test Harness DSL

Tests use `gametest.TestGame` with two scripted players (`PlayerA`, `PlayerB`). Card test files go in the same package as the cards (e.g., `cards/arabian/creatures_test.go`).

```go
import (
    . "github.com/mage/mage/pkg/mage/core"
    "github.com/mage/mage/pkg/mage/gametest"
)

func TestCard(t *testing.T) {
    g := gametest.NewTestGame(t)
    g.AddCard(ZoneBattlefield, gametest.PlayerA, "Card Name")
    g.StopAt(1, EndStep)
    g.Execute()
    g.AssertLife(gametest.PlayerB, 20)
}
```

### Setup

- `NewTestGame(t)` — creates game, both players at 20 life, empty board
- `AddCard(zone, player, name, count...)` — put a card in any zone (battlefield, hand, graveyard, library, exile). Optional count parameter for multiple copies.
- `SetLife(player, n)` — set starting life
- `AddCounters(turn, step, player, cardName, counterType, n)` — add counters to a permanent

### Actions (turn number, phase/step, player, args...)

- `CastSpell(turn, step, player, spellName, targets...)` — cast from hand
- `CastSpellWithX(turn, step, player, spellName, x, targets...)` — cast with X
- `CastInResponseTo(player, spellName, targets...)` — cast in response (no turn/step needed)
- `CastInResponseToWithX(player, spellName, x, targets...)` — cast in response with X
- `ActivateAbility(turn, step, player, permanentName, targets...)` — activate an ability
- `ActivateAbilityWithX(turn, step, player, permanentName, x, targets...)` — activate with X
- `ActivateInResponseTo(player, permanentName, targets...)` — activate in response
- `Attack(turn, player, attackers...)` — declare attackers
- `Block(turn, player, blocker, attacker)` — declare a block
- `FormBand(turn, player, creatures...)` — form a band of attackers

### Choices (scripted decisions for AI-free testing)

- `ChoosePermanent(player, permanentName)` — choose a permanent when prompted
- `ChooseDiscard(player, cardNames...)` — choose cards to discard
- `ChooseManaColor(player, color)` — choose a mana color
- `ChooseFromLibrary(player, cardName)` — choose a card from library
- `ChooseMode(player, mode)` — choose a mode (e.g., charm modes)
- `ChooseBandingDistribution(player, distribution)` — assign banding damage (map[string]int)

### Control

- `StopAt(turn, step)` — stop execution at this point
- `Execute()` — run the game to the stop point
- `PlayToEnd(maxTurns...)` — run until game over (optional max turns)

### Assertions

- `AssertLife(player, n)`
- `AssertPoisonCounters(player, n)`
- `AssertPermanentCount(player, name, n)`
- `AssertGraveyardCount(player, name, n)`
- `AssertExileCount(name, n)` — no player param, checks global exile
- `AssertHandCount(player, name, n)`
- `AssertLibraryCount(player, name, n)`
- `AssertPowerToughness(player, name, power, toughness)`
- `AssertCounterCount(player, name, counterType, n)`
- `AssertTapped(player, name, tapped)`
- `AssertHasAbility(player, name, keyword, has)` — keyword is `core.Keyword`, has is bool
- `AssertAttachedTo(player, auraName, targetName)`
- `AssertLibraryTop(player, names...)`
- `AssertGraveyardOrder(player, names...)`
- `AssertAnteCount(player, name, n)`
- `AssertBanded(player1, creature1, player2, creature2, want)` — want is bool
- `AssertWinner(player)`
- `AssertGameOver(want)` — want is bool
- `AssertTotalTurns(n)`
