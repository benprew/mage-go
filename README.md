# mage-go

A Magic: The Gathering rules engine written in Go, inspired by [XMage](https://github.com/magefree/mage)'s approach to open-source rules enforcement. The engine is a ground-up reimplementation — not a port — using idiomatic Go patterns: flat interfaces, composition over inheritance, and functional options instead of deep class hierarchies.

The engine is 2-player only. The card pool targets early Magic, from Alpha through roughly the Odyssey/Onslaught block era. Mechanics and rules from later sets (planeswalkers, transform, energy, sagas, etc.) are not in scope.

## How the Engine Works

**Effects** are the core abstraction. One-shot effects (deal damage, draw cards, destroy target) implement a simple `Effect` interface and resolve immediately from the stack. Continuous effects (lord boosts, aura enchantments, control-change) re-apply from scratch every cycle in MTG layer order (copy → control → type → color → ability → P/T), ensuring correct interaction regardless of timestamp.

**Abilities** come in three flavors — activated (`{cost}: effect`), triggered (`when/whenever...`), and static (continuous effects that apply while the source is on the battlefield). Triggered abilities listen for typed game events and filter via condition predicates, so new triggers don't require engine changes.

**The attr system** unifies keywords, capabilities, and type identity into additive integer counts on each permanent. `HasAttr(Flying)` replaces scattered boolean flags; granting and revoking compose naturally (base flying + "loses flying" = net zero). The entire granted-attr map resets and recomputes each effect cycle.

**GameMutator** is the interface boundary between effects and game state. Effects never touch `*Game` directly — they call explicit mutation methods, keeping the engine's internals shielded and the set of legal mutations auditable.

**The stack** is a standard LIFO queue unifying spells and abilities. Priority passes between players; when both pass with objects on the stack, the top resolves. State-based actions run between each resolution.

**Combat** handles banding, first/double strike, trample, and damage assignment with prevention/redirection/reflection via the DamageSystem subsystem.

## Status

The rules engine (`pkg/mage/`) is relatively stable and heavily tested — 800+ tests cover card interactions, combat, triggered abilities, continuous effects, banding, protection, and more. Every card is implemented test-first.

The TUI and SSH server (`cmd/tui/`, `cmd/server/`) are rough prototypes. They work well enough to play games against the AI or another human over SSH, but the interface is bare-bones and the AI is basic.

## Package Layout

```
pkg/mage/              # engine (package mage)
pkg/mage/core/         # enums, value types (package core)
pkg/mage/gametest/     # test harness DSL (package gametest)
pkg/mage/interactive/  # TUI/AI player layer
cards/limited/         # Alpha (Limited Edition)
cards/arabian/         # Arabian Nights
cards/antiquities/     # Antiquities
cards/legends/         # Legends
cards/custom/          # custom/test cards
cmd/tui/               # terminal UI
cmd/server/            # SSH multiplayer server
cmd/fetchset/          # set data fetcher
```

## Build & Test

```bash
go test ./...    # all tests
go build ./...   # build check
go vet ./...     # vet
```

The only external dependency is `github.com/google/uuid`.

## Credits

This project is inspired by [XMage](https://github.com/magefree/mage), which pioneered open-source MTG rules enforcement in Java. The engine's architectural bones — layered continuous effects, event-driven triggers, stack-based resolution — are drawn from XMage's design, reimplemented from scratch in idiomatic Go. XMage's card implementations were consulted as a reference for rules correctness throughout development.

Magic: The Gathering is a trademark of Wizards of the Coast LLC.

## License

GPL-2.0. See [LICENSE](LICENSE) for full text.
