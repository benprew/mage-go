# Alpha (LEA) — Remaining Work

Updated 2026-07-28 after reconciling the package against `IMPLEMENTATION_GAPS.md` and the current DSL.

## Cards still absent from `cards/limited`

- **Creature Bond** — needs a pipeline `ValueSource` for the dying attached creature's last-known toughness and a selector for its last-known controller.
- **Stone Giant** — needs a target filter for "creature you control with toughness less than this source's power." The flying and next-end-step destruction effects otherwise have reusable operators.
- **Gaea's Liege** — needs an attacking-aware defending-player count source and a source-duration land subtype override operator.
- **Demonic Hordes** — needs an opponent-chooses-permanent effect for the unpaid upkeep branch.
- **Time Vault** — needs a skip-your-next-turn cost/replacement operator.
- **Volcanic Eruption** — `TargetXPermanents` now covers target declaration, but the pipeline still needs to count which targeted Mountains were actually put into graveyards and use that count for damage.
- **Power Leak** — needs a pay-any-amount-of-mana choice whose paid amount can be stored and used by later prevention/damage steps.

## Registered cards with Oracle gaps

### Artifacts

- **Forcefield** — target selection must be restricted to an unblocked attacker.
- **Jade Statue** — needs a combat-only activation timing rule. The current timing model supports one named step, not the whole combat phase.

### Creatures

- **Personal Incarnation** — the damage redirect direction is now correct, but only the creature's owner may activate the ability; controller-only activation is still used when ownership and control differ.
- **Vesuvan Doppelganger** — copying must preserve its existing color rather than copying the chosen creature's color.

### Enchantments

- **Gloom** — activated abilities of white enchantments must cost `{3}` more.
- **Consecrate Land** — the enchanted land must be unable to be enchanted by other Auras.

### Instants and sorceries

- **Guardian Angel** — after the initial X prevention, its controller must be able to pay `{1}` repeatedly to prevent another 1 damage through end of turn.
- **Blue Elemental Blast** and **Red Elemental Blast** — each lacks its destroy-permanent mode; modal choices currently cannot declare targets in different zones.
- **Drain Life** — X must use only black mana. Its life-gain cap is implemented, but a pipeline value for the capped damage/life amount is still missing.
- **Fireball** — needs any-number targeting, the extra generic cost for each target beyond the first, and evenly divided damage rounded down.
- **Disintegrate** — needs the "if it would die this turn, exile it instead" replacement effect.
- **Sacrifice** — sacrificing a creature must be an additional casting cost, not a targeted resolving effect.
- **Natural Selection** — needs ordered top-three library selection plus the optional full-library shuffle.
- **Drain Power** — needs forced activation of each land's mana ability, draining all unspent mana, and adding the mana lost this way.
- **Blaze of Glory** — needs its cast window and forced-block-each-attacker behavior.
- **False Orders** — needs declare-blockers-only timing and the complete block reassignment/unblocking rules.
- **Siren's Call** — needs its cast restriction and an attack-if-able effect in addition to the delayed destruction.

## Ante cards

These cards are registered, but their current direct zone manipulation is not Oracle-complete and does not use the ante APIs consistently:

- **Contract from Below** — needs ante-enabled legality/removal handling and pipeline steps for discarding the hand, moving the library top to ante, and drawing seven.
- **Darkpact** — needs a target in the shared ante zone, ownership validation/change, and an exact exchange with the library top.
- **Demonic Attorney** — needs ante-enabled legality/removal handling and a reusable each-player top-card-to-ante step.
