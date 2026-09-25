# Composable DSL Outstanding Implementation Work

This document tracks active, incomplete implementation items, engine blockers, and acceptance criteria for the composable DSL migration. All completed items and historical slice records have been archived in git history.

The architectural specification is in [composable-dsl-design.md](composable-dsl-design.md), and authoring reference is in [dsl-reference.md](../dsl-reference.md).

## Outstanding Card Migrations

- [ ] **Icatian Moneychanger**: Audit entry damage source identity. (Sacrifice activation with credit counter capture is complete).
- [ ] **Spore Creatures**: Migrate the five spore creatures' remaining activated definitions after token color, damage-source LKI, and sacrifice-cost audits. Elvish Farmer and Thallid Devourer have pre-existing omitted Saproling-sacrifice abilities; record exact implementations separately from upkeep migration.
- [ ] **Whirling Dervish**: Audit end-step timing, per-turn opponent-damage condition, and source identity before retiring `DealsDamageToOpponentTrigger`. (It is the remaining production caller; Nicol Bolas is already migrated).
- [ ] **Wheel of Fortune & Contract from Below**: Migrate before retiring the legacy `DiscardHand` selector surface. Audit simultaneous multi-player hand discards and the remaining draw/ante definitions. (Singular typed hand discard is available as `DiscardEntireHand(subject)`).
- [ ] **Modal Cards with Graveyard References**: Add typed graveyard references and remaining modal effects. Trusty Retriever, Entomber Exarch, and Biblioplex Tomekeeper retain their existing modal builders until their remaining effects have exact adapters.
- [ ] **Deep Spawn**: Audit optional upkeep mill payment before migrating the last production `MillTargetPlayer` caller. (Selhoff Occultist, Towering-Wave Mystic, Millstone, and Whimsy's mill branch are migrated; `MillstoneEffect` is retired).
- [ ] **Cauldron of Essence**: Migrate reanimation ability after typed graveyard references are available. Preserve target selection before sacrifice payment, graveyard ownership, sorcery timing, and return under resolving controller. (Death trigger is migrated; `AnyCreatureDiesTrigger` is retired).
- [ ] **First Main Phase Callers**: Migrate other first-main callers before retiring `BeginningOfFirstMainPhaseTrigger`: Scathing Shadelock needs typed prepared state and prepared casting audit; Abstract Paintmage needs missing instant/sorcery mana spending restriction.
- [ ] **Surrounding Card Definitions**: Migrate remaining surrounding definitions for Blindblast, Exclude, Thirst for Knowledge, and Rhystic Study. Damage-source LKI, stack references, hidden-zone choices, and event participants still need exact adapters.
- [ ] **Rhystic Study**: Audit optional draw and payer choice order when migrating the trigger. Legacy payment branches expose nested typed effects to inspection/validation; event participant capture remains open.
- [ ] **Elemental Uprising**: Audit land animation, target-controller restriction, color behavior, and lifetime.
- [ ] **Neyith of the Dire Hunt**: Audit combat trigger, optional hybrid payment, power doubling, and end-of-combat blocking requirement before removing `TargetMustBeBlockedIfAble`.
- [ ] **Glorious Decay**: Audit damage mode and graveyard target before retiring `DestroyTargetArtifact`.
- [ ] **Positional Return Helpers**: Continue positional return helper migrations. Witch Hunter's separate damage ability awaits damage-source LKI. Rakalite's delayed source identity audit and migration are complete; audit Prismari Charm's modes before retiring `ReturnToHandTarget`.
- [ ] **String-Bound Action Migrations**: Continue destruction, return, tap-or-untap, attachment, and other string-bound action migrations where exact typed forms exist.

## Engine Blockers

### Combat Removal
- [x] Preserve blocked status when the final blocker is removed. Correct the shared combat operation after a failing regression; keep explicit-unblocking legacy effects separate. See the typed combat-removal slice below.
- [ ] Audit explicit unblocking and historical sole-blocker tracking for False Orders and Ydwen Efreet before retiring their legacy removal helpers.

### Zone LKI Retention
- [ ] Audit zone LKI retention after final cleanup and damage source/recipient incarnation before claiming exact damage-condition adapters. The legacy card-ID cache still resets on final cleanup and can be replaced by a later departure of the same card. Typed counter and attachment histories have their own durable incarnation records; general damage references do not.

### Mana-Cost Representation
- [ ] Support monocolored hybrid symbols in the shared parser and payment path. `ParseManaCost("{2/W}").CMC()` currently returns 0 instead of 2. The typed mana-value adapter reuses this service; it does not add symbol support.

### Prepared Spell Payment
- [ ] Resolve prepared spell mana payment before migrating Infirmary Healer's prepared Stream of Life factory. `WithPreparedSpell` installs a zero-cost activated ability, and `CastPreparedSpellCopy` pushes a copy without paying its mana cost. The current X=0 test supplies no mana; a failing cost regression is still required.
- [ ] Test the actual prepared casting entry path: {G} at X=0, {4}{G} at X=4, failed payment and target rollback, stored X, and copied targets.

### Damage-Source Identity & LKI
- [ ] Correct unprevented planeswalker damage outcomes separately: `DamageSystem.ExecuteDamageToCreature`, also used for planeswalker recipients, marks damage instead of removing loyalty. Rakalite target eligibility and full prevention work through the existing pipeline; this migration does not claim to resolve downstream unprevented damage outcomes.
- [ ] Retain source incarnation and last-known characteristics through the existing damage and replacement pipeline, including death, return, and copying. `DamageSystem.ExecuteDamageToPlayer` and `ExecuteDamageToCreature` use `FindPermanent(sourceID)` for lifelink and deathtouch. Protection in `DealDamageToPermanent` also looks up the physical card ID; `LookupObject` prefers a live object with that ID. Add regressions for returned sources and departed sources with granted keywords before adding the typed adapter.

### Destruction Regeneration Scope
- [x] Preserve turn-long regeneration prohibitions through ability removal. `CantRegenerate` is a rules restriction, not a removable ability; typed duration metadata reports a detrimental effect. See the Hurr Jackal slice below.
- [x] Resolve repeated destruction and intermediate state observations in the Disk helper. One union collection now uses the existing simultaneous destruction executor.
- [x] Keep the collection no-regeneration prohibition on each destruction action; preserve permanent characteristics and simultaneous death snapshots.
- [x] Correct singular typed, positional, step, and string-bound no-regeneration execution to use the action-scoped prohibition. Migrate Terror; see the completed slice below.
- [ ] Migrate remaining no-regeneration callers before retiring positional exports: Abu Ja'far needs combat-opponent capture and simultaneous destruction; Detonate needs X-constrained targets and damage-source LKI; Aswan Jaguar needs a chosen-subtype target adapter. Crumble's saved-value migration is complete and the string-bound no-regeneration constructor is retired.

### Temporary Blocking Restrictions
- [x] Preserve resolved blocking prohibitions through ability removal and source departure. Use the existing combat-restriction storage, keeping printed unblockable abilities removable. `TestBlockingProhibitionSurvivesAbilityRemoval` failed before the fix.
- [x] Migrate Dwarven Warriors and Suspicious Bookcase to typed targets, activation costs, and `Until(EndOfTurn, CantBeBlocked(creature))`. See the activated blocking slice below. No engine gap.
- [x] Migrate Teleport to typed `Spell`, `OnlyDuring(DeclareAttackers)`, and `Until(EndOfTurn, CantBeBlocked(creature))`. Enforce the casting restriction in normal, alternate, free, and card-copy casting paths and spell availability; retire `MakeUnblockableUntilEndOfTurn`. See the Teleport slice below.

### Untap Skipping
- [ ] Audit Goblin Rock Sled's last-turn attack condition, then migrate its attack trigger before retiring `StunCreature`. Deep Spawn's optional upkeep mill payment remains open under the milling audit.

### Life Gain & Payment Boundaries
- [x] Migrate the nine legacy replacement registration calls in `fasting_step_test.go`, `fasting_controller_test.go`, and `replacement_life_outcome_test.go`; retire `AddFastingReplacement`. Six integration registrations now use `Static(SkipDrawStepForLife(2))`. Three explicit lifetime registrations use private executor records, preserving absent incarnation, phasing, leave-and-return, and clone assertions. No engine gap or production card migration remains for this helper.
- [ ] Audit card callback `PlayerGainLife` callers and direct `GainLife` mutations for missing events. The low-level API intentionally retains its no-event contract; callers that already emit events must not receive duplicates. The singular DSL and string-variable executors report actual outcomes through one shared event path.
- [ ] Audit life-gain replacement ordering and recipient/source identity. Legacy amount modifiers still run before the general replacement pipeline.
- [ ] Establish simultaneous collection gain and replacement handling, including events that must observe all final life totals.

### Mill Replacement Semantics
- [ ] Integrate legacy mill modifiers with replacement choice and rechecks. Explicit registrations run first, followed by static abilities in battlefield and ability order. General replacement ordering remains unsupported.
- [ ] Define once-per-action replacement identities and actual mill results, including general zone replacements and events, before adding typed rules.

### Event & Delayed Capture
- [ ] General future event capture and general delayed events/reflexive triggers remain open.
- [x] Resolve the Blight attachment-tap audit: event-time permanent capture uses existing trigger records; `TapPermanent` now emits events only on a tap transition.
- [ ] Continue attachment-tap audits for Psychic Venom, Kudzu, Artifact Possession, and Relic Bind, plus remaining regeneration callers. Damage identity, attachment manipulation, and modal/player-or-planeswalker targeting remain separate work.

## Remaining DSL Families (Architectural Scope)

| Family | Outstanding Work |
| :--- | :--- |
| **Bindings** | Multiple variable target groups; card and stack references; other saved-value domains; divided-value declarations; explicit branch results. |
| **Actions** | Typed damage sources with LKI; payment/action result bindings and result amounts; typed token definitions; remaining action families and legacy callers. |
| **Milling** | General zone-change replacements, events, result bindings, and replacement ordering beyond the legacy amount modifier and direct player-zone updates. |
| **Choices & Hidden Zones** | Card choices; search, reveal, and order bindings; zone-lifetime references; visibility contracts; scry/surveil result bindings and surveil zone-change events/replacements. |
| **Continuous Effects** | Remaining modifications and durations; complete compound-effect identity; cross-layer condition continuity; dependency loops and audit; general multi-keyword summary behavior. |
| **Triggers** | Event participant bindings and other event patterns; event-based and reflexive delayed capture; linked state; legacy caller migrations. |
| **Spells & Activations** | Remaining legacy builders and callers; multi-mode selection; remaining casting and copy variants. |
| **Mana Abilities** | Dynamic amount planning; conditional, modal, and loop-based classification; automatic planning for mixed bodies and abilities without tap costs; fixed and string-bound legacy callers. |
| **Replacements** | General typed replacement/prevention rules and outcomes; retirement of legacy replacement registration test callers. |
| **Life Collections** | Collection life gain and its event/replacement audit; multiplayer team selection. |
| **Author Boundary** | Mutable aliases; callback declarations and cost conditions; remaining direct engine imports; obsolete author exports. |

## Legacy Helper Audits

- [ ] Migrate the remaining `ExileTargetStep` caller before retiring that export. (`ExileTarget` is retired from both author surfaces; internal executor remains in use by typed `Exile` and legacy step).
- [ ] Migrate Hazezon Tamar's `ExileTargetStep` loop and the string-bound `ExileGathered` callers in Tawnos's Coffin and Emeritus of Truce's Swords to Plowshares spell. Preserve timing, identity, and linked state. Audit graveyard-source exile separately.
- [ ] Audit remaining targeted, event-player, and string-variable life helpers before removing `GainLifeTarget`. The typed gain adapter still uses its executor.
- [ ] Migrate remaining UUID-based subtype helpers and positional destruction, return, modal, spell, PT, and keyword helpers before removing their exports.
- [ ] Close the immutable author boundary. Re-exporting mutable engine types through `dsl` does not complete it. Recheck direct imports and caller counts when each family is migrated.

## Outstanding Validation and Acceptance

- [ ] Maintain the High-Leverage 3-Tier Testing Model (exhaustive Tier 1 engine tests, 100% Tier 2 `TestRegisteredDefinitions` static validation, and Tier 3 risk-based card tests).
- [ ] Add exact caller migration evidence for each unfinished family, remove obsolete exports only after all callers have replacements, and close the mutable author surface before marking the full design complete.
