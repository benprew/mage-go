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
cmd/genset/            # stub generator from Scryfall JSON
data/                  # Scryfall JSON card data per set
.claude/skills/        # Claude Code skills for card implementation
```

## Adding a New Set

The project includes a pipeline for implementing entire card sets:

1. **Fetch** card data from Scryfall:
   ```bash
   go run ./cmd/fetchset -o data/DRK.json DRK
   ```

2. **Generate** Go stubs from the JSON. This creates one file per card type (`creatures.go`, `spells.go`, etc.) with full Oracle text as comments and `// TODO: implement` markers. Reprints already registered in other sets are automatically skipped.
   ```bash
   go run ./cmd/genset "The Dark" data/DRK.json cards/thedark/
   ```

3. **Implement** cards. Each card is implemented TDD-style: write a failing test that exercises the card's behavior, then implement the card to make it pass. The `pkg/mage/doc.go` file is the comprehensive API reference for all available effects, triggers, targets, and costs.

4. **Validate** completeness by checking every implementation against Scryfall Oracle text, auditing test coverage, and running the full test suite.

[Claude Code](https://claude.com/claude-code) skills (`.claude/skills/`) automate steps 3 and 4 — `/implement-set` handles the full implementation workflow and `/validate-set` audits the result.

## Build & Test

```bash
make test        # Unit tests.
go build ./...   # Build check.
go vet ./...     # Vet.
```

`make test-ai` compares the working tree AI with committed `HEAD`. Each run
resolves and prints the full HEAD commit hash. It copies the committed
interactive packages into a separate package tree in a temporary directory.
Historical AI, combat solver, evaluation, personality inference, and player
methods use that tree. No historical package can import current interactive
code. Both opponents use the committed HEAD rules engine and card implementations.
This also pins engine-level decision helpers. Working tree changes outside
the interactive packages do not change the engine or cards in this comparison.
An incompatible historical API causes a build failure.

The command first runs a HEAD-versus-HEAD control, then the working tree
comparison. Each suite runs 30,000 paired mirror games by default. Each pair
uses the same opening deck orders and swaps the AI strategies between seats. The limit
is 200 individual turns. Reports separate actual draws from turn-limit games
and use game pairs to calculate confidence intervals.

The primary suite uses WG Aggro, RB Burn, Aga Galneer, Guardian of the Tusk,
and the existing Merfolk Shaman rogue deck. `SIM_INCLUDE_PRIESTESS=1` adds
Priestess for slow-game stress tests. `SIM_GAMES`, `SIM_MAX_TURNS`, and
`SIM_SEED` override the game count, turn limit, and opening shuffle seed.
For example, `SIM_GAMES=10 make test-ai` runs a quick comparison check.
Each opponent infers its own personality from the deck with its own version
of the inference code and presets. The report lists both personality names
for each deck. This inference does not search for the best personality.
The temporary source tree is removed after the run.

The rules engine uses only the Go standard library.

## Credits

This project is inspired by [XMage](https://github.com/magefree/mage), which pioneered open-source MTG rules enforcement in Java. The engine's architectural bones — layered continuous effects, event-driven triggers, stack-based resolution — are drawn from XMage's design, reimplemented from scratch in idiomatic Go. XMage's card implementations were consulted as a reference for rules correctness throughout development.

Magic: The Gathering is a trademark of Wizards of the Coast LLC.

## License

GPL-2.0. See [LICENSE.txt](LICENSE.txt) for full text.
