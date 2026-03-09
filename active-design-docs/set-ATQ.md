 ---                                                                                                                                                                                                                   
Set Validation Report: Antiquities (ATQ)

Summary

- Total unique cards in set: 85 (non-reprint)
- Registered in engine: 83
- Fully implemented (COMPLETE): 66
- Partially implemented (XXX): 15
- Stubs (TODO/empty): 2
- Not registered: 2 (Bronze Tablet — ante; Golgothian Sylex — metatextual)
- Test coverage: 83 of ~65 testable cards have tests (all testable cards covered)
- All 143 test cases PASS
- go vet clean

Stubs (Not Implemented)

Artifacts:
Candelabra of Tawnos — XXX: X-targeting for untap lands (engine gap)

Enchantments:
Power Artifact — XXX: cost reduction for attached artifact (engine gap)

Not Registered (2 cards)

    Bronze Tablet — ante card (out of scope)
    Golgothian Sylex — references "originally printed in Antiquities" (metatextual, out of scope)

Partial Implementations (XXX Markers)

Duration/Timing Issues:

┌───────────────────┬───────────────────────────────────────────────┬───────────────────────────────────────────────────────┬───────────────────────────────────────────────────┐
│       Card        │                  Oracle says                  │                       Code does                       │                       Issue                       │
├───────────────────┼───────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────┤
│ Xenic Poltergeist │ "Until your next upkeep"                      │ TemporaryAnimate (end of turn)                        │ Wrong duration — effect ends too early            │
├───────────────────┼───────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────┤
│ Battering Ram     │ "destroy that Wall at end of combat"          │ g.DestroyPermanent(wall) immediately on block trigger │ Destroys Wall immediately instead of delayed      │
├───────────────────┼───────────────────────────────────────────────┼───────────────────────────────────────────────────────┼───────────────────────────────────────────────────┤
│ Clockwork Avian   │ "At end of combat, if it attacked or blocked" │ AttacksTrigger / BlocksTrigger (on declaration)       │ Counter removed at declaration, not end of combat │
└───────────────────┴───────────────────────────────────────────────┴───────────────────────────────────────────────────────┴───────────────────────────────────────────────────┘

Missing Token Generation:

┌──────────┬────────────────────────────────────────────────────────────────────────────────────────────────────────────┬───────────────────────────────────────────────────────────┐
│   Card   │                                                Oracle says                                                 │                         Code does                         │
├──────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────────────┤
│ Tetravus │ Upkeep: remove +1/+1 counters → create 1/1 Tetravite tokens with flying; also absorb tokens → add counters │ Only ETB with 3 counters + flying; no token create/absorb │
└──────────┴────────────────────────────────────────────────────────────────────────────────────────────────────────────┴───────────────────────────────────────────────────────────┘

Missing Abilities on Mishra's Factory:

┌────────────────────────────────────────────────────┬─────────────────────────────────┐
│                   Oracle ability                   │             Status              │
├────────────────────────────────────────────────────┼─────────────────────────────────┤
│ {T}: Add {C}                                       │ ✓                               │
├────────────────────────────────────────────────────┼─────────────────────────────────┤
│ {1}: Becomes 2/2 Assembly-Worker artifact creature │ Missing Assembly-Worker subtype │
├────────────────────────────────────────────────────┼─────────────────────────────────┤
│ {T}: Target Assembly-Worker gets +1/+1 until EOT   │ Missing entirely                │
└────────────────────────────────────────────────────┴─────────────────────────────────┘

Missing Triggered Ability Component (3 cards):

┌─────────────────────┬─────────────────────────────────────────────────────────────────┬──────────────────────┐
│        Card         │                           Oracle says                           │      Code does       │
├─────────────────────┼─────────────────────────────────────────────────────────────────┼──────────────────────┤
│ Artifact Possession │ Triggers on tap or ability activation without {T} cost          │ Only triggers on tap │
├─────────────────────┼─────────────────────────────────────────────────────────────────┼──────────────────────┤
│ Haunting Wind       │ Triggers on tap or ability activation without {T} cost          │ Only triggers on tap │
├─────────────────────┼─────────────────────────────────────────────────────────────────┼──────────────────────┤
│ Powerleech          │ Triggers on opponent tap or ability activation without {T} cost │ Only triggers on tap │
└─────────────────────┴─────────────────────────────────────────────────────────────────┴──────────────────────┘

Simplified Enchantment Effects:

┌────────────────────────────┬────────────────────────────────────────────────────────────────────────────────────────────────┬───────────────────────────────────────────────────┬───────────────────────────────┐
│            Card            │                                          Oracle says                                           │                     Code does                     │             Issue             │
├────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────┼───────────────────────────────┤
│ Artifact Ward              │ 3 abilities: can't be blocked by artifact creatures, prevent artifact damage, can't be         │ Only implements blocking restriction              │ 2 of 3 abilities missing      │
│                            │ targeted by artifact abilities                                                                 │                                                   │                               │
├────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────┼───────────────────────────────┤
│ Circle of Protection:      │ "next time an artifact source of your choice" — single chosen source                           │ AddTypePrevention prevents ALL artifact damage    │ Prevents too much damage      │
│ Artifacts                  │                                                                                                │                                                   │                               │
├────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────┼───────────────────────────────┤
│ Energy Flux                │ Per-artifact "sacrifice unless you pay {2}" choice                                             │ Auto-pays sequentially from mana pool             │ No player agency per artifact │
├────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────┼───────────────────────────────┤
│ Titania's Song             │ Noncreature artifacts "lose all abilities"; effect continues until EOT after Song leaves       │ Animates but doesn't remove abilities; no         │ Missing ability removal +     │
│                            │                                                                                                │ lingering effect                                  │ lingering                     │
├────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────┼───────────────────────────────────────────────────┼───────────────────────────────┤
│ Tawnos's Coffin            │ "Exile target creature and all Auras attached"; return Auras attached to creature              │ Does not exile or return Auras                    │ Aura handling missing         │
└────────────────────────────┴────────────────────────────────────────────────────────────────────────────────────────────────┴───────────────────────────────────────────────────┴───────────────────────────────┘

2-Player Simplifications (Acceptable):

┌─────────────┬────────────────────────────────────────────────────────────────────┐
│    Card     │                               Issue                                │
├─────────────┼────────────────────────────────────────────────────────────────────┤
│ Cursed Rack │ No ETB opponent choice; auto-picks opponent (correct for 2-player) │
├─────────────┼────────────────────────────────────────────────────────────────────┤
│ The Rack    │ No ETB opponent choice; auto-picks opponent (correct for 2-player) │
└─────────────┴────────────────────────────────────────────────────────────────────┘

Oracle Text Violations (Non-XXX-Marked)

1. Ashnod's Battle Gear — Oracle: "Target creature you control" — code uses TargetCreature() without the ControlledBy filter, allowing targeting opponent creatures.

Missing Tests

All testable cards have tests. No cards are missing test coverage.

Weak Tests

┌─────────────────────┬────────────────────────────────────────────┬─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│        Card         │               What's tested                │                                                               What's missing                                                                │
├─────────────────────┼────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Goblin Artisans     │ Lose-flip counter test, stat check         │ Win-flip draw test commented out (XXX: drawn card consumed by autoPlayLands)                                                                │
├─────────────────────┼────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Martyrs of Korlis   │ Redirect while untapped, non-artifact      │ "Does not redirect when tapped" test is broken — Martyrs untaps before turn 4, so the test actually verifies redirect works (same assertion │
│                     │ negative                                   │  as positive case)                                                                                                                          │
├─────────────────────┼────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Ashnod's            │ Counter + artifact type added              │ No negative case — doesn't verify target must be nonartifact                                                                                │
│ Transmogrant        │                                            │                                                                                                                                             │
├─────────────────────┼────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Tawnos's Wand       │ Unblockable when power ≤ 2                 │ No negative case — doesn't test refusal when power > 2                                                                                      │
├─────────────────────┼────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Jalum Tome          │ Taps, draws and discards                   │ Doesn't verify net hand size or specific discard                                                                                            │
├─────────────────────┼────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Urza's Miter        │ Non-sacrifice trigger + sacrifice          │ Doesn't verify card was actually drawn (only checks Ornithopter died)                                                                       │
│                     │ non-trigger                                │                                                                                                                                             │
├─────────────────────┼────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Primal Clay         │ 2/2 flying mode tested                     │ Missing test for default 3/3 mode and 1/6 Wall/defender mode                                                                                │
├─────────────────────┼────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Dwarven Weaponsmith │ Upkeep-only activation + counter           │ No negative case — doesn't test that activation fails outside upkeep                                                                        │
└─────────────────────┴────────────────────────────────────────────┴─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

Test Failures

None — all 143 test cases pass.

Engine Feature Gaps

┌─────────────────────────────────────────────┬────────────────────────────────────────────────┐
│               Feature needed                │                 Cards blocked                  │
├─────────────────────────────────────────────┼────────────────────────────────────────────────┤
│ "Until your next upkeep" duration           │ Xenic Poltergeist                              │
├─────────────────────────────────────────────┼────────────────────────────────────────────────┤
│ "At end of combat" delayed trigger          │ Battering Ram, Clockwork Avian                 │
├─────────────────────────────────────────────┼────────────────────────────────────────────────┤
│ Token creation from counters                │ Tetravus                                       │
├─────────────────────────────────────────────┼────────────────────────────────────────────────┤
│ X-targeting (target X permanents)           │ Candelabra of Tawnos                           │
├─────────────────────────────────────────────┼────────────────────────────────────────────────┤
│ Activated ability cost reduction (attached) │ Power Artifact                                 │
├─────────────────────────────────────────────┼────────────────────────────────────────────────┤
│ "Ability activated without {T}" event       │ Artifact Possession, Haunting Wind, Powerleech │
├─────────────────────────────────────────────┼────────────────────────────────────────────────┤
│ "Loses all abilities" continuous effect     │ Titania's Song                                 │
├─────────────────────────────────────────────┼────────────────────────────────────────────────┤
│ Exile/return attached Auras                 │ Tawnos's Coffin                                │
├─────────────────────────────────────────────┼────────────────────────────────────────────────┤
│ Per-source damage prevention choice         │ Circle of Protection: Artifacts                │
└─────────────────────────────────────────────┴────────────────────────────────────────────────┘

  ---
Suggested Next Steps

1. Quick Wins (code-only fixes, no engine work):
- Fix Ashnod's Battle Gear target to TargetCreature(ControlledBy(controller)) — one-line fix
- Add Assembly-Worker subtype to Mishra's Factory animate effect
- Fix Martyrs of Korlis tapped test (it currently passes for the wrong reason)

2. Test Gaps:
- Add Primal Clay 3/3 default and 1/6 Wall mode tests
- Add Tawnos's Wand negative case (power > 2 rejection)
- Add Ashnod's Transmogrant negative case (can't target artifact creature)
- Add Dwarven Weaponsmith outside-upkeep negative case
- Fix Goblin Artisans win-flip draw test (harness timing issue)

3. Oracle Fixes (medium effort):
- Add Mishra's Factory third ability ({T}: +1/+1 to Assembly-Worker)
- Fix Battering Ram Wall destruction to use delayed trigger at end of combat
- Fix Clockwork Avian counter removal to trigger at end of combat instead of on declaration

4. Engine Work (grouped by feature, with impact):
- "At end of combat" delayed triggers (2 cards: Battering Ram, Clockwork Avian)
- "Until your next upkeep" duration (1 card: Xenic Poltergeist)
- EvtAbilityActivated event for non-tap abilities (3 cards: Artifact Possession, Haunting Wind, Powerleech)
- Token creation (1 card: Tetravus)
- "Loses all abilities" continuous effect (1 card: Titania's Song)
- X-targeting (1 card: Candelabra of Tawnos)
- Ability cost reduction (1 card: Power Artifact)

5. Hard Problems:
- Circle of Protection: Artifacts needs a "choose a source, prevent next damage from that source" mechanic — different from current type-based prevention
- Energy Flux needs per-artifact player choice during upkeep
- Tawnos's Coffin Aura tracking requires exile zone metadata enhancement