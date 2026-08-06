# Alpha (LEA) — Remaining Work

This inventory includes unimplemented and partially implemented cards. A reprint is considered implemented when its current Oracle behavior is fully registered by another card package.

## Unimplemented cards

- **Demonic Hordes** — needs an opponent-chooses-permanent effect for the unpaid upkeep branch.
- **Time Vault** — needs a skip-your-next-turn cost/replacement operator.

## Registered cards with Oracle gaps

### Artifacts

- **Forcefield** — models the choice as a target and permits any attacking creature rather than requiring an unblocked creature.
- **Jade Statue** — can be activated outside combat and does not gain the Golem subtype.

### Creatures

- **Personal Incarnation** — controller-only activation is used instead of owner-only activation, a whole damage event is redirected instead of the next 1 damage, and the death trigger uses the controller rather than the owner when control differs.
- **Vesuvan Doppelganger** — the enters choice is modeled as a spell target, copied color is not excluded, and the upkeep ability auto-selects a creature instead of making the optional targeted choice Oracle requires.

### Enchantments

- **Gloom** — activated abilities of white enchantments must cost `{3}` more.
- **Consecrate Land** — the enchanted land must be unable to be enchanted by other Auras.

### Instants and sorceries

- **Guardian Angel** — after the initial X prevention, its controller must be able to pay `{1}` repeatedly to prevent another 1 damage through end of turn.
- **Blue Elemental Blast** — lacks its destroy-permanent mode; modal choices currently cannot declare targets in different zones.
- **Drain Life** — X can use nonblack mana, and life gain is calculated from X rather than the damage actually dealt after prevention or replacement effects.
- **Fireball** — needs any-number targeting, the extra generic cost for each target beyond the first, and evenly divided damage rounded down.
- **Disintegrate** — needs the "if it would die this turn, exile it instead" replacement effect.
- **Sacrifice** — sacrificing a creature is implemented as a targeted resolving effect rather than an additional cost and is not restricted to a creature the caster controls.
- **Natural Selection** — randomly shuffles only the top three cards instead of letting the caster order them and optionally shuffle the full library.
- **Red Elemental Blast** — lacks its destroy-permanent mode; modal choices currently cannot declare targets in different zones.
- **Drain Power** — needs forced activation of each land's mana ability, draining all unspent mana, and adding the mana lost this way.
- **Blaze of Glory** — lacks its cast window, defending-player target restriction, and forced-block-each-attacker behavior.
- **False Orders** — lacks declare-blockers-only timing, the defending-player target restriction, and the complete block reassignment/unblocking choice.
- **Siren's Call** — lacks its opponent-turn/pre-attack cast restriction, attack-if-able requirement, and continuously-controlled exception to delayed destruction.
