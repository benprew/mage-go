  ---
Set Validation Report: Legends

Summary

┌──────────────────────────────┬────────────────────────────────────────┐
│            Metric            │                 Count                  │
├──────────────────────────────┼────────────────────────────────────────┤
│ Total cards in set           │ ~310                                   │
├──────────────────────────────┼────────────────────────────────────────┤
│ Fully implemented (COMPLETE) │ ~267                                   │
├──────────────────────────────┼────────────────────────────────────────┤
│ Partially implemented (XXX)  │ ~30                                    │
├──────────────────────────────┼────────────────────────────────────────┤
│ Stubs (TODO)                 │ ~10                                    │
├──────────────────────────────┼────────────────────────────────────────┤
│ Test coverage                │ ~175 of ~220 testable cards have tests │
├──────────────────────────────┼────────────────────────────────────────┤
│ All tests pass               │ Yes                                    │
├──────────────────────────────┼────────────────────────────────────────┤
│ go vet clean                 │ Yes                                    │
└──────────────────────────────┴────────────────────────────────────────┘

  ---
Stubs (Not Implemented)

Creatures:
- Quarum Trench Gnomes (per-land mana replacement)
- Wood Elemental ("as enters" replacement with variable sacrifice)
- Ayesha Tanaka (counter activated ability from artifact source)

Artifacts:
- Gauntlets of Chaos (control exchange)
- Knowledge Vault (exile tracking + hand swap)
- North Star (mana color flexibility)
- Nova Pentacle (damage redirection choice)
- Ring of Immortals (counter instant/Aura)

  ---
Partial Implementations (XXX Markers)

Creatures (6):
- Firestorm Phoenix — "plays with that card revealed and can't play it until next turn" not enforced
- Wall of Shadows — "can't be the target of spells that can target only Walls" not implemented
- Tetsuo Umezawa — "can't be the target of Aura spells" not implemented
- Shelkin Brownie — "bands with other" approximated as Banding
- Marble Priest — "All Walls able to block do so" forced-block not implemented

Artifacts (1):
- Forethought Amulet — damage cap from instants/sorceries not implemented

Enchantments (15):
- Anti-Magic Aura, Arboria, Caverns of Despair, Chains of Mephistopheles, Divine Intervention, Equinox, Imprison, Land Equilibrium, Land's Edge, Revelation, Sylvan Library, Takklemaggot (partial —
  missing death trigger)

Spells (12):
- All Hallow's Eve, Avoid Fate, Backdraft, Dwarven Song/Enchantment Alteration (Aura migration), Eureka, Falling Star (physical dexterity), Part Water, Psychic Purge (discard trigger), Pyrotechnics (divided
  damage), Juxtapose, Remove Enchantments, Reverberation, Rust, Silhouette, Untamed Wilds, Visions, Winter Blast

Lands (3):
- Adventurers' Guildhouse, Cathedral of Serra, Seafarer's Quay — "bands with other legendary creatures" approximated as Banding

  ---
Oracle Text Violations

HIGH — Incorrect Game Behavior

1. Cosmic Horror — Sacrifice vs. Destroy + missing damage
- Oracle: "At the beginning of your upkeep, destroy this creature unless you pay {3}{B}{B}{B}. If this creature is destroyed this way, it deals 7 damage to you."
- Code: Uses SacrificeAtUpkeepUnlessPay (sacrifice ≠ destroy; regeneration can prevent destruction but not sacrifice). The 7 damage clause is completely absent.

2. Firestorm Phoenix — Triggered ability instead of replacement effect
- Oracle: "If this creature would die, return it to its owner's hand instead."
- Code: Uses EvtCreatureDied (after death), then moves from graveyard. Oracle is a replacement effect — creature should never hit the graveyard. Other "when a creature dies" triggers incorrectly fire.

3. Axelrod Gunnarson — Hits all opponents instead of targeting one
- Oracle: "...you gain 1 life and Axelrod Gunnarson deals 1 damage to target player or planeswalker."
- Code: DealDamageToPlayers(Fixed(1), SelectEachOpponent()) — no targeting, hits all opponents.

4. Boris Devilboon — Token missing colors
- Oracle: "Create a 1/1 black and red Demon creature token named Minor Demon."
- Code: CreateToken("Minor Demon", 1, 1, ...) — no color override. Token has no colors.

5. Master of the Hunt — Token missing color and ability
- Oracle: "Create a 1/1 green Wolf creature token named Wolves of the Hunt. It has 'bands with other creatures named Wolves of the Hunt.'"
- Code: Token has no green color assignment and no "bands with other" ability.

6. Hyperion Blacksmith — Can target own artifacts
- Oracle: "{T}: You may tap or untap target artifact an opponent controls."
- Code: WithTarget(TargetPermanent(IsArtifact)) — targets any artifact, not just opponent's.

7. Mana Drain — Wrong delayed trigger timing
- Oracle: "At the beginning of your next main phase, add {C} equal to that spell's mana value."
- Code: Uses EvtUpkeep — adds mana at upkeep instead of main phase.

8. Urborg — Removes both abilities instead of choosing one
- Oracle: "{T}: Target creature loses first strike or swampwalk until end of turn."
- Code: Creates effects revoking BOTH FirstStrike AND Swampwalk. Should be a choice of one.

9. Voodoo Doll — Missing {X}{X} mana cost on damage ability
- Oracle: "{X}{X}, {T}: This artifact deals damage equal to pin counters... X is the number of pin counters."
- Code: Only uses Tap(). The mana cost is missing — ability is free to activate.

10. Greater Realm of Preservation — Double prevention instead of single choice
- Oracle: "The next time a black or red source of your choice would deal damage to you..."
- Code: Adds TWO prevention shields (one for black, one for red). Should be a single source choice.

11. Triassic Egg — Missing counter condition on sacrifice
- Oracle: "Sacrifice this artifact: Choose one. Activate only if there are two or more hatchling counters."
- Code: No counter check — can activate with zero counters.

MEDIUM — Missing Conditions or Player Agency

12. Enchanted Being — Prevents all damage, not just combat
- Oracle: "Prevent all combat damage that would be dealt to this creature by enchanted creatures."
- Code: Prevents ALL damage (combat + non-combat) from enchanted creatures.

13. Lady Evangela — Same issue: prevents all damage, not just combat
- Oracle: "Prevent all combat damage that would be dealt by target creature this turn."
- Code: Prevents ALL damage from that creature.

14. Horn of Deafening — Same pattern
- Oracle: "Prevent all combat damage that would be dealt by target creature."
- Code: Prevents all damage.

15. Gaseous Form — Same pattern
- Oracle: "Prevent all combat damage that would be dealt to and dealt by enchanted creature."
- Code: Prevents all damage.

16. Demonic Torment — Same pattern
- Oracle: "Prevent all combat damage that would be dealt by enchanted creature."
- Code: Prevents all damage.

17. Subdue — Same pattern
- Oracle: "Prevent all combat damage that would be dealt by target creature this turn."
- Code: Prevents all damage.

18. Marble Priest — Same pattern (prevents all Wall damage, not just combat)

19. Elder Spawn — Forced sacrifice instead of player choice
- Oracle: "Unless you sacrifice an Island" — player should choose.
- Code: Auto-sacrifices first available Island.

20. Mold Demon — Same issue: forced sacrifice
- Oracle: "Unless you sacrifice two Swamps" — player should choose.
- Code: Auto-sacrifices two Swamps if available.

21. Primordial Ooze — "You may pay" is mandatory
- Oracle: "Then you may pay {X}..."
- Code: Auto-pays whenever lands are available.

22. Rohgahh of Kher Keep — Same: "you may pay" is mandatory

23. Lesser Werewolf — Target too broad
- Oracle: "target creature blocking or blocked by this creature"
- Code: Targets any attacking or blocking creature.

24. Sentinel — Same issue: target too broad (any attacker/blocker vs. in-combat-with-self)

25. Ramses Overdark — "Enchanted" check too broad
- Oracle: "Destroy target enchanted creature."
- Code: Checks len(p.Attachments) > 0 — accepts Equipment too, not just enchantments.

26. Kry Shield — Missing "you control" target restriction
- Oracle: "target creature you control"
- Code: Can target any creature.

27. Glyph of Reincarnation — Missing "can't be regenerated"
- Oracle: "Destroy all creatures that were blocked by target Wall this turn. They can't be regenerated."
- Code: Uses plain DestroyPermanent without preventing regeneration.

28. Recall — Graveyard retrieval is not player-chosen

29. Nicol Bolas — Triggers on any player damage, not just opponent

LOW — Timing/Edge Cases

30. Clergy of the Holy Nimbus — Controller can activate opponent-only ability (WithAnyPlayerMay instead of opponents-only)

31. Angus Mackenzie — Missing "only before combat damage step" timing restriction

32. Gwendlyn Di Corci — Missing "only during your turn" timing restriction

33. Rasputin Dreamweaver — "Started the turn untapped" checked at upkeep, not at turn start

34. Cocoon — Untap prevention is unconditional (should only prevent while pupa counters remain)

35. Hellfire — Counts creatures sent to DestroyPermanent even if they regenerate

36. The Tabernacle at Pendrell Vale — Trigger on land rather than granted to each creature (removing Tabernacle stops the effect immediately instead of letting already-triggered abilities resolve)

  ---
Missing Tests

Creatures needing triggered/activated ability tests (no tests found):
- Abomination, Blazing Effigy, Cosmic Horror, Aisling Leprechaun, Floral Spuzzem, Giant Turtle, Willow Satyr, Axelrod Gunnarson, Gabriel Angelfire, Gosta Dirk, Lady Caleria, Lady Evangela, Princess Lucrezia,
  Ragnar, Sentinel, Marble Priest, Hyperion Blacksmith, The Lady of the Mountain

Artifacts needing tests:
- Forethought Amulet, Sword of the Ages (complex exile+damage), Triassic Egg (counter-gated modes)

Enchantments needing tests:
- Horror of Horrors, Invoke Prejudice, Living Plane (has test but limited), Puppet Master (limited), Spirit Shackle

Spells needing tests:
- Reset, Sea Kings' Blessing, Alabaster Potion

  ---
Weak Tests

- Mana Drain — Tests the counter but the delayed mana trigger test (TestManaDrainDelayedMana) doesn't verify the mana arrives at main phase (it arrives at upkeep per the bug)
- Cocoon — Tests counter addition but doesn't verify the creature can untap after counters are removed
- Blood Lust — Tests toughness >= 5 case but doesn't test the "if toughness is less than 5" branch behavior

  ---
Engine Feature Gaps

These are the missing engine capabilities blocking multiple cards:

┌────────────────────────────────────────────┬───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                  Feature                   │                                                   Blocked Cards                                                   │
├────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ "Combat damage" prevention (vs all damage) │ Enchanted Being, Lady Evangela, Horn of Deafening, Gaseous Form, Demonic Torment, Subdue, Marble Priest (7 cards) │
├────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Draw replacement effects                   │ Chains of Mephistopheles, Sylvan Library, Caverns of Despair (3 cards)                                            │
├────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Revealed hand/library top                  │ Revelation                                                                                                         │
├────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
├────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Aura re-attachment/migration               │ Enchantment Alteration, Dwarven Song, Takklemaggot (3 cards)                                                      │
├────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ X-count targeting                          │ Part Water, Winter Blast, Pyrotechnics (3 cards)                                                                  │
├────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ "As enters" replacement                    │ Wood Elemental (1 card)                                                                                           │
├────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Counter activated abilities                │ Ayesha Tanaka, Ring of Immortals, Rust (3 cards)                                                                  │
├────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Per-land mana replacement                  │ Quarum Trench Gnomes (1 card)                                                                                     │
├────────────────────────────────────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Modal triggered abilities                  │ Relic Bind, Quagmire (2 cards)                                                                                    │
└────────────────────────────────────────────┴───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

  ---
Suggested Next Steps

1. Quick Wins (< 30 min each)

- Fix Boris Devilboon + Master of the Hunt token colors (add WithColor)
- Fix Hyperion Blacksmith target filter (add ControlledByOpponent)
- Fix Kry Shield target filter (add ControlledBySelf)
- Fix Urborg to be a mode choice instead of both
- Fix Ramses Overdark to check for Aura attachments specifically
- Fix Glyph of Reincarnation to prevent regeneration
- Fix Nicol Bolas trigger to check opponent only

2. Systemic Fix: Combat Damage Prevention

Adding a WithCombatOnly flag to AddDamagePreventionRule would fix 7 cards at once (Enchanted Being, Lady Evangela, Horn of Deafening, Gaseous Form, Demonic Torment, Subdue, Marble Priest).

3. Test Gaps

Write tests for the 18+ creatures with non-trivial abilities that lack tests, prioritizing cards with known Oracle violations.

4. "You May Pay" Pattern

Several cards auto-pay costs when Oracle says "you may pay." Consider a g.OfferPayCost() helper that asks the player. Affects: Elder Spawn, Mold Demon, Primordial Ooze, Rohgahh of Kher Keep.

5. Engine Work (by impact)

1. Combat-only damage prevention — 7 cards
2. X-count targeting — 3 cards
3. Counter abilities on stack — 3 cards
4. Draw replacement effects — 3 cards
5. Aura migration — 3 cards
6. Card naming — 2 cards

6. Hard Problems

- Firestorm Phoenix replacement effect (needs "would die" replacement, not death trigger)
- Cosmic Horror destroy-not-sacrifice + conditional self-damage
- Voodoo Doll dynamic mana cost based on counters
- Triassic Egg counter-gated activation
- The Tabernacle granting abilities vs. land-based trigger
