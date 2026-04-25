# Alpha (LEA) — Remaining Work

## Missing Cards (13 + 3 ante)

### Needs engine work

**Dingus Egg** `{4}` Artifact
Whenever a land is put into a graveyard from the battlefield, deals 2 damage to that land's controller.
*Needs: card type info on EvtPutIntoGraveyardFromBattlefield event, or a land-specific death event.*

**Creature Bond** `{1}{U}` Enchantment — Aura
Enchant creature. When enchanted creature dies, deal damage equal to its toughness to its controller.
*Needs: last-known-information for dying creature's toughness in aura's death trigger.*

**Stone Giant** `{2}{R}{R}` Creature — Giant 3/4
{T}: Target creature you control with toughness less than Stone Giant's power gains flying until end of turn. Destroy that creature at the beginning of the next end step.
*Needs: toughness-less-than-source-power filter for targeting.*

**Library of Leng** `{1}` Artifact
You have no maximum hand size. If an effect causes you to discard a card, you may put it on top of your library instead.
*Needs: discard replacement effect.*

**Living Artifact** `{G}` Enchantment — Aura
Enchant artifact. Whenever you're dealt damage, put vitality counters on this. At upkeep, remove one for 1 life.
*Needs: "whenever you're dealt damage" trigger.*


**Gaea's Liege** `{3}{G}{G}{G}` Creature — Avatar \*/\*
P/T = Forests you control (or defending player's Forests if attacking). {T}: Target land becomes a Forest.
*Needs: attacking-dependent P/T, permanent land type change effect.*

**Demonic Hordes** `{3}{B}{B}{B}` Creature — Demon 5/5
{T}: Destroy target land. Upkeep: pay {B}{B}{B} or tap self and opponent chooses a land you sac.
*Needs: opponent-chooses-permanent effect.*

**Time Vault** `{2}` Artifact
Enters tapped. Doesn't untap. Skip your turn to untap it. {T}: Extra turn.
*Needs: skip-turn replacement effect.*

**Volcanic Eruption** `{X}{U}{U}{U}` Sorcery
Destroy X target Mountains. Deal damage equal to Mountains actually destroyed.
*Needs: multi-target X, count-destroyed-for-damage pattern.*

**Power Leak** `{1}{U}` Enchantment — Aura
Enchant enchantment. At upkeep, controller may pay mana; deals 2 minus amount paid.
*Needs: pay-any-amount-of-mana mechanic.*

**Smoke** — IMPLEMENTED but needs creature untap limit engine feature (added `CreatureUntapMax` and `LimitCreatureUntaps`).

### Ante cards (out of scope)

- Contract from Below
- Darkpact
- Demonic Attorney

---

## Oracle Text Violations on Implemented Cards

### Needs engine work

**Personal Incarnation** — WRONG DIRECTION
Oracle: "{0}: The next 1 damage that would be dealt to this creature this turn is dealt to its owner instead."
Current: Redirects damage FROM player TO creature (opposite).
*Needs: one-shot damage redirect replacement FROM creature TO owner, as an activated ability.*

**Fireball** — single target instead of multi-target
Oracle: "Fireball deals X damage divided evenly, rounded down, among any number of targets."
Current: Single target, full X damage.
*Needs: multi-target spell framework, additional cost per target, damage division.*

**Sacrifice** (spell) — effect vs additional cost
Oracle: "As an additional cost to cast this spell, sacrifice a creature."
Current: Targets a creature and sacrifices on resolution.
*Needs: additional-cost-sacrifice on cast.*

**Drain Life** — missing life gain cap and mana restriction
Oracle: "Spend only black mana on X. You gain life equal to the damage dealt, but not more than the target's life/toughness."
Current: Gains life equal to X unconditionally.
*Needs: mana color restriction on X, life gain capped to actual damage dealt.*

**Natural Selection** — wrong effect
Oracle: "Look at the top three cards of target player's library, then put them back in any order."
Current: Shuffles the entire library.
*Needs: top-N-cards reorder API.*

**Rock Hydra** — missing 3 abilities
Oracle: Damage-to-counter replacement, {R}: prevent 1 damage, {R}{R}{R}: add counter (upkeep only).
Current: Only enters with X counters.
*Needs: damage-removes-counters replacement, prevention activated ability, upkeep-only activation restriction.*

**Clockwork Beast** — missing repair ability
Oracle: "{X}, {T}: Put up to X +1/+0 counters (max 7). Activate only during your upkeep."
Current: ETB + attack/block counter removal only.
*Needs: X-cost activated ability, counter cap, upkeep-only activation restriction.*

**Blaze of Glory** — missing timing + forced blocking
Oracle: "Cast only during combat before blockers declared. It blocks each attacking creature this turn if able."
Current: Just grants CanBlockAny.
*Needs: cast-timing restriction, forced-block-all-attackers.*

**Gloom** — missing activated ability cost increase
Oracle: "Activated abilities of white enchantments cost {3} more to activate."
Current: Only increases spell costs.
*Needs: activated-ability cost increase engine support.*

**Consecrate Land** — missing enchant restriction
Oracle: "Enchanted land can't be enchanted by other Auras."
Current: Only grants indestructible.
*Needs: "can't be enchanted by other Auras" restriction.*

**Farmstead** — unconditional life gain
Oracle: "At the beginning of your upkeep, you may pay {W}{W}. If you do, you gain 1 life."
Current: Gains 1 life unconditionally on upkeep.
*Needs: pay-or-don't-gain conditional trigger.*
