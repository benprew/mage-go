# CLAUDE.md

## What This Is

A Go reimplementation of [XMage](https://github.com/magefree/mage) — an open-source MTG rules engine. 2-player only. The only external dependency is `github.com/google/uuid`.

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

**Read `pkg/mage/doc.go` first.** It is the comprehensive reference for every engine subsystem: card constructors, effects (50+), targets, filters, costs, activated abilities, triggered abilities (20+ convenience constructors), continuous effects, the layer system, attrs, the replacement effect system, and the GameMutator API. It includes complete card examples.

**Replacement effects**: Cards that prevent, redirect, or replace game actions (damage prevention, regeneration, Lich, etc.) use the `ReplacementEffect` pipeline. See the "Replacement Effect System" section in `doc.go` for the `Action` types, `ReplacementEffect` interface, the 17 built-in replacements, and how to implement custom ones. Most cards use the existing `GameMutator` proxy methods (`AddPreventionShield`, `AddRegenerationShield`, etc.) which create replacements internally.

## Package Layout

```
pkg/mage/              # engine (package mage)
pkg/mage/core/         # enums, value types (package core)
pkg/mage/gametest/     # test harness DSL (package gametest)
pkg/mage/interactive/  # TUI/AI player layer
cards/limited/         # Alpha cards
cards/arabian/         # Arabian Nights cards
cards/antiquities/     # Antiquities cards
cards/custom/          # custom/test cards
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
    g.StopAt(1, EndOfTurn)
    g.Execute()
    g.AssertLife(gametest.PlayerB, 20)
}
```

### Setup

- `NewTestGame(t)` — creates game, both players at 20 life, empty board
- `AddCard(zone, player, name)` — put a card in any zone (battlefield, hand, graveyard, library, exile). Call multiple times for multiple copies.
- `AddCardCount(zone, player, name, n)` — add n copies
- `SetLife(player, n)` — set starting life
- `AddCounters(player, cardName, counterType, n)` — add counters to a permanent

### Actions (turn number, phase/step, player, args...)

- `CastSpell(turn, step, player, spellName, targets...)` — cast from hand
- `CastSpellWithX(turn, step, player, spellName, x, targets...)` — cast with X
- `CastInResponseTo(turn, step, player, spellName, triggerDesc, targets...)` — cast in response to a trigger/spell
- `ActivateAbility(turn, step, player, permanentName, abilityText, targets...)` — activate an ability
- `ActivateAbilityWithX(turn, step, player, name, text, x, targets...)` — activate with X
- `Attack(turn, player, attackers...)` — declare attackers
- `Block(turn, player, blocker, attacker)` — declare a block
- `FormBand(turn, player, creatures...)` — form a band of attackers

### Choices (scripted decisions for AI-free testing)

- `ChoosePermanent(turn, step, player, permanentName)` — choose a permanent when prompted
- `ChooseDiscard(turn, step, player, cardNames...)` — choose cards to discard
- `ChooseManaColor(turn, step, player, color)` — choose a mana color
- `ChooseFromLibrary(turn, step, player, cardName)` — choose a card from library
- `ChooseMode(turn, step, player, mode)` — choose a mode (e.g., charm modes)
- `ChooseBandingDistribution(turn, player, assignments...)` — assign banding damage

### Control

- `StopAt(turn, step)` — stop execution at this point
- `Execute()` — run the game to the stop point
- `PlayToEnd()` — run until game over (no stop point needed)

### Assertions

- `AssertLife(player, n)`
- `AssertPermanentCount(player, name, n)`
- `AssertGraveyardCount(player, name, n)`
- `AssertExileCount(player, name, n)`
- `AssertHandCount(player, n)`
- `AssertLibraryCount(player, n)`
- `AssertPowerToughness(player, name, power, toughness)`
- `AssertCounterCount(player, name, counterType, n)`
- `AssertTapped(player, name, tapped)`
- `AssertHasAbility(player, name, attr)`
- `AssertAttachedTo(player, auraName, targetName)`
- `AssertLibraryTop(player, name)`
- `AssertGraveyardOrder(player, names...)`
- `AssertAnteCount(player, n)`
- `AssertBanded(player, creature, bandmates...)`
- `AssertWinner(player)`
- `AssertGameOver()`
- `AssertTotalTurns(n)`
