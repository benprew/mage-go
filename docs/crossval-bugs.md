# Cross-Validation Results

## Architecture

XMage drives: XMage's AI makes all decisions (play land, cast spell, pass).
At each decision point, XMage emits game state as JSON. The Go driver reads
the stream and checks invariants. Both engines use the same deck (no shuffle).

## Run: 1000 games, seed=1, 10 turns each

~80 games completed before a game hung (XMage infinite loop on complex board state).
Each game produces ~200 decision points.

**Result: ~16,000 decision points checked, 0 engine divergences found.**

The only false positives were life totals > 20 (legitimate lifegain from cards
like Ivory Tower), which was fixed by adjusting the invariant check.

## Earlier Runs (mage-go driven)

When mage-go drove decisions and both engines were compared at each step:

### Confirmed matching (no bugs)
- Basic land plays
- Vanilla/French vanilla creatures (Flying Men, Benalish Hero, Plague Rats, etc.)
- Non-targeted artifacts (Disrupting Scepter, Winter Orb, Jade Monolith)
- Non-targeted enchantments (Angelic Voices, Living Lands)
- Non-targeted sorceries (Visions)
- Turn structure, priority, draw, untap, upkeep triggers
- Library ordering, hand contents, graveyard contents

### Harness gaps (not engine bugs)
- **X-cost spells**: Need X value in wire protocol (excluded from decks)
- **Aura targeting**: Both engines auto-target differently
- **Shuffle effects** (Winds of Change): XMage shuffle is no-op, causes draw divergence
- **Library reorder** (Natural Selection): Different ordering choices
- **Mana payment**: Different land tapping order (cosmetic, same result)
- **Stack item names**: mage-go uses "Ability", XMage uses full rule text
- **Activated ability names**: Different formatting between engines

### Potential engine bug (from mage-go driven run, seed=200)
**Winter Orb + Divine Offering life divergence** — P0 at 19 life in mage-go but 20
in XMage after no attacks. Needs investigation with a focused test.

## Files

- `cmd/crossval/` — Go driver
- `../xmage/Mage.Tests/.../crossval/` — Java CrossValPlayer + CrossValOracle
- Run with: `go build -o crossval ./cmd/crossval/ && ./crossval -xmage ../xmage -turns 10 -games 100`
