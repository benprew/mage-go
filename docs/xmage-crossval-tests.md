# Cross-Validation Results

## Run 1: seed=42, turns=10, 1 game, 20 decisions
**Result**: 86 mismatches (cascading from Earthquake X-value)

Root cause: Earthquake cast with X=0 in mage-go (harness didn't propagate X value),
XMage auto-selected X=2. All subsequent mismatches cascade from the life total difference.

**Harness fix**: Exclude X-cost spells from random decks until X-value protocol is added.

## Run 2: seed=100, turns=10, 3 games, 38 decisions (X-cost spells excluded)
**Result**: Game 1 OK (13 decisions), Games 2-3 had mismatches

### Game 1: OK (13 decisions, 10 turns)
Both engines agree on all state: life, hand, battlefield, graveyard, library, stack.
Cards cast: Angelic Voices, Disrupting Scepter, Singing Tree.
Mana-payment tap differences correctly ignored.

### Game 2: Instill Energy targeting divergence (T6)
- mage-go: Instill Energy resolved to battlefield (auto-targeted Plague Rats)
- XMage: Instill Energy stuck on stack (target not provided by wire protocol)
- **Root cause**: Aura spells need target selection in the wire protocol. The Go driver
  sends `cast_spell` without specifying the target, so XMage's CrossValPlayer can't
  match the target and the spell doesn't resolve properly.
- **Harness fix needed**: Propagate target selection for targeted spells.

### Game 3: Blight + Living Lands complex interaction (T9)
Actions: Shelkin Brownie cast T5, Berserk T6, Blight T7, Living Plane T8, Living Lands T9.
- mage-go: Blight triggered (enchanted Plains became tapped), destroyed Plains. Living Lands resolved.
- XMage: Blight trigger and Living Lands both on stack (different resolution timing).
- **Potential engine bug**: Stack resolution ordering difference. Blight's "when enchanted land becomes
  tapped, destroy it" trigger fires immediately in mage-go but is still on the stack in XMage.
  May also be a target-selection issue (Blight is an Aura targeting a land).

## Confirmed Working
- Basic land plays: both engines agree perfectly
- Creature spells (vanilla/French vanilla): both engines agree (Plague Rats, Shelkin Brownie, Singing Tree)
- Non-targeted artifact spells: both engines agree (Disrupting Scepter, Jade Monolith)
- Non-targeted enchantment spells: both engines agree (Angelic Voices)
- Ability activation (upkeep triggers): compared without issues
- Turn structure, priority, draw: fully synchronized
- Library ordering: matched after reorder fix

## Harness Improvements Needed
1. **X-cost spells**: Add X value to wire protocol (currently excluded from decks)
2. **Targeted spells**: Propagate target selection (Auras, removal, etc.)
3. **Combat**: Currently attacks with all creatures, doesn't block — need smarter combat
4. **Mid-resolution choices**: Auto-accept first option, should match both engines
