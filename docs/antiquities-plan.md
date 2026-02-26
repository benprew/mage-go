# Antiquities XXX Gap Implementation Plan

Red/green TDD: write failing tests first, then implement minimal code to pass, commit after each card.

## Ordering Rationale

Cards are grouped so engine features are built incrementally. Pure card-level fixes come first, then cards sharing an engine prerequisite are clustered together.

---

## ~~Phase 1: Pure Card-Level Fixes (no engine changes)~~ DONE

- ~~Card 1: Obelisk of Undoing~~ — committed
- ~~Card 2: Transmute Artifact~~ — committed
- ~~Card 3: Goblin Artisans~~ — committed (WIP: win-flip draw test needs harness fix for autoPlayLands consuming drawn lands)

### Card 4: Tawnos's Coffin — exile/return Auras

**Tests** (`artifacts_test.go`):
- `TestTawnosCoffinWithAura`: Attach an Aura to a creature, exile creature with Coffin → both creature and Aura should be exiled. When Coffin leaves → creature returns tapped with Aura attached.
- `TestTawnosCoffinWithAuraAndCounters`: Creature has counters AND an Aura → all restored on return.

**Implementation** (`artifacts.go`):
- In the exile effect: before removing the creature, find all Auras attached to it (`g.FilterBattlefield(AttachedTo(target.ID()))`). Exile them alongside the creature (store in `ExiledCard` metadata or as additional `ExiledCard` entries with same `ExiledBy`).
- In the return effect: after putting creature back on battlefield, also return each exiled Aura and re-attach it.

---

## Phase 2: Combat Timing Fixes

### Card 5: Battering Ram — wall destruction at end of combat

**Tests** (`creatures_test.go`):
- `TestBatteringRamWallDestroyed`: Ram attacks, Wall blocks → at EndCombat, Wall is destroyed.
- `TestBatteringRamWallDealsDamage`: Ram attacks, Wall of Swords (3/5) blocks → Wall deals 3 combat damage to Ram (Ram dies from lethal), but Wall is ALSO destroyed at end of combat (both die). This proves the Wall survives through combat damage step.
- `TestBatteringRamNonWallBlock`: Ram attacks, non-Wall creature blocks → no destruction trigger.

**Implementation** (`creatures.go`):
- Change the `EvtDeclaredBlocker` trigger to record the wall's ID, then use a `DelayedTrigger` with `EvtEndCombat` (or similar) to destroy it at end of combat instead of immediately. Alternatively, use a continuous effect with `EndOfCombat` duration that marks the wall for destruction in a cleanup callback.

---

### Card 6: Clockwork Avian — counter removal at end of combat

**Tests** (`creatures_test.go`):
- `TestClockworkAvianAttackRemovesCounter`: Avian (4 +1/+0 counters) attacks → at EndCombat, has 3 counters.
- `TestClockworkAvianBlockRemovesCounter`: Avian blocks → at EndCombat, has 3 counters.
- `TestClockworkAvianNoAttackNoRemoval`: Avian doesn't attack or block → stays at 4 counters after combat.
- `TestClockworkAvianCounterRemovedAfterDamage`: Avian attacks, creature blocks. Verify Avian's power during CombatDamage step still includes the counter (not removed yet). Counter removed at EndCombat.

**Implementation** (`creatures.go`):
- Replace `AttacksTrigger` / `BlocksTrigger` with a flagging mechanism: use `EvtDeclaredAttacker` / `EvtDeclaredBlocker` to set a flag (e.g., stored value or a counter), then an `EvtEndCombat` trigger that checks the flag and removes a +1/+0 counter if set. Clear the flag after.
- Alternative: use a single `EvtBeginCombat` trigger that registers a delayed trigger for `EvtEndCombat` which checks if Avian attacked or blocked.

---

## Phase 3: "Until Your Next Upkeep" Duration

### Engine prerequisite: `UntilYourNextTurn` duration removal

- The constant `UntilYourNextTurn` exists in `core/layer.go` but no removal logic fires at upkeep.
- **Add to game loop** (`game.go` or wherever upkeep is processed): call `g.Effects.RemoveUntilYourNextTurn(activePlayerID)` at the beginning of each upkeep step for the active player.
- **Add method** to `EffectManager`: `RemoveUntilYourNextTurn(playerID)` that removes effects with `UntilYourNextTurn` duration whose controller matches `playerID`.

### Card 7: Xenic Poltergeist — duration "until your next upkeep"

**Tests** (`creatures_test.go`):
- `TestXenicPoltergeistAnimatesArtifact`: Activate on PlayerA's turn targeting a noncreature artifact → artifact becomes a creature with P/T = CMC until PlayerA's next upkeep.
- `TestXenicPoltergeistPersistsThroughOpponentTurn`: Activate on turn 1, check at turn 2 (opponent's turn) → artifact is still animated.
- `TestXenicPoltergeistExpiresAtNextUpkeep`: Activate on turn 1, check at turn 3 (PlayerA's next upkeep) → artifact is no longer a creature.

**Implementation** (`creatures.go`):
- Replace `TemporaryAnimate` (which uses `EndOfTurn`) with a version using `UntilYourNextTurn` duration. Use `TargetEffect(LayerType, UntilYourNextTurn, ...)` with the controller set so the removal logic knows whose upkeep expires it.

---

## Phase 4: Animate with Subtypes (Mishra's Factory)

### Engine prerequisite: `TemporaryAnimateWithSubtypes`

- Extend `temporaryAnimate` in `continuous_effects.go` to accept optional subtypes.
- Add a variant: `TemporaryAnimateWithSubtypes(targetID, power, toughness, subtypes []string)` that also sets subtypes on the animated permanent.
- In the `TargetEffect` apply func, add `target.AddSubTypeOverride(subtype)` or directly manipulate the card's subtypes in the type-changing layer.

### Card 8: Mishra's Factory — Assembly-Worker subtype + pump ability

**Tests** (`lands_test.go` or `creatures_test.go`):
- `TestMishrasFactoryAnimate`: Activate {1} → becomes 2/2 Assembly-Worker artifact creature until end of turn, still a land.
- `TestMishrasFactoryAssemblyWorkerSubtype`: After animating, verify it has "Assembly-Worker" subtype.
- `TestMishrasFactoryPumpAbility`: Animate Factory, then activate {T} pump on itself (or another Assembly-Worker) → target gets +1/+1 until end of turn.
- `TestMishrasFactoryPumpOther`: Have two Factories. Animate both. Use one's pump on the other.

**Implementation** (`lands.go`):
- Change the animate ability to use `TemporaryAnimateWithSubtypes(sourceID, 2, 2, []string{"Assembly-Worker"})`.
- Add a third activated ability: `WithActivatedAbility(BoostUntilEndOfTurn(Fixed(1), Fixed(1), SelectTarget), TapSourceCost(), WithTarget(TargetCreature(HasSubType("Assembly-Worker"))))`.

---

## Phase 5: Non-Tap Ability Activation Trigger

### Engine prerequisite: fire event for non-tap ability activations

- Currently `EvtAbilityActivated` exists in `core/event.go`. Check if it's fired when abilities are activated.
- If not fired or doesn't distinguish tap vs. non-tap: ensure `SimpleActivatedAbility.Resolve()` fires `EvtAbilityActivated` with metadata indicating whether the ability's cost included tapping (check `hasTapCost` flag or similar).
- The event should carry: `SourceID` = the permanent whose ability was activated, `PlayerID` = activating player, `Flag` = whether {T} was part of the cost.

### Card 9: Artifact Possession — add non-tap ability trigger

**Tests** (`enchantments_test.go`):
- `TestArtifactPossessionTapped`: Tap enchanted artifact → 2 damage to controller (already works).
- `TestArtifactPossessionNonTapAbility`: Enchant an artifact with a non-tap activated ability (e.g., Atog's sacrifice ability). Activate Atog's ability → Artifact Possession deals 2 damage to Atog's controller.

**Implementation** (`enchantments.go`):
- Add a second trigger on `EvtAbilityActivated` that checks: (a) the activated permanent is the enchanted artifact, (b) the ability did not include {T} in its cost. If both true, deal 2 damage.

### Card 10: Haunting Wind — add non-tap ability trigger

**Tests** (`enchantments_test.go`):
- `TestHauntingWindTapped`: Any artifact tapped → 1 damage (already works).
- `TestHauntingWindNonTapAbility`: Artifact with non-tap ability activated → 1 damage to that artifact's controller.

**Implementation** (`enchantments.go`):
- Add a second trigger on `EvtAbilityActivated` checking the source is an artifact and the ability didn't include {T}. Deal 1 damage to the artifact's controller.

### Card 11: Powerleech — add non-tap ability trigger

**Tests** (`enchantments_test.go`):
- `TestPowerleechTapped`: Opponent's artifact tapped → gain 1 life (already works).
- `TestPowerleechNonTapAbility`: Opponent activates a non-tap ability on their artifact → gain 1 life.

**Implementation** (`enchantments.go`):
- Add a second trigger on `EvtAbilityActivated` checking the source is an opponent's artifact and the ability didn't include {T}. Gain 1 life.

---

## Phase 6: Artifact Ward — damage prevention + targeting restriction

### Card 12: Artifact Ward

**Tests** (`enchantments_test.go`):
- `TestArtifactWardCantBeBlocked`: (Already works) Enchanted creature can't be blocked by artifact creatures.
- `TestArtifactWardPreventsDamage`: Enchanted creature is dealt damage by an artifact source → damage prevented.
- `TestArtifactWardPreventsCombatDamage`: Artifact creature blocks enchanted creature → combat damage from artifact creature is prevented.
- `TestArtifactWardCantBeTargeted`: Artifact source tries to target enchanted creature with an ability → targeting fails / no effect.

**Implementation** (`enchantments.go`):
- Add `PreventDamageFromTo(IsArtifact, func(sourceID) { return IsAttachedTarget(sourceID) })` as a second static ability.
- Add targeting restriction: continuous effect that makes the attached creature untargetable by artifact sources. This may require adding a `TargetingRestriction` to the effect manager or granting a "hexproof from artifacts" equivalent attr.

---

## Phase 7: Circle of Protection: Artifacts — per-source prevention

### Card 13: Circle of Protection: Artifacts

**Tests** (`enchantments_test.go`):
- `TestCoPArtifactsPreventsDamage`: Activate COP, artifact source deals damage → damage prevented.
- `TestCoPArtifactsNextTimeOnly`: Activate once, two different artifact sources deal damage → only the first instance is prevented, second goes through.
- `TestCoPArtifactsMultipleActivations`: Activate twice → two separate damage instances prevented.

**Implementation** (`enchantments.go`):
- Replace `AddTypePrevention(controller, TypeArtifact)` with a more targeted prevention: `AddNextDamagePrevention(controller, TypeArtifact)` that only prevents the next instance from an artifact source, not all artifact damage for the turn. This may require a new prevention type in the effect manager that tracks "next N instances" rather than blanket prevention.

---

## Phase 8: Energy Flux — per-artifact player choice

### Card 14: Energy Flux

**Tests** (`enchantments_test.go`):
- `TestEnergyFluxPaysForArtifact`: Player has mana and one artifact → pays {2}, artifact survives.
- `TestEnergyFluxSacrificesWhenCantPay`: Player has no mana and one artifact → artifact is sacrificed.
- `TestEnergyFluxPlayerChoosesWhichToKeep`: Player has two artifacts and only {2} mana → one survives, one is sacrificed. (Test that the player gets to choose which one to keep.)

**Implementation** (`enchantments.go`):
- Refactor the trigger to iterate per-artifact, asking the player whether to pay {2} for each. Use `player.ChooseMayAbility("pay {2} for <artifact>")` for each artifact. If yes and can pay → pay. If no or can't → sacrifice.
- Ideally, grant each artifact its own upkeep trigger (as Oracle text states), but batch-processing with per-artifact choice is functionally equivalent.

---

## Phase 9: Titania's Song — loses abilities + persists after leaving

### Engine prerequisite: "loses all abilities" effect

- Add a mechanism to strip abilities from a permanent. Options:
  - Add an `AttrLosesAllAbilities` attr that `RuntimeAbilities()` checks — if set, return empty list.
  - Or add `EffectManager.SilencePermanent(id)` that filters abilities in the abilities lookup.

### Card 15: Titania's Song

**Tests** (`enchantments_test.go`):
- `TestTitaniasSongAnimatesArtifacts`: Noncreature artifacts become creatures with P/T = CMC.
- `TestTitaniasSongLosesAbilities`: Artifact with an activated ability (e.g., Jayemdae Tome's draw ability) → that ability can't be activated while Titania's Song is on the battlefield.
- `TestTitaniasSongDoesntAffectCreatureArtifacts`: Artifact creatures are not affected (they're already creatures).
- `TestTitaniasSongPersistsAfterLeaving`: Destroy Titania's Song → effect continues until end of turn. Artifacts are still animated this turn, revert next turn.

**Implementation** (`enchantments.go`):
- Add "loses all abilities" to the continuous effect (in ability layer, before PT layer).
- For "persists after leaving": when Titania's Song leaves the battlefield, create a temporary continuous effect with `EndOfTurn` duration that does the same animation. This could be a leaves-battlefield trigger that spawns the temporary effect.

---

## Phase 10: Power Artifact — activated ability cost reduction

### Engine prerequisite: per-permanent ability cost reduction

- Need a way for a continuous effect to reduce the mana cost of a specific permanent's activated abilities by {2} (minimum 1 mana).
- Add `EffectManager.SetAbilityCostReduction(permanentID, amount, minMana)` or similar, checked in `SimpleActivatedAbility.CanActivate()` and `Pay()`.

### Card 16: Power Artifact

**Tests** (`enchantments_test.go`):
- `TestPowerArtifactReducesCost`: Enchant an artifact with a {4} activated ability → ability now costs {2} to activate.
- `TestPowerArtifactMinimumOneMana`: Enchant an artifact with a {2} activated ability → ability costs {1} (not {0}, minimum 1 mana).
- `TestPowerArtifactOneManaAbility`: Enchant an artifact with a {1} activated ability → still costs {1} (can't reduce below 1).

**Implementation** (`enchantments.go`):
- Add a continuous effect (ability layer) that sets cost reduction on the enchanted artifact's activated abilities.

---

## Phase 11: Remaining Complex Cards

### Card 17: Tetravus — token creation/absorption

**Tests** (`creatures_test.go`):
- `TestTetravusETB`: Enters with 3 +1/+1 counters and flying. P/T = 4/4.
- `TestTetravusCreateTokens`: At upkeep, remove 2 +1/+1 counters → create 2 Tetravite 1/1 flying artifact creature tokens.
- `TestTetraviteTokenProperties`: Tetravite tokens are 1/1, colorless, artifact creature, have flying, have "can't be enchanted".
- `TestTetravusAbsorbTokens`: At upkeep (after creating tokens earlier), exile 1 Tetravite token → put 1 +1/+1 counter on Tetravus.
- `TestTetravusCreateAndAbsorbSameTurn`: Both triggers fire at beginning of upkeep — can remove counters to create tokens AND absorb tokens in the same upkeep.

**Implementation** (`creatures.go`):
- Add two `BeginningOfUpkeepTrigger` abilities:
  1. **Create**: Ask player how many +1/+1 counters to remove (0 to current count). Remove that many, create that many Tetravite tokens.
  2. **Absorb**: Ask player how many Tetravite tokens to exile (0 to token count). Exile them, add that many +1/+1 counters.
- Register "Tetravite" as a token with flying + "can't be enchanted" (use an attr or enchant-prevention).

### Card 18: Candelabra of Tawnos — X-targeting untap

**Tests** (`artifacts_test.go`):
- `TestCandelabraUntapsLands`: Pay X=2, tap, target 2 tapped lands → both untap.
- `TestCandelabraXZero`: Pay X=0 → no effect (no targets needed).
- `TestCandelabraTargetsOnlyLands`: Can only target lands, not other permanents.

**Implementation** (`artifacts.go`):
- Need X-count targeting: `WithTarget(TargetNPermanents(XValue, IsLand))` or similar. If the engine doesn't support variable-count targeting, this may require a custom effect that finds X tapped lands and untaps them (using player choice if there are more tapped lands than X).

---

## ~~Phase 12: Multiplayer-Only Gaps (Skipped)~~

Cards 19-20 (Cursed Rack & The Rack) work correctly in 2-player. Skipped per user instruction.

---

## Commit Strategy

Each card gets its own commit:
```
feat(antiquities): fix <Card Name> — <brief description>
```

For engine prerequisites, commit them together with the first card that needs them:
```
feat(engine+antiquities): implement <engine feature> for <Card Name>
```

## Progress

- [x] 1. Obelisk of Undoing
- [x] 2. Transmute Artifact
- [x] 3. Goblin Artisans (WIP: win-flip draw test)
- [ ] 4. Tawnos's Coffin (Aura handling)
- [ ] 5. Battering Ram
- [ ] 6. Clockwork Avian
- [ ] 7. Xenic Poltergeist (+ engine: UntilYourNextTurn removal)
- [ ] 8. Mishra's Factory (+ engine: animate with subtypes)
- [ ] 9. Artifact Possession (+ engine: non-tap ability event)
- [ ] 10. Haunting Wind
- [ ] 11. Powerleech
- [ ] 12. Artifact Ward
- [ ] 13. Circle of Protection: Artifacts
- [ ] 14. Energy Flux
- [ ] 15. Titania's Song (+ engine: loses all abilities)
- [ ] 16. Power Artifact (+ engine: ability cost reduction)
- [ ] 17. Tetravus
- [ ] 18. Candelabra of Tawnos
- ~~19. Cursed Rack (multiplayer, skipped)~~
- ~~20. The Rack (multiplayer, skipped)~~
