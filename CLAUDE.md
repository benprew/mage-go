# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Run all tests
go test ./...

# Run only the core engine tests (harness, banding, DSL)
go test ./pkg/mage/gametest/

# Run only the cards package tests
go test ./cards/...

# Run a single test by name
go test ./cards/limited/ -run TestAlphaWalls

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

- **`pkg/mage/core/` (package `core`)**: Fundamental value types and enums with no engine dependencies — `Zone`, `PhaseStep`, `Keyword`, `CounterType`, `EventType`, `GameEvent`, `CardType`, `Color`, `ManaCost`, `Mana`, `Layer`, `Duration`, `AttachType`, `AbilityType`. External consumers import `core` directly (e.g. `core.Flying`, `core.ZoneBattlefield`). Internal `mage` files use a dot import.
- **`pkg/mage/` (package `mage`)**: Game engine — game state, cards, abilities, effects, combat, mana pool, targeting, the stack.
- **`pkg/mage/gametest/` (package `gametest`)**: Test harness DSL (`TestGame`, `TestPlayer`, `PlayerA`/`PlayerB` refs). Imported by card tests.
- **`pkg/mage/interactive/` (package `interactive`)**: TUI/AI player layer — `AIPlayer`, `RunGameLoop`, `SnapshotGameState`, action types, prompt types. Imported by `cmd/tui/`.
- **`cards/limited/` (package `limited`)**: Alpha (Limited Edition) card definitions. Registers cards into the global registry via `init()`.
- **`cards/arabian/` (package `arabian`)**: Arabian Nights card definitions.
- **`cards/custom/` (package `custom`)**: Custom/test cards (e.g., Wraithbloom).
- **`cmd/tui/`**: Terminal UI (package `main`).
- **`cmd/fetchset/`**: Set fetcher utility (package `main`).

### Core Types and Their Relationships

**Game loop**: `Game` (pkg/mage/game.go) holds all state — players, battlefield, exile, stack, combat, effects. Turns progress through `PhaseStep` values defined in core/turn.go. The game fires `GameEvent` values (core/event.go) that triggered abilities listen for.

**Card system**: The `Card` interface (pkg/mage/card.go) is implemented by `BaseCard`. Cards are created through factory functions registered with `Register(name, factory)` in registry.go and instantiated with `CreateCard(name)`. Card constructors like `NewCreature(name, manaCost, subtypes...)` are in card.go.

**Abilities**: Three-layer hierarchy — `Ability` interface (ability.go) for keywords/static abilities, `ActivatedAbility` (activated.go) for activated abilities with costs/targets/effects, and `TriggeredAbility` (triggered.go) for event-driven triggers. All use builder-pattern chaining (`AddCost`, `AddTarget`, `AddEffect`).

**Effects**: One-shot effects implement the `Effect` interface (effect.go). Continuous effects use the MTG layer system (copy→control→text→type→color→ability→P/T) managed by `EffectManager` (continuous.go).

**Combat**: `Combat` struct (combat.go) handles attacker/blocker declaration, damage assignment, and evasion rules (flying/reach, menace, fear, landwalk, protection).

**Targeting**: `Target` interface (target.go) with implementations for creatures, players, permanents, and flexible "any target" patterns. Filters (filter.go) are composable predicates (`And`, `Or`, `Not`) used by both targeting and game queries.

**Mana**: `ManaCost` parsed from strings like `"{2}{W}{B}"` via `ParseManaCost()`. `ManaPool` tracks available mana with `CanPay`/`Pay` (mana.go). Costs (cost.go) wrap mana payments, tap, sacrifice, life payment, and counter removal.

### Adding a New Card

Register a factory in the appropriate file under `cards/` (e.g., `cards/limited/creatures.go`, `cards/limited/spells.go`):

```go
mage.Register("Card Name", func() mage.Card {
    return mage.NewCreature("Card Name", "{2}{G}", 3, 3,
        mage.WithSubTypes("Beast"),
        mage.WithKeyword(core.Trample),
    )
})
```

Registration functions are called from `init()` in each file. The `cards/limited/test.go` file holds blank-identifier references (e.g., `var _ = registerCreatures`) to ensure all registration functions are called even when only test files import the package.

### Test Harness DSL

Tests use `TestGame` (pkg/mage/gametest/) with two scripted `TestPlayer` instances (`PlayerA`, `PlayerB`). Card test files import `core` and `gametest`:

```go
import (
    "github.com/mage/mage/pkg/mage/core"
    "github.com/mage/mage/pkg/mage/gametest"
)

g := gametest.NewTestGame(t)
g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serra Angel")
g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
g.Attack(1, gametest.PlayerA, "Serra Angel")
g.Block(1, gametest.PlayerB, "Grizzly Bears", "Serra Angel")
g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
g.StopAt(1, core.EndCombat)
g.Execute()
g.AssertLife(gametest.PlayerB, 16)
g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
```

Key harness methods: `AddCard`, `SetLife`, `AddCounters`, `CastSpell`, `ActivateAbility`, `Attack`, `Block`, `StopAt`, `Execute`, and `Assert*` family.
