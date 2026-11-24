# mage-go

A Magic: The Gathering rules engine written in Go, inspired by [XMage](https://github.com/magefree/mage)'s approach to open-source rules enforcement. The engine is a ground-up reimplementation — not a port — using idiomatic Go patterns: flat interfaces, composition over inheritance, and functional options instead of deep class hierarchies.

The engine is 2-player only. The card pool targets early Magic, from Alpha through roughly the Odyssey/Onslaught block era.

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
pkg/mage/mtgish/        # tagged JSON parser and compiler
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
.agents/skills/        # Agent skills for the repo
```

## Card Definitions and Sets

Use the sibling mtgish checkout to inspect card definitions from the mage-go root.

```bash
../mtgish/show_card "Sol Ring"
```

The compiler in `pkg/mage/mtgish` consumes tagged JSON from `../mtgish/data/mtgish.lines.json`. Its rule coverage is still growing. Existing Go definitions in `cards/` supply cards during the migration.

Use Scryfall JSON for the expected set roster and Oracle text. The repo skills cover [rules and card support](.agents/skills/implement-engine/SKILL.md), [set implementation](.agents/skills/implement-set/SKILL.md), and [set validation](.agents/skills/validate-set/SKILL.md).

### Card rulings

Use the Scryfall rulings snapshot in `data/rulings/` with Oracle text and the
Comprehensive Rules when you implement or audit a card. Look up the exact card
name in `index.json` and read the listed `<oracle-id>.json` file in the same
directory. Reprints share one file. [Rulings index](data/rulings/index.json)

The record's `retrieved_at` field gives the date and time of the Scryfall lookup.
Each item in `rulings` stores the text in `comment`, the publication date in
`published_at`, and the source in `source`. A source of `wotc` identifies Wizards
of the Coast, and `scryfall` identifies a Scryfall note. An empty `rulings` array
means Scryfall returned no rulings at retrieval time. If an entry or file is
missing, consult the card's published rulings when you need them to establish
its behavior.

## Build & Test

```bash
make test        # Unit tests.
go build ./...   # Build check.
go vet ./...     # Vet.
```

`go run ./cmd/aibench` compares the working tree AI with committed `HEAD`. Each run
resolves and prints the full HEAD commit hash. It copies the committed
interactive packages into a separate package tree in a temporary directory.
Historical AI, combat solver, evaluation, personality inference, and player
methods use that tree. No historical package can import current interactive
code. Both opponents use the committed HEAD rules engine and card implementations.
This also pins engine-level decision helpers. Working tree changes outside
the interactive packages do not change the engine or cards in this comparison.
An incompatible historical API causes a build failure.

The command first runs a HEAD-versus-HEAD control, then the working tree
comparison. Set `SIM_GAMES` to a positive even number of games per suite. Each pair
uses the same opening deck orders and swaps the AI strategies between seats. The limit
is 200 individual turns. Reports separate actual draws from turn-limit games
and use game pairs to calculate confidence intervals.

The primary suite uses WG Aggro, RB Burn, Aga Galneer, Guardian of the Tusk,
and the existing Merfolk Shaman rogue deck. `SIM_INCLUDE_PRIESTESS=1` adds
Priestess for slow-game stress tests. Use `SIM_MAX_TURNS` and `SIM_SEED`
to override the turn limit and opening shuffle seed.
For a smoke test, run `SIM_GAMES=10 go run ./cmd/aibench`.
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
