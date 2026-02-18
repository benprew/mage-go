# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Run all tests
go test ./...

# Run only the core engine tests
go test .

# Run only the cards package tests
go test ./cards/...

# Run a single test by name
go test ./cards/... -run TestAlphaWalls

# Build check (library only, no binary)
go build ./...

# Vet
go vet ./...
```

## Origin

This is a Go reimplementation of [XMage](https://github.com/magefree/mage) (Magic, Another Game Engine), an open-source Java engine that enforces full MTG rules for 28,000+ cards. A local clone of XMage lives at `~/mage` and serves as the reference for rules behavior, card implementations, and test scenarios. When implementing new cards or mechanics, consult the Java source in `~/mage/Mage.Sets/src/mage/cards/` and `~/mage/Mage/src/main/java/mage/` for how XMage handles them.

## Architecture

This is a Go library (`github.com/mage/mage`) implementing the MTG game engine. There is no `main` package — it is imported as a library. The only external dependency is `github.com/google/uuid`.

### Package Layout

- **Root package (`mage`)**: The entire game engine — game state, cards, abilities, effects, combat, mana, targeting, the stack, and the test harness.
- **`cards/`**: Card definitions organized by set and type. Registers cards into the global registry via `init()` functions.

### Core Types and Their Relationships

**Game loop**: `Game` (game.go) holds all state — players, battlefield, exile, stack, combat, effects. Turns progress through `PhaseStep` values defined in turn.go. The game fires `GameEvent` values (event.go) that triggered abilities listen for.

**Card system**: The `Card` interface (card.go) is implemented by `BaseCard`. Cards are created through factory functions registered with `Register(name, factory)` in registry.go and instantiated with `CreateCard(name)`. Card constructors like `NewCreature(name, manaCost, subtypes...)` are in card.go.

**Abilities**: Three-layer hierarchy — `Ability` interface (ability.go) for keywords/static abilities, `ActivatedAbilityI` (activated.go) for activated abilities with costs/targets/effects, and `TriggeredAbility` (triggered.go) for event-driven triggers. All use builder-pattern chaining (`AddCost`, `AddTarget`, `AddEffect`).

**Effects**: One-shot effects implement the `Effect` interface (effect.go). Continuous effects use the MTG layer system (copy→control→text→type→color→ability→P/T) managed by `EffectManager` (continuous.go).

**Combat**: `Combat` struct (combat.go) handles attacker/blocker declaration, damage assignment, and evasion rules (flying/reach, menace, fear, landwalk, protection).

**Targeting**: `Target` interface (target.go) with implementations for creatures, players, permanents, and flexible "any target" patterns. Filters (filter.go) are composable predicates (`And`, `Or`, `Not`) used by both targeting and game queries.

**Mana**: `ManaCost` parsed from strings like `"{2}{W}{B}"` via `ParseManaCost()`. `ManaPool` tracks available mana with `CanPay`/`Pay` (mana.go). Costs (cost.go) wrap mana payments, tap, sacrifice, life payment, and counter removal.

### Adding a New Card

Register a factory in the appropriate file under `cards/` (e.g., `cards/creatures.go`, `cards/alpha_spells.go`):

```go
mage.Register("Card Name", func() mage.Card {
    c := mage.NewCreature("Card Name", "{2}{G}", "Beast")
    c.Power_ = 3
    c.Toughness_ = 3
    c.AddAbility(mage.HasKeyword(mage.Trample))
    return c
})
```

Registration functions are called from `init()` in each file. Alpha set cards use separate `registerAlpha*` functions referenced by blank identifiers in test files to ensure loading.

### Test Harness DSL

Tests use `TestGame` (harness.go) with two scripted `TestPlayer` instances (`PlayerA`, `PlayerB`). The DSL supports:

```go
g := mage.NewTestGame(t)
g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Serra Angel")
g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears")
g.Attack(1, mage.PlayerA, "Serra Angel")
g.Block(1, mage.PlayerB, "Grizzly Bears", "Serra Angel")
g.Cast(1, mage.PrecombatMain, mage.PlayerA, "Lightning Bolt", "Grizzly Bears")
g.StopAt(1, mage.EndCombat)
g.Execute()
g.AssertLife(mage.PlayerB, 16)
g.AssertPermanentCount(mage.PlayerB, "Grizzly Bears", 0)
```

Key harness methods: `AddCard`, `SetLife`, `AddCounters`, `Cast`, `Activate`, `Attack`, `Block`, `StopAt`, `Execute`, and `Assert*` family.
