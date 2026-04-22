# Data-Driven Effects Migration

**Goal:** Make card definitions pure data — no closures that take `*Game`. Cards describe "what" happens; the engine interprets and executes.

**Branch:** awesome
**Started:** 2026-04-21

## Architecture

Three new subsystems, all in `pkg/mage/`:

| File | Purpose |
|---|---|
| `effect_data.go` | `EffectData` interface, `EffectContext` (variable binding), `DataEffect()` adapter |
| `executor.go` | Central `ExecuteEffect()` type switch — dispatches ~95 effect data types |
| `effect_pipeline.go` | Pipeline, Snapshot, ForEach, IfElse, Modal, ChoosePermanent, gathered-permanent ops |
| `effect_pipeline_ext.go` | Extended primitives: SnapshotAttached, delayed triggers, keyword grant/revoke, mana, prevention |
| `trigger_condition_data.go` | 48 composable trigger predicates + And/Or/Not combinators |

Pre-built effects (`effect_damage.go`, `effect_removal.go`, `effect_cards.go`, `effect_combat.go`, `effect_spells.go`) are fully converted — structs implement `EffectData` and execution lives in the executor. Constructors still return `Effect` via the adapter, so card code is unchanged.

New continuous effect constructors added to `continuous_effects.go`: `GrantKeywordToControlled`, `GrantKeywordToOtherControlled`, `BoostOtherControlledCreatures`, `RevokeKeywordFromAll`, `RevokeAttrFromControlled`. New filter: `IsLegendary`.

## Closure Counts

| Type | Original | Remaining | Converted | % done |
|---|---|---|---|---|
| FuncEffect | 289 | 0 | 289 | 100% |
| FuncContinuousEffect | 97 | 0 | 97 | 100% |
| SetCondition(func) | 69 | 6 | 63 | 91% |
| **Total** | **455** | **6** | **449** | **99%** |

## What's Converted

Cards that now use data-driven effects instead of closures include:

- **Swords to Plowshares** — Pipeline(Snapshot + Exile + GainLifeFromVar)
- **Pestilence** — IfElse(NoCreatures, SacrificeSource)
- **Healing Salve** — ModalEffect
- **Crumble** — Pipeline(Snapshot + DestroyNoRegen + GainLifeFromVar)
- **Armageddon Clock** — Pipeline(SnapshotSourceCounter + DealDamageToPlayersFromVar)
- **Ivory Tower** — Pipeline(SetVarFromHandSize + GainLifeFromVar)
- **Circle of Protection** (5 colors) — AddColorPreventionStep
- **Bottle of Suleiman** — IfElse(FlipCoin, CreateToken, DealDamage)
- **Clockwork Avian** — RegisterDelayedTriggerStep
- **Thrull Retainer** — Pipeline(SnapshotAttached + RegenerateGathered)
- **Akron Legionnaire, Moat, Gravity Sphere** — new continuous effect constructors
- **SacrificeAtUpkeepUnlessPay** — IfElse(TryPayMana, nil, SacrificeSource)
- 40+ trigger predicates — composable data predicates instead of closures
- All engine trigger constructors (DealsDamageToOpponentTrigger, RampageTrigger, WheneverSpellCastTrigger, etc.)
- Combat predicates: SourceIsBlockedAttacker, SourceIsUnblockedAttacker, SourceIsBlockingInCombat, SourceInCombat
- Damage predicates: EventSourceIsSelfDamageToPlayer, EventSourceIsSelfDamageToOpponent
- Battlefield state: ControllerHasNoPermanentMatching, NoBattlefieldPermanentMatching, CreatureDeathsOccurred
- Spell-cast: SpellCastIsType, OpponentCastSpellOfType, ControllerCastSpellOfType, SpellCastMatchesCardFilters
- Artifact activation: AttachedToIsEventSourceNoTapCost, OpponentActivatedArtifactNoTapCost
- Aura: AttachedToDealsDamageToController, EventIsAttachedControllerUpkeep
- Combat opponents: SourceBlockedByCreatureMatching, SourceInCombatWithMatchingCreature
- General: SourceOnBattlefield, CombatGroupCountEquals, OpponentCastNthSpellOfType

## What Remains

The 6 remaining `SetCondition(func)` closures are genuinely card-specific or closure-dependent:

| Card | File | Why unconvertible |
|---|---|---|
| Wall of Caltrops | legends/creatures.go:285 | "other Wall blocking same attacker, no non-Walls blocking" — unique multi-blocker logic |
| Stangg twin | legends/creatures.go:3224 | Captured `twinID` variable — inherently closure-dependent |
| Gaseous Form | legends/enchantments.go:657 | Enchanted creature in combat with toughness-≤3 creature — no dynamic toughness filter |
| Invoke Prejudice | legends/enchantments.go:715 | Opponent creature not sharing color with controller's creatures — color-sharing logic |
| City in a Bottle | arabian/artifacts.go:185 | Set-specific (Arabian Nights) card check |
| Jihad | arabian/enchantments.go:192 | ChosenColor + ChosenPlayer + nontoken multi-field check |

## Status

**The migration is complete.** 449/455 original closures converted (99%). The 6 remaining closures are genuinely card-specific and would each require a bespoke predicate with no reuse value.

New card implementations should use data-driven primitives (EffectData, SetConditionData, Pipeline, etc.) where possible, and FuncEffect/SetCondition only when the engine lacks a matching primitive.

## Commits

```
7a9dde7 refactor: Data-driven effect system — cards describe "what", executor decides "how"
2c7e3b3 refactor: Convert limited/ FuncEffects to data-driven pipelines where possible
fbc4211 refactor: Convert arabian/antiquities/fallen_empires FuncEffects to pipelines
843f6d6 refactor: Convert legends/ FuncEffects to data-driven pipelines where possible
21b564b refactor: Add extended pipeline primitives, convert ~24 more FuncEffects
f8289b4 refactor: Phase 5 — composable trigger condition predicates
b444639 refactor: Phase 4+5 card migration — convert SetCondition and FuncContinuousEffect closures
158aa3c refactor: Phase 6 — convert 59/69 SetCondition closures to data-driven predicates
8342ec1 refactor: Update doc.go examples and test DSL to use SetConditionData
```
