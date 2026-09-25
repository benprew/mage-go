# Composable DSL Outstanding Implementation Work

This document tracks active, incomplete implementation items, engine blockers, and acceptance criteria for the composable DSL migration. All completed items and historical slice records have been archived in git history.

The architectural specification is in [composable-dsl-design.md](composable-dsl-design.md), and authoring reference is in [dsl-reference.md](../dsl-reference.md).

## Outstanding Card Migrations

- [x] **Ramses Overdark**: Migrate enchanted-creature destruction to typed targets and activation with the existing `IsEnchanted` filter. See the completed slice below.
- [ ] **Icatian Moneychanger**: Audit entry damage source identity. (Sacrifice activation with credit counter capture is complete).
- [ ] **Spore Creatures**: Migrate the five spore creatures' remaining activated definitions after token color, damage-source LKI, and sacrifice-cost audits. Elvish Farmer and Thallid Devourer have pre-existing omitted Saproling-sacrifice abilities; record exact implementations separately from upkeep migration.
- [ ] **Whirling Dervish**: Audit end-step timing, per-turn opponent-damage condition, and source identity before retiring `DealsDamageToOpponentTrigger`. (It is the remaining production caller; Nicol Bolas is already migrated).
- [ ] **Wheel of Fortune & Contract from Below**: Migrate before retiring the legacy `DiscardHand` selector surface. Audit simultaneous multi-player hand discards and the remaining draw/ante definitions. (Singular typed hand discard is available as `DiscardEntireHand(subject)`).
- [ ] **Modal Cards with Graveyard References**: Add typed graveyard references and remaining modal effects. Trusty Retriever, Entomber Exarch, and Biblioplex Tomekeeper retain their existing modal builders until their remaining effects have exact adapters.
- [ ] **Deep Spawn**: Audit optional upkeep mill payment before migrating the last production `MillTargetPlayer` caller. (Selhoff Occultist, Towering-Wave Mystic, Millstone, and Whimsy's mill branch are migrated; `MillstoneEffect` is retired).
- [ ] **Cauldron of Essence**: Migrate reanimation ability after typed graveyard references are available. Preserve target selection before sacrifice payment, graveyard ownership, sorcery timing, and return under resolving controller. (Death trigger is migrated; `AnyCreatureDiesTrigger` is retired).
- [ ] **First Main Phase Callers**: Migrate other first-main callers before retiring `BeginningOfFirstMainPhaseTrigger`: Scathing Shadelock needs typed prepared state and prepared casting audit; Abstract Paintmage needs missing instant/sorcery mana spending restriction.
- [x] **Goblin Lore**: Migrate the remaining random-discard callback and spell builder to typed `Spell`, `Sequence`, and `DiscardAtRandom`. See the completed slice below.
- [ ] **Surrounding Card Definitions**: Migrate remaining surrounding definitions for Blindblast, Exclude, Thirst for Knowledge, and Rhystic Study. Damage-source LKI, stack references, hidden-zone choices, and event participants still need exact adapters.
- [x] **Explore Land Allowance**: Add `GrantLandPlays(PlayerRef, ResolutionValue)` and migrate Explore to typed `Spell` and `Sequence`. Existing turn allowance and clone services need no engine fix. See the completed slice below.
- [ ] **Rhystic Study**: Audit optional draw and payer choice order when migrating the trigger. Legacy payment branches expose nested typed effects to inspection/validation; event participant capture remains open.
- [ ] **Elemental Uprising**: Audit land animation, target-controller restriction, color behavior, and lifetime.
- [ ] **Neyith of the Dire Hunt**: Audit combat trigger, optional hybrid payment, power doubling, and end-of-combat blocking requirement before removing `TargetMustBeBlockedIfAble`.
- [ ] **Glorious Decay**: Audit damage mode and graveyard target before retiring `DestroyTargetArtifact`.
- [ ] **Positional Return Helpers**: Continue positional return helper migrations. Witch Hunter's separate damage ability awaits damage-source LKI. Rakalite's delayed source identity audit and migration are complete; audit Prismari Charm's modes before retiring `ReturnToHandTarget`.
- [x] **Divine Offering**: Add `ManaValueOf`, capture with `SnapshotNumber`, and migrate destruction and controller life gain to typed actions. Retire the sole-caller helpers `DestroyTargetStep` and `GainLifeControllerFromVar`.
- [x] **Blight Attachment-Tap Capture**: Add `AttachedBecomesTapped`, capture the event land before trigger placement, migrate Blight, and retire `DestroyAttachedStep` and `DestroyAttachedData`. Correct repeated tap-event emission.
- [x] **Targeted Permanent Destruction**: Migrate all six production `DestroyTargetPermanent` callers to typed `Destroy` targets and retire the export. See the completed slice below.
- [x] **Targeted Land Destruction**: Migrate Sinkhole, Stone Rain, and Ice Storm to typed land targets and `Destroy`; retire `NewLandDestruction` and `DestroyTargetLand`. No engine blocker. See the completed slice below.
- [x] **Land-Wide Destruction**: Migrate Armageddon and Tsunami to typed `DestroyEach` collections; retire `DestroyAllLands`. No engine blocker. See the completed slice below.
- [x] **Collection Destruction Without Regeneration**: Add `DestroyEachWithoutRegeneration`, migrate Wrath of God and Shatterstorm, and retire both collection no-regeneration helpers. Resolve the action-scoping engine gap with a regression test first. See the completed slice below.
- [x] **Filtered Destruction Spells**: Migrate Flashfires, Acid Rain, and Cleanse, the remaining direct card callers of `DestroyAllMatching`, to typed collections. No engine blocker. See the completed slice below.
- [x] **Remaining Collection Destruction Helpers**: Migrate Nevinyrral's Disk, Tranquility, and Whimsy's Disk branch; retire `NevinyrralsDiskEffect`, `DestroyAllCreatures`, `DestroyAllEnchantments`, and `DestroyAllMatching`. Resolve the Disk's separate destruction batches. See the completed slice below.
- [x] **Terror and Singular Regeneration Scope**: Migrate Terror to typed destruction and correct all singular no-regeneration executors. Retain legacy exports for the remaining callers; see the completed slice below.
- [x] **Crumble Saved-Value Migration**: Migrate Crumble and Whimsy's Crumble branch to typed controller/mana-value captures and no-regeneration destruction. Retire `CrumbleEffect` and `DestroyGatheredNoRegen`; see the completed slice below.
- [x] **Tabernacle String-Bound Destruction**: Migrate its granted upkeep body to typed payment and source destruction; retire `DestroyGathered`. Correct returned-source identity and include Tabernacle when it is a creature. See the completed slice below.
- [x] **Rakalite Delayed Return**: Migrate prevention and delayed source return to typed targets, activation, and `AtNext`; preserve the original source incarnation. See the completed slice below.
- [x] **Aura Entry Tap**: Migrate Paralyze and Waterknot to typed entry triggers and attachment taps; retire `TapAttachedCreature`. Resolve departed and returned attachment identity with existing typed references. See the completed slice below.
- [x] **Whimsy Random Tap**: Migrate the remaining random-tap branch to typed selection and tap; exclude phased-out candidates. See the completed slice below.
- [x] **Temporary Regeneration Prohibition**: Migrate Hurr Jackal and Faerie Dragon's matching branch, retire `HurrJackalEffect`, and preserve prohibition through source departure and ability removal. See the completed slice below.
- [x] **Tawnos’s Wand Blocking Restriction**: Add typed `CantBeBlocked`, migrate the Wand and Faerie Dragon’s matching branch, and retire `TawnosWandEffect`. Correct ability-removal and source-departure behavior; see the completed slice below.
- [x] **Swords to Plowshares Saved-Value Migration**: Migrate the spell and Faerie Dragon’s matching branch to typed controller/power capture, exile, and life gain; retire `SwordsToPlowsharesEffect`. Resolve exile metadata and phased-out random selection; see the completed slice below.
- [x] **Sorceress Queen Base P/T**: Add typed base power/toughness setting and source exclusion, migrate the Queen and Faerie Dragon’s matching branch, and retire `SorceressQueenEffect`. See the completed slice below.
- [x] **Base Power Activations**: Add typed `SetBasePower`, migrate Singing Tree and Island of Wak-Wak, and retire `SetPowerUntilEndOfTurn`. See the completed slice below.
- [x] **Typed Combat Removal**: Add `RemoveFromCombat(PermanentRef)`, migrate Mijae Djinn’s coin-flip body and Disharmony’s removal instruction, and correct shared removal to preserve blocked status. Keep explicit-unblocking legacy callers pending their full audits. See the completed slice below.
- [x] **Disharmony Casting Window**: Extend `OnlyDuring` to multiple steps and enforce beginning-of-combat or declare-attackers timing. The control-effect adapter remains separate. See the completed slice below.
- [x] **Procrastinate Tap and Counters**: Add typed `MultiplyValues`, migrate the spell to typed tap and twice-X stun counters, and remove its positional callback. No engine blocker. See the completed slice below.
- [x] **Controlled Creature Team Boosts**: Migrate Fortify, Inspired Charge, and Assault Formation’s team activation to typed collection modifications and durations; retire `BoostMatchingUntilEndOfTurn`, including its test fixture caller. No engine blocker. See the completed slice below.
- [x] **Limited Edition Simple Self-Pump Activations**: Migrate Frozen Shade, Granite Gargoyle, Shivan Dragon, Wall of Fire, and Wall of Water to typed activation costs and temporary source modifications. No engine gap; see the completed slice below.
- [x] **Targeted Instant P/T Boosts**: Migrate Howl from Beyond and Righteousness to typed targets, resolution-time X, and temporary modifications. No engine gap; see the completed slice below.
- [x] **Combat-Wide P/T Boost Spells**: Migrate Army of Allah, Piety, and Morale to typed combat collections and end-of-turn modifications. No engine blocker; see the completed slice below.
- [x] **Targeted Artifact +2/+2 Activations**: Migrate Coral Helm and Elven Lyre to typed targets, cost groups, and end-of-turn modifications. No engine blocker; see the completed slice below.
- [x] **Antiquities Self-Pump Activations**: Migrate Atog and Dragon Engine to typed costs and source modifications. Correct Atog's source-inclusive artifact sacrifice using the existing cost API. See the completed slice below.
- [x] **Global P/T Reductions**: Migrate Hell Swarm, Marsh Gas, Bone Flute, and Languish to typed creature collections and durations. Correct negative temporary P/T metadata after a failing regression; see the completed slice below.
- [x] **Targeted Land Boost Activations**: Migrate Mishra’s Factory’s boost and Pendelhaven to typed targets, tap costs, and temporary P/T modifications. No engine blocker; see the completed slice below.
- [x] **Affa Guard Hound Entry Boost**: Migrate the targeted entry boost to `WhenEvent`, `WithTargets`, and temporary `ModifyPT`. Correct target choice timing in the card definition; no engine gap. See the completed slice below.
- [x] **Shield Wall Team Boost**: Migrate the remaining positional team spell to a typed controlled-creature collection and explicit end-of-turn duration. No engine gap; see the completed slice below.
- [x] **Jumpstart Single-Target P/T Spells**: Migrate Aegis of the Heavens, Auger Spree, and Last Gasp to typed creature targets and explicit end-of-turn modifications. No engine gap; see the completed slice below.
- [x] **Jumpstart Simple Self-Pump Activations**: Migrate Dragon Hatchling, Furnace Whelp, Minotaur Sureshot, Molten Ravager, Tibalt’s Rager, and Jousting Dummy to typed mana costs and temporary source modifications. No engine gap; see the completed slice below.
- [x] **Legends Simple Mana Self-Pump Activations**: Migrate eight activations across six cards to typed mana costs and temporary source modifications. No engine gap; see the completed slice below.
- [x] **Legends Targeted Tap P/T Activations**: Migrate Ghosts of the Damned, Pradesh Gypsies, and Tuknir Deathlock to typed targets, grouped costs, and temporary modifications. No engine gap; see the completed slice below.
- [x] **Limited Self-Pump Activations**: Migrate Fire Drake, Vampire Bats, and Mindful Biomancer to typed mana costs, source modifications, and per-turn activation limits. No engine gap; see the completed slice below.
- [x] **Jumpstart Triggered Team Boosts**: Migrate Inspiring Captain and Inspiring Unicorn to typed controlled-creature collections and explicit end-of-turn P/T modifications. Captain also uses `WhenEvent(Enters(This), ...)`; Unicorn retains its attack-event wrapper pending that adapter. No engine gap; see the completed slice below.
- [x] **Angelic Page Combat Target**: Migrate its tap activation to a typed attacking-or-blocking creature target and end-of-turn P/T modification. No engine gap; see the completed slice below.
- [ ] **String-Bound Action Migrations**: Continue destruction, return, tap-or-untap, attachment, and other string-bound action migrations where exact typed forms exist.
- [x] **Target Collection Boosts**: Migrate Dauntless Onslaught and Tandem Tactics to typed target collections, spells, and temporary collection boosts. See the completed slice below.
- [x] **Osai Vultures Counter-Payment Boost**: Migrate its activation to typed counter costs and an explicit temporary source modification. No engine gap; the end-step trigger remains a separate audit. See the completed slice below.
- [x] **Wyluli Wolf Targeted Tap Boost**: Migrate its activation to a typed creature target, tap cost, and explicit end-of-turn P/T modification. Existing adapters cover the full ability; no engine gap. See the completed slice below.

- [x] **Fallen Angel Sacrifice Boost**: Migrate its activation to typed costs and a temporary source modification. Correct source exclusion with the existing sacrifice-cost adapter; see the completed slice below.
- [x] **Dwarven Lieutenant Subtype Target Boost**: Migrate its activation to a typed Dwarf-creature target, mana cost, and explicit end-of-turn modification. No engine gap; see the completed slice below.

- [x] **Icatian Targeted Mana Boosts**: Migrate Icatian Lieutenant and Icatian Priest to typed creature targets, mana costs, and explicit end-of-turn P/T modifications. No engine gap; see the completed slice below.

- [x] **Fallen Empires Orders**: Migrate Order of Leitbur and Order of the Ebon Hand to typed first-strike and power-boost activations with explicit end-of-turn durations. No engine gap; see the completed slice below.

- [x] **Fallen Empires Self-Keyword Activations**: Migrate the five remaining mana-only self-keyword activations on Icatian Infantry, River Merfolk, Vodalian Knights, and Orcish Veteran to typed costs and explicit durations. No engine gap; see the completed slice below.

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

## Completed Slice: Fasting Registration Retirement

| Retired API and callers | Replacement and semantics | Evidence |
| :--- | :--- | :--- |
| `Game.AddFastingReplacement`: three step-test calls, one controller-test call, two life-outcome-test calls | `Static(SkipDrawStepForLife(2))`; no targets; optional whole draw-step replacement for the current controller; gain first in the next actual step. No damage LKI is needed. Static inspection and benefit metadata remain unchanged. | `TestFastingSkipsWholeStep`, `TestFastingDrawStepEligibility`, `TestFastingGainWaitsThroughSkippedSteps`, `TestFastingReplacementCurrentController`, `TestReplacementLifeGainOutcomes`, `TestFastingLifeGainRequiresAcceptedDrawStep` |
| Three explicit lifetime-test calls | Private `drawStepLifeReplacement` records with explicit `WhileOnBattlefield` duration and captured incarnation; a missing incarnation stays inactive. These fixtures test the executor independently of static rule regeneration. | `TestFastingReplacementSourceLifetime` |
| Fasting Oracle: “If you would begin your draw step, you may skip that step instead. If you do, you gain 2 life.” | Existing `cards/thedark` definition already uses `Static(SkipDrawStepForLife(2))`; no card change required. CR 614.10–614.10b in the repository rules remain the contract. | Existing `TestStaticDrawStepReplacement*` engine tests and `TestDrawStepReplacementComposition` author-surface validation |

No Go callers or declaration of the retired API remain. General typed replacement rules and the other life-gain audits remain open.

Validation: focused Fasting, static replacement, and life-outcome tests; `go test ./pkg/mage/... -short -count=1`; `make test`; `go build ./...`; `go vet ./...`; and `make lint` all passed with zero failures (lint: zero issues).

## Completed Slice: Divine Offering Mana-Value Capture

- [x] Add and export `ManaValueOf(PermanentRef)` with typed subject inspection, registration validation, and compile-time rejection of players and collections.
- [x] Migrate Divine Offering to `Spell`, a typed artifact target, `SnapshotNumber`, `Destroy`, and `GainLife(You, FromNumber(...))`.
- [x] Retire `DestroyTargetStep` and `GainLifeControllerFromVar` from both engine and author surfaces. Each had exactly one caller, Divine Offering. Retain the shared destruction and string-player life executors for other callers.

| Oracle clause / retired API | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| “Destroy target artifact.” / `DestroyTargetStep` | `Destroy(artifact)` with `TargetPermanent("artifact", IsArtifact)` and `WithTargets`; either player's artifact; ordinary spell target legality, incarnation checks, and regeneration. Destruction metadata remains available. | `TestDivineOffering`, `TestDivineOfferingOutcomes`, `TestDivineOfferingTypedDefinition` |
| “You gain life equal to its mana value.” / `GainLifeControllerFromVar` | `SnapshotNumber(ManaValueOf(artifact), value)` before destruction, then `GainLife(You, FromNumber(value))`; stored spell controller gains through the existing event/replacement path even if destruction fails. No gain if the only target is illegal. The number survives departure without a card-ID LKI lookup. | `TestDivineOfferingOutcomes`: destruction, indestructible, regeneration, own artifact, illegal target, leave-and-return, zero mana value, and X |
| Typed mana value and saved value | Reuse `PermanentRef.resolvePermanent`, `ManaCost.CMC()`, and number frames. Live values exclude phased/absent/new incarnations; saved numbers remain unchanged. Typed target groups and cloned resolutions retain independent values. | `TestManaValueOfPermanent`, `TestManaValueOfTargetClone`, `TestManaValueOfValidation`, `TestManaValueComposition`, `TestClosedExpressionRoles` |

Oracle source: `cards/legends/LEG.json`. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 202.3/202.3e (mana value and X), 400.7 (new objects), and 608.2b/608.2h (target legality and object information). No underlying engine change was needed for this migration; the new adapter uses the existing mana-cost representation. Tests first failed for the missing constructor and the legacy untyped card target.

Separate audit finding: `ParseManaCost("{2/W}").CMC()` returns 0 instead of 2. The parser supports two-color hybrid symbols but not monocolored hybrid symbols. Full monocolored-hybrid parsing/payment support remains an engine gap outside this slice; `ManaValueOf` does not correct or claim support for unrepresented mana symbols. General copying and zone-LKI audits remain open.

Validation: focused mana-value, Divine Offering, composition, and compile-fail tests; `go test ./pkg/mage/... ./cards/legends -short -count=1`; `go test ./cards/... -short -count=1`; `make test`; `go build ./...`; `go vet ./...`; and `make lint` passed with zero failures (lint: zero issues). No Go declarations or callers remain for either retired export.


## Completed Slice: Blight Attachment-Tap Capture

- [x] Add and export `AttachedBecomesTapped(PermanentBinding)` with binding inspection, registration validation, modal composition, and compile-time category checks.
- [x] Fix `Game.TapPermanent` to require an untapped-to-tapped transition. `TestTapPermanentRequiresTransition` first failed with two triggers for one transition; it also checks that a later transition can trigger again.
- [x] Migrate Blight and remove the sole-caller `DestroyAttachedStep` constructor and `DestroyAttachedData` executor. No Go declarations or callers remain.

| Oracle clause / retired API | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| “Enchant land” | Existing `WithCastTarget(TargetLand())` remains the Aura casting restriction. | Existing target validation; `TestRegisteredDefinitions` |
| “When enchanted land becomes tapped” | `WhenEvent(AttachedBecomesTapped(land), ...)`; capture at detection, before pending-trigger placement; no trigger targets. Tap source must be the attachment; unattached and phased sources do not trigger. | `TestAttachedBecomesTappedCapture`, `TestAttachedBecomesTappedDelayedModes`, `TestTapPermanentRequiresTransition` |
| “destroy it.” / `DestroyAttachedStep` (Blight only) | `Destroy(From(land))`; destroy the captured incarnation even if the Aura moves or leaves. Ignore a returned land or phased-out host; allow regeneration and indestructible. Uses the shared destruction executor and exposes destruction metadata. | `TestBlightCapturesTappedLand`: normal resolution, Aura move/departure, land return, regeneration, indestructible. Move, departure, and return cases failed with the old definition. |
| Event bindings, copies, and composition | Existing `eventPermanentCapture` and resolution frames preserve separate occurrences, pending and stack clones, copied controllers, phasing, control changes, and delayed capture. Registration rejects missing/empty bindings and cross-ability reads. | `TestAttachedBecomesTappedCapture`, `TestAttachedBecomesTappedDelayedModes`, `TestAttachedBecomesTappedValidation`, `TestAttachedBecomesTappedComposition`, `TestClosedExpressionRoles` |

Oracle source: `cards/legends/LEG.json`. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 603.2e (becomes), 603.3 (pending triggers), 400.7 (new objects), 701.8 (destroy), and 701.26a (tap). The new adapter uses the existing event detector, captured-permanent records, pending trigger and copy paths, and destruction executor. The only engine behavior fix is the tap-transition guard.

`WhenAttachedBecomesTappedTrigger` remains for four production callers: Psychic Venom, Kudzu, Artifact Possession, and Relic Bind. Their outstanding damage, attachment, and modal audits are not resolved by this slice. Blight's file retains unrelated legacy definitions and engine imports.

Validation: focused tap-transition, attachment-capture, Blight, composition, and compile-fail tests; `go test ./pkg/mage/... ./cards/legends -short -count=1`; `make test`; `go build ./...`; `go vet ./...`; and `make lint` all passed with zero failures (lint: zero issues).


## Completed Slice: Targeted Permanent Destruction

- [x] Migrate Northern Paladin, Ronom Unicorn, Unstable Obelisk, Ravenous Chupacabra, Crushing Canopy, and Grapple with Death to typed permanent targets and `Destroy`.
- [x] Retire `DestroyTargetPermanent` from the engine and `dsl` exports. No Go declarations or callers remain.
- [x] Audit the existing target, payment, modal, and destruction adapters. No engine fix or new DSL constructor is required for these callers.

| Oracle clause / previous caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Northern Paladin: “{W}{W}, {T}: Destroy target black permanent.” | `Activate(Costs(Mana("{W}{W}"), TapCost(This)), Destroy(target), WithTargets(target))`; black filter, either controller, normal tap and mana payment. | `TestNorthernPaladin`; typed target and activation contract tests |
| Ronom Unicorn: “Sacrifice this creature: Destroy target enchantment.” | `Activate` with `SacrificeSourceCost`, enchantment target, and `Destroy`; select the target before sacrificing the source. No source characteristics are needed at resolution. | `TestRonomUnicorn`; existing transactional activation tests |
| Unstable Obelisk: “{7}, {T}, Sacrifice this artifact: Destroy target permanent.” | `Activate` with mana, tap, and sacrifice costs; `AnyPermanent` typed target; destruction survives source departure. Its separate “{T}: Add {C}.” ability is unchanged. | `TestUnstableObelisk`; existing transactional activation tests |
| Ravenous Chupacabra: “When this creature enters, destroy target creature an opponent controls.” | `WhenEvent(Enters(This), Destroy(creature), WithTargets(creature))` with `TargetOpponentControls`; mandatory target at trigger placement, stored controller, legality rechecked at resolution. | `TestRavenousChupacabra_DestroysOnETB`; `TestTypedOpponentTargetCandidatesAndActivation`, `TestTypedOpponentTargetCopy`, `TestTypedOpponentTargetTriggerController` |
| Crushing Canopy: “Choose one — • Destroy target creature with flying. • Destroy target enchantment.” | `SpellModes` with two `ModeOf` definitions; each has its own typed target and `Destroy`. Only the selected mode supplies targets. Both former destruction helper calls are migrated. | `TestCrushingCanopy_DestroyFlying`, `TestCrushingCanopy_DestroyEnchantment`; `TestModalCastRejectsTargetsBeforePayment`, `TestModalCastKeepsModeAndTargets` |
| Grapple with Death: “Destroy target artifact or creature. You gain 1 life.” | `Spell(Sequence(Destroy(target), GainLife(You, Fixed(1))), WithTargets(target))`; artifact-or-creature target; gain still occurs if destruction fails, but not if the sole target is illegal. | `TestGrappleWithDeath`, `TestGrappleWithDeathTypedDefinition`, `TestGrappleWithDeathOutcomes`: destruction, indestructible, regeneration, illegal target, returned target, own artifact |
| Five test-only helper calls: three legacy modal tests, one legacy/typed modal comparison fixture, one AI removal fixture | Legacy modal fixtures use the remaining `DestroyTarget` positional executor to retain coverage of `NewModalSpell`. The AI fixture uses typed `Spell` and `Destroy`, retaining detrimental/destruction metadata and threat preference. | `TestNewModalSpell*`, `TestModalCast*`, `TestAutoSelectTargets_GenericRemovalPrefersCreatureOverLand` |

All migrated targets identify the captured battlefield incarnation. They cannot destroy a later incarnation of the same card. Destruction uses the existing executor, permits regeneration, and respects indestructible. No damage-source LKI is required. The internal `destroyTargetPermanentEffect` remains for `DestroyTargetLand` and `DestroyTargetArtifact`; those helpers and other positional destruction callers are outside this slice. The changed card files retain unrelated legacy definitions and imports.

Oracle sources: `data/LEA.json`, `data/JMP.json`, and `data/SOS.json`. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 400.7 (new objects), 601–602 (targets and payment), 603.3 (trigger placement), 608.2b (target legality), and 701.8 (destroy). The Grapple definition test failed before migration because the target was untyped; its new outcome tests passed against the legacy implementation and preserve that behavior.

Validation: focused card, modal, target, compile-fail, and AI tests; all three affected card packages; `go test ./cards/... -short -count=1`; `go test ./pkg/mage/... -short -count=1`; `make test`; `go build ./...`; `go vet ./...`; and `make lint` passed with zero failures (lint: zero issues).


## Completed Slice: Targeted Land Destruction

- [x] Migrate all three `NewLandDestruction` callers to `NewSorcery`, `Spell`, `TargetPermanent("land", IsLand)`, and `Destroy` through `pkg/mage/dsl`.
- [x] Retire `NewLandDestruction` from the engine and author surface, and retire the engine-only `DestroyTargetLand` export after migrating its test fixture. No Go declarations or callers remain.
- [x] Audit the existing typed target and destruction adapters. No new constructor or engine behavior fix is needed. The shared `destroyTargetPermanentEffect` remains for `DestroyTargetArtifact`.

| Oracle clause / retired API and callers | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Sinkhole, Stone Rain, Ice Storm: “Destroy target land.” / all three `NewLandDestruction` callers | `Spell(Destroy(land), WithTargets(land))` with `TargetPermanent("land", IsLand)`; one land controlled by either player, including nonbasic lands; printed costs and sorcery timing remain unchanged. The action allows regeneration and respects indestructible. | `TestLandDestructionTypedDefinitions` checks all three typed definitions, land-only candidates from both controllers, and detrimental/destruction metadata; existing `TestAlphaDestroyEffects` covers Sinkhole resolution. |
| `DestroyTargetLand`: `NewLandDestruction` and the Stone Rain fixture in `rules_ch03_card_types_test.go` | Typed land target and `Destroy`; target legality and captured incarnation are checked by the existing spell/target services. No source LKI is required. The fixture retains its existing cost and sorcery type. | `TestTypedTargetGroups`, `TestTypedTargetCopyAndClone`, `TestTargetIncarnationAfterReturn`, `TestComposedRemovalMetadata`; existing card-type rules suite. |

Oracle source: `data/LEA.json`; full text is now above each registration. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 400.7 (new objects), 608.2b (target legality), and 701.8 (destruction and regeneration). The new definition test failed for all three cards before migration because their land targets were untyped. These single-action cards use the existing engine contract tests and registration validation instead of additional game simulations. The changed card file retains unrelated legacy definitions and imports. Other destruction migrations and engine blockers remain open.

Validation: focused land-definition and destruction tests; `go test ./cards/limited -count=1`; `go test ./cards/... -short -count=1`; `go test ./pkg/mage/... -short -count=1`; `go build ./...`; `go vet ./...`; `make test`; and `make lint` all passed with zero failures (lint: zero issues).

## Completed Slice: Land-Wide Destruction

- [x] Migrate Armageddon and Tsunami to `Spell(DestroyEach(MatchingPermanents(filter)))` through `pkg/mage/dsl`.
- [x] Retire `DestroyAllLands` from the engine and author surface after migrating its sole caller, Armageddon. Only compile-fail fixtures mention the removed Go export.
- [x] Audit collection selection and simultaneous destruction. Existing adapters and `Game.DestroyPermanents` support these cards; no new constructor or engine fix is required.

| Oracle clause / previous caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Armageddon: “Destroy all lands.” / sole `DestroyAllLands` caller | `DestroyEach(MatchingPermanents(IsLand))`; all lands of both players, selected at resolution, without targets. Printed cost and sorcery timing remain unchanged. | `TestLandSweepersTypedDefinitions`, `TestDestroyEachMatchingLands`, existing `TestAlphaDestroyEffects` and Consecrate Land tests |
| Tsunami: “Destroy all Islands.” / `DestroyAllMatching` caller | `DestroyEach(MatchingPermanents(HasSubType("Island")))`; both players' Islands, including nonbasic Islands; subtype matching does not depend on the card name. | `TestLandSweepersTypedDefinitions`, `TestDestroyEachMatchingLands` |
| Collection execution and metadata | The existing `destroyEachEffect` selects current, unphased permanents and calls `DestroyPermanents` once. Regeneration and indestructible apply. No targets, saved references, or source LKI are needed. Detrimental, mass, and destruction metadata remain available. | `TestDestroyEachMatchingLands`: empty collection, later entries, both controllers, subtype exclusion, indestructible, regeneration, phasing, and clone isolation; `TestDeathObservationSimultaneousAndClone`: grouped observation and departing death-trigger sources |

Oracle source: `data/LEA.json`; full Oracle text is above each registration. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 608.2c (instructions), 701.8 (destruction and regeneration), and 702.26b (phasing). The new definition test failed for both legacy definitions before migration. These declarative spells reuse engine contract tests and universal registration validation instead of new card simulations.

`DestroyAllMatching` remains for other callers, including non-land destruction. Other destruction helpers, unrelated legacy definitions/imports in the changed card file, and the listed engine blockers remain open.

Validation: focused definition, collection, and simultaneous-destruction tests; `go test ./cards/limited -count=1`; `go test ./cards/... -short -count=1`; `go test ./pkg/mage/... -short -count=1` (including retired-export compile-fail fixtures); `go build ./...`; `go vet ./...`; `make test`; and `make lint` all passed with zero failures (lint: zero issues).


## Completed Slice: Collection Destruction Without Regeneration

- [x] Add and export `DestroyEachWithoutRegeneration(PermanentSet)` with collection inspection, destruction metadata, registration validation, composition tests, and compile-time category checks.
- [x] Migrate Wrath of God and Shatterstorm and retire `DestroyAllCreaturesNoRegen`, `DestroyAllMatchingNoRegen`, and their private executor. Only retired-export compile-fail fixtures mention those Go APIs.
- [x] Resolve the collection engine gap: forbid regeneration on the destruction action instead of granting the permanent a keyword. `TestCollectionNoRegenerationPreservesCharacteristics` first failed against the legacy executor because the last-known creature incorrectly had `CantRegenerate`; it passed after the minimal action/replacement fix, before helper retirement.

| Oracle clause / retired API and callers | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Wrath of God: “Destroy all creatures. They can't be regenerated.” / sole `DestroyAllCreaturesNoRegen` card caller | `Spell(DestroyEachWithoutRegeneration(MatchingPermanents(IsCreature)))`; both players' creatures selected at resolution, no targets, printed cost and sorcery timing preserved. | `TestLandSweepersTypedDefinitions`, `TestDestroyEachWithoutRegenerationCollections` |
| Shatterstorm: “Destroy all artifacts. They can't be regenerated.” / sole direct `DestroyAllMatchingNoRegen` card caller (the other caller was the retired creature helper) | `Spell(DestroyEachWithoutRegeneration(MatchingPermanents(IsArtifact)))`; both players' artifacts, no targets; indestructible and phasing apply. | `TestShatterstormTypedDefinition`, existing `TestShatterstorm`, `TestDestroyEachWithoutRegenerationCollections` |
| “They can't be regenerated.” / private collection executor | Shared `destroyEachEffect` calls the existing simultaneous destruction service with an action-local prohibition. Regeneration replacements decline that action. Permanent characteristics and the pre-death snapshot remain unchanged. | `TestCollectionNoRegenerationPreservesCharacteristics`; collection tests cover empty sets, later entries, exclusions, shields, both controllers, phasing, indestructible, and clone isolation. |
| Simultaneous deaths and composed definitions | Reuse the existing zone-change batch, target/set resolution, replacement pipeline, and definition visitor. No source LKI is required. Detrimental and mass metadata remain; `Destroys` is now correctly exposed. | `TestNoRegenerationSimultaneousDeathObservation` covers departing observers, final death counts, and state-trigger atomicity; `TestCollectionNoRegenerationComposition` covers spells, sequences, activations, triggers, modes, and invalid nil collections; compile-fail fixtures reject singular/player subjects and retired exports. |

Oracle sources: `data/LEA.json` and `cards/antiquities/ATQ.json`. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 701.8 (destroy), 701.19c (regeneration prohibition), 603.6c (death observation), and 702.26b (phasing). Both typed-definition tests failed against the legacy card definitions before migration. These declarative cards reuse engine contract tests and universal registration validation.

The changed card files retain unrelated legacy definitions and imports. Singular positional no-regeneration executors still grant `CantRegenerate`; their migration and action-scope audit remain separate work. The general damage/zone-LKI blockers are unchanged.

Validation: focused regression and typed-definition tests; both affected card packages; `go test ./cards/... -short -count=1`; `go test ./pkg/mage/... -short -count=1`; `make test`; `go build ./...`; `go vet ./...`; and `make lint` all passed with zero failures (lint: zero issues).


## Completed Slice: Filtered Destruction Spells

- [x] Migrate Flashfires, Acid Rain, and Cleanse to `Spell(DestroyEach(MatchingPermanents(filter)))` through `pkg/mage/dsl`.
- [x] Verify land subtype and compound creature/color filters against the existing collection executor. No new constructor or engine behavior fix is required.
- [x] Audit remaining callers: no production card calls `DestroyAllMatching` directly. Keep that export and its executor for `DestroyAllCreatures`, `DestroyAllEnchantments`, and `NevinyrralsDiskEffect`; their migration and the Disk's simultaneous destruction audit remain open.

| Oracle clause / previous caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Flashfires: “Destroy all Plains.” / `DestroyAllMatching` | `DestroyEach(MatchingPermanents(HasSubType("Plains")))`; includes both players' basic and nonbasic Plains, selected at resolution by subtype. | `TestLandSweepersTypedDefinitions`, `TestDestroyEachMatchingLands` |
| Acid Rain: “Destroy all Forests.” / `DestroyAllMatching` | `DestroyEach(MatchingPermanents(HasSubType("Forest")))`; same subtype contract as Flashfires. Removes the redundant land-type conjunction. | `TestFilteredDestructionTypedDefinitions`, `TestDestroyEachMatchingLands` |
| Cleanse: “Destroy all black creatures.” / `DestroyAllMatching` | `DestroyEach(MatchingPermanents(And(IsCreature, HasColorFilter(Black))))`; includes multicolored black creatures of both players; excludes nonblack creatures and black noncreatures. | `TestFilteredDestructionTypedDefinitions`, `TestDestroyEachMatchingBlackCreatures` |
| Collection behavior and metadata | One simultaneous destruction group, no targets or source LKI; regeneration and indestructible apply, phased permanents are excluded, and cloned games remain isolated. Printed mana costs and sorcery timing are unchanged. Detrimental, mass, and destruction metadata remain available. | Expanded land-subtype and black-creature engine tests cover empty collections, later entries, exclusions, regeneration, indestructible, phasing, both controllers, and clone isolation; existing `TestDeathObservationSimultaneousAndClone` covers grouped deaths. |

Oracle sources: `data/LEA.json` and `cards/legends/LEG.json`. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 205.3i (land subtypes), 608.2c (instructions), 701.8 (destruction and regeneration), and 702.26b (phasing). All three typed-definition tests failed against the legacy definitions before migration. These declarative spells reuse engine contract tests and universal registration validation instead of new card simulations. Unrelated legacy definitions and engine imports remain in the changed card files. No engine blocker is claimed resolved by this slice.

Validation: focused typed-definition and filtered-collection tests; both affected card packages; `go test ./cards/... -short -count=1`; `go test ./pkg/mage/... -short -count=1`; `make test`; `go build ./...`; `go vet ./...`; and `make lint` all passed with zero failures (lint: zero issues).


## Completed Slice: Remaining Collection Destruction Helpers

- [x] Migrate Nevinyrral's Disk to `Activate`, typed mana/tap costs, and one `DestroyEach` union collection; migrate Tranquility to `Spell(DestroyEach(...))` and Whimsy's Disk branch to the same union action.
- [x] Resolve the Disk helper defect under `pkg/mage` after two failing regressions. The old composite destroyed creatures, enchantments, and artifacts in separate batches. Reuse the existing simultaneous executor; no new DSL constructor or zone subsystem is needed.
- [x] Retire `NevinyrralsDiskEffect`, `DestroyAllCreatures`, `DestroyAllEnchantments`, `DestroyAllMatching`, and the private `destroyAllMatchingEffect` executor from engine and author surfaces. Migrate every test caller and add retired-export compile-fail fixtures.

| Oracle clause / retired API and callers | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Disk: “This artifact enters tapped.” | Preserve `WithKeyword(EntersTapped)`. | Existing `TestNevinyrralsDisk`; `TestDiskTypedActivation` |
| Disk: “{1}, {T}: Destroy all artifacts, creatures, and enchantments.” / `NevinyrralsDiskEffect` | `Activate(Costs(Mana("{1}"), TapCost(This)), DestroyEach(MatchingPermanents(Or(IsArtifact, IsCreature, IsEnchantment))))`; no targets; includes the Disk; membership selected at resolution. The ability resolves after its source leaves. | `TestDiskTypedActivation`: entry tapped, rejected tapped activation, mana consumption, tap payment, stack placement, late entries, source departure, regeneration, and destruction metadata |
| Whimsy's Disk branch / second production `NevinyrralsDiskEffect` caller | The same typed union action in the existing random-action callback. Preserve selection and surrounding Whimsy behavior. | `TestWhimsyDiskRegeneration`; existing `TestWhimsy_EachRandomActionCanBeSelected` |
| Tranquility: “Destroy all enchantments.” / sole direct card caller of `DestroyAllEnchantments` | `Spell(DestroyEach(MatchingPermanents(IsEnchantment)))`; both players' enchantments, no targets; printed cost and sorcery timing unchanged. | `TestLandSweepersTypedDefinitions/Tranquility`; existing `TestAlphaDestroyEffects` |
| `DestroyAllCreatures`: former Disk constituent and four test calls; `DestroyAllEnchantments`: former Disk constituent and Tranquility; `DestroyAllMatching`: those wrappers and the former Disk artifact constituent | Typed `DestroyEach` collections; keep the existing game-test spell construction paths and engine audit assertions. The classic-effect text fixture now uses the union expression. | Existing `TestDestroyAllCollapse`, action, casting/resolution, and engine audit suites; author compile-fail suite |
| Union destruction contract | One simultaneous action group destroys each matching permanent once. A single regeneration shield protects a permanent with all three types. State triggers cannot observe partial type batches. No source LKI or saved reference is required. | `TestUnionDestructionRegeneratesOnce` and `TestUnionDestructionIsSimultaneous` failed against the legacy helper, passed after its minimal engine fix, and remain against the canonical typed action after retirement. `TestUnionDestructionMembership` covers empty sets, later entries, both controllers, each included type, excluded lands/planeswalkers, indestructible, phasing, and clone isolation. |

Oracle source: `data/LEA.json`; full Oracle text is above both changed registrations. Whimsy retains its existing random-action definitions apart from the Disk branch. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 602 (activation costs), 608.2c (instructions), 701.8 (destruction), 701.19 (regeneration), and 603.8 (state triggers). Tranquility's new definition check failed against its legacy untyped effect before migration. Detrimental, mass, and destruction metadata remain available.

No declarations or calls of the retired APIs remain in `cards` or `pkg/mage`, apart from negative compile fixtures. The unrelated `cmd/forge` effect-kind enum named `DestroyAllCreatures` remains. Unrelated legacy definitions/imports in changed card files and all other listed engine blockers remain open.

Validation: both failing engine regressions passed after the minimal helper fix and again after retirement; focused card/collection tests, both affected card packages, `go test ./cards/... -short -count=1`, `go test ./pkg/mage/... -short -count=1` (including compile-fail fixtures), `make test`, `go build ./...`, `go vet ./...`, and `make lint` all passed with zero failures (lint: zero issues).


## Completed Slice: Singular Destruction Without Regeneration and Terror

- [x] Resolve the singular regeneration-scope defect with failing regressions first. Typed, positional, step, and string-bound paths no longer grant `CantRegenerate`.
- [x] Reuse `Game.destroyPermanents` with one permanent and an action-scoped prohibition. Keep the existing replacement, death observation, and zone services.
- [x] Migrate Terror to `Spell(DestroyWithoutRegeneration(creature), WithTargets(creature))` with a typed nonartifact, nonblack creature target.

| Oracle clause / affected API | Replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Terror: “Destroy target nonartifact, nonblack creature. It can't be regenerated.” | Either player's creature; exclude artifacts and all black creatures, including multicolored black creatures. Preserve instant timing and {1}{B}. No source LKI is required. | `TestTerrorTypedDefinition` first failed for the legacy target; verifies registration, typed action, destruction metadata, and target eligibility for both players. |
| `DestroyWithoutRegeneration`, `DestroyTargetNoRegen`, `DestroyTargetNoRegenStep`, `DestroyGatheredNoRegen` | Shared action-scoped prohibition bypasses regeneration shields without changing permanent characteristics or death LKI. | All four cases of `TestSingularNoRegenerationPreservesCharacteristics` failed before the fix and passed after it. |
| Typed saved reference and cloning | The existing resolver excludes absent, phased, and returned incarnations; phasing cycles and control changes preserve references. Indestructible survives and retains its shield. | `TestSingularNoRegenerationSavedLifetime`; existing regeneration and collection contract tests. |

Oracle source: `data/LEA.json`; the complete text is above Terror's registration. Rules contract: repository CR 701.8 (destruction), 701.19c (regeneration prohibition), 400.7 (new objects), and 702.26 (phasing). Terror is declarative and uses engine contract tests and registration validation.

No export is retired in this slice. Remaining production callers are Abu Ja'far and Detonate (step), Aswan Jaguar (positional), and Crumble's helper (string-bound). Their combat, target, damage, and saved-value migrations remain open. This fix corrects their regeneration scope without claiming those surrounding definitions are fully audited. General damage and zone-LKI blockers remain open.

Validation: focused engine regressions and Terror definition test; `go test ./cards/limited -count=1`; `go test ./pkg/mage/... ./cards/... -short -count=1`; `make test`; `go build ./...`; `go vet ./...`; and `make lint` all passed with zero failures (lint: zero issues).


## Completed Slice: Crumble Saved Values

- [x] Migrate Crumble to a typed artifact target, controller and mana-value snapshots, `DestroyWithoutRegeneration`, and `GainLife`.
- [x] Migrate Whimsy's Crumble branch to `ChooseRandomPermanent` and the same typed sequence.
- [x] Retire `CrumbleEffect` and `DestroyGatheredNoRegen` from engine and author surfaces. Remove the unused string-bound no-regeneration executor branch; retain ordinary `DestroyGathered` and shared life executors for other callers.
- [x] Add retired-export compile-fail fixtures for both surfaces. Replace the old gathered no-regeneration regression case with a typed saved permanent.

| Oracle clause / retired API and callers | Replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Crumble: “Destroy target artifact. It can't be regenerated.” / `CrumbleEffect` | `Spell` with `TargetPermanent("artifact", IsArtifact)` and `DestroyWithoutRegeneration`; either controller; instant timing and {G} unchanged. Action-scoped regeneration prohibition leaves characteristics unchanged. | `TestCrumbleTypedDefinition` first failed for the legacy target; `TestCrumbleOutcomes` and existing `TestCrumble` cover destruction, indestructible, and regeneration. |
| “That artifact's controller gains life equal to its mana value.” | `SnapshotPlayer(ControllerOf(artifact), controller)` and `SnapshotNumber(ManaValueOf(artifact), value)` before destruction; `GainLife(FromPlayer(controller), FromNumber(value))` afterward. The resolving controller of the artifact gains even if destruction fails. Values survive destruction without general zone LKI. | `TestCrumbleOutcomes` covers opposing and own artifacts, control changes before resolution, illegal and returned targets, zero mana value, and X. Existing `TestPlayerSnapshotLifetime`, `TestManaValueOfPermanent`, and life-outcome contracts cover snapshot lifetime, clone isolation, and gain events/replacements. |
| Whimsy's Crumble branch / second production `CrumbleEffect` caller | `ChooseRandomPermanent(MatchingPermanents(IsArtifact), chosen)` followed by typed captures and actions. No spell target declaration; preserve random choice at resolution. Empty collections do nothing. | `TestWhimsyCrumble` covers ordinary destruction, regeneration, indestructible, empty collections, controller life gain, and nonartifact exclusion; existing random-action selection tests remain. |
| `DestroyGatheredNoRegen`: Crumble helper and one engine regression | `DestroyWithoutRegeneration` on a typed target or saved permanent; destruction metadata is now available on the composed card effect. | `TestSingularNoRegenerationPreservesCharacteristics`, `TestSingularNoRegenerationSavedLifetime`, and retired-export compile-fail fixtures. |

Oracle source: `cards/antiquities/ATQ.json`; full text remains above the registration. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 202.3/202.3e (mana value and X), 400.7 (new objects), 608.2b/608.2h (target legality and object information), 701.8 (destruction), and 701.19c (regeneration prohibition). Existing typed adapters and engine contracts provide the complete migration; no engine gap was encountered or claimed resolved. General mana-symbol, damage, and zone-LKI blockers remain open.

No live Go callers or declarations of either retired constructor remain outside negative compile fixtures. Unrelated legacy card definitions and imports remain in the edited files.

Validation: focused Crumble and Whimsy tests; both affected card packages; `go test ./pkg/mage/... ./cards/... -short -count=1` (including universal registration validation and compile-fail fixtures); `make test`; `go build ./...`; `go vet ./...`; and `make lint` passed with zero failures (lint: zero issues).

## Completed Slice: Tabernacle String-Bound Destruction

- [x] Replace the sole production `DestroyGathered` call with `UnlessPaid(You, Costs(Mana("{1}")), Destroy(This))`.
- [x] Remove `DestroyGathered` and its executor from the engine and author surfaces; add compile-fail retirement fixtures for both surfaces.
- [x] Correct the grant to include its source when Tabernacle is a creature, using the existing `GrantTriggeredAbilityToAllIncludingSource` executor.
- [x] Verify returned-source identity before trigger placement and while on the stack. Existing typed source capture resolves both failures without an engine change.

| Oracle clause / retired API | Replacement and semantics | Evidence |
| :--- | :--- | :--- |
| “All creatures have” | Existing source-inclusive continuous grant, filtered by `IsCreature`; each creature receives its own trigger. An animated Tabernacle also receives the ability. | `TestTabernacleGrantsToItselfWhenCreature`, `TestTabernacleGrantedSource` |
| “At the beginning of your upkeep” | Retained upkeep grant and `EventPlayerIsController`; the creature is the source, and its controller at triggering controls the nontargeted ability. | `TestTheTabernacleAtPendrellVale`, `TestTabernacleGrantedSource` |
| “destroy this creature” / sole `DestroyGathered("self")` caller | `Destroy(This)` uses the captured source incarnation. A returned or phased-out source is unaffected; a control change does not prevent destruction. Regeneration and indestructible apply. Removal of Tabernacle does not cancel an existing trigger. No damage LKI is needed. | `TestTabernacleGrantedSource` covers each case, including departure before trigger placement and after placement. |
| “unless you pay {1}” | `UnlessPaid(You, Costs(Mana("{1}")), ...)` offers the stored trigger controller an optional payment through the existing transaction. Declined or failed payment runs destruction. The typed body retains detrimental metadata and exposes payment and source children to inspection. | `TestTheTabernacleAtPendrellVale` covers payment, decline, and inability; `TestTabernaclePaymentAfterControlChange` checks payer and exact mana consumption; `TestTabernacleGrantedSource` checks runtime body metadata. |

Oracle source: `cards/legends/LEG.json`, with the full comment preserved above registration. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 118.12a (unless payment), 400.7 (new objects), 603.2b/603.3a (upkeep and trigger controller), and 701.8 (destruction).

Before migration, regressions failed for both leave-and-return cases and the creature Tabernacle grant. All are card-wiring corrections supported by existing engine APIs; no engine blocker was introduced or deferred. The surrounding legacy continuous trigger-grant constructor remains opaque at factory inspection and is not claimed as a completed typed grant family. Its runtime granted body is typed and inspected. Other grant callers and the broader string-bound action checklist remain open.

Validation: focused Tabernacle tests; `go test ./cards/legends -count=1`; `go test ./cards/... -short -count=1`; `go test ./pkg/mage/... -short -count=1`; `make test`; `go build ./...`; `go vet ./...`; and `make lint` all passed with zero failures (lint: zero issues).


## Completed Slice: Rakalite Delayed Source Return

- [x] Migrate Rakalite to `Activate(Costs(Mana("{2}")), ...)`, `TargetDamageRecipient`, `PreventNextDamage`, and `AtNext(EachEndStep, ReturnToHand(This))`.
- [x] Resolve its delayed source-identity audit with existing typed capture. Before migration, regressions returned a new incarnation when Rakalite left and returned before activation resolution or before the next end step. Both now pass without an engine change.
- [x] Replace the old self-name Oracle comment with the full text from `cards/antiquities/ATQ.json` ("Return this artifact ...").

| Oracle clause / legacy caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| "{2}: Prevent the next 1 damage that would be dealt to any target this turn." | Typed activation, generic mana cost, and damage-recipient binding; either player's creature or player and planeswalkers are eligible. The existing prevention executor supplies a one-damage shield. An illegal sole target prevents the entire ability from resolving. | `TestRakaliteTargetsAndPayment`: legal recipient categories, invalid artifact target, insufficient mana rollback, stack use, and prevention; `TestRakaliteDelayedSource`: exact consumption and illegal target. Existing `TestTypedPreventionConsumptionAndClone` and `TestPreventionShieldNewIncarnation` cover shield lifecycle. |
| "Return this artifact to its owner's hand at the beginning of the next end step." / Rakalite's `RegisterDelayedTriggerStep` and `ReturnToHandTarget` calls | One-shot `AtNext(EachEndStep, ReturnToHand(This))`; no target declaration; retain the activation source incarnation and controller; return to the owner even after a control change. An ability resolving after an end-step event waits for the next event. | `TestRakaliteDelayedSource`: return before activation resolution, before end step, and on delayed stack; absence, phasing, phase cycle, control change, end-step creation, opponent's end step, owner/controller distinction, clone isolation, and one-shot consumption. Existing `TestRakalite` covers repeated activations and shields. |

Rules contract: repository CR 400.7 and 603.7a–e. No new DSL constructor or engine fix was required. The positional return export remains for Prismari Charm, its engine test callers, and the typed adapter's shared executor. The generic legacy delayed-trigger export still has other callers. Neither is retired by this slice. A separate unprevented planeswalker damage audit finding is recorded above; general damage-source LKI remains open.

Validation: focused Rakalite regressions, `go test ./cards/antiquities -count=1`, `go test ./cards/... -short -count=1`, `make test`, `go build ./...`, `go vet ./cards/antiquities`, and `make lint` all passed with zero failures (lint: zero issues).

## Completed Slice: Aura Entry Tap

- [x] Migrate both production `TapAttachedCreature` callers, Paralyze and Waterknot, to `WhenEvent(Enters(This), Tap(AttachedPermanent))` through the author surface.
- [x] Retire `TapAttachedCreature` from engine and author exports and remove its private executor. Add negative compile fixtures for both surfaces.
- [x] Resolve entry-tap source and host identity with existing typed attachment history. Regressions failed before migration for source departure before trigger placement, source departure on the stack, source return, and host return. No engine gap or new constructor was required.

| Oracle clause / retired API | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Both Auras: “When this Aura enters, tap enchanted creature.” / both `TapAttachedCreature` calls | `WhenEvent(Enters(This), Tap(AttachedPermanent))`; one nontargeted stack trigger; read the current attachment at resolution or the departed source incarnation's last attachment. Control changes do not prevent tapping. | `TestAuraEntryTapLifetime`: unchanged attachment, moved Aura, departure before placement and resolution, returned Aura, returned host, phased host, phase cycle, control change, and clone isolation. The returned Aura's separate entry trigger is not resolved by the old-trigger assertion. |
| Typed attachment action | Reuse `TapSubject`, the incarnation-aware `AttachedPermanent` resolver, and `Game.TapPermanent`; preserve detrimental and tap metadata. | `TestAuraEntryTapTypedDefinitions`; existing `TestAttachedPermanentResolutionLifetime`, `TestComposableSavedTapLifetime`, and `TestTapPermanentRequiresTransition`; retired-export compile-fail fixtures. |
| “Enchant creature” and “Enchanted creature doesn't untap during its controller's untap step.” | Preserve Aura construction, casting targets, and the existing static untap restriction. Costs remain {B} and {1}{U}{U}. | Existing `TestParalyze` and `TestWaterknot`; universal registration validation. |
| Paralyze's upkeep payment clause | Retain the existing separate callback. Its payment, event-player capture, and attachment identity audit remain outside this entry-tap slice. | Existing `TestParalyze`; no claim that the surrounding callback is fully migrated. |

Oracle sources: `data/LEA.json` and `cards/jumpstart/JMP.json`; full Oracle comments precede the registrations. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 303.4m (enchanted object), 400.7 (new objects), and 608.2h (source information at resolution). General damage LKI and other attachment-event audits remain open. Unrelated legacy definitions and direct engine imports remain in both card files.

Validation: focused entry-tap regressions and Waterknot tests; both affected card packages; `go test ./cards/... -short -count=1`; `go test ./pkg/mage/... -short -count=1` (including registration validation and compile-fail fixtures); `make test`; `go build ./...`; `go vet ./...`; and `make lint` all passed with zero failures (lint: zero issues).


## Completed Slice: Whimsy Random Tap

- [x] Replace Whimsy's `ApplyToRandomPermanent(Tap(), ...)` branch with `ChooseRandomPermanent`, a saved permanent binding, and `Tap(From(chosen))` through `dsl`.
- [x] Resolve phased-out membership with the existing typed collection. Before migration, both phased-out regression cases consumed a random result despite having no eligible permanent. No engine fix or new constructor was required.

| Oracle clause / affected caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Whimsy: “Play X random fast effects.” / random-tap branch | Preserve the existing action table and X iterations. Choose one untapped artifact, creature, or land from either player at resolution, then tap its captured incarnation. The choice is not a target, so shroud does not exclude it. | `TestWhimsyRandomTapMembership`; existing `TestWhimsy_EachRandomActionCanBeSelected` and `TestWhimsy_PerformsXIndependentActionsInOrder` |
| Empty or phased-out candidates | `MatchingPermanents` excludes phased-out objects; an empty choice consumes no randomness. Already tapped permanents and enchantments without an eligible type are excluded. | `TestWhimsyRandomTapMembership`; existing `TestWhimsy_NoEligiblePermanentIsSafe` |
| Tap action and repeat selection | `Tap` uses the existing tap-transition event path. Later iterations recompute membership, so an already tapped permanent is not selected again. The composed body retains tap/detrimental metadata through `Sequence`. | `TestWhimsyRandomTapEventsAndRepeatedSelection` verifies one trigger, one random choice, and clone isolation; existing `TestTapPermanentRequiresTransition` and typed saved-tap contracts cover the shared executor. |

Oracle source: `cards/astral/PAST.json`; the complete comment remains above registration. The Oracle text does not enumerate the random action table; this slice preserves the repository's existing table and branch restrictions. Rules contract: repository CR 702.26 (phasing), 701.26 (tap and untap), and 400.7 (object identity). Other Whimsy branches remain outside this slice.

No export is retired: `ApplyToRandomPermanent` still has six Faerie Dragon production callers and one engine test. The legacy `Tap()` cost/effect surface also has other callers. Their separate audits and the broader string-bound action checklist remain open.

Validation: the phased-out regression failed before migration and passed afterward; `go test ./cards/astral -count=1`; `go test ./cards/... -short -count=1`; `go test ./pkg/mage/... -short -count=1`; `make test`; `go build ./...`; `go vet ./cards/astral`; and `make lint` passed with zero failures (lint: zero issues).


## Completed Slice: Temporary Regeneration Prohibition

- [x] Migrate both production `HurrJackalEffect` callers: Hurr Jackal and Faerie Dragon's Hurr Jackal branch.
- [x] Retire the helper from engine and author surfaces, remove its redundant text-only test case, and add negative compile fixtures for both exports.
- [x] Fix ability removal and typed detrimental metadata with failing engine regressions first. Reuse attribute storage, continuous durations, and regeneration replacement handling.
- [x] Correct source-departure duration and phased-out random membership with existing typed adapters. Card regressions failed before migration and pass afterward.

| Oracle clause / retired API and callers | Replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Hurr Jackal: “{T}: Target creature can't be regenerated this turn.” / `HurrJackalEffect` | `Activate(Costs(TapCost(This)), Until(EndOfTurn, GrantKeyword(creature, CantRegenerate)), WithTargets(creature))`; either player's creature; tap cost, stack resolution, target legality, and captured incarnation use existing contracts. The restriction remains after source departure and expires at cleanup. | `TestHurrJackal`, `TestHurrJackalTypedDefinition`, existing typed target and duration contract tests |
| Faerie Dragon: “{1}{G}{G}: Play a random effect.” / second helper caller | Preserve the existing action table. `ChooseRandomPermanent(MatchingPermanents(IsCreature), chosen)` followed by the same typed duration on `From(chosen)`; choose at resolution without targeting, so shroud does not exclude a creature. Exclude phased-out creatures and consume no random choice for an empty collection. | `TestFaerieDragonRegenerationProhibitionMembership`, `TestFaerieDragonRandomKeywordEffects`, `TestFaerieDragonTemporaryKeywordLifetime` |
| Ability removal and regeneration shields | `IsAbilityAttr(CantRegenerate)` is false. Removing abilities before or after the restriction does not remove it. Shields can still be created but cannot replace destruction while the prohibition applies. | `TestRegenerationProhibitionSurvivesAbilityRemoval`: both timestamp orders, ordinary flying removal, clone-local expiry, destruction with an active prohibition, and successful regeneration after expiry |
| Typed effect metadata | `Until` reports `OutcomeDetriment` with `GrantedKeyword: CantRegenerate`, preserving the retired helper's AI summary. | `TestComposableRegenerationProhibitionMetadata` |

Oracle sources: `cards/arabian/ARN.json` and `cards/astral/PAST.json`; full registration comments are preserved. Faerie Dragon's Oracle does not enumerate its random action table; this slice preserves the repository's table. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 611.1–611.2 (continuous restrictions and duration), 701.19c (regeneration prohibitions), 400.7 (object identity), and 702.26 (phasing).

No live callers or declarations of `HurrJackalEffect` remain outside negative compile fixtures. `ApplyToRandomPermanent` retains five Faerie Dragon callers and its engine test; it is not retired. Other random branches and broader continuous-effect audits remain open.

Validation: failing card and engine regressions passed after their fixes; affected card packages; `go test ./cards/... ./pkg/mage/... -short -count=1` (including registration validation and compile-fail fixtures); `make test`; `go build ./...`; `go vet ./...`; and `make lint` all passed with zero failures (lint: zero issues).

## Completed Slice: Tawnos’s Wand Blocking Restriction

- [x] Add and export `CantBeBlocked(PermanentRef)` as a continuous modification, with typed subject inspection, validation, and player/collection compile-fail fixtures.
- [x] Migrate Tawnos’s Wand and Faerie Dragon’s matching random branch; retire both author and engine exports of `TawnosWandEffect` and its redundant text-only test case.
- [x] Correct the shared legacy blocking executor after a failing engine regression: retain the restriction through ability removal and source departure. Reuse `composedLayerEffect`, captured incarnations, durations, and existing combat-restriction storage.
- [x] Exclude phased-out random candidates using the existing typed collection. Both ownership cases failed before migration because they consumed a random result with no eligible permanent.

| Oracle clause / retired API and callers | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Tawnos’s Wand: “{2}, {T}: Target creature with power 2 or less can't be blocked this turn.” / first `TawnosWandEffect` caller | `Activate(Costs(Mana("{2}"), TapCost(This)), Until(EndOfTurn, CantBeBlocked(creature)), WithTargets(creature))`; typed creature target with `HasPowerLTE(2)`; either controller; target legality and power rechecked on resolution. A later power increase does not end the restriction. | Existing `TestTawnossWand` and `TestTawnossWandNegative`; `TestTawnosWandResolution` checks exact payment, power before/after resolution, returned and phased targets, source departure, ability removal, and expiry. |
| Faerie Dragon: “{1}{G}{G}: Play a random effect.” / second helper caller | Preserve its action table. `ChooseRandomPermanent(MatchingPermanents(And(IsCreature, HasPowerLTE(2))), chosen)` followed by `Until(EndOfTurn, CantBeBlocked(From(chosen)))`; no target, so shroud does not exclude the choice. | `TestFaerieDragonWandMembership`: either controller, power boundary, noncreature, phased and empty choices, and random consumption. The old keyword-attribute assertion is replaced by actual block legality. |
| Blocking restriction and lifetime | Store an empty allowed-blocker set in the existing combat-restriction service. The restriction is not a granted ability. New blockers must also obey it; returned subjects are new objects. Preserve beneficial effect metadata and subject inspection. No damage LKI is required. | `TestBlockingProhibitionSurvivesAbilityRemoval` failed for removal-after and source departure before the fix; both timestamp orders now pass. `TestTypedBlockingProhibitionLifetime` covers incarnation, phasing, control, power, ability removal, later blockers, and clone-local cleanup. `TestBlockingProhibitionPreservesPrintedAbilityRemoval` ensures printed unblockable abilities remain removable. `TestBlockingRestrictionComposition` covers author validation and metadata. |

Oracle sources: `cards/antiquities/ATQ.json` and `cards/astral/PAST.json`; full registration comments are preserved. Faerie Dragon's Oracle does not enumerate its random action table; this slice preserves that table. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 400.7, 509.1b, and 611.2a–c.

No live callers or declarations of `TawnosWandEffect` remain outside negative compile fixtures. `MakeUnblockableUntilEndOfTurn` remains for Dwarven Warriors, Suspicious Bookcase, and Teleport; its executor now uses the corrected captured restriction. Teleport's pre-existing casting-timing gap and the remaining caller migrations are not resolved here. `ApplyToRandomPermanent` retains four Faerie Dragon production callers and its engine test. Other legacy definitions and imports remain outside this slice.

Validation: failing engine and phased-choice regressions passed after the fixes; affected card packages; `go test ./pkg/mage/... ./cards/... -short -count=1` (including universal registration validation and compile-fail fixtures); `make test`; `go build ./...`; `go vet ./...`; and `make lint` all passed with zero failures (lint: zero issues).


## Completed Slice: Activated Blocking Restrictions

- [x] Migrate the remaining two activated `MakeUnblockableUntilEndOfTurn` callers, Dwarven Warriors and Suspicious Bookcase, through `pkg/mage/dsl`.
- [x] Preserve exact Oracle text above registration, target restrictions, tap and mana costs, and end-of-turn duration.
- [x] Check the Warriors' power boundary before and after resolution, and strengthen the Bookcase test with an actual blocking attempt and tap-cost assertion before migration.

| Oracle clause / migrated caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Dwarven Warriors: “{T}: Target creature with power 2 or less can't be blocked this turn.” | `Activate(Costs(TapCost(This)), Until(EndOfTurn, CantBeBlocked(creature)), WithTargets(creature))`; `TargetPermanent("creature", And(IsCreature, HasPowerLTE(2)))`. Either player's creature; power is checked at selection and resolution. A later power increase does not end the restriction. | `TestDwarvenWarriors`; `TestDwarvenWarriorsPowerChange` covers an opponent's creature, tap payment without mana, and power increasing before versus after resolution. Existing `TestTawnossWandNegative` covers the shared power-filter selection boundary. |
| Suspicious Bookcase: “Defender” and “{3}, {T}: Target creature can't be blocked this turn.” | Preserve `WithKeyword(Defender)` and the artifact creature Wall definition. `Activate(Costs(Mana("{3}"), TapCost(This)), Until(EndOfTurn, CantBeBlocked(creature)), WithTargets(creature))` with `TargetPermanent("creature", IsCreature)`; no power or controller restriction. | `TestSuspiciousBookcase_MakesUnblockable` now attempts a block and checks source tapping. Universal `TestRegisteredDefinitions` validates the composed definition. |
| Shared activation, target, and duration contracts | Existing typed adapters use the activation payment and target-resolution services and the captured combat restriction. Beneficial metadata is preserved. No damage LKI is needed. | `TestBlockingRestrictionComposition`, `TestTypedBlockingProhibitionLifetime`, `TestBlockingProhibitionSurvivesAbilityRemoval`, and existing activation/target contracts cover validation, metadata, incarnation, phasing, source departure, ability removal, later blockers, and clone-local expiry. |

Oracle sources: `data/LEA.json` and `data/JMP.json`. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 602.2 (activation and payment), 608.2b (target recheck), 509.1b (blocking restrictions), and 611.2a–c (duration and affected objects).

No engine gap was found for these two activations, and no new DSL synonym is needed. `MakeUnblockableUntilEndOfTurn` retains one production caller, Teleport, and one engine regression caller. Its exports and executor remain until Teleport's existing casting-timing gap is resolved. The migrated files retain direct engine imports for unrelated definitions; the immutable author boundary remains open.

Validation: focused Warriors and Bookcase checks passed before migration; affected card packages passed after migration; `go test ./cards/... ./pkg/mage/... -short -count=1` (including registration validation and compile-fail fixtures), `make test`, `go build ./...`, `go vet ./...`, and `make lint` all passed with zero failures (lint: zero issues).

## Completed Slice: Teleport Casting Step and Typed Blocking

- [x] Confirm the engine gap with failing `TestSpellStepRestriction`: existing spell step timing was ignored in every casting path and in `GetCastableSpells`.
- [x] Check the existing action timing fields before choices and payment in normal and alternate/free casts, and in the shared card-copy casting path. No new timing subsystem is needed.
- [x] Extend `OnlyDuring` with the closed `StepTimingOption` role, accepted by spells and existing activation contexts. Inspect spell timing as `spell-step`; reject invalid steps and duplicate restrictions.
- [x] Migrate Teleport and the remaining engine regression caller, then retire the positional constructor, executor, and author export. No Go callers remain.

| Oracle clause / retired API | Replacement and semantics | Evidence |
| :--- | :--- | :--- |
| “Cast this spell only during the declare attackers step.” | `Spell(..., OnlyDuring(core.DeclareAttackers))`; either player's turn; normal, alternate, free, and card-copy casts obey the restriction before choices/payment. This does not grant sorcery timing permission. | `TestSpellStepRestriction` fails before the fix and passes afterward; `TestSpellStepDoesNotGrantSorceryTiming`; `TestTeleportCastingRestriction` |
| “Target creature can't be blocked this turn.” / Teleport's `MakeUnblockableUntilEndOfTurn` call | Typed creature target plus `Until(EndOfTurn, CantBeBlocked(creature))`; no controller or attacking restriction; captured battlefield incarnation; no damage LKI required; beneficial effect metadata preserved. | `TestTeleport`, `TestTeleportCastingRestriction`, `TestTypedBlockingProhibitionLifetime`, `TestSpellStepComposition` |
| `TestBlockingProhibitionSurvivesAbilityRemoval` positional helper call | Typed activation and target through the actual activation/resolution path; restriction survives ability removal and source departure, with independent clone expiry. | `TestBlockingProhibitionSurvivesAbilityRemoval` |
| Casting versus copying and resolution | Card copies retain the restriction; existing stack copies and later resolution do not recheck it. | `TestSpellStepCopyAndResolution` |
| Author-surface timing | `StepTimingOption` composes with `Spell`; existing activation, modal activation, and grant use remains supported. Activation-only turn options do not become spell options. | `TestSpellStepComposition`, `TestSpellStepOptionValidation`, existing combined/granted timing tests, `spell_activation_turn.go` compile-fail fixture |

Oracle source: `cards/legends/LEG.json`. Rules baseline: repository Comprehensive
Rules effective February 27, 2026, CR 601.3 and 707.10. Teleport's Oracle comment
is unchanged; its obsolete timing blocker is removed from the card and set TODO.
The surrounding Legends file still needs engine imports for unrelated cards.

Validation: focused spell-step, Teleport, blocking, and timing tests;
`go test ./pkg/mage/... ./cards/... -short -count=1`; `make test`;
`go build ./...`; `go vet ./...`; and `make lint` all passed with zero failures
(lint: zero issues).


## Completed Slice: Swords to Plowshares Saved-Value Migration

- [x] Migrate both production `SwordsToPlowsharesEffect` callers through `dsl` and retire the helper, author export, and redundant text-only test case. Add negative compile fixtures for both surfaces.
- [x] Preserve controller and effective power before exile with existing typed snapshots. No damage-source LKI or new binding type is needed.
- [x] Resolve the metadata gap with failing `TestComposableExileMetadata` first: the shared exile executor now reports `AITargetExile`, including through a sequence, without reporting regenerable destruction.
- [x] Resolve phased-out random membership with the existing typed collection. The corrected regression fails against the legacy branch because it consumes a random choice with no eligible creature; it passes after migration.

| Oracle clause / retired API and callers | Replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Swords to Plowshares: “Exile target creature. Its controller gains life equal to its power.” / first helper caller | `Spell` with a typed creature target; save `ControllerOf` and `PowerOf`, then `Exile` and `GainLife` from the saved values. Either player's creature; current power and controller at resolution; nonpositive power gains no life. | `TestSwordsToPlowsharesOutcomes`: own/opponent, control change, power increase, zero/negative power, indestructible, departed and returned targets; verifies exile destination and life totals. |
| Faerie Dragon: “{1}{G}{G}: Play a random effect.” / second helper caller | Preserve the existing action table. `ChooseRandomPermanent(MatchingPermanents(IsCreature), chosen)` followed by the same typed snapshots and actions. No target, so shroud does not prevent selection. Empty and phased-out collections consume no randomness. | `TestFaerieDragonSwordsMembership`: either controller, creature/noncreature, shroud, phasing, empty collection, life gain, and random consumption. |
| Typed definition and metadata | Existing `SnapshotPlayer`, `SnapshotNumber`, `Exile`, and `GainLife` executors; detrimental/exile metadata survives composition. | `TestSwordsToPlowsharesTypedDefinition`, `TestComposableExileMetadata`, `TestTypedExileComposition`, existing player/number snapshot lifetime and clone tests, and universal registration validation. |

Oracle sources: `data/LEA.json` and `cards/astral/PAST.json`; full Oracle comments precede registration. Faerie Dragon's Oracle does not enumerate its random effects; this slice preserves the repository's action table. Rules baseline: repository Comprehensive Rules effective February 27, 2026, CR 107.1b, 400.7, 608.2b/h, and 702.26b.

No live callers or declarations of `SwordsToPlowsharesEffect` remain outside negative compile fixtures. `ApplyToRandomPermanent` retains three production Faerie Dragon callers and its engine test. The separate Emeritus of Truce prepared spell, other string-bound exile callers, general damage LKI, and mutable author boundary remain open. Both migrated files retain engine imports for unrelated definitions.

Validation: focused card and engine regressions; affected card packages;
`go test ./cards/... ./pkg/mage/... -short -count=1` (including registration
validation and compile-fail fixtures); `make test`; `go build ./...`;
`go vet ./...`; and `make lint` all passed with zero failures (lint: zero issues).


## Completed Slice: Sorceress Queen Base Power and Toughness

- [x] Add `SetBasePowerToughness(PermanentRef, ResolutionValue, ResolutionValue)`
  through the existing composed layer executor and base-value override storage.
- [x] Add `TargetOtherThanSource()` over permanent target selection. Preserve
  either controller's creatures and allow a different creature with the same name.
- [x] Migrate both `SorceressQueenEffect` production callers and retire its
  engine and author exports and redundant text-only test case.
- [x] Correct Faerie Dragon's phased-out random membership with the existing
  typed collection. The regression failed for both owners before migration.

| Oracle clause / retired caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Sorceress Queen: “{T}: Target creature other than this creature has base power and toughness 0/2 until end of turn.” | `Activate(Costs(TapCost(This)), Until(EndOfTurn, SetBasePowerToughness(creature, Fixed(0), Fixed(2))), WithTargets(creature))`; creature target with `TargetOtherThanSource()`; either controller; normal target legality recheck. | Existing `TestSorceressQueen` checks 0/2 and expiry; `TestTypedOtherThanSourceTargets` checks source exclusion, same-name creatures, both controllers, and phasing. |
| Faerie Dragon: “{1}{G}{G}: Play a random effect.” / matching action-table branch | `ChooseRandomPermanent(WithoutPermanent(MatchingPermanents(IsCreature), This), chosen)` followed by typed base P/T setting. No target; shroud does not exclude a choice. Preserve the action table and source exclusion. | `TestFaerieDragonQueenMembership`: either owner, source, creature, noncreature, shroud, phasing, empty set, and random consumption. |
| Base values and lifetime | Reuse `BasePTOverride`, captured incarnations, and existing duration/clone handling. Preserve bonuses and counters. Preserve detrimental metadata without reporting base values as additive boosts. No damage LKI needed. | `TestTypedBasePowerToughness`: bonuses, counters, overwrite, returned objects, phasing cycle, source departure, control change, and clone-local expiry. `TestBasePowerToughnessComposition`: registration and metadata; player/collection and retired-export compile-fail fixtures. |

Oracle sources: `cards/arabian/ARN.json` and `cards/astral/PAST.json`. The Queen's
registration comment now matches the inventory's “this creature” wording.
Faerie Dragon's Oracle does not enumerate its random action table. Rules baseline:
repository Comprehensive Rules effective February 27, 2026, CR 400.7, 608.2b,
611.2a–c, 613.4b–c, and 702.26b.

No engine defect blocked these adapters. Source exclusion extends the existing
permanent target service with the restriction already used by creature targets.
`SetPTUntilEndOfTurn` remains for its other callers, and `ApplyToRandomPermanent`
retains two production callers. Broader layer and source-identity audits remain
open. Arabian card definitions already use `dsl`; Astral retains engine imports
for unrelated definitions.

Validation: the phased-choice regression failed before migration and passed
afterward; focused Queen checks; `go test ./pkg/mage/... ./cards/arabian
./cards/astral -short -count=1`; `make test` (including universal registration
validation and compile-fail fixtures); `go build ./...`; `go vet ./...`; and
`make lint` all passed with zero failures (lint: zero issues).


## Completed Slice: Base Power Activations

- [x] Add `SetBasePower(PermanentRef, ResolutionValue)` through the existing
  composed layer executor, with explicit duration, inspection, and metadata.
- [x] Migrate both production `SetPowerUntilEndOfTurn` callers and retire its
  constructor, executor, and author export. Replace the old UUID-based
  `SetBasePower` engine constructor, whose only caller was that executor.
- [x] Preserve base toughness during each layer pass, including after an earlier
  base P/T effect expires. No separate engine defect blocked this adapter.

| Oracle clause / retired caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Singing Tree: “{T}: Target attacking creature has base power 0 until end of turn.” | `Activate(Costs(TapCost(This)), Until(EndOfTurn, SetBasePower(creature, Fixed(0))), WithTargets(creature))`; creature filter includes `IsAttacking`; either controller; normal selection and resolution rechecks. | Existing `TestSingingTree`: opponent's attacker, unchanged toughness, zero combat damage, and expiry. |
| Island of Wak-Wak: “{T}: Target creature with flying has base power 0 until end of turn.” | Same activation with a creature-and-flying target filter; no attack restriction or mana cost. | Existing `TestIslandOfWakWak`: opponent's flyer, unchanged toughness, and nonflyer exclusion. |
| Base power, identity, and lifetime | Existing captured incarnation and duration handling; retain bonuses and counters, preserve current base toughness, and report detrimental metadata without additive boosts. No damage LKI is required. | `TestTypedBasePower`: bonuses, counters, overwrite, return, phasing cycle, source departure, control change, and clone-local expiry. `TestTypedBasePowerPreservesEarlierToughness`: prior base P/T setting and independent expiry. |
| Author surface | Typed modification composes with spells, activations, and static rules; invalid player and collection subjects fail compilation. | `TestBasePowerComposition`, new compile-fail fixtures, and universal registration validation. |

Oracle source: `cards/arabian/ARN.json`; both full registration comments are
unchanged. Rules baseline: repository Comprehensive Rules effective February 27,
2026, CR 400.7, 608.2b, 611.2a–d, and 613.4. The tests first demonstrated the
missing typed API, then passed after implementation. No live calls or declarations
of `SetPowerUntilEndOfTurn` remain outside negative compile fixtures. Both card
files use the `dsl` author surface for these definitions; direct engine imports
remain for unrelated definitions. `SetPTUntilEndOfTurn`, broader layer
audits, and the immutable author boundary remain open.

Validation: focused base-power and card tests; `go test ./cards/...
./pkg/mage/... -short -count=1` (including registration and compile-fail checks);
`make test`; `go build ./...`; `go vet ./...`; and `make lint` all passed with
zero failures (lint: zero issues).

## Completed Slice: Typed Combat Removal

| Existing caller or API | Replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Mijae Djinn: `SnapshotPermanent` / `RemoveFromCombatGathered` pipeline | `FlipCoin(You, OnLoss(Sequence(RemoveFromCombat(This), Tap(This))))`; attack trigger, no targets, stored controller and source incarnation; no damage LKI required. Exact Oracle comment from `data/ARN.json`. Neutral removal metadata is retained. | `TestMijaeDjinn`, `TestTypedCombatRemovalLifetime`, `TestCombatRemovalComposition` |
| Disharmony: positional removal instruction | `RemoveFromCombat(creature)` reads its existing typed attacking-creature target. Untap and control instructions are unchanged. Its casting-timing gap is resolved by the Disharmony casting-window slice below. | `TestDisharmony`, `TestCombatRemovalComposition` |
| `Combat.RemoveFromCombat` | Removes combat membership without clearing the attacker's blocked status (CR 509.1h, repository rules effective February 27, 2026). | `TestCombatRemovalKeepsAttackerBlocked` failed for sole-blocker removal before the fix. |
| Legacy explicit-unblocking effects | False Orders now names the retained engine helper directly; Ydwen Efreet retains its string-bound helper. Both use a private compatibility operation for their explicit unblocking clause. No legacy engine exports are retired while these callers remain. | `TestFalseOrders`, `TestYdwenEfreet` |

The public DSL `RemoveFromCombat` now requires a permanent reference. Composition tests cover saved references, targets, modes, loops, triggers, and activations; validation rejects absent producers, nil subjects, and undeclared targets. Lifetime tests cover absent, returned, phased, and phased-in references and clone isolation for attackers and blockers.

Remaining audits: Ydwen Efreet's full Oracle text includes an explicit unblocking clause omitted from its current comment; migrate that clause together with its turn-long blocking prohibition. False Orders still needs exact defending-player targeting, casting timing, historical sole-blocker tracking, and optional reassignment. The private compatibility operation preserves existing behavior and does not claim to solve those audits. Disharmony's control adapter remains separate; its timing is resolved below. These cards retain their existing legacy surfaces; Mijae Djinn remains DSL-only.

Validation: focused combat-removal and affected card tests; `go test ./pkg/mage/... ./cards/arabian ./cards/legends ./cards/limited -short -count=1`; `go test ./pkg/mage/dsl -count=1` (including player/collection compile-fail fixtures); `make test`; `go build ./...`; `go vet ./...`; and `make lint` all passed with zero failures. Lint reported zero issues.


## Completed Slice: Disharmony Casting Window

- [x] Extend `OnlyDuring(step, moreSteps...)` to accept a union of turn steps.
- [x] Use the existing action timing checks for spells, activations, and mana
  abilities. Keep the legacy single-step field and store additional steps in an
  immutable bit mask. Copy the author-supplied slice before storing the definition.
- [x] Migrate Disharmony's casting restriction and remove its resolved card and
  set timing markers. Preserve its full Oracle comment.

| Oracle clause / capability | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| “Cast this spell only during combat before blockers are declared.” | `OnlyDuring(core.BeginCombat, core.DeclareAttackers)`; either player's turn; no permission to bypass target selection or normal casting rules. | `TestDisharmonyCastingRestriction` failed before migration for main phases, declare blockers, both damage steps, and end of combat. It uses a synthetic attacker to isolate timing from target availability. |
| Multiple allowed steps | Existing shared action timing checks accept either step before payment in normal, free, alternate-cost, and card-copy casts. Spell availability uses the same restriction. | `TestSpellMultipleStepsRestriction` first failed to compile against the missing API, then passed across every step, both players, and all four paths; rejected casts preserve hand, mana, and stack. |
| Activation compatibility and definition lifetime | Multiple steps compose with `OnlyOnYourTurn` in either order and with activation limits. Later mutation of the caller's step slice does not change the definition. | `TestMultipleStepActivationRestrictions`; existing activation, granted-activation, and mana contracts. |
| Copies, resolution, validation, and inspection | Card copies retain timing; stack copies and resolution do not recheck it. Invalid additional steps are rejected. Inspection lists all allowed steps. | `TestSpellStepCopyAndResolution`, `TestSpellStepOptionValidation`, `TestSpellStepComposition`, universal registration validation. |
| Untap, combat removal, and temporary control | Existing typed attacking-creature target, `Untap`, `RemoveFromCombat`, and legacy control body are unchanged. No damage LKI is required. | `TestDisharmony` and the typed combat-removal contracts. |

Oracle source: `cards/legends/LEG.json`. Rules baseline: repository Comprehensive
Rules effective February 27, 2026, CR 507, 508, 509.1, and 601.3. Beginning of
combat remains an allowed step even when there is no legal attacking target.
The existing timing service needed only a multiple-step representation and shared
membership check; no casting subsystem was added. No legacy export is retired.
The typed control adapter and unrelated Legends engine imports remain open.

Validation: focused timing and Disharmony regressions; engine and Legends suites
with `-short -count=1`; `make test`; `go build ./...`; `go vet ./...`; and
`make lint` passed with zero failures.

## Completed Slice: Explore Land Allowance

Added `GrantLandPlays(PlayerRef, ResolutionValue)` over `Game.GrantExtraLandPlay`.
The subject resolves before the amount; absent subjects and nonpositive amounts
have no effect. The action adds permission for this turn, does not play a land,
and does not override timing or zone restrictions. Grants accumulate by player,
clone independently, and clear through existing turn cleanup. No engine blocker
was found. Static land allowances remain a separate family. Existing
`TestGrantExtraLandPlay_AllowsSecondLand`, `TestGrantExtraLandPlay_ResetAtEndOfTurn`,
and `TestGrantExtraLandPlay_Cumulative` establish the shared executor baseline.

| Previous caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Explore's allowance `FuncEffect` and `NewSpellAbility` | `Spell(Sequence(GrantLandPlays(You, Fixed(1)), Draw(You, Fixed(1))))`; no targets; stored resolving controller; no source LKI required; benefit and draw metadata preserved | `TestExplore_AllowsSecondLand` checks the second land and drawn Plains; `TestGrantLandPlaysLifetime`, `TestGrantLandPlaysAmountAndAbsence` |

| Oracle clause | Expression | Contract and evidence |
| :--- | :--- | :--- |
| You may play an additional land this turn. | `GrantLandPlays(You, Fixed(1))` | CR 305.2–305.3, repository rules effective February 27, 2026; cumulative player allowances, cleanup, clone isolation, and absent subjects covered by the adapter tests |
| Draw a card. | `Draw(You, Fixed(1))` after the grant | Existing draw executor; Explore integration test checks the card in hand |

Oracle text matches `cards/jumpstart/JMP.json`. No legacy export is retired:
`GrantExtraLandPlay` remains the shared executor. Explore's definition now uses
`dsl`; other definitions in its file retain engine imports for unrelated work.
`TestGrantLandPlaysValidation`, `TestLandPlayAllowanceComposition`, and two
compile-fail fixtures cover binding validation, metadata, lifecycle composition,
and rejection of permanent/collection subjects.

Verification: focused engine, DSL, and Jumpstart suites with `-short -count=1`;
`make test`; `go build ./...`; `go vet ./...`; and `make lint` all passed.
Lint reported zero issues.

## Completed Slice: Procrastinate Tap and Counters

`MultiplyValues(ResolutionValue, ResolutionValue)` extends the existing arithmetic
node. Both operands resolve in the current frame; inspection preserves their
binding reads and rejects missing producers, undeclared targets, and nil operands.
No new engine service or engine blocker was required.

| Migrated caller | Replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Procrastinate's `NewTargetedSpell`, positional `Tap()`, and counter callback | `Spell(Sequence(Tap(creature), AddCounters(creature, Stun, MultiplyValues(Fixed(2), ValueAtResolution(XValue())))), WithTargets(creature))`. One creature target, normal sorcery timing, tap followed by twice the announced X counters through the existing counter replacement service. No source LKI is required. Typed target identity, legality, and spell-copy X use existing services. | `TestProcrastinate`: X=0, X=3, and stun replacing the next untap. Oracle text matches `cards/secretsofstrixhaven/SOS.json`. |
| New typed multiplication | Existing arithmetic executor and named operand inspection; signed and zero products; saved-number reads; no mutable definition state. Arithmetic contributes no separate effect summary; tap and counter actions retain their existing metadata. | `TestComposableArithmeticSignedValues`, `TestComposableArithmeticValidation`, `TestComposableMultiplySavedValue`, `TestTypedMultiplyComposition`, and `multiply_legacy.go` compile-fail fixture. |

The focused tests were added before migration. Arithmetic tests first failed
because the constructor was absent; the expanded card scenarios passed on the
legacy definition. No shared export was retired: remaining legacy spell, tap,
and callback callers belong to later slices. The surrounding card file still
uses legacy engine imports for those definitions.

Oracle clause coverage: `Tap target creature` maps to the target declaration and
first sequence instruction; `Put twice X stun counters on it` maps to the second
instruction and multiplication; the reminder text maps to the existing stun
untap replacement. `TestProcrastinate` covers all three clauses.

Validation: focused arithmetic, DSL composition, compile-fail, and card tests;
full `make test`; `go build ./...`; and `make lint`.


## Completed Slice: Controlled Creature Team Boosts

| Retired API and callers | Replacement and semantics | Evidence |
| :--- | :--- | :--- |
| `BoostMatchingUntilEndOfTurn`: Fortify's two modes | `SpellModes` and `ModeOf`, each installing `Until(EndOfTurn, ModifyEachPT(ControlledPermanents(IsCreature), ...))`; choose exactly one mode; no targets | `TestFortify_PowerMode`, `TestFortify_ToughnessMode`; both assert the opponent's creature is unchanged |
| Same helper: Inspired Charge | `Spell` with the same typed collection and duration; +2/+1 for controlled creatures only. The old `AnyPermanent` argument implicitly selected creatures; the new filter states that restriction | `TestInspiredCharge`; controller boost and opponent exclusion |
| Same helper: Assault Formation's final activation | `Activate(Costs(Mana("{2}{G}")), Until(...))`; +0/+1 for controlled creatures; no targets or tap cost | `TestAssaultFormation_TeamBoost`; activated ability index, controller restriction, and end-of-turn expiry |
| Same helper: `Ch06 Rally` test fixture | Typed `Spell` and `Until` with `ControlledPermanents(And(IsCreature, HasColorFilter(White)))`; preserves the white-creature restriction | `pkg/mage/gametest` suite |

All collections are captured at resolution under CR 611.2c, and modifications
apply in layer 7c. No damage-source LKI is needed. `TestComposableContinuousCapture`
checks later entry and leave-and-return identity. `TestTeamBoostComposition`
checks public DSL composition, registration validation, beneficial outcome,
collection metadata, and +2/+1 value summaries. Existing engine services support
the slice without a rules fix. Oracle comments match `data/JMP.json`.

The helper has no remaining Go callers and is removed. Assault Formation's
unrelated combat-damage static ability and defender activation retain their
legacy definitions and direct engine import; this slice does not claim their
migration. The shared Jumpstart files retain imports for other legacy cards.

Validation: focused card and DSL tests, the Jumpstart package, engine and rules
harness tests, universal registration validation through `make test`,
`go build ./...`, `go vet ./cards/jumpstart`, and `make lint`.

## Completed Slice: Limited Edition Simple Self-Pump Activations

- [x] Replace the five simple `Boost(...).Targeting(ToSource())` activations with `Activate(Costs(Mana(...)), Until(EndOfTurn, ModifyPT(This, ...)))` through `pkg/mage/dsl`.
- [x] Restore complete Oracle comments from `data/LEA.json` immediately above each registration.
- [x] Verify stacking, expiration, clone isolation, phasing, and returned-object exclusion with `TestComposableSelfPumpLifetime`; verify public composition, registration, and beneficial single-subject P/T metadata with `TestSelfPumpActivationComposition`.
- [x] Reuse existing activation/payment and continuous-effect services; no engine blocker or new executor was needed.

| Oracle clause / callers | Typed expression and rules contract | Evidence |
| :--- | :--- | :--- |
| Frozen Shade: {B}, +1/+1 | `Costs(Mana("{B}"))`; `ModifyPT(This, Fixed(1), Fixed(1))` | Activation/payment contract suite; self-pump composition and lifetime tests |
| Granite Gargoyle: {R}, +0/+1 | `Costs(Mana("{R}"))`; `ModifyPT(This, Fixed(0), Fixed(1))` | Same contract tests; printed flying retained |
| Shivan Dragon, Wall of Fire: {R}, +1/+0 | `Costs(Mana("{R}"))`; `ModifyPT(This, Fixed(1), Fixed(0))` | Same contract tests; printed flying/defender retained |
| Wall of Water: {U}, +1/+0 | `Costs(Mana("{U}"))`; `ModifyPT(This, Fixed(1), Fixed(0))` | Same contract tests; printed defender retained |
| All five: until end of turn | `Until(EndOfTurn, ...)`, additive layer 7c bonus on the source incarnation, no targets or damage-source LKI | `TestComposableSelfPumpLifetime`, `TestRegisteredDefinitions` |

No export is retired: `Boost`, `ToSource`, and the legacy activation builder still have other callers. Dragon Whelp's activation-count and delayed-sacrifice clauses require a separate audit. Other legacy definitions still require the direct engine import in `cards/limited/creatures.go`.

Validation: focused adapter tests, engine and Limited Edition package tests, `make test`, `go build ./...`, and `make lint` pass with zero failures.


## Completed Slice: Targeted Instant P/T Boosts

- [x] Migrate Howl from Beyond and Righteousness to `Spell`, typed permanent targets, and `Until(EndOfTurn, ModifyPT(...))` through `pkg/mage/dsl`.
- [x] Restore complete Oracle comments from `data/LEA.json` immediately above both registrations.
- [x] Verify target restrictions, resolution-time X, duration, clone isolation, and returned-object exclusion. Existing target and continuous-effect services require no engine fix.

| Oracle clause / migrated caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Howl from Beyond: “Target creature gets +X/+0 until end of turn.” | `TargetPermanent("creature", IsCreature)` and `ModifyPT(creature, ValueAtResolution(XValue()), Fixed(0))` inside `Until(EndOfTurn, ...)`; either controller, instant timing, and {X}{B} retained. X is captured when the effect resolves. | `TestHowlFromBeyond` exercises casting with X=5; `TestTargetedBoostResolution` checks X=0 and X=5 and changes the resolution X afterward. |
| Righteousness: “Target blocking creature gets +7/+7 until end of turn.” | `TargetPermanent("blocking creature", And(IsCreature, IsBlocking))` and `ModifyPT(creature, Fixed(7), Fixed(7))` with the same duration; either controller, instant timing, and {W} retained. Blocking is a target restriction, not a duration condition. | `TestTargetedBoostTypedDefinitions` checks both controllers and excludes noncreatures and nonblockers; `TestTargetedBoostResolution` checks a creature removed from combat before resolution and a bonus that remains after combat removal. |
| Both legacy `NewTargetedSpell` / `Boost` calls | Typed targets capture the battlefield incarnation; layer 7c bonuses expire at end of turn and do not follow a returned card. No source LKI is needed. Beneficial, single-subject metadata remains available. | `TestTargetedBoostResolution` checks expiration, clone isolation, and leave-and-return; `TestTargetedBoostComposition` checks public spell, sequence, and modal composition and registration validation. |

Rules contract: repository Comprehensive Rules effective February 27, 2026,
CR 400.7 (new objects), 608.2b (target legality), 611.2c (affected objects), and
613 (power/toughness layers). Both typed-definition tests failed before migration;
the new resolution cases passed against the legacy definitions and preserve those
outcomes. No export is retired: the legacy targeted spell and boost constructors
still have other callers. Other definitions in `cards/limited/spells.go` retain
their engine import. No outstanding engine blocker is claimed resolved.

Validation: focused card and public DSL tests; full `make test`; `go build ./...`;
`go vet ./cards/limited`; and `make lint` passed with zero failures (lint: zero
issues).


## Completed Slice: Combat-Wide P/T Boost Spells

- [x] Migrate Army of Allah, Piety, and Morale to `Spell` with `Until(EndOfTurn, ModifyEachPT(...))` through `pkg/mage/dsl`.
- [x] Check the complete Oracle comments against `data/ARN.json` and `data/4ED.json`; all three already match.
- [x] Audit combat membership capture and duration using existing collection and continuous-effect services. No engine gap was found.

| Oracle clause / migrated caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Army of Allah: “Attacking creatures get +2/+0 until end of turn.” | `MatchingPermanents(And(IsCreature, IsAttacking))`, +2/+0 in layer 7c; instant timing and {1}{W}{W} retained | `TestArmyOfAllah`; `TestCombatBoostResolutionMembership` |
| Piety: “Blocking creatures get +0/+3 until end of turn.” | `MatchingPermanents(And(IsCreature, IsBlocking))`, +0/+3 in layer 7c; instant timing and {2}{W} retained | `TestPiety`; `TestCombatBoostResolutionMembership` |
| Morale: “Attacking creatures get +1/+1 until end of turn.” | Same attacking collection, +1/+1 in layer 7c; instant timing and {1}{W}{W} retained | `TestMorale`; `TestCombatBoostResolutionMembership` |
| All three legacy `NewSpellAbility` / `Boost` / `ToAllMatching` expressions | No targets or controller restriction. Capture the matching permanent incarnations at resolution. Removing a creature from combat before resolution excludes it; removing it afterward does not end the bonus. Later combat participants are excluded. End-of-turn expiry, clone isolation, and returned-object exclusion are preserved. No damage-source LKI is required. | `TestCombatBoostResolutionMembership` tests each spell with either player casting; `TestCombatBoostTypedDefinitions` checks typed duration, registration validity, no targets, beneficial mass metadata, and exact P/T summaries |

Rules contract: repository Comprehensive Rules effective February 27, 2026,
CR 400.7 (new objects), 611.2c (affected objects), and 613 (P/T layers).
The typed-definition checks failed on all three legacy forms before migration;
the new behavioral cases passed before and after migration. The existing
collection and duration APIs provide the complete DSL expression, so no new
constructor or executor is needed. No export is retired: the legacy builders,
boosts, and selectors still have other callers. The surrounding card files
retain legacy definitions and imports. No outstanding engine blocker is claimed
resolved.

Validation: focused migration and card tests; affected card packages; engine
suite; universal registration validation and full `make test`; `go build ./...`;
`go vet ./cards/arabian ./cards/fourthedition`; and `make lint` passed with zero
failures.


## Completed Slice: Targeted Artifact +2/+2 Activations

- [x] Migrate Coral Helm and Elven Lyre to `Activate`, `Costs`, `TargetPermanent`, and `Until(EndOfTurn, ModifyPT(...))` through `pkg/mage/dsl`.
- [x] Check full Oracle comments against `data/ATQ.json` and `data/FEM.json`; both match exactly.
- [x] Verify target scope, payment outcomes, timing, and metadata. Existing payment, target, and continuous-effect services support the slice without an engine fix.

| Oracle clause / migrated caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Coral Helm: {3}, discard a card at random | `Costs(Mana("{3}"), DiscardRandomCost(1))`; no tap cost or sorcery restriction | `TestCoralHelm` now checks the discarded card in the graveyard, its absence from hand, and the untapped Helm |
| Elven Lyre: {1}, {T}, sacrifice this artifact | `Costs(Mana("{1}"), TapCost(This), SacrificeSourceCost())`; payment removes the source before resolution | `TestElvenLyre_BoostsCreature` checks the boost and the sacrificed Lyre in the graveyard |
| Both: target creature gets +2/+2 until end of turn | `TargetPermanent("creature", IsCreature)` and `Until(EndOfTurn, ModifyPT(creature, Fixed(2), Fixed(2)))`; either controller; layer 7c; no source-presence condition or damage-source LKI | `TestArtifactBoostTypedDefinitions` checks typed targets, either controller, noncreature exclusion, instant timing, registration validation, and beneficial single-subject +2/+2 metadata; `TestCoralHelm` checks cleanup expiry |

The definition test failed on both legacy targets before migration. Strengthened
payment assertions passed before and after migration. Existing
`TestComposablePaymentAtomicity`, `TestComposableContinuousCapture`, and
`TestComposableSelfPumpLifetime` cover shared payment and continuous-effect
contracts, including captured incarnations, expiration, and clone isolation.
Rules contract: repository Comprehensive Rules effective February 27, 2026,
CR 602 (activation costs), 608.2b (target legality), 611.2c (affected objects),
and 613 (P/T layers).

No export is retired: `Boost`, legacy activation builders, and positional target
helpers still have other callers. The remaining artifact abilities retain their
existing forms. No outstanding engine blocker is claimed resolved.

Validation: focused migration and card tests; affected card packages and engine
suite; full `make test`; `go build ./...`; `go vet ./cards/antiquities
./cards/fallen_empires`; and `make lint` passed with zero failures.


## Completed Slice: Antiquities Self-Pump Activations

- [x] Migrate Atog and Dragon Engine to `Activate`, `Costs`, and `Until(EndOfTurn, ModifyPT(This, ...))` through `pkg/mage/dsl`.
- [x] Restore full Oracle comments from `data/ATQ.json` immediately above both registrations.
- [x] Correct Atog's artifact sacrifice selection. The old `SacrificeArtifactCost` excludes the source; Oracle permits sacrificing Atog if it is an artifact. The existing `SacrificeMatchingIncludingSourceCost` supports this requirement, so no engine fix is needed.

| Oracle clause / caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Atog: Sacrifice an artifact | `Costs(SacrificeMatchingIncludingSourceCost(IsArtifact, "Sacrifice an artifact"))`; one artifact controlled by the activating player, including the source if it is an artifact; no tap cost or timing restriction | `TestAtogSacrificePayment` checks own, opposing, phased, and nonartifact candidates, plus an artifact source; source sacrifice failed before migration. Successful payment is checked before resolution, with a tapped source during the opponent's upkeep. |
| Atog: This creature gets +2/+2 until end of turn | `Until(EndOfTurn, ModifyPT(This, Fixed(2), Fixed(2)))`; no targets; layer 7c source bonus; absent source has no effect | Existing `TestAtog` checks the bonus and expiry; `TestAtogSacrificePayment` checks source sacrifice and resolution. |
| Dragon Engine: {2}: This creature gets +1/+0 until end of turn | `Costs(Mana("{2}"))` and the same duration with `ModifyPT(This, Fixed(1), Fixed(0))`; no targets or tap cost | Existing `TestDragonEngine` checks repeated activation and expiry; shared payment contracts cover mana payment. |
| Both legacy activation / boost definitions | Typed source incarnation and duration; no damage-source LKI required; beneficial single-subject P/T metadata retained | `TestAntiquitiesSelfPumpTypedDefinitions` failed on both legacy effects before migration. Existing `TestComposableSelfPumpLifetime` covers stacking, expiry, clone isolation, phasing, and returned-object exclusion. |

Rules contract: repository Comprehensive Rules, CR 400.7 (new objects), 602
(activation and costs), 701 (sacrifice), and 613 (P/T layers). The existing cost
and continuous-effect services support this complete slice. No shared export
is retired: legacy activation, boost, and sacrifice helpers still have callers.
The remaining Antiquities abilities and outstanding engine blockers remain open.

Validation: `go test ./cards/antiquities -count=1`; card registration and engine
regressions through `make test`; `go build ./...`; `go vet ./cards/antiquities`;
and `make lint` passed with zero failures.


## Completed Slice: Global P/T Reductions

- [x] Migrate all four global reduction callers to `Until(EndOfTurn, ModifyEachPT(MatchingPermanents(IsCreature), ...))` through `pkg/mage/dsl`.
- [x] Check full Oracle comments against `data/LEG.json`, `data/4ED.json`, `data/DRK.json`, and `data/JMP.json`; all four match exactly.
- [x] Resolve the temporary P/T metadata gap: a negative summed fixed power or toughness modifier now reports `OutcomeDetriment`. `TestTemporaryPTMetadata` failed before the minimal `untilEffect.Properties` fix. No rules executor change is required.

| Oracle clause / migrated caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Hell Swarm: All creatures get -1/-0 until end of turn | `Spell` with the typed collection and -1/0 modifiers; instant timing and {B} retained | `TestHellSwarm` checks both players' creatures |
| Marsh Gas: All creatures get -2/-0 until end of turn | Same form with -2/0; instant timing and {B} retained | `TestMarshGas` checks both players and zero resulting power |
| Bone Flute: {2}, {T}: All creatures get -1/-0 until end of turn | `Activate(Costs(Mana("{2}"), TapCost(This)), ...)`; no targets or sorcery restriction; source departure does not end the effect | `TestBoneFlute` checks both players and now asserts the tap payment; shared transactional payment tests cover mana costs |
| Languish: All creatures get -4/-4 until end of turn | `Spell` with -4/-4; sorcery timing and {2}{B}{B} retained; the explicit creature filter replaces the legacy selector's implicit creature restriction | `TestLanguishWipes` checks both players' creatures die through state-based actions |
| All four legacy boost definitions | Both players' unphased creatures captured at resolution; layer 7c; no targets or source LKI; later entries and returned objects excluded; end-of-turn expiry | `TestGlobalReductionTypedDefinitions` checks typed duration, registration, targetlessness, exact signed modifiers, and detrimental mass metadata; existing `TestComposableContinuousCapture`, `TestComposableSelfPumpLifetime`, and `TestCombatBoostResolutionMembership` cover shared lifetime contracts |
| Typed temporary P/T metadata | Summed fixed negative power or toughness reports detriment for singular and collection effects, including mixed-sign modifiers. Positive and zero summaries retain benefit. No dynamic value evaluation is added | `TestTemporaryPTMetadata` covers both subject cardinalities and both duration constructors; `TestTypedTemporaryPTComposition` checks spell, activation, trigger, sequence, modal, and random-choice composition |

The four definition checks failed on legacy effects before migration. Existing
card behavior tests and the added Bone Flute tap assertion passed before and
after migration. These declarative cards reuse engine contracts and registration
validation rather than adding duplicate simulations. Rules baseline: repository
Comprehensive Rules effective February 27, 2026, CR 611.2a–c, 613, and 704.

No shared export is retired: `ToAllMatching` still serves Inniaz's flying-attacker
activation, and `Boost` and the legacy ability builders retain other callers.
The surrounding files still need their existing engine imports. General compound
metadata, dynamic-value classification, and other outstanding blockers remain open.

Validation: focused regression and card tests; affected card packages;
`go test ./pkg/mage/... -short -count=1`; full `make test` (including AI
simulations and card registration validation); `go build ./...`; `go vet ./...`;
and `make lint` passed with zero failures. Lint reported zero issues.


## Completed Slice: Targeted Land Boost Activations

- [x] Migrate Mishra’s Factory’s boost and Pendelhaven to `Activate`, `TargetPermanent`, `Costs(TapCost(This))`, and `Until(EndOfTurn, ModifyPT(...))` through `pkg/mage/dsl`.
- [x] Audit target restrictions, activation timing, payment, source departure, and duration. Existing engine services support this slice; no engine fix or new constructor is required.

| Oracle clause / previous caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Mishra’s Factory: “{T}: Target Assembly-Worker creature gets +1/+1 until end of turn.” / positional `Boost` | Typed target with `And(IsCreature, HasSubType("Assembly-Worker"))`; tap cost; +1/+1 at layer 7c until end of turn. Either controller’s creature qualifies; a noncreature Assembly-Worker does not. | Antiquities `TestLandBoostTypedDefinition`, `TestLandBoostResolution`; existing `TestMishrasFactory` |
| Pendelhaven: “{T}: Target 1/1 creature gets +1/+2 until end of turn.” / positional `Boost` | Typed creature target with current power and toughness each equal to 1; tap cost; +1/+2 at layer 7c until end of turn. No controller restriction. Check target characteristics again at resolution; the resulting 2/3 keeps the bonus. | Legends `TestLandBoostTypedDefinition`, `TestLandBoostResolution`; existing `TestPendelhaven` |
| Shared target and duration contract | Instant activation timing, one target, tap payment before stack resolution, target legality recheck, and boost survival after source departure. End-of-turn removal in a clone does not remove the original game’s bonus. Single-subject beneficial P/T metadata remains available. No source LKI is needed; the existing typed target and continuous-effect services retain incarnation identity. | Both new test pairs; existing `TestTypedTargetGroups`, `TestTargetIncarnationAfterReturn`, and `TestTypedTemporaryPTComposition` |

Oracle sources: `data/ATQ.json` and `cards/legends/LEG.json`. The full Factory Oracle comment now uses the inventory’s “This land” wording. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 602 (activation), 608.2b (target legality), 611.2a (duration), and 613 (continuous layers).

The new definition tests failed before migration because both legacy targets were untyped. Behavioral tests passed before and after migration. No export is retired: `Boost` and `WithActivatedAbility` still have other callers. Factory animation and mana definitions, Pendelhaven’s mana definition, other positional P/T callers, and unrelated legacy imports remain outside this slice.

Validation: focused land-boost and existing land tests; both affected card packages; `go test ./cards/... ./pkg/mage/... -short -count=1`; `go build ./...`; `go vet ./...`; `make test`; and `make lint` passed with zero failures (lint: zero issues).


## Completed Slice: Affa Guard Hound Entry Boost

- [x] Migrate Affa Guard Hound from `WithCastTarget` and positional `WithETBEffect(Boost(...))` to `WhenEvent(Enters(This), Until(EndOfTurn, ModifyPT(...)), WithTargets(...))` through `pkg/mage/dsl`.
- [x] Correct the card definition's premature target requirement. The existing casting, event, typed target, and continuous-effect services support the exact behavior; no engine fix or new constructor is required.

| Oracle clause / previous caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| “Flash (You may cast this spell any time you could cast an instant.)” | Preserve `WithKeyword(Flash)` and the printed {2}{W} cost. The creature spell has no targets and can be cast onto an empty battlefield. | `TestAffaGuardHoundFlash`, `TestAffaGuardHoundEntryTarget` |
| “When this creature enters, target creature gets +0/+3 until end of turn.” / Affa Guard Hound's `WithETBEffect` and positional `Boost` | `WhenEvent(Enters(This), ...)` with a typed creature target; mandatory target choice at trigger placement, after entry. Either controller's creature or the Hound itself qualifies. Entry without casting also triggers. | `TestAffaGuardHoundTypedDefinition`, `TestAffaGuardHoundEntryTarget`, `TestAffaGuardHoundWithoutCasting`; existing `TestAffaGuardHound` |
| Target identity, duration, and metadata | `Until(EndOfTurn, ModifyPT(creature, Fixed(0), Fixed(3)))`; layer 7c bonus, independent of source survival. A returned or phased-out target is illegal. Expire at cleanup. No damage-source LKI is needed; the existing typed target retains battlefield incarnation. Preserve single-subject beneficial +0/+3 metadata. | Entry-target cases cover source departure, target return, target phasing, and expiry. Definition test excludes noncreatures. `TestTypedTemporaryPTComposition` now includes +0/+3 across supported compositions; existing `TestTypedTargetCopyAndClone` and `TestComposableSelfPumpLifetime` cover independent cloned state. |

Oracle source: `cards/jumpstart/JMP.json`; the complete Oracle comment remains unchanged. Rules contract: repository Comprehensive Rules effective February 27, 2026, CR 115.1d and 603.3d (trigger targets), 400.7 (new objects), 608.2b (target legality), and 611.2a–c (continuous duration and capture).

`TestAffaGuardHoundEntryTarget` failed before migration in every case with “not enough targets for spell.” The corrected definition passes, including selecting a creature that entered after casting. The existing flash and cast tests now script the trigger choice explicitly. No exports are retired: `WithETBEffect`, `WithCastTarget`, and `Boost` have other callers. Other positional P/T definitions and unrelated legacy imports in the card file remain outside this slice.

Validation: focused Affa Guard Hound and typed P/T composition tests; `go test ./cards/jumpstart -count=1`; `go test ./cards/... ./pkg/mage/... -short -count=1`; `go build ./...`; `go vet ./...`; `make test`; and `make lint` passed with zero failures (lint: zero issues).


## Completed Slice: Shield Wall Team Boost

- [x] Migrate Shield Wall to `Spell(Until(EndOfTurn, ModifyEachPT(ControlledPermanents(IsCreature), Fixed(0), Fixed(2))))` through `pkg/mage/dsl`.
- [x] Check the complete Oracle comment against `cards/legends/LEG.json`; it matches exactly.
- [x] Audit controller scope, collection capture, duration, and metadata with existing engine contracts. No engine gap or new constructor is required.

| Oracle clause / previous caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| “Creatures you control get +0/+2 until end of turn.” / Shield Wall's positional `Boost` | `ControlledPermanents(IsCreature)` captures the resolving controller's creatures; `ModifyEachPT` installs +0/+2 at layer 7c with `Until(EndOfTurn, ...)`. No targets. Instant timing and {1}{W} remain unchanged. | `TestShieldWall` checks own and opposing creatures and now checks cleanup expiry; both cases passed before and after migration. |
| Temporary membership and identity | Capture members at resolution; later entries and returned objects do not receive the bonus. No source LKI is required. | Existing `TestComposableContinuousCapture` and `TestComposableSelfPumpLifetime` cover membership, incarnation, expiry, and clone isolation. |
| Typed spell and AI metadata | Explicit duration and beneficial mass +0/+2 summary; no target declaration. | `TestShieldWallTypedDefinition` checks registration validity, typed duration, targets, and metadata. |

Rules contract: repository Comprehensive Rules effective February 27, 2026,
CR 400.7 (new objects), 611.2a–c (duration and captured membership), and 613
(continuous layers). The explicit creature filter replaces the legacy boost's
implicit creature restriction. No helper is retired: `Boost`, `ToMatching`,
and `NewSpellAbility` still have other callers. The surrounding card file
retains its engine import for unmigrated definitions. No outstanding engine
blocker is claimed resolved.

Validation: focused Shield Wall definition and behavior tests; Legends package;
shared collection and lifetime tests; `go build ./...`; `go vet ./cards/legends`;
`make test`; and `make lint` passed with zero failures (lint: zero issues).


## Completed Slice: Jumpstart Single-Target P/T Spells

- [x] Migrate Aegis of the Heavens, Auger Spree, and Last Gasp from positional `NewTargetedSpell` / `Boost` to `Spell`, `TargetPermanent`, and `Until(EndOfTurn, ModifyPT(...))` through `pkg/mage/dsl`.
- [x] Check full Oracle comments and mana costs against `cards/jumpstart/JMP.json`; all three match exactly.
- [x] Audit target scope, incarnation, duration, and signed P/T metadata. Existing engine contracts support the slice; no engine fix or new constructor is required.

| Oracle clause / migrated caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Aegis of the Heavens: “Target creature gets +1/+7 until end of turn.” | One typed creature target, +1/+7 at layer 7c, explicit end-of-turn duration; instant and {1}{W} retained | `TestAegisOfTheHeavens` |
| Auger Spree: “Target creature gets +4/-4 until end of turn.” | Same target and duration, +4/-4; instant and {1}{B}{R} retained | `TestAugerSpree` checks lethal toughness reduction |
| Last Gasp: “Target creature gets -3/-3 until end of turn.” | Same target and duration, -3/-3; instant and {1}{B} retained | `TestLastGasp` checks lethal toughness reduction |
| Shared target and duration contract | Either player's creature qualifies. Target legality is checked at resolution; a returned card is a different incarnation. Effects expire at cleanup and retain independent clone state. Signed modifier metadata reports detriment for negative toughness, including mixed-sign changes. | `TestTargetIncarnationAfterReturn`, `TestComposableContinuousCapture`, `TestComposableSelfPumpLifetime`, `TestTemporaryPTMetadata`, and `TestTypedTemporaryPTComposition` |

These declarative spells reuse existing engine contracts, card behavior tests,
and registration validation. No new simulation tests are needed. No export is
retired: `NewTargetedSpell`, `TargetCreature`, and `Boost` retain other callers.
The remaining positional P/T migrations and recorded engine blockers stay open.

Validation: focused card and shared engine contract tests; Jumpstart package;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` passed with zero
failures (lint: zero issues).


## Completed Slice: Jumpstart Simple Self-Pump Activations

- [x] Migrate six positional `WithActivatedAbility` / `Boost` definitions to `Activate(Costs(Mana(...)), Until(EndOfTurn, ModifyPT(This, ...)))` through `pkg/mage/dsl`.
- [x] Check complete Oracle comments and printed mana costs against `cards/jumpstart/JMP.json`; all six match exactly.
- [x] Audit source scope, costs, duration, and metadata. Existing engine services support these abilities; no engine fix or new constructor is required.

| Oracle clause / migrated caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Dragon Hatchling, Furnace Whelp, Molten Ravager: “{R}: This creature gets +1/+0 until end of turn.” | `Costs(Mana("{R}"))` and `ModifyPT(This, Fixed(1), Fixed(0))` with explicit end-of-turn duration | `TestDragonHatchling_BoostsForR`, `TestFurnaceWhelp_BoostsForR`, `TestMoltenRavager_BoostsForR` |
| Minotaur Sureshot: “{1}{R}: This creature gets +1/+0 until end of turn.” | Same source modification with `Costs(Mana("{1}{R}"))` | `TestMinotaurSureshot_BoostsForOneR` |
| Tibalt’s Rager: “{1}{R}: This creature gets +2/+0 until end of turn.” | `Costs(Mana("{1}{R}"))` and `ModifyPT(This, Fixed(2), Fixed(0))` | `TestTibaltsRager_BoostsForOneR`; `TestSelfPumpActivationComposition` now includes +2/+0 |
| Jousting Dummy: “{3}: This creature gets +1/+0 until end of turn.” | `Costs(Mana("{3}"))` and the same +1/+0 source modification | `TestJoustingDummy_BoostsPower` |
| Shared source and duration contract | No targets, tap costs, timing restrictions, or activation limits. Each resolution adds a layer 7c bonus to the original source incarnation. Bonuses expire at cleanup, survive phasing, and remain independent in cloned games. No damage-source LKI is required. Benefit, single-subject, and P/T metadata remain available. | `TestComposableSelfPumpLifetime`, `TestSelfPumpActivationComposition`, `TestTemporaryPTMetadata`; universal `TestRegisteredDefinitions` |

Rules contract: repository Comprehensive Rules effective February 27, 2026,
CR 400.7 (new objects), 602 (activation), 611.2a (duration), and 613
(continuous layers). Existing card and shared contract tests passed before
migration. These declarative abilities need no new card simulations.

No exports are retired: `WithActivatedAbility`, `Boost`, and `ToSource`
still have other callers. Tibalt’s Rager’s damage trigger and the file’s
remaining legacy definitions and engine import stay outside this slice.
No outstanding engine blocker is claimed resolved.

Validation: focused card and shared source-lifetime tests before migration;
Jumpstart package; `go test ./pkg/mage/... -short -count=1`;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` passed with zero
failures (lint: zero issues).


## Completed Slice: Legends Simple Mana Self-Pump Activations

- [x] Migrate Carrion Ants, Wall of Opposition, Killer Bees, Arcades Sabboth, Pavel Maliki, and Vaevictis Asmadi to `Activate(Costs(Mana(...)), Until(EndOfTurn, ModifyPT(This, ...)))` through `pkg/mage/dsl`.
- [x] Check full Oracle comments against `cards/legends/LEG.json`; all six match exactly. Preserve printed costs, keywords, and other abilities.
- [x] Audit source scope, mana payment, duration, and metadata against existing contracts. No engine fix or new constructor is required.

| Oracle clause / migrated caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Carrion Ants: “{1}: This creature gets +1/+1 until end of turn.” | `Costs(Mana("{1}"))`, `ModifyPT(This, Fixed(1), Fixed(1))`, explicit end-of-turn duration | `TestCarrionAnts` verifies repeated activation |
| Wall of Opposition: “{1}: This creature gets +1/+0 until end of turn.” | Same form with generic mana and +1/+0 | Shared activation and lifetime contracts; registration validation; `TestWallOfOpposition` retains base characteristics coverage |
| Killer Bees: “{G}: This creature gets +1/+1 until end of turn.” | Same form with green mana and +1/+1 | `TestKillerBees` verifies repeated activation |
| Arcades Sabboth: “{W}: Arcades Sabboth gets +0/+1 until end of turn.” | Same form with white mana and +0/+1 | Shared activation and lifetime contracts; registration validation; existing `TestArcadesSabboth` covers the separate upkeep ability |
| Pavel Maliki: “{B}{R}: Pavel Maliki gets +1/+0 until end of turn.” | Same form with both required colors and +1/+0 | `TestPavelMaliki` |
| Vaevictis Asmadi: three separate {B}, {R}, and {G} +1/+0 activations | Three separate `Activate` definitions, each with its own mana cost and end-of-turn source bonus | `TestVaevictisAsmadi` verifies three cumulative activations and the separate upkeep ability |
| Shared source identity, duration, and metadata | No targets, tap costs, or activation limits. Each resolution adds a layer 7c bonus to the original source incarnation. Expire at cleanup; retain phasing and clone isolation. Beneficial, single-subject P/T metadata remains available. No damage-source LKI is needed. | `TestComposableSelfPumpLifetime`, `TestSelfPumpActivationComposition`, `TestTemporaryPTMetadata`, universal `TestRegisteredDefinitions` |

Rules contract: repository Comprehensive Rules effective February 27, 2026,
CR 400.7 (new objects), 602 (activation), 611.2a (duration), and 613
(continuous layers). Existing focused card and shared contract tests passed
before migration. These declarative abilities use the established engine
contracts and registration validation; no new simulation tests are needed.

No export is retired: `WithActivatedAbility`, `Boost`, `ToSource`,
`GenericCost`, and `ManaCostOf` retain other callers. Legends abilities with
additional costs, activation limits, or other effects remain separate work.
Arcades Sabboth's other abilities, Vaevictis Asmadi's upkeep ability, and the
file's remaining engine imports are outside this slice. No outstanding engine
blocker is claimed resolved.

Validation: focused card and shared source-lifetime tests before migration;
Legends package; `go test ./cards/... -short -count=1`;
`go build ./...`; `go vet ./cards/legends`; `make test`; and `make lint`
passed with zero failures (lint: zero issues).


## Completed Slice: Legends Targeted Tap P/T Activations

- [x] Migrate all three simple targeted tap P/T activations in Legends from positional `WithActivatedAbility` / `Boost` to `Activate`, `TargetPermanent`, `Costs`, and `Until(EndOfTurn, ModifyPT(...))` through `pkg/mage/dsl`.
- [x] Check full Oracle comments and printed costs against `cards/legends/LEG.json`; all three match exactly. Preserve other characteristics and abilities.
- [x] Audit targeting, payment, duration, and signed metadata against existing contracts. No engine fix or new constructor is required.

| Oracle clause / migrated caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Ghosts of the Damned: “{T}: Target creature gets -1/-0 until end of turn.” | `Costs(TapCost(This))`, typed creature target, `ModifyPT(creature, Fixed(-1), Fixed(0))`, explicit end-of-turn duration | `TestGhostsOfTheDamned` checks opposing creature reduction and now checks tap payment |
| Pradesh Gypsies: “{1}{G}, {T}: Target creature gets -2/-0 until end of turn.” | Same target and duration, `Costs(Mana("{1}{G}"), TapCost(This))`, -2/0 modifier | `TestPradeshGypsies` checks opposing creature reduction and now checks tap payment |
| Tuknir Deathlock: “{R}{G}, {T}: Target creature gets +2/+2 until end of turn.” | Same target and duration, `Costs(Mana("{R}{G}"), TapCost(This))`, +2/+2 modifier | `TestTuknirDeathlock` checks friendly creature boost, retained flying, and now checks tap payment |
| Shared target, cost, and lifetime contract | Either player's creature, including the source, qualifies. One target is chosen before transactional payment; normal instant activation timing and tap restrictions apply. Legality is rechecked at resolution. Layer 7c effects retain the target incarnation, survive source departure, expire at cleanup, and retain independent clone state. No damage-source LKI is needed. | `TestLandBoostResolution`, `TestTargetIncarnationAfterReturn`, `TestComposableContinuousCapture`, `TestComposableSelfPumpLifetime`; shared payment contracts and registration validation |
| Signed P/T metadata and composition | Single-subject benefit for +2/+2 and detriment for negative power; explicit duration remains inspectable | `TestTypedTemporaryPTComposition` now includes -1/0 and +2/+2 alongside -2/0 across activation, spell, trigger, sequence, modal, and random-choice contexts; `TestTemporaryPTMetadata` |

Rules contract: repository Comprehensive Rules effective February 27, 2026,
CR 400.7 (new objects), 602 (activation), 608.2b (target legality),
611.2a–c (duration and capture), and 613 (continuous layers). Existing card
behavior tests, including the added tap assertions, passed before migration.
These declarative abilities reuse shared contracts without new simulations.

No exports are retired: `Boost`, `WithActivatedAbility`, `TargetCreature`, and
legacy cost constructors retain other callers. Other Legends abilities and
the file's engine import remain outside this slice. No outstanding engine
blocker is claimed resolved.

Validation: focused card and shared lifetime tests; Legends package;
`go test ./cards/... ./pkg/mage/... -short -count=1`; `go build ./...`;
`go vet ./cards/legends`; `make test`; and `make lint` passed with zero
failures (lint: zero issues).


## Completed Slice: Limited Self-Pump Activations

- [x] Migrate Fire Drake, Vampire Bats, and Mindful Biomancer from positional `WithActivatedAbility` / `Boost` to `Activate`, `Costs(Mana(...))`, `Until(EndOfTurn, ModifyPT(This, ...))`, and `ActivationLimit` through `pkg/mage/dsl`.
- [x] Check complete Oracle comments and printed costs against `cards/thedark/DRK.json`, `cards/legends/LEG.json`, and `data/SOS.json`; all three match exactly.
- [x] Audit payment, activation limits, source lifetime, duration, and metadata. The existing engine supports this slice; no engine fix or new constructor is required.

| Oracle clause / migrated caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Fire Drake: “{R}: This creature gets +1/+0 until end of turn. Activate only once each turn.” | Red mana payment, +1/+0 source modification, explicit end-of-turn duration, `ActivationLimit(1)` | `TestFireDrake` |
| Vampire Bats: “{B}: This creature gets +1/+0 until end of turn. Activate no more than twice each turn.” | Black mana payment, +1/+0 source modification, explicit end-of-turn duration, `ActivationLimit(2)` | `TestVampireBats`, `TestVampireBatsActivationLimit` |
| Mindful Biomancer: “{2}{G}: This creature gets +2/+2 until end of turn. Activate only once each turn.” | Generic and green mana payment, +2/+2 source modification, explicit end-of-turn duration, `ActivationLimit(1)` | `TestMindfulBiomancer_ActivatedBoost`; entry life gain remains covered by `TestMindfulBiomancer_ETBGainsLife` |
| Payment and activation limits | No targets or tap costs; activation is allowed during the opponent's upkeep. Failed payment consumes no allowance. Successful activations count before resolution; an excess activation consumes no mana and adds no stack object. Resolution does not reset the limit. Cleanup removes the bonus and resets the limit independently in clones. | New `TestComposableLimitedPumpPayment`, tested before migration for both one and two activations; existing `TestCleanupHistoryCloneAndActivationLimits` and `TestCleanupHistoryPhasedActivationReset` |
| Source identity, duration, and metadata | Layer 7c bonus on the original source incarnation, with phasing and clone isolation. Beneficial single-subject P/T metadata remains available. | `TestComposableSelfPumpLifetime`; `TestSelfPumpActivationComposition` now includes +2/+2 and limits of one and two; universal `TestRegisteredDefinitions` |

Rules contract: repository Comprehensive Rules effective February 27, 2026,
CR 400.7 (new objects), 602 and 602.5 (activation restrictions),
611.2a (duration), and 613 (continuous layers). Existing card behavior tests
and the new shared payment test passed before migration. These declarative
abilities reuse shared contracts without new card simulations.

All three direct, mana-only self-pump callers with a hard per-turn activation
limit are migrated. No exports are retired: `Boost`, `WithActivatedAbility`,
`WithOncePerTurn`, and `WithMaxActivationsPerTurn` retain other callers.
Mindful Biomancer's separate entry trigger and remaining direct engine imports
stay outside this slice. No outstanding engine blocker is claimed resolved.

Validation: focused card and shared contract tests before migration; all three
affected card packages; `go test ./cards/... -short -count=1`;
focused shared activation, lifetime, cleanup, and composition tests;
`go build ./...`; `go vet ./cards/thedark ./cards/legends ./cards/secretsofstrixhaven`;
`make test`; and `make lint` passed with zero failures (lint: zero issues).


## Completed Slice: Jumpstart Triggered Team Boosts

- [x] Migrate both +1/+1 team-trigger bodies from positional `Boost` / `ToMatching` to `Until(EndOfTurn, ModifyEachPT(ControlledPermanents(IsCreature), Fixed(1), Fixed(1)))` through `pkg/mage/dsl`.
- [x] Migrate Inspiring Captain's entry trigger to `WhenEvent(Enters(This), ...)`.
- [x] Check full Oracle comments, mana costs, and base characteristics against `cards/jumpstart/JMP.json`; both match exactly.
- [x] Audit controller scope, collection capture, and duration against existing engine contracts. No engine fix or new constructor is required.

| Oracle clause / previous caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Inspiring Captain: “When this creature enters, creatures you control get +1/+1 until end of turn.” | Mandatory entry trigger, no targets; typed controlled-creature collection and +1/+1 modification with explicit end-of-turn duration | `TestInspiringCaptain` checks the Captain, another friendly creature, opposing-creature exclusion, and cleanup expiry |
| Inspiring Unicorn: “Whenever this creature attacks, creatures you control get +1/+1 until end of turn.” | Existing mandatory `AttacksTrigger` detects declaration of this source as an attacker; same typed collection and duration | `TestInspiringUnicorn` checks the Unicorn and another friendly creature, opposing-creature exclusion, and cleanup expiry |
| Collection and lifetime | Use the stored resolving controller. Capture creatures at resolution, including the source if still controlled. Later entries and returned objects do not receive a resolved bonus. Source departure does not end the bonus; expire at cleanup. No damage-source LKI is required. | `TestComposableContinuousCapture`, `TestComposableSelfPumpLifetime`, existing trigger contracts and universal `TestRegisteredDefinitions` |

Both existing card tests were extended with opposing-creature and cleanup
assertions and passed before migration. The declarative definitions reuse the
established collection and trigger services; no new simulation tests are needed.

No exports are retired: `Boost`, `ToMatching`, `EntersBattlefieldTrigger`, and
`AttacksTrigger` retain other callers. The typed attack-event adapter remains
open under the Triggers family; this slice completes both team-boost bodies,
not all trigger migrations. No outstanding engine blocker is claimed resolved.

Validation: affected card package; focused shared collection, lifetime, and
trigger tests; `go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` passed with zero
failures (lint: zero issues).


## Completed Slice: Angelic Page Combat Target

- [x] Migrate Angelic Page from positional `WithActivatedAbility` / `Boost` to `Activate`, `Costs(TapCost(This))`, `TargetPermanent`, and `Until(EndOfTurn, ModifyPT(...))` through `pkg/mage/dsl`.
- [x] Check full Oracle text, mana cost, base characteristics, and flying against `cards/jumpstart/JMP.json`; all match exactly.
- [x] Audit combat eligibility, target rechecks, source departure, duration, and metadata. Existing engine contracts support this slice; no engine fix or new constructor is required.

| Oracle clause / previous caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| “{T}:” | `Activate(Costs(TapCost(This)), ...)`; normal tap restrictions and instant activation timing | `TestAngelicPageCombatTargets` checks tap payment; existing tap-cost and summoning-sickness contracts |
| “Target attacking or blocking creature” | `TargetPermanent("creature", And(IsCreature, Or(IsAttacking, IsBlocking)))` with `WithTargets`; either player's combat creature; recheck at resolution | `TestAngelicPageCombatTargets` covers both controllers, attackers and blockers, idle-creature and noncreature exclusion, and removal from combat before resolution |
| “gets +1/+1 until end of turn.” | `Until(EndOfTurn, ModifyPT(creature, Fixed(1), Fixed(1)))`; layer 7c bonus persists after combat and source departure, expires at cleanup | `TestAngelicPageCombatTargets`, existing `TestAngelicPage`; shared `TestTargetIncarnationAfterReturn` and `TestComposableContinuousCapture` cover incarnation and clone contracts |
| Typed definition and metadata | One typed target, explicit duration, beneficial single-subject +1/+1 metadata | `TestAngelicPageTypedDefinition`, universal `TestRegisteredDefinitions` |

Rules contract: repository Comprehensive Rules effective February 27, 2026,
CR 602 (activation), 608.2b (target legality), 611.2a–c (duration and capture),
and 613 (continuous layers). New combat behavior cases passed before migration;
the definition test first failed for the legacy target and effect role.

No exports are retired: `Boost`, `WithActivatedAbility`, `TargetCreature`,
and combat filters retain other callers. Unrelated Jumpstart definitions and
engine imports remain. No outstanding engine blocker is claimed resolved.

Validation: Jumpstart package; `go test ./cards/... -short -count=1`;
`go build ./...`; `go vet ./cards/jumpstart`; `make test`; and `make lint`
passed with zero failures (lint: zero issues).

## Completed Slice: Osai Vultures Counter-Payment Boost

- [x] Migrate the activation from `WithActivatedAbility` / `Boost` to `Activate`, `Costs(RemoveCountersCost(Carrion, 2))`, and `Until(EndOfTurn, ModifyPT(This, ...))` through `pkg/mage/dsl`.
- [x] Check the full Oracle comment, mana cost, base characteristics, and flying against `cards/legends/LEG.json`; all match exactly.
- [x] Audit payment, timing, source identity, duration, and metadata. Existing counter-cost and continuous-effect services support this slice; no engine fix or new constructor is required.

| Oracle clause / previous caller | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| “Remove two carrion counters from this creature:” | `Costs(RemoveCountersCost(Carrion, 2))`; source-only payment before resolution, no tap or mana payment, no target or per-turn limit | `TestOsaiVulturesCounterPayment` covers zero, one, two, and four counters; unrelated charge counters; payment before resolution; repeated activations; tapped source on the opponent's upkeep; failed payment preserves counters and stack |
| “This creature gets +1/+1 until end of turn.” | `Until(EndOfTurn, ModifyPT(This, Fixed(1), Fixed(1)))`; each resolution adds a layer 7c bonus to the original incarnation | `TestOsaiVulturesCounterPayment` checks stacking and expiration; shared `TestComposableSelfPumpLifetime` covers return, phasing, and clone isolation |
| Typed definition and metadata | Explicit duration, untargeted instant-speed activation, beneficial single-subject +1/+1 metadata | `TestOsaiVulturesTypedActivation`, universal `TestRegisteredDefinitions` |

Rules contract: repository Comprehensive Rules effective February 27, 2026,
CR 602 (activation and payment), 611.2a–c (duration and capture), and 613
(continuous layers). The payment tests passed before migration; the definition
test first failed because the old effect had the `legacy` role.

No exports are retired: `Boost`, `WithActivatedAbility`, and
`RemoveCountersCost` retain other callers. Osai Vultures' legacy end-step
trigger and counter-addition body remain pending a separate trigger audit;
this slice does not claim their migration or resolve an outstanding engine blocker.

Validation: focused Osai Vultures tests; Legends package;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/legends`; focused engine payment and source-lifetime contracts;
`make test`; and `make lint` passed with zero failures (lint: zero issues).

## Completed Slice: Wyluli Wolf Targeted Tap Boost

- [x] Migrate Wyluli Wolf from `WithActivatedAbility` / positional `Boost` to `Activate`, `Costs(TapCost(This))`, `TargetPermanent`, and `Until(EndOfTurn, ModifyPT(...))` through `pkg/mage/dsl`.
- [x] Check the complete Oracle comment, mana cost, creature type, and base P/T against `cards/arabian/ARN.json`; all match exactly.
- [x] Audit target scope, payment, identity, duration, and metadata against existing adapters. No new constructor or engine fix is required.

| Oracle clause | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| “{T}:” | `Activate(Costs(TapCost(This)), ...)`; normal tap restrictions and instant activation timing, no mana cost or activation limit | Existing activation and tap-cost contracts; `TestWyluliWolf` |
| “Target creature” | `TargetPermanent("creature", IsCreature)` and `WithTargets`; either player's creature, including the Wolf; target legality and incarnation checked at resolution | Shared typed-target validation and `TestTargetIncarnationAfterReturn`; universal `TestRegisteredDefinitions` |
| “gets +1/+1 until end of turn.” | `Until(EndOfTurn, ModifyPT(creature, Fixed(1), Fixed(1)))`; layer 7c bonus on the captured target, independent of the source after resolution; expires at cleanup | `TestWyluliWolf` covers the bonus and expiry; `TestComposableContinuousCapture`, `TestComposableSelfPumpLifetime`, and `TestTemporaryPTMetadata` cover shared identity, lifetime, and metadata contracts |

This declarative ability uses established engine primitives and registration
validation, so no new simulation test is required. Existing card and shared
contract tests passed before migration. The complete Wyluli Wolf definition
is migrated. Legacy boost and activation exports still have other callers;
no export is retired and no outstanding engine blocker is claimed resolved.

Validation: focused Wyluli Wolf and shared target, lifetime, metadata, and
composition tests; Arabian Nights package; `go test ./cards/... -short -count=1`;
`go build ./...`; `go vet ./cards/arabian`; `make test`; and `make lint`
passed with zero failures (lint: zero issues).


## Completed Slice: Fallen Angel Sacrifice Boost

- [x] Migrate Fallen Angel from `WithActivatedAbility` / string-bound `Boost` to `Activate`, `Costs`, and `Until(EndOfTurn, ModifyPT(This, ...))` through `pkg/mage/dsl`.
- [x] Correct the legacy cost's exclusion of Fallen Angel itself with `SacrificeMatchingIncludingSourceCost(IsCreature, "Sacrifice a creature")`. No engine fix or new constructor is required.
- [x] Check the complete Oracle comment, mana cost, creature type, flying, and base P/T against `cards/legends/LEG.json`.

| Oracle clause / previous API | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| “Sacrifice a creature:” / `SacrificeCreatureCost` | `Costs(SacrificeMatchingIncludingSourceCost(IsCreature, "Sacrifice a creature"))`; controller's unphased creature, including the source; payment before resolution, no mana or tap cost | `TestFallenAngelCanSacrificeItself` first failed because the legacy cost excluded the source; now verifies payment and resolution after source departure on the opponent's upkeep with a tapped source. Existing `TestFallenAngel` covers sacrificing another creature; `TestMatchingSacrificeCostExcludesPhasedPermanents` covers phased exclusions. |
| “This creature gets +2/+1 until end of turn.” / `Boost(...).Targeting(ToSource())` | `Until(EndOfTurn, ModifyPT(This, Fixed(2), Fixed(1)))`; layer 7c bonus on the original incarnation; no target or source LKI needed | `TestFallenAngel`; shared `TestComposableSelfPumpLifetime` covers repeated bonuses, expiration, returned source, phasing, and clone isolation |
| Definition and metadata | Untargeted instant-speed activation; explicit duration; beneficial single-subject +2/+1 metadata | `TestFallenAngelTypedActivation` first failed on the legacy effect role; `TestTemporaryPTMetadata`; universal `TestRegisteredDefinitions` |

Rules contract: repository Comprehensive Rules effective February 27, 2026,
CR 602 (activation and costs), 400.7 (new objects), 611.2 (duration), and
613 (continuous layers). The existing engine supports the exact payment;
the defect was in card wiring. No outstanding engine blocker is claimed resolved.

Fallen Angel's complete definition is migrated. No export is retired because
`SacrificeCreatureCost`, `Boost`, and `WithActivatedAbility` retain other callers.
The changed Legends file retains unrelated legacy definitions and imports.

Validation: focused Fallen Angel and shared sacrifice, source-lifetime, and
metadata tests; Legends package; `go test ./cards/... -short -count=1`;
`go build ./...`; `go vet ./cards/legends`; `make test`; and `make lint`
passed with zero failures (lint: zero issues).

## Completed Slice: Dwarven Lieutenant Subtype Target Boost

- [x] Migrate the complete Dwarven Lieutenant definition from `WithActivatedAbility`, `Boost`, and positional `TargetCreature` to `Activate`, `Costs`, `TargetPermanent`, `WithTargets`, and `Until(EndOfTurn, ModifyPT(...))` through `pkg/mage/dsl`.
- [x] Check the complete Oracle comment, mana cost, creature types, and base P/T against `data/FEM.json`.
- [x] Audit target scope, payment, duration, identity, and metadata against existing adapters. No new constructor or engine fix is required.

| Oracle clause / previous API | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| “{1}{R}:” / `ManaCostOf` | `Activate(Costs(Mana("{1}{R}")), ...)`; instant activation timing, no tap cost or per-turn limit | Existing activation/payment contracts; `TestDwarvenLieutenant_BoostsDwarf` |
| “Target Dwarf creature” / `TargetCreature(HasSubType("Dwarf"))` | `TargetPermanent("dwarf", And(IsCreature, HasSubType("Dwarf")))` with `WithTargets`; either player's creature, including the source; both type restrictions checked at activation and resolution | Existing shared filter and typed-target contracts; `TestDwarvenLieutenant_BoostsDwarf` covers self-targeting; `TestTargetIncarnationAfterReturn` covers returned-target rejection |
| “gets +1/+0 until end of turn.” / `Boost` | `Until(EndOfTurn, ModifyPT(dwarf, Fixed(1), Fixed(0)))`; layer 7c bonus on the captured incarnation, independent of source presence and later subtype changes; beneficial single-subject metadata | `TestDwarvenLieutenant_BoostsDwarf`; `TestComposableSelfPumpLifetime`, `TestComposableContinuousCapture`, and `TestTemporaryPTMetadata` cover shared duration, incarnation, clone, and metadata contracts |

This declarative ability uses established primitives and universal
`TestRegisteredDefinitions` validation. Existing card and shared contract tests
passed before migration; no new simulation test is required. Rules contracts
are CR 602 (activation), 608.2b (target legality), 400.7 (new objects),
611.2 (duration), and 613 (layers), using the repository rules effective
February 27, 2026.

No export is retired: the legacy constructors retain other callers. The
Fallen Empires creature file retains unrelated legacy definitions and its
direct engine import. No outstanding engine blocker is claimed resolved.

Validation: focused Dwarven Lieutenant and shared target, lifetime, metadata,
and composition tests; Fallen Empires package; `go test ./cards/... -short -count=1`;
`go build ./...`; `go vet ./cards/fallen_empires`; `make test`; and `make lint`
passed with zero failures (lint: zero issues).


## Completed Slice: Icatian Targeted Mana Boosts

- [x] Migrate the complete Icatian Lieutenant and Icatian Priest definitions from `WithActivatedAbility`, positional `Boost`, and `TargetCreature` to `Activate`, `Costs`, `TargetPermanent`, `WithTargets`, and `Until(EndOfTurn, ModifyPT(...))` through `pkg/mage/dsl`.
- [x] Check full Oracle comments, mana costs, creature types, and base P/T against `data/FEM.json`; both match exactly.
- [x] Audit payment, target scope, duration, identity, and metadata against existing adapters. No new constructor or engine fix is required.

| Oracle clause / previous API | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Lieutenant: “{1}{W}: Target Soldier creature gets +1/+0 until end of turn.” | `Costs(Mana("{1}{W}"))`, `TargetPermanent("soldier", And(IsCreature, HasSubType("Soldier")))`, and explicit end-of-turn +1/+0 modification | `TestIcatianLieutenant_BoostsSoldier`; shared target and filter contracts |
| Priest: “{1}{W}{W}: Target creature gets +1/+1 until end of turn.” | `Costs(Mana("{1}{W}{W}"))`, `TargetPermanent("creature", IsCreature)`, and explicit end-of-turn +1/+1 modification | `TestIcatianPriest_BoostsTarget` |
| Activation and targeting | Instant activation timing, no tap cost or activation limit; either player's qualifying creature, including the source; target restrictions checked again at resolution | Existing activation and typed-target contracts; universal `TestRegisteredDefinitions` |
| Identity, duration, and metadata | Layer 7c bonus on the captured target incarnation, independent of source presence and later subtype changes; beneficial single-subject metadata | `TestTargetIncarnationAfterReturn`, `TestComposableContinuousCapture`, `TestComposableSelfPumpLifetime`, and `TestTemporaryPTMetadata` |

These declarative abilities use established primitives and registration
validation. Existing card and shared contract tests passed before migration;
no new simulation test is required. Rules contracts are CR 602 (activation),
608.2b (target legality), 400.7 (new objects), 611.2 (duration), and 613
(layers), using the repository rules effective February 27, 2026.

This completes the remaining Fallen Empires targeted mana-only P/T boost
activations. Legacy constructors retain other callers, so no export is retired.
Unrelated definitions and direct engine imports remain. No outstanding engine
blocker is claimed resolved.

Validation: focused card and shared target, lifetime, and metadata tests;
Fallen Empires package; `go test ./cards/... -short -count=1`;
`go build ./...`; `go vet ./cards/fallen_empires`; `make test`; and
`make lint` passed with zero failures (lint: zero issues).

## Completed Slice: Fallen Empires Orders

- [x] Migrate both activations on Order of Leitbur and Order of the Ebon Hand from `WithActivatedAbility`, string-bound keyword grants, and `Boost` to `Activate`, `Costs`, and typed source modifications through `pkg/mage/dsl`.
- [x] Check full Oracle comments, mana costs, creature types, protection colors, and base P/T against `data/FEM.json`; both match exactly.
- [x] Audit payment, source identity, duration, layers, and metadata against existing adapters. No new constructor or engine fix is required.

| Oracle clause | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| “{W}: This creature gains first strike until end of turn.” / corresponding `{B}` ability | `Activate(Costs(Mana(...)), Until(EndOfTurn, GrantKeyword(This, FirstStrike)))`; untargeted layer 6 grant on the source incarnation | `TestOrderOfLeitbur_FirstStrikePump`; `TestComposableKeywordLayerOrder`, `TestComposableKeywordSourceExit`; DSL composition tests |
| “{W}{W}: This creature gets +1/+0 until end of turn.” / corresponding `{B}{B}` ability | Separate activation with the printed mana cost and `Until(EndOfTurn, ModifyPT(This, Fixed(1), Fixed(0)))`; cumulative layer 7c bonus | `TestComposableSelfPumpLifetime`, `TestComposableContinuousCapture`, `TestTemporaryPTMetadata` |
| Activation and lifetime | Instant activation timing, no tap cost, target, or activation limit; explicit cleanup expiration and captured source incarnation | Shared activation and continuous-effect contracts; universal `TestRegisteredDefinitions` |
| Protection and base characteristics | Existing protection abilities, types, costs, and 2/1 base values preserved | Existing Order protection tests; Oracle comparison |

These declarative abilities use established primitives and registration
validation, so no new simulation test is required. Existing focused card and
shared lifetime, keyword, and metadata tests passed before migration. Both
card definitions are migrated. Legacy constructors retain other callers;
no export is retired and no outstanding engine blocker is claimed resolved.

Validation: focused Order and shared continuous-effect tests; Fallen Empires
package; `go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/fallen_empires`; `make test`; and `make lint` passed with
zero failures (lint: zero issues).


## Completed Slice: Fallen Empires Self-Keyword Activations

- [x] Migrate five activations on four cards from `WithActivatedAbility` and string-bound keyword grants to `Activate`, `Costs(Mana(...))`, and `Until(EndOfTurn, GrantKeyword(This, ...))` through `pkg/mage/dsl`.
- [x] Check full Oracle comments, printed mana costs, creature types, and base P/T against `data/FEM.json`; all match exactly.
- [x] Audit payment, source identity, duration, layers, and metadata against existing adapters. No new constructor or engine fix is required.

| Oracle clause | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Icatian Infantry: “{1}: This creature gains first strike until end of turn.” and its separate banding activation | Separate `Activate` definitions, each with `Mana("{1}")`; `GrantKeyword(This, FirstStrike)` or `GrantKeyword(This, Banding)` inside `Until(EndOfTurn, ...)` | `TestIcatianInfantry_FirstStrike`; shared keyword contracts include banding and clone-local expiry |
| River Merfolk: “{U}: This creature gains mountainwalk until end of turn.” | `Mana("{U}")` and temporary `GrantKeyword(This, Mountainwalk)` | `TestRiverMerfolk_GainsMountainwalk` |
| Vodalian Knights: “{U}: This creature gains flying until end of turn.” | `Mana("{U}")` and temporary `GrantKeyword(This, Flying)` | `TestVodalianKnights_GainsFlying`; existing Island restriction and state-trigger tests |
| Orcish Veteran: “{R}: This creature gains first strike until end of turn.” | `Mana("{R}")` and temporary `GrantKeyword(This, FirstStrike)` | `TestOrcishVeteran_FirstStrike`; existing blocking-restriction tests |
| Activation and lifetime | Untargeted, instant-speed activation; no tap cost or activation limit. Grant in layer 6 to the captured source incarnation; expire at cleanup. Ability removal follows timestamp order. | `TestComposableKeywordLayerOrder`, `TestComposableKeywordSourceExit`, `TestComposableSelfPumpLifetime`, `TestComposableContinuousCapture`, and universal `TestRegisteredDefinitions` |

These declarative activations reuse established engine primitives. Focused
card and shared contract tests passed before migration; no new simulation
tests are required. Existing keyword metadata is supplied by the typed
continuous modification. Rules contracts are CR 602 (activation), 400.7
(new objects), 611.2 (duration), and 613 (layers).

This completes the remaining Fallen Empires mana-only self-keyword activations.
The Infantry and Merfolk definitions are fully migrated. The Knights and
Veteran retain their separate static restrictions. The file retains unrelated
legacy definitions and its direct engine import. Legacy keyword and activation
exports still have callers; no export is retired and no outstanding engine
blocker is claimed resolved.

Validation: focused card and shared keyword/lifetime tests; Fallen Empires
package; `go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/fallen_empires`; `make test`; and `make lint` passed with
zero failures (lint: zero issues).


## Completed Slice: Fallen Empires Targeted Keyword Activations

- [x] Migrate Icatian Scout and Svyelunite Priest from string-bound keyword grants and positional activation targets to typed `Activate`, `Costs`, `TargetPermanent`, `WithTargets`, and `Until(EndOfTurn, GrantKeyword(...))` through `pkg/mage/dsl`.
- [x] Check full Oracle comments, mana costs, creature types, and base P/T against `data/FEM.json`; both match exactly.
- [x] Preserve Svyelunite Priest's controller-upkeep restriction with `OnlyDuring(core.Upkeep)` and `OnlyOnYourTurn()`.
- [x] Audit target scope, payment, timing, identity, duration, and metadata against the existing engine contracts. No engine gap or new constructor is required.

| Oracle clause / previous API | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Scout: “{1}, {T}: Target creature gains first strike until end of turn.” | `Costs(Mana("{1}"), TapCost(This))`, creature target, and temporary `GrantKeyword(creature, FirstStrike)` | `TestIcatianScout_GrantsFirstStrike` |
| Priest: “{U}{U}, {T}: Target creature gains shroud until end of turn.” | `Costs(Mana("{U}{U}"), TapCost(This))`, creature target, and temporary `GrantKeyword(creature, Shroud)` | `TestSvyelunitePriest_GrantsShroud` |
| Priest: “Activate only during your upkeep.” / `WithUpkeepOnly` | Combined step and active-player restrictions apply at activation, with no timing recheck at resolution | `TestCombinedActivationTiming` covers both option orders, controller upkeep, opponent upkeep, and other steps |
| `WithTarget(TargetCreature())` | `TargetPermanent("creature", IsCreature)` and `WithTargets(creature)`; either player's creature, including the source; recheck legality at resolution | Existing typed-target contracts and universal `TestRegisteredDefinitions` |
| Legacy `mage.GrantKeyword` | Typed layer 6 modification on the captured target incarnation; explicit cleanup expiration, source-independent lifetime, and beneficial keyword metadata | `TestTargetIncarnationAfterReturn`, `TestComposableKeywordSourceExit`, `TestComposableKeywordLayerOrder`, `TestComposableContinuousCapture`, and `TestComposableSelfPumpLifetime` |

These declarative abilities reuse established primitives. Existing card and
shared contract tests pass; no new simulation test is required. Rules contracts
are CR 602 (activation), 608.2b (target legality), 400.7 (new objects), 611.2
(duration), and 613 (layers), using the repository rules effective February 27,
2026.

Both card definitions are fully migrated. Legacy activation, target, keyword,
and upkeep helpers retain other callers, so no export is retired. The file
retains unrelated legacy definitions and its direct engine import. No open
engine blocker is claimed resolved.

Validation: focused card and shared keyword, target, lifetime, and combined
timing tests; Fallen Empires package; `go test ./cards/... -short -count=1`;
`go build ./...`; `go vet ./cards/fallen_empires`; `make test`; and
`make lint` passed with zero failures (lint: zero issues).


## Completed Slice: Jump and Pixie Queen Flying Grants

- [x] Migrate Jump and Pixie Queen's targeted flying grant to `TargetPermanent`, `WithTargets`, and `Until(EndOfTurn, GrantKeyword(...))` through `pkg/mage/dsl`.
- [x] Use `Spell` for Jump and `Activate` with explicit mana and tap costs for Pixie Queen. Preserve Pixie Queen's printed flying ability.
- [x] Verify complete Oracle comments and card characteristics against `data/LEA.json` and `data/LEG.json`.
- [x] Move the existing `TestJumpGrantsFlying` from `test.go` to `spells_test.go`, so Go discovers and runs both grant and expiration cases. Confirm both cases pass before migrating the definition.
- [x] Reuse the existing target, layer, duration, and payment services. No engine gap or new constructor is required.

| Oracle clause / previous API | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Jump: “Target creature gains flying until end of turn.” / `NewTargetedSpell`, positional `mage.GrantKeyword(Flying)` | `Spell`, typed creature target, and `Until(EndOfTurn, GrantKeyword(creature, Flying))` | `TestJumpGrantsFlying` checks flying and cleanup expiration |
| Pixie Queen: “Flying” | Keep `WithKeyword(Flying)` | Oracle inventory and universal `TestRegisteredDefinitions` |
| Pixie Queen: “{G}{G}{G}, {T}: Target creature gains flying until end of turn.” / `WithActivatedAbility`, `WithCost`, `WithTarget` | `Activate(Costs(Mana("{G}{G}{G}"), TapCost(This)), ...)` with the same typed target and flying body | `TestPixieQueen`, shared activation contracts, and registration validation |
| “Target creature” | Either player's creature, including Pixie Queen itself; check target legality again at resolution | Shared typed-target contracts, including `TestTargetIncarnationAfterReturn` |
| “until end of turn” | Layer 6 grant captures the recipient incarnation; source departure does not end it; cleanup removes it | `TestComposableKeywordSourceExit`, `TestComposableKeywordLayerOrder`, `TestComposableContinuousCapture`, and `TestComposableSelfPumpLifetime` |

The adapter reports a beneficial flying grant through existing inspection and
AI metadata. Rules contracts are CR 601–602, 608.2b, 400.7, 611.2, and 613,
using the repository rules effective February 27, 2026. Existing contract tests
cover these declarative definitions; no new simulation is needed.

Both card definitions are fully migrated. Legacy keyword and ability helpers
still have other callers, so no export is removed. Both files retain direct
engine imports for unrelated definitions. The broader action migration remains
open. Blaze of Glory remains separate: its existing definition omits the casting
window, defending-player target restriction, and mandatory blocking clause;
this slice does not migrate or approximate those requirements.

Validation: focused card and shared target/layer/lifetime tests; Limited and
Legends package tests; `go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/limited ./cards/legends`; `make test`; and `make lint` passed
with zero failures (lint: zero issues).

## Completed Slice: Royal Assassin and Exorcist Destruction Activations

- [x] Migrate Royal Assassin and Exorcist from positional `DestroyTarget`, `WithActivatedAbility`, and `WithTarget` to typed `Activate`, `Costs`, `TargetPermanent`, `WithTargets`, and `Destroy` through `pkg/mage/dsl`.
- [x] Verify full Oracle text, mana costs, creature types, and base P/T against `data/LEA.json` and `data/DRK.json`. Add Royal Assassin's full Oracle text above registration.
- [x] Move `TestAlphaRoyalAssassin` from `test.go` to `creatures_test.go` so Go discovers it. Confirm this test and `TestExorcist` pass before migration.
- [x] Audit the existing target, payment, and destruction adapters. No new constructor or engine fix is required.

| Oracle clause / previous API | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Royal Assassin: “{T}: Destroy target tapped creature.” | `Costs(TapCost(This))`, `TargetPermanent("creature", And(IsCreature, IsTapped))`, and `Destroy(creature)` | `TestAlphaRoyalAssassin` |
| Exorcist: “{1}{W}, {T}: Destroy target black creature.” | `Costs(Mana("{1}{W}"), TapCost(This))`, `TargetPermanent("creature", And(IsCreature, HasColorFilter(Black)))`, and `Destroy(creature)` | `TestExorcist` |
| Target scope and identity | Either player's qualifying creature; black includes multicolored black creatures. Target legality is checked again at resolution. Captured incarnations prevent destruction of a returned card. | Shared target contracts, including `TestTargetIncarnationAfterReturn`, `TestTypedTargetGroups`, and `TestTypedTargetCopyAndClone` |
| Destruction | Use the existing destruction executor; regeneration and indestructible still apply. The typed action preserves destruction metadata. | Shared regeneration and indestructible tests; `TestComposedRemovalMetadata` |

These are declarative single-action abilities. Existing engine contracts and
universal `TestRegisteredDefinitions` cover the shared primitives; no new
simulation is required. Both card definitions are fully migrated. Other callers
still use the legacy exports, so no export is retired. The broader action
migration and all unrelated engine blockers remain open.

Validation: both card packages, focused shared target and destruction contracts,
`go test ./cards/... -short -count=1`, `go build ./...`,
`go vet ./cards/limited ./cards/thedark`, `make test`, and `make lint` passed
with zero failures (lint: zero issues).

## Completed Slice: Merfolk Assassin and Savaen Elves Destruction Activations

- [x] Migrate both complete definitions from positional destruction and activation helpers to `Activate`, `Costs`, `TargetPermanent`, `WithTargets`, and `Destroy` through `pkg/mage/dsl`.
- [x] Verify full Oracle comments, mana costs, creature types, and base P/T against `data/DRK.json`.
- [x] Resolve the Aura target-filter blocker: `IsAuraOnLand` now requires the Aura subtype as well as the enchantment type and attachment to a current land. `TestIsAuraOnLand/non-aura_enchantment_on_land` failed before the minimal engine fix.

| Oracle clause / previous API | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Merfolk Assassin: “{T}: Destroy target creature with islandwalk.” | `Costs(TapCost(This))`, `TargetPermanent("creature", And(IsCreature, HasKeywordFilter(Islandwalk)))`, and `Destroy(creature)` | `TestMerfolkAssassin` |
| Savaen Elves: “{G}{G}, {T}: Destroy target Aura attached to a land.” | `Costs(Mana("{G}{G}"), TapCost(This))`, `TargetPermanent("aura", IsAuraOnLand)`, and `Destroy(aura)` | `TestSavaenElves`; `TestIsAuraOnLand` covers Aura, enchantment, attachment, land, and host-presence requirements |
| Target scope and identity | Either player's qualifying permanent; no controller restriction on the Aura or its host. Recheck filters at resolution and reject returned target incarnations. | Existing typed-target contracts: `TestTypedTargetGroups`, `TestTypedTargetCopyAndClone`, and `TestTargetIncarnationAfterReturn` |
| Destruction and inspection | Existing destruction executor preserves regeneration, indestructible, and destruction metadata. Instant activation timing; mana and tap costs remain payments. | `TestComposedRemovalMetadata`, shared engine contracts, and universal `TestRegisteredDefinitions` |

The shared filter correction also applies to the existing Pyramids target;
its land-or-Aura choice is unchanged. No new constructor is needed. These
single-action definitions use established primitives, so existing card tests
and the new engine filter regression provide the required coverage. Rules
contracts are CR 303.4 (Aura subtype), 602 (activation), 608.2b (target
legality), and 400.7 (new objects), using the repository Comprehensive Rules.

Both card definitions are fully migrated. Legacy destruction, activation,
and target helpers retain other callers, so no export is retired. The broader
action migration and unrelated engine blockers remain open.

Validation: focused regression and typed-target tests; The Dark package;
`go test ./pkg/mage/... -short -count=1`; `go test ./... -short -count=1`
(including all card registration checks); `go build ./...`; `go vet ./...`;
`make test`; and `make lint` passed with zero failures (lint: zero issues).


## Completed Slice: Alabaster Mage and Emerald Dragonfly Keyword Activations

- [x] Migrate both complete definitions to typed `Activate`, `Costs`, `GrantKeyword`, and `Until(EndOfTurn, ...)` through `pkg/mage/dsl`.
- [x] Replace Alabaster Mage's positional target with `TargetPermanent`, `TargetYouControl`, and `WithTargets`. Use `This` for Emerald Dragonfly's untargeted grant.
- [x] Verify full Oracle comments, mana costs, creature types, and base P/T against `data/JMP.json` and `data/LEG.json`.
- [x] Audit existing target, payment, identity, duration, and layer adapters. No engine gap or new constructor is required.

| Oracle clause / previous API | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| Alabaster Mage: “{1}{W}: Target creature you control gains lifelink until end of turn.” / `WithActivatedAbility`, positional keyword grant | `Costs(Mana("{1}{W}"))`, a creature target with `TargetYouControl()`, and `Until(EndOfTurn, GrantKeyword(creature, Lifelink))` | `TestAlabasterMage`; typed target contracts |
| “Target creature you control” / `TargetControlledCreature` | Includes the Mage itself; checks controller and creature type at announcement and resolution. A control change after resolution does not end the grant. | `TestTypedTargetYouOwnCandidates`, `TestTypedTargetYouOwnActivation` cover the shared controller option; `TestTypedTargetGroups` |
| Emerald Dragonfly: “Flying” | Keep `WithKeyword(Flying)` | Oracle inventory and universal `TestRegisteredDefinitions` |
| Emerald Dragonfly: “{G}{G}: This creature gains first strike until end of turn.” / source selector | `Costs(Mana("{G}{G}"))`, `GrantKeyword(This, FirstStrike)`, and explicit `Until(EndOfTurn, ...)`; no target, tap cost, or activation limit | `TestEmeraldDragonfly`; `TestComposableSelfPumpLifetime` |
| Temporary keyword grants | Layer 6 grants capture the recipient incarnation, survive source departure, expire at cleanup, and expose keyword metadata. Returned cards are new objects. | `TestComposableKeywordLayerOrder`, `TestComposableKeywordSourceExit`, `TestComposableContinuousCapture`, `TestTargetIncarnationAfterReturn`, `TestTypedTargetCopyAndClone` |

Both are declarative single-action abilities that reuse proven primitives.
Existing card and shared engine tests cover the migration; no new simulation
is required. Rules contracts are CR 602, 608.2b, 400.7, 611.2, and 613.
The damage-source LKI blocker remains open; this slice grants lifelink through
the existing layer service and does not change damage execution.

Both definitions are fully migrated. Legacy keyword and activation helpers
retain other callers, so no export is retired. The files retain engine imports
for unrelated definitions. The broader action migration remains open.

Validation: focused card and shared target/layer/lifetime tests; Jumpstart and
Legends package tests; `go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart ./cards/legends`; `make test`; and `make lint` passed
with zero failures (lint: zero issues).

## Completed Slice: Scarwood Hag and Typed Keyword Removal

- [x] Add `RemoveKeyword(PermanentRef, Attr)` as a typed layer 6 modification. Use the existing `RevokeAttr` service; no engine gap or rules-service change is required.
- [x] Migrate both Scarwood Hag activations to `Activate`, `Costs`, `TargetPermanent`, `WithTargets`, and explicit `Until(EndOfTurn, ...)` through `pkg/mage/dsl`.
- [x] Verify the complete Oracle comment, mana cost, creature type, and base P/T against `data/DRK.json`.
- [x] Cover keyword ordering, removal of multiple grants, unrelated keywords, cleanup, source departure and return, recipient phasing and return, clone isolation, metadata, binding validation, public composition, and compile-time category rejection.

| Oracle clause / previous API | Typed replacement and semantics | Evidence |
| :--- | :--- | :--- |
| “{G}{G}{G}{G}, {T}: Target creature gains forestwalk until end of turn.” / positional keyword grant | `Costs(Mana("{G}{G}{G}{G}"), TapCost(This))`, a creature target, and `Until(EndOfTurn, GrantKeyword(creature, Forestwalk))` | `TestScarwoodHag/grant_forestwalk` |
| “{T}: Target creature loses forestwalk until end of turn.” / `RevokeKeyword(Forestwalk)` | `Costs(TapCost(This))` and `Until(EndOfTurn, RemoveKeyword(creature, Forestwalk))` | `TestScarwoodHag/lose_forestwalk`; `TestComposableKeywordRemoval` |
| Target scope and lifetime | Either player's creature, including the Hag. Shared target services recheck legality at resolution. Removal captures the recipient incarnation; source departure does not end it, phasing preserves it, and a returned recipient is unaffected. | `TestComposableKeywordRemovalLifetime`; existing typed-target contracts |
| Layer order and duration | Remove every instance of the selected keyword in layer 6. A later grant restores it; a later removal removes it again. Cleanup restores printed abilities. | `TestComposableKeywordRemoval` |
| Author surface and inspection | Accept a permanent reference, reject players and collections, validate binding scope, report removal as detrimental, and exclude it from the keyword-grant AI shortcut. | `TestComposableKeywordRemovalMetadata`, `TestTypedKeywordRemovalComposition`, `TestClosedExpressionRoles` |

The adapter contract tests initially failed to compile because `RemoveKeyword`
did not exist. The implementation adds syntax over the established layer and
incarnation services. No engine blocker is claimed resolved. Rules contracts
are CR 400.7, 602, 608.2b, 611.2, 613.1f, 613.7, and 702.14.

Scarwood Hag is fully migrated. Legacy `RevokeKeyword` still has callers in
Arabian Nights and Legends, so its export remains. The broader continuous
modification and action migrations remain open. Existing card tests cover this
declarative wiring; new behavior contracts reside in the shared adapter tests.

Validation: focused adapter, public API, compile-failure, and Scarwood Hag tests;
The Dark package; `go test ./pkg/mage/... -short -count=1`;
`go test ./cards/... -short -count=1` (including registration validation);
`go build ./...`; `go vet ./cards/thedark`; `make test`; and `make lint`
passed with zero failures (lint: zero issues).


## Completed Slice: Radjan Spirit Flying Removal

- [x] Migrate Radjan Spirit's complete definition to typed `Activate`, `Costs`, `TargetPermanent`, `WithTargets`, and `Until(EndOfTurn, RemoveKeyword(...))` through `pkg/mage/dsl`.
- [x] Verify the full Oracle comment, mana cost, creature type, and base P/T against `data/LEG.json`.
- [x] Audit target scope, payment, layer order, duration, identity, and inspection against existing contracts. No new constructor or engine fix is required.

The Oracle ability, “{T}: Target creature loses flying until end of turn,”
uses `Costs(TapCost(This))` and a creature target with no controller restriction.
Either player's creature, including Radjan Spirit itself, qualifies. The target
need not have flying. Target legality is checked again at resolution. The
layer 6 removal removes all flying instances; a later grant can restore flying.
The resolved removal survives source departure, expires at cleanup, and does
not affect a returned recipient incarnation. Inspection reports a detrimental
removal, not a keyword grant.

`TestRadjanSpirit` passed before and after migration. Existing shared contracts
`TestComposableKeywordRemoval`, `TestComposableKeywordRemovalLifetime`,
`TestComposableKeywordRemovalMetadata`, `TestTypedKeywordRemovalComposition`,
and `TestTargetIncarnationAfterReturn` cover the adapter. This declarative
single-action ability needs no duplicate simulation test. Rules contracts are
CR 400.7, 602, 608.2b, 611.2, and 613.

Radjan Spirit is fully migrated. `RevokeKeyword` retains production callers,
so no export is retired. The broader action migration remains open. Tolaria
and Shelkin Brownie still require exact “bands with other” support; removing
ordinary banding alone does not satisfy their Oracle text. No unrelated engine
blocker is claimed resolved.

Validation: focused card and shared adapter tests; Legends package tests;
`go test ./cards/... -short -count=1` (including registration validation);
`go build ./...`; `go vet ./cards/legends`; `make test`; and `make lint`
passed with zero failures (lint: zero issues).


## Completed Slice: Spinal Villain Typed Destruction

- [x] Migrate Spinal Villain from positional destruction and activation helpers to `Activate`, `Costs`, `TargetPermanent`, `WithTargets`, and `Destroy` through `pkg/mage/dsl`.
- [x] Verify the complete Oracle comment, mana cost, creature type, and base P/T against `data/LEG.json`.
- [x] Audit target scope, tap payment, resolution legality, object identity, and destruction behavior against existing adapters. No engine gap or new constructor is required.

The Oracle ability, “{T}: Destroy target blue creature,” uses
`Costs(TapCost(This))` and `TargetPermanent("creature", And(IsCreature,
HasColorFilter(Blue)))`. Either player's blue creature qualifies, including a
multicolored creature or Spinal Villain itself if it becomes blue. There is no
mana payment, opponent restriction, or regeneration prohibition. Target legality
is checked again at resolution; a returned creature is a different incarnation.
The typed action delegates to the existing destruction service, preserving
regeneration, indestructible, and detrimental effect metadata.

`TestSpinalVillain` passed before and after migration. Shared contracts
`TestTypedTargetGroups`, `TestTypedTargetValidation`,
`TestTypedTargetCopyAndClone`, `TestTargetIncarnationAfterReturn`,
`TestTypedRegenerationConsumption`, and `TestComposableActivationRestrictions`
cover the reused services. This declarative single-action ability needs no
new simulation test. Rules contracts are CR 400.7, 602, 608.2b, and 701.8.

Spinal Villain is fully migrated. `DestroyTarget` and the legacy activation
helpers retain other production callers, so no export is retired. The file
retains its direct engine import for unrelated definitions. The broader action
migration remains open; no engine blocker is claimed resolved.

Validation: focused card and shared adapter tests; Legends package tests;
`go test ./cards/... -short -count=1` (including registration validation);
`go build ./...`; `go vet ./cards/legends`; `make test`; and `make lint`
passed with zero failures (lint: zero issues).


## Completed Slice: Strip Mine Typed Activations

- [x] Migrate Strip Mine's complete definition to typed `Activate`, `Costs`, `AddManaAmount`, `TargetPermanent`, `WithTargets`, and `Destroy` through `pkg/mage/dsl`.
- [x] Verify the full Oracle comment and land characteristics against `data/ATQ.json`; use the exact wording “Sacrifice this land.”
- [x] Audit target selection, payment, source departure, target identity, mana classification, and automatic mana planning. Existing services support both abilities; no engine fix or new constructor is required.

The mana ability uses `Costs(TapCost(This))` and
`AddManaAmount(Colorless, Fixed(1))`. It has no target, resolves immediately,
and exposes its fixed production to automatic mana planning. The separate
destruction ability uses `Costs(TapCost(This), SacrificeSourceCost())` and
`TargetPermanent("land", IsLand)`. Select the target before payment; tapping
and sacrificing are costs. Either player's land, including Strip Mine itself,
is a legal target. The ability remains on the stack after its source is
sacrificed. A self-target is absent at resolution. Other targets are checked
again for legality and incarnation; regeneration and indestructible still
apply. There is no mana cost, basic-land restriction, or sorcery restriction.

`TestStripMine` passed before migration and passes afterward with explicit
ability indices for mana and destruction. Its automatic selector recognizes
legacy concrete mana abilities, so it selected the new typed mana ability
when the destruction test omitted an index. The test now specifies its intent.
Existing contracts `TestTypedManaAmountActivation`,
`TestTypedManaAmountPlanning`, `TestComposableManaProductionAndResolution`,
`TestTypedTargetGroups`, `TestTypedTargetCopyAndClone`,
`TestTargetIncarnationAfterReturn`, `TestComposedRemovalMetadata`,
`TestActivateAbility_TotalCostFailureIsAtomic`, and
`TestActivationNumberCaptureSacrificeAndCopies` cover the reused primitives.
This declarative wiring needs no duplicate simulation. Rules contracts are
CR 400.7, 602, 605, 608.2b, and 701.8.

Strip Mine is fully migrated, and the land file no longer imports `pkg/mage`
directly. Unrelated definitions still use mutable aliases from the DSL, so this
does not close the author boundary. Legacy mana, destruction, and activation
helpers retain production callers; no export is retired. The broader action
and mana migrations remain open. No engine blocker is claimed resolved.

Validation: focused card and shared contract tests; Antiquities package tests;
`go test ./cards/... -short -count=1` (including registration validation);
`go build ./...`; `go vet ./cards/antiquities`; `make test`; and `make lint`
passed with zero failures (lint: zero issues).


## Completed Slice: Scavenger Folk Typed Sacrifice Activation

- [x] Migrate Scavenger Folk's complete definition from positional destruction and activation helpers to `Activate`, `Costs`, `TargetPermanent`, `WithTargets`, and `Destroy` through `pkg/mage/dsl`.
- [x] Verify the full Oracle comment, mana cost, creature type, and base P/T against `data/DRK.json`.
- [x] Audit target selection, payment, source departure, target identity, and destruction against existing contracts. No new constructor or engine fix is required.

The Oracle ability, “{G}, {T}, Sacrifice this creature: Destroy target
artifact,” uses `Costs(Mana("{G}"), TapCost(This), SacrificeSourceCost())`
and `TargetPermanent("artifact", IsArtifact)`. Select the target before
payment. Either player's artifact qualifies, including Scavenger Folk itself
if it becomes an artifact. Tapping and sacrificing are costs; normal creature
tap restrictions apply. Failed payment rolls back the transaction. The ability
resolves after its source leaves. A self-target is then absent; other targets
must still be legal and have the captured incarnation. Destruction allows
regeneration and respects indestructible. There is no sorcery restriction.

`TestScavengerFolk` passes before and after migration and checks both the
source sacrifice and the target's destruction. Existing shared contracts
`TestTypedTargetGroups`, `TestTypedTargetValidation`,
`TestTypedTargetCopyAndClone`, `TestTargetIncarnationAfterReturn`,
`TestTypedRegenerationConsumption`, `TestComposedRemovalMetadata`,
`TestActivateAbility_TotalCostFailureIsAtomic`, and
`TestActivationNumberCaptureSacrificeAndCopies` pass and cover the reused
services. This declarative ability needs no duplicate simulation test.
Rules contracts are CR 400.7, 602, 608.2b, and 701.8.

Scavenger Folk is fully migrated. Legacy helpers retain other callers, so no
export is retired. The file retains its engine import for unrelated definitions.
The broader action migration remains open; no engine blocker is claimed resolved.

Validation: focused card and shared contract tests; The Dark package tests;
`go test ./cards/... -short -count=1` (including registration validation);
`go build ./...`; `go vet ./cards/thedark`; `make test`; and `make lint`
passed with zero failures (lint: zero issues).


## Completed Slice: Goblin Digging Team Typed Wall Destruction

- [x] Migrate Goblin Digging Team's complete definition to `Activate`, `Costs`, `TargetPermanent`, `WithTargets`, and `Destroy` through `pkg/mage/dsl`.
- [x] Verify the full Oracle comment, mana cost, creature type, and base P/T against `data/DRK.json`.
- [x] Correct the legacy creature-only target restriction after a failing card regression. Oracle says “target Wall,” so the typed target uses `HasSubType("Wall")` without `IsCreature`.
- [x] Audit target scope, payment, source departure, object identity, and destruction. Existing engine services suffice; no engine fix or new constructor is required.

The Oracle ability, “{T}, Sacrifice this creature: Destroy target Wall,”
uses `Costs(TapCost(This), SacrificeSourceCost())` and a typed Wall
permanent target. Either player's Wall qualifies. CR 109.2 refers to a
permanent of the named subtype, and CR 205.3m allows creature subtypes on
kindreds as well as creatures. Do not add a creature-type restriction.

Select the target before paying the tap and sacrifice costs. Normal creature
tap restrictions apply. The ability resolves after its source leaves.
A self-target, possible if the source becomes a Wall, is then absent.
Other targets must still be legal and have the captured incarnation.
Destruction allows regeneration and respects indestructible. There is no
mana cost, sorcery restriction, or regeneration prohibition.

`TestGoblinDiggingTeamNoncreatureWall` failed before migration because the
legacy ability had no legal target. It now passes and checks source sacrifice
and target destruction. The test fixture supplies the relevant characteristics
directly: an enchantment with the Wall subtype. It tests the target restriction,
not general kindred type support. The existing `TestGoblinDiggingTeam`
continues to cover a creature Wall.

Shared contracts `TestTypedTargetGroups`, `TestTypedTargetValidation`,
`TestTypedTargetCopyAndClone`, `TestTargetIncarnationAfterReturn`,
`TestTypedRegenerationConsumption`, `TestComposedRemovalMetadata`,
`TestActivateAbility_TotalCostFailureIsAtomic`, and
`TestActivationNumberCaptureSacrificeAndCopies` cover the reused services.
Rules contracts are CR 109.2, 205.3m, 400.7, 602, 608.2b, and 701.8.

Goblin Digging Team is fully migrated. Legacy helpers retain other callers,
including Miracle Worker in the same file, so no export is retired.
The broader action migration and unrelated engine blockers remain open.

Validation: focused card and shared contract tests; The Dark package tests;
`go test ./cards/... -short -count=1` (including registration validation);
`go build ./...`; `go vet ./cards/thedark`; `make test`; and `make lint`
passed with zero failures (lint: zero issues).


## Completed Slice: Dwarven Demolition Team Typed Wall Destruction

- [x] Migrate the complete definition to `Activate`, `Costs`, `TargetPermanent`, `WithTargets`, and `Destroy` through `pkg/mage/dsl`.
- [x] Verify the full Oracle comment, mana cost, creature type, and base P/T against `data/LEA.json`.
- [x] Correct the legacy creature-only target restriction after a failing card regression. Oracle says “target Wall,” so use `HasSubType("Wall")` without `IsCreature`.
- [x] Audit targeting, tap payment, resolution legality, identity, and destruction against existing contracts. No engine gap or new constructor is required.

The Oracle ability, “{T}: Destroy target Wall,” uses `Costs(TapCost(This))`
and a typed Wall permanent target. Either player's Wall qualifies, including
the source if it gains that subtype. CR 109.2 and 205.3m do not restrict this
target to creatures. Select the target before paying the tap cost. Normal
creature tap restrictions apply. There is no mana cost, sacrifice cost, or
sorcery restriction. Resolution checks target legality and incarnation again;
destruction allows regeneration and respects indestructible.

`TestDwarvenDemolitionTeamNoncreatureWall` failed before migration because the
legacy ability found no legal target. It now passes and checks that the target
is destroyed while the source remains. The fixture supplies an enchantment
with the Wall subtype to isolate the target restriction; it does not test
general kindred support. Existing shared contracts `TestTypedTargetGroups`,
`TestTypedTargetValidation`, `TestTypedTargetCopyAndClone`,
`TestTargetIncarnationAfterReturn`, `TestTypedRegenerationConsumption`,
`TestComposedRemovalMetadata`, and `TestComposableActivationRestrictions`
pass and cover the reused services. Rules contracts are CR 109.2, 205.3m,
400.7, 602, 608.2b, and 701.8.

Dwarven Demolition Team is fully migrated. Legacy destruction and activation
helpers retain other callers, so no export is retired. The broader action
migration and unrelated engine blockers remain open.

Validation: focused card and shared contract tests; Limited Edition package;
`go test ./cards/... -short -count=1` (including registration validation);
`go build ./...`; `go vet ./cards/limited`; `make test`; and `make lint`
passed with zero failures (lint: zero issues).


## Completed Slice: Miracle Worker Attachment Targets

- [x] Add `IsAttachedTo(filter)` and `TargetAttachedToYouControl()` to the typed author surface. Keep host characteristics and host control as separate constraints.
- [x] Migrate Miracle Worker's complete definition to typed `Activate`, `Costs`, `TargetPermanent`, `WithTargets`, and `Destroy`.
- [x] Verify the full Oracle comment, mana cost, creature types, and base P/T against `data/DRK.json`.
- [x] Correct the legacy selector's acceptance of attached non-Aura enchantments after a failing regression. Retire `TargetAuraAttachedToCreatureYouControl` and its private target implementation after migrating its one production caller and one test caller.

The ability, “{T}: Destroy target Aura attached to a creature you control,”
requires an Aura enchantment attached to a current creature controlled by the
stored ability controller. The Aura may have either controller. The host is
not a target, so its targeting restrictions do not apply. Target selection
precedes tap payment; normal creature tap restrictions apply. Destruction
allows regeneration and respects indestructible.

The new constraints use the existing permanent-target selection and resolution
checks. Host type, attachment, and control are checked again at resolution.
Moving the same Aura to another qualifying creature keeps the target legal.
A returned Aura is a new incarnation. Phased-out targets and hosts do not
qualify. A copy uses its own controller for the host-control restriction.
The new option exposes host-controller inspection metadata.

The non-Aura regression failed against the old selector, passed after its
minimal predicate fix, and remains in `TestTypedAuraAttachedToCreatureYouControl`
after retirement. `TestAttachmentTargetResolution` covers opposing Aura
control, host control and type, attachment moves, detachment, phasing, returned
incarnations, and clone isolation. `TestAttachmentTargetCopyController` covers
copy retargeting under another controller. `TestAttachmentTargetComposition`
checks registration, metadata, and undeclared binding rejection.
`TestMiracleWorker` covers the card activation and destruction.
Shared typed-target, incarnation, and destruction contracts also pass.

Rules contracts are CR 400.7, 602, 608.2b, and 701.8. The broader action
migration and unrelated engine blockers remain open. The creature file retains
unrelated legacy definitions and its direct engine import.

Validation: focused engine and DSL contracts; The Dark package;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/thedark`; `make test`; and `make lint` passed with zero
failures (lint: zero issues).

## Completed Slice: King Suleiman Subtype Targeting

- [x] Migrate King Suleiman's positional destruction activation to typed
  `TargetPermanent`, `Activate`, `Costs`, `Destroy`, and `WithTargets`.
- [x] Remove the incorrect creature-only target restriction. Oracle says
  “target Djinn or Efreet,” so either subtype qualifies without a creature-type
  or controller restriction (CR 109.2 and 205.3m).
- [x] Verify the full Oracle comment, mana cost, subtypes, and P/T against
  `data/ARN.json`.
- [x] Add noncreature Djinn and Efreet regression cases. Both fail before the
  migration because no ability can activate, then pass with the typed target.
  These synthetic subtype fixtures test target scope; they do not claim full
  Kindred card-type support. Existing cases cover both creature subtypes and
  rejection of an unrelated creature.

The existing typed target and destruction services supply resolution legality,
incarnation capture, regeneration, and indestructible behavior. The activation
keeps its tap cost and uses the stack. No engine fix or new constructor is
required. Other positional destruction callers remain, so no export is retired.
This completes one subtype-targeting slice of the outstanding action migrations;
the broader migration and engine blockers remain open.

Validation: Arabian Nights tests, all card tests and registration validation,
`go build ./...`, `go vet ./cards/arabian`, `make test`, and `make lint` pass
with zero failures.


## Completed Slice: Stand Up for Yourself Typed Destruction

- [x] Migrate the complete spell definition from `NewTargetedSpell` and
  `DestroyTarget` to `dsl.Spell`, `TargetPermanent`, `WithTargets`, and `Destroy`.
- [x] Preserve the creature restriction and current power threshold with
  `And(IsCreature, HasPowerGTE(3))`. Either player's qualifying creature can
  be targeted. Check the filter again at resolution.
- [x] Verify the full Oracle comment, instant card type, and `{2}{W}` mana cost
  against `data/SOS.json`.
- [x] Audit the existing target and destruction adapters. No engine fix or new
  constructor is required.

The existing services preserve target incarnation, copying, clone isolation,
regeneration, and indestructible behavior (CR 400.7, 601, 608.2b, and 701.8).
`HasPowerGTE` reads current power, including continuous effects and counters;
it does not use printed power. Destruction metadata remains available through
the typed action. The complete definition uses the DSL author surface; other
cards in the file still require the engine import.

`TestStandUpForYourself` passes. Shared `TestTypedTargetGroups`,
`TestTypedTargetValidation`, `TestTypedTargetCopyAndClone`,
`TestTargetIncarnationAfterReturn`, `TestComposedRemovalMetadata`, and
`TestTypedRegenerationConsumption` pass. This declarative spell reuses existing
contracts, so no new simulation test is required.

This completes one slice of the outstanding action migrations. Legacy spell
and destruction helpers retain other callers, so no export is retired and
no open engine blocker is marked resolved.

Validation: Secrets of Strixhaven package tests; focused shared contracts;
`go test ./cards/... -short -count=1` (including registration validation);
`go build ./...`; `go vet ./cards/secretsofstrixhaven`; `make test`; and
`make lint` pass with zero failures (lint: zero issues).


## Completed Slice: Gate to Phyrexia Typed Activation

- [x] Migrate the complete definition to `Activate`, `Costs`, `TargetPermanent`,
  `WithTargets`, and `Destroy` through `pkg/mage/dsl`.
- [x] Preserve “during your upkeep” with `OnlyDuring(core.Upkeep)` and
  `OnlyOnYourTurn()`, and “only once each turn” with `ActivationLimit(1)`.
- [x] Verify the full Oracle comment, enchantment type, and `{B}{B}` mana cost
  against `data/ATQ.json`.
- [x] Correct the sacrifice cost after a failing card regression: “a creature”
  includes the source if it becomes a creature. Use the existing
  `SacrificeMatchingIncludingSourceCost` instead of `SacrificeCreatureCost`,
  which excludes the source. No engine fix or new constructor is required.

Either player's artifact can be targeted. Choose the target before sacrificing
one creature you control; the cost has no tap or mana requirement. The ability
resolves after its source leaves. A sacrificed target is absent at resolution;
other targets must still be legal and have the captured incarnation. Destruction
allows regeneration and respects indestructible. Timing is checked at activation,
not again at resolution. The activation limit resets for the next turn.

`TestGateToPhyrexiaCanSacrificeAnimatedSource` failed with “cannot activate
ability” before the cost correction and now passes. Its synthetic creature
source tests sacrifice eligibility and resolution after source departure, not
an animation effect. The existing `TestGateToPhyrexia` also passes.
Shared timing, activation-limit cleanup and clone, target identity and copy,
regeneration, removal metadata, atomic payment, and phased-sacrifice contracts
pass. Rules contracts are CR 400.7, 602, 608.2b, 701.8, and 701.21.

The enchantment file no longer imports the engine directly. Other definitions
still use legacy aliases; the author boundary remains open. Legacy helpers
retain other callers, so no export is retired. This completes one slice of the
outstanding action migrations; no open engine blocker is claimed resolved.

Validation: Antiquities package tests; focused shared contracts;
`go test ./cards/... -short -count=1` (including registration validation);
`go build ./...`; `go vet ./cards/antiquities`; `make test`; and `make lint`
pass with zero failures (lint: zero issues).

## Completed Slice: Blighted Bat Haste Activation

- [x] Migrate the complete Blighted Bat definition from `WithActivatedAbility`,
  `GenericCost`, and the positional keyword helper to `Activate`, `Costs`,
  `Mana`, and `Until(EndOfTurn, GrantKeyword(This, Haste))` through `pkg/mage/dsl`.
- [x] Verify the full Oracle comment, mana cost, creature types, printed flying,
  and 2/1 base characteristics against `data/JMP.json`.
- [x] Audit payment, source identity, duration, and layer behavior against the
  existing adapters. No engine fix or new constructor is required.

The `{1}` activation has no target, tap cost, timing restriction, or activation
limit. It grants haste in layer 6 to the captured source incarnation until
cleanup. A returned card is a new object; phasing preserves the incarnation.
The legacy helper already defaulted to end of turn; the typed definition makes
that duration explicit. Printed flying remains separate.

`TestBlightedBat_GainsHaste` and the shared `TestComposableKeywordSourceExit`,
`TestComposableKeywordLayerOrder`, `TestComposableSelfPumpLifetime`, and
`TestComposableContinuousCapture` pass. These contracts cover keyword grants,
timestamp order, incarnation, phasing, expiry, and clone isolation. This simple
declarative ability uses established primitives and universal registration
validation, so no new simulation test is required. Rules contracts are CR 602,
400.7, 611.2, and 613.

This completes one slice of the remaining legacy action migrations. Other cards
still use the legacy helpers and direct engine import, so no export is retired
and no outstanding engine blocker is claimed resolved.

Validation: Jumpstart package tests; focused shared contracts;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Kels, Fight Fixer Sacrifice Activation

- [x] Migrate Kels's activated ability to `Activate`, `Costs`, `Mana`, and
  `Until(EndOfTurn, GrantKeyword(This, Indestructible))` through `pkg/mage/dsl`.
- [x] Correct sacrifice eligibility after a failing card regression. Oracle
  says “Sacrifice a creature,” which includes Kels itself. Use the existing
  `SacrificeMatchingIncludingSourceCost(IsCreature, "Sacrifice a creature")`.
- [x] Verify the full Oracle comment, mana cost, legendary supertype, creature
  types, menace, and base 4/3 characteristics against `data/JMP.json`.
- [x] Audit payment, source identity, and duration against existing contracts.
  No new constructor or engine fix is required.

Pay `{1}` and sacrifice one creature you control before the ability resolves.
The activation has no target, tap cost, timing restriction, or activation limit.
If Kels remains, the captured source incarnation gains indestructible in layer 6
until cleanup. Sacrificing Kels still triggers its optional paid draw; the
indestructible grant has no remaining subject. A returned Kels is a new object.

`TestKelsFightFixer_CanPayWithSelf` failed before migration because no ability
could activate with Kels as the only creature. It now passes and checks both
self-sacrifice and the draw after paying for the sacrifice trigger. Existing
Kels tests cover sacrificing another creature, granting indestructible, and
accepting or declining the draw. Shared keyword, duration, source capture,
clone, atomic payment, and phased-sacrifice contracts pass. Rules contracts
are CR 400.7, 602, 611.2, 613, and 701.21.

This completes the activation slice of the legacy action migrations. Kels's
sacrifice trigger and payment branch retain their existing builders; their
migration remains within the outstanding event and payment families. Other
legacy callers remain, so no export is retired or engine blocker marked resolved.

Validation: focused Kels and shared contract tests; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Wildheart Invoker Typed Activation

- [x] Migrate Wildheart Invoker's complete definition to `Activate`, `Costs`,
  `TargetPermanent`, `WithTargets`, and typed continuous modifications through
  `pkg/mage/dsl`.
- [x] Install +5/+5 and trample together with one `Until(EndOfTurn, ...)`,
  using `ModifyPT` and `GrantKeyword` on the same creature binding.
- [x] Verify the full Oracle comment, mana cost, Elf Shaman subtypes, and 4/3
  base characteristics against `data/JMP.json`.
- [x] Audit target selection, payment, identity, layers, and duration against
  existing adapters. No engine fix or new constructor is required.

Pay `{8}` to target any creature, including the source or an opponent's
creature. The ability has no tap cost, timing restriction, or activation limit.
Target selection precedes payment; resolution checks legality and incarnation
again. The resolved modifications capture the same target, apply in layers 6
and 7c, survive source departure, and expire at cleanup. Later activations add
another +5/+5. A returned target is a new object. Existing metadata reports
both the power/toughness bonus and trample. Rules contracts are CR 400.7, 602,
608.2b, 611.2, and 613.

`TestWildheartInvoker_PumpsAndGrantsTrample` passes before and after migration.
Shared typed-target, copy, incarnation, keyword, continuous capture, source
lifetime, temporary P/T metadata, and atomic payment contracts pass. This
simple declaration reuses established primitives and registration validation;
no new simulation test is required.

This completes one slice of the outstanding legacy action migrations. Other
cards retain the legacy constructors and direct engine import, so no export
is retired and no outstanding engine blocker is marked resolved.

Validation: Jumpstart package tests; focused shared contracts;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Shattered Acolyte Typed Sacrifice Activation

- [x] Migrate Shattered Acolyte's complete definition to the author DSL, with
  `Activate`, `Costs`, `TargetPermanent`, `WithTargets`, and `Destroy`.
- [x] Preserve the full Oracle comment, lifelink, mana cost, Dwarf Warlock
  subtypes, and 2/2 base characteristics from `data/SOS.json`.
- [x] Audit the target, payment, and destruction adapters. No engine fix or new
  constructor is required.

The activation costs `{1}` and sacrifice of the source. Its single target can
be an artifact or enchantment controlled by either player, including the source
if it has a qualifying type. Choose the target before payment. Destruction
resolves after sacrifice, checks target legality and incarnation again, and
allows regeneration and indestructible. A sacrificed self-target is absent.
There is no tap cost, timing restriction, or activation limit. The typed action
retains the existing destruction metadata. Rules contracts are CR 400.7, 602,
608.2b, and 701.7.

Existing `TestShatteredAcolyte_Lifelink` and
`TestShatteredAcolyte_SacrificeDestroyArtifact` pass before and after migration.
Shared target validation, copy, incarnation, capture-before-payment, and payment
contracts also pass. This declaration uses established primitives; it does not
need a new simulation test.

This completes one slice of the outstanding legacy action migrations. Other
cards still use the legacy helpers and the file's direct engine import, so no
export is retired and no engine blocker is marked resolved.

Validation: Secretsofstrixhaven package tests; focused shared contracts;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/secretsofstrixhaven`; `make test`; and `make lint` pass with
zero failures (lint: zero issues).

## Completed Slice: Rattlechains Entry Hexproof Grant

- [x] Migrate Rattlechains's entry ability from `WithCastTarget`, `WithETBEffect`,
  and string-bound keyword targeting to `WhenEvent(Enters(This), ...)`,
  `TargetPermanent`, `WithTargets`, and
  `Until(EndOfTurn, GrantKeyword(spirit, Hexproof))`.
- [x] Correct target selection timing: cast the creature without a target, then
  choose the Spirit when its entry trigger goes on the stack. Entry without
  casting also triggers the ability, and Rattlechains can target itself.
- [x] Match “target Spirit” with `HasSubType("Spirit")`, without an added
  creature or controller restriction. An opponent's noncreature Spirit qualifies.
- [x] Verify the complete Oracle comment, mana cost, types, base 2/1, flash,
  flying, and Spirit casting permission against `data/JMP.json`.
- [x] Audit the typed target, keyword grant, and duration contracts. Existing
  engine services support this slice; no engine blocker requires a fix.

`TestRattlechainsEntryTarget` failed before migration in all three cases:
casting without an entry target was rejected, and entry without casting produced
no trigger for either the self or noncreature Spirit case. All cases now verify
that the grant waits for trigger resolution and expires at cleanup. The existing
Rattlechains tests now choose the target for the trigger and still verify the
Spirit grant and casting Rattlechains during the opponent's combat.

The shared typed-target and keyword contracts cover target incarnation,
resolution legality, source departure, layer order, and duration. Rules contracts
are CR 115.1d, 603.3d, 603.6a, 400.7, 608.2b, 611.2a, and 613. The printed
keywords and existing static Spirit casting permission remain intact; this slice
completes the entry ability within the outstanding legacy action migrations.
Other callers still need the legacy APIs, so no export is retired.

Validation: focused Rattlechains tests; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).

## Completed Slice: Controller Life-Gain Events and Kalastria Nightwatch

- [x] Add `ControllerGainsLife(playerBinding)` to the typed DSL. Capture the
  gaining player at event detection without declaring a target.
- [x] Migrate Kalastria Nightwatch's complete definition to `WhenEvent` and
  `Until(EndOfTurn, GrantKeyword(This, Flying))`.
- [x] Verify the full Oracle comment, `{4}{B}` cost, Vampire Warrior Ally
  subtypes, and 4/5 characteristics against `data/JMP.json`.
- [x] Audit the existing life-event, trigger capture, and keyword services.
  No engine fix is required for this adapter.

The pattern matches positive `EvtLifeGained` outcomes for the source's
controller at detection. Opponent gains, life loss, and gains replaced with
zero or no gain do not trigger it. Each gain produces one trigger regardless
of its amount. The saved player remains fixed through control changes, copies,
source departure, and clones; `You` still uses the resolving controller.
Kalastria Nightwatch grants flying to its captured source incarnation through
cleanup. A returned card is a new object; phasing preserves its incarnation.
Rules contracts are CR 119.9, 400.7, 603.2, 603.3a, 611.2, and 613.

`TestControllerLifeGainEventFiltering`, `Lifecycle`, `SourceIdentity`, and
`Validation` cover outcomes, controller changes, pending and stack clones,
copies, modal and delayed composition, phasing, source return, and binding
scope. DSL composition and compile-failure tests cover the public surface.
The tests first failed to compile because the new constructor was absent.
`TestKalastriaNightwatch_GainsFlyingOnLifeGain` passes before and after migration;
existing keyword and duration contracts cover its declarative body.

This completes one slice of event participant and legacy action migrations.
Six production `WheneverYouGainLifeTrigger` callers remain: Bloodbond Vampire,
Malakir Familiar, Cradle of Vitality, Comforting Counsel, Blech, Loafing Pest,
and Pest Mascot. The helper remains exported. General life-event emission,
replacement ordering, collection gains, and other engine blockers remain open.

Validation: focused adapter and card tests; engine short tests; Jumpstart
package tests; all card tests and registration validation; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).

## Completed Slice: Bloodbond Vampire and Malakir Familiar Life-Gain Triggers

- [x] Migrate Bloodbond Vampire to `WhenEvent(ControllerGainsLife(player),
  AddCounters(This, P1P1, Fixed(1)))` through `pkg/mage/dsl`.
- [x] Migrate Malakir Familiar to the same event pattern with
  `Until(EndOfTurn, ModifyPT(This, Fixed(1), Fixed(1)))`, removing its direct
  engine callback and UUID-based temporary boost.
- [x] Verify complete Oracle comments, mana costs, types, base P/T, and printed
  keywords against `data/JMP.json`.
- [x] Audit event filtering, source identity, counter replacement, and temporary
  P/T lifetime against existing engine contracts. No engine fix is required.

Both abilities trigger once for each positive life-gain event for the source's
controller, regardless of the amount gained. Opponent gains do not trigger them.
The trigger captures the original source incarnation. A card that leaves and
returns is a new object. Bloodbond Vampire places one counter through the
counter replacement service; the counter has no end-of-turn expiration.
Malakir Familiar adds a separate layer 7c bonus for each resolution, and the
bonus expires at cleanup. Flying and deathtouch remain printed abilities.
Rules contracts are CR 119.9, 400.7, 603.2, 611.2, and 613.

Existing `TestBloodbondVampire_GainsCounterOnLifeGain`,
`TestBloodbondVampire_NoCounterOnOpponentLifeGain`, and
`TestMalakirFamiliar_PumpsOnLifeGain` pass before and after migration.
`TestControllerLifeGainEvent*`, `TestTypedCounterAmount*`,
`TestComposableCounterMetadata`, and `TestComposableSelfPumpLifetime` cover
filtering, capture, copies, clones, replacement, stacking, phasing, and duration.
These declarative compositions need no additional card simulation tests.

This completes the two remaining Jumpstart creature callers of
`WheneverYouGainLifeTrigger`. Four production callers remain: `Cradle of
Vitality`, `Comforting Counsel`, `Blech, Loafing Pest`, and `Pest Mascot`. The legacy helper remains exported. General life-event
emission, replacement ordering, and collection gains remain open.

Validation: focused card and shared contract tests; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Comforting Counsel and Pest Mascot Life-Gain Counters

- [x] Migrate both life-gain triggers to `WhenEvent(ControllerGainsLife(player),
  AddCounters(This, counter, Fixed(1)))` through `pkg/mage/dsl`.
- [x] Use `Growth` for Comforting Counsel and `P1P1` for Pest Mascot. Pest
  Mascot's complete definition now uses the author DSL and keeps trample.
- [x] Verify full Oracle comments, costs, types, and Pest Mascot's 2/3 base
  characteristics against `data/SOS.json`.
- [x] Audit event filtering, source identity, and counter replacement against
  existing contracts. No engine fix or new constructor is required.

Each positive life-gain event for the source's controller creates one trigger,
regardless of the amount gained. Opponent gains and gains replaced with zero
or no gain do not trigger these abilities. The effects have no targets and add
one counter through the counter replacement service to the captured source
incarnation. Source departure and return do not redirect an old trigger to the
new object. Phasing preserves identity. Rules contracts are CR 119.9, 400.7,
603.2, and 122.

Existing `TestComfortingCounsel`, `TestPestMascot_LifegainCounterTrigger`, and
`TestPestMascot_NoCounterWithoutLifegain` pass before and after migration.
Shared `TestControllerLifeGainEvent*`, `TestTypedCounterAmount*`, and
`TestComposableCounterMetadata` cover event filtering, controller changes,
copies, clones, source identity, replacement, binding validation, and metadata.
These declarations reuse tested primitives and registration validation; no new
card simulation test is required.

This completes the two singular counter triggers in Secrets of Strixhaven.
Comforting Counsel's conditional static boost remains on its existing callback
and still needs a separate migration. Two production callers of
`WheneverYouGainLifeTrigger` remain: Cradle of Vitality and Blech, Loafing Pest.
The legacy helper remains exported. General life-event emission, replacement
ordering, and collection handling remain open; no engine blocker is marked
resolved.

Validation: Secrets of Strixhaven package tests; focused shared contracts;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/secretsofstrixhaven`; `make test`; and `make lint` pass with
zero failures (lint: zero issues).


## Completed Slice: Cave People Temporary Effects

- [x] Migrate the attack-trigger body to `Until(EndOfTurn, ModifyPT(This,
  Fixed(1), Fixed(-2)))` through the author DSL.
- [x] Migrate the mountainwalk activation to `Activate`, `Costs`,
  `TargetPermanent`, `WithTargets`, and an explicit end-of-turn `GrantKeyword`.
- [x] Match the full Oracle comment, including reminder text, mana cost, Human
  subtype, and 1/4 characteristics against `data/4ED.json`.
- [x] Audit existing target, payment, source capture, layer, and duration
  contracts. No new constructor or engine fix is required.

The attack trigger gives its captured source incarnation +1/-2 in layer 7c.
It is mandatory and lasts through the end of the turn, not just combat.
The activation costs `{1}{R}{R}` and tapping the source. It can target either
player's creature, including Cave People itself. Target selection precedes
payment; legality and incarnation are checked again at resolution. Mountainwalk
is granted in layer 6 and lasts through cleanup even if Cave People leaves.
A returned recipient is a new object. Rules contracts are CR 400.7, 602, 603,
608.2b, 611.2, 613, and 702.14.

`TestCavePeople` passes before and after migration, covering the attack boost,
mountainwalk grant, and interaction with The Brute and Ironclaw Orcs. Shared
`TestComposableKeywordSourceExit`, `TestComposableKeywordLayerOrder`,
`TestTargetIncarnationAfterReturn`, `TestComposableContinuousCapture`,
`TestComposableSelfPumpLifetime`, and `TestComposablePaymentAtomicity` pass.
These declarative effects reuse established primitives and registration
validation; no new simulation test is required.

This completes one slice of the remaining string-bound action migrations and
removes the creature file's last direct engine import. The attack event still
uses `AttacksTrigger`; its typed event adapter remains separate work. Legacy
helpers retain other callers, so no export is retired or engine blocker marked
resolved.

Validation: Fourth Edition package tests; focused shared contracts;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/fourthedition`; `make test`; and `make lint` pass with zero
failures (lint: zero issues).


## Completed Slice: Moment of Heroism Typed Spell

- [x] Migrate the complete spell definition to `TargetPermanent`, `Spell`,
  `WithTargets`, and `Until(EndOfTurn, ...)` through `pkg/mage/dsl`.
- [x] Install +2/+2 and lifelink together with `ModifyPT` and `GrantKeyword`
  on the same creature binding.
- [x] Restore the full Oracle comment, including reminder text, and verify
  the instant type and `{1}{W}` cost against `data/JMP.json`.
- [x] Audit target identity, layers, payment, and duration against existing
  engine contracts. No engine fix or new constructor is required.

Either player's creature can be targeted. Resolution checks target legality
and incarnation again. The modifications capture the recipient at resolution,
apply in layers 6 and 7c, and expire at cleanup. A later control change does
not end either effect; lifelink uses the creature's controller when it deals
damage. A returned target is a new object. Rules contracts are CR 400.7,
601.2, 608.2b, 611.2, 613, and 702.15.

`TestMomentOfHeroism` passes before and after migration and checks both the
power/toughness bonus and lifelink. Shared `TestComposableKeywordSourceExit`,
`TestComposableKeywordLayerOrder`, `TestComposableContinuousCapture`,
`TestComposableSelfPumpLifetime`, `TestTargetIncarnationAfterReturn`, and
`TestComposablePaymentAtomicity` pass. This declaration reuses tested
primitives and registration validation; no new simulation test is required.

This completes one slice of the outstanding action migrations. Other spells
retain legacy helpers and the file's direct engine import, so no export is
retired. Damage-source LKI and other engine blockers remain open.

Validation: Jumpstart package tests; focused shared contracts;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero
failures (lint: zero issues).


## Completed Slice: Arbor Armament and Battlefield Promotion

- [x] Migrate both complete spell definitions to `TargetPermanent`, `Spell`,
  `WithTargets`, and `Sequence` through `pkg/mage/dsl`.
- [x] Use typed `AddCounters` for the +1/+1 counter and
  `Until(EndOfTurn, GrantKeyword(...))` for reach or first strike.
- [x] Preserve Battlefield Promotion's final `GainLife(You, Fixed(2))` action.
- [x] Restore full Oracle comments, including reminder text, and verify instant
  types and mana costs against `data/JMP.json`.
- [x] Audit existing counter, target, sequence, keyword, and life-gain adapters.
  No new constructor or engine fix is required.

Either player's creature can be targeted. Resolution checks legality and the
captured target incarnation. If the sole target is illegal, the spell does not
resolve, including Battlefield Promotion's life gain. Otherwise, the instructions
run in printed order: add the counter through the counter replacement service,
install the layer 6 keyword grant, then gain life for the resolving controller
where instructed. The counter remains after cleanup; the keyword grant expires.
A later control change does not end the grant, and a returned card is a new
object. Rules contracts are CR 122, 400.7, 601.2, 608.2b, 608.2c, 611.2, and 613.

Existing `TestArborArmament` and `TestBattlefieldPromotion` pass before and
after migration. Shared `TestTypedCounterAmount*`,
`TestComposableCounterMetadata`, keyword lifetime and layer-order tests,
`TestTargetIncarnationAfterReturn`, `TestComposableContinuousCapture`,
`TestComposableSequenceSourceIdentity`, `TestComposablePaymentAtomicity`,
`TestLifeGainOutcomeEvents`, and `TestComposableLifeGainReplacement` pass.
These declarations reuse tested primitives; no new simulation test is required.

This completes one slice of the outstanding action migrations. Other callers
retain the legacy counter, keyword, and spell helpers, so no export is retired.
The file's direct engine import remains for unrelated definitions. No outstanding
engine blocker is marked resolved.

Validation: focused card and shared contract tests; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero
failures (lint: zero issues).

## Completed Slice: Jumpstart Typed Scry Creature Abilities

- [x] Migrate Cloudreader Sphinx, Octoprophet, and Sage's Row Savant to
  `WhenEvent(Enters(This), Scry(Fixed(2)))`.
- [x] Migrate Sigiled Starfish to
  `Activate(Costs(TapCost(This)), Scry(Fixed(1)))`.
- [x] Verify each full Oracle comment, mana cost, creature types, power,
  toughness, and printed keywords against `data/JMP.json`.
- [x] Audit the existing scry, trigger, and activation adapters. These
  single-action abilities use proven engine services; no engine fix is needed.

The entry triggers have no targets and scry for their stored controller.
The Starfish pays its tap cost before its ability goes on the stack. Scry
uses the current library at resolution, permits top and bottom placement,
and handles short or empty libraries through the shared library service.

The four existing card tests pass before and after migration. Shared library
choice tests cover controller context, amounts, placement, completion events,
empty libraries, clone isolation, and typed trigger/activation composition.
Registration validation checks all four declarations. No duplicate simulation
tests are needed for these single-action definitions.

This completes one slice of the remaining trigger and activation migrations.
Other cards still use the legacy builders; no export is retired. Hidden-zone
result bindings and the other recorded blockers remain open.

Validation: focused card and shared contract tests; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).

## Completed Slice: Bloodhunter Bat Typed Entry Life Loss and Gain

- [x] Migrate the complete entry ability to `WhenEvent(Enters(This), ...)`,
  `TargetPlayer`, `WithTargets`, `Sequence`, `LoseLife`, and `GainLife`.
- [x] Remove the direct life mutations and event-free `PlayerGainLife` callback.
  Existing typed actions cover the full ability; no engine fix is required.
- [x] Verify the full Oracle comment, `{3}{B}` cost, Bat subtype, 2/2 stats,
  and flying against `data/JMP.json`.
- [x] Resolve this card's missing life-gain event with a regression that failed
  before migration.

`TestBloodhunterBatEntryResolution` covers opponent and self targets, target
selection on the entry trigger, life loss before gain, source departure, and
normal, doubled, or suppressed life gain. The stored controller gains life;
the chosen player loses life. The gain event reports the actual positive gain.
The existing cast-path test also passes.

This completes one card in the life-gain callback audit. Other callbacks and
replacement-order blockers remain open. No legacy export is retired.

Validation: focused regressions; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).

## Completed Slice: Book of Rass and Fountain of Youth Typed Activations

- [x] Migrate both complete abilities to `Activate` with explicit `Costs`.
  Book of Rass uses `Mana("{2}")`, `LifePayCost(2)`, and `Draw(You, Fixed(1))`.
  Fountain of Youth uses `Mana("{2}")`, `TapCost(This)`, and
  `GainLife(You, Fixed(1))`.
- [x] Verify both full Oracle comments, artifact types, and casting costs
  against `data/DRK.json`.
- [x] Audit payment, controller context, and action outcomes. Existing typed
  adapters and engine services support both abilities; no engine fix is needed.

Both abilities have no targets and use the stack after payment. Book of Rass
has no tap cost or activation limit; paying life is a cost before the draw.
Fountain of Youth requires an untapped source. The stored controller draws or
gains life even if the source leaves before resolution. The shared actions
apply replacements and report actual outcomes through their existing events.

`TestBookOfRass`, `TestFountainOfYouth`, and the shared payment, draw, life,
life-gain-outcome, and life-payment-boundary tests pass before and after
migration. Shared contracts cover failed-payment rollback, payment at exactly
the required life total, controller context, clone isolation, and replacements.
These single-action declarations use proven primitives and need no duplicate
simulation tests.

This completes one slice of the remaining activation and action migrations.
Other callers retain the legacy helpers, so no export is retired. No outstanding
engine blocker is marked resolved.

Validation: focused card and shared contract tests; The Dark package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/thedark`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).

## Completed Slice: Mire Triton Typed Entry Mill and Life Gain

- [x] Migrate the complete entry ability to `WhenEvent(Enters(This), ...)`,
  `Sequence`, `Mill(You, Fixed(2))`, and `GainLife(You, Fixed(2))`.
- [x] Remove the card callback's direct library updates and event-free
  `PlayerGainLife` call. Existing engine services need no change.
- [x] Verify the complete Oracle comment, mana cost, types, power, toughness,
  and deathtouch against `data/JMP.json`.
- [x] Resolve this card's library-order, mill-modifier, life-gain-event, and
  departed-source defects with a regression that failed before migration.

`TestMireTritonEntryResolution` covers top-first milling, empty and one-card
libraries, doubled milling, doubled and suppressed life gain, mill-before-gain
order, and source departure before resolution. The entry trigger uses the
stored controller and still gains life when fewer than two cards can be milled.
Milling does not count as drawing. The opponent's life and graveyard stay intact.
The existing cast-path test also passes.

This completes one card in the life-gain callback and milling caller audits.
General mill replacement ordering, zone-change replacements, and mill events
remain open as recorded above. No legacy export is retired.

Validation: focused regressions; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).

## Completed Slice: Scroll of Avacyn Typed Conditional Life Gain

- [x] Migrate the complete definition to `Activate`, typed mana and sacrifice
  costs, `Sequence`, `Draw`, `IfState`, `Controls`, and `GainLife`.
- [x] Replace the card callback's event-free `PlayerGainLife` call with the
  existing event-aware typed action. No engine change is required.
- [x] Check for an Angel permanent without an additional creature restriction,
  as required by CR 109.2.
- [x] Verify the full Oracle comment, artifact type, and `{1}` mana cost against
  `data/JMP.json`.

The controller pays `{1}` and sacrifices the artifact before the ability goes
on the stack. At resolution, that player draws first, then gains 5 life if they
currently control an Angel. Multiple Angels do not multiply the gain. The
condition does not use the battlefield state at activation. Life gain reports
its actual positive amount after modifiers and replacements through the shared
executor. Rules contracts are CR 109.2, 602.2, and 608.2c.

`TestScrollOfAvacynResolution` failed before migration because successful gains
emitted no life-gain event. It now covers source sacrifice and mana payment,
failed payment, draw-before-gain order, no Angel, an opponent's Angel, Angels
entering or leaving before resolution, multiple Angels, a noncreature Angel
permanent, and doubled or suppressed gains. Existing `TestScrollOfAvacyn` and
shared life-outcome, condition, and sacrifice-capture contracts pass.

This completes one card in the life-gain callback audit. Other callbacks and
the broader replacement-order audit remain open. No legacy export is retired.

Validation: focused regressions and shared contracts; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).

## Completed Slice: Valorous Stance Typed Modes and Toughness Filter

- [x] Add `HasToughnessGTE(n)` to the author DSL using the existing current
  toughness service. Combine it with `IsCreature` for creature targets.
- [x] Migrate Valorous Stance to `SpellModes`, `ModeOf`, `TargetPermanent`,
  `WithTargets`, typed `Destroy`, and an end-of-turn `GrantKeyword`.
- [x] Remove the card-local `toughness4OrGreater` engine callback.
- [x] Restore the complete Oracle reminder text and verify the instant type
  and `{1}{W}` cost against `data/JMP.json`.
- [x] Audit modal selection, target identity, destruction, and keyword duration.
  No engine behavior fix or outstanding blocker resolution is required.

Choose one mode before selecting its target and paying. Each mode can target
an opponent's creature or your own. Only the chosen mode supplies targets.
The destruction mode checks current toughness of at least 4 at casting and
resolution; it permits regeneration and respects indestructible. The grant
mode captures the target incarnation, applies indestructible in layer 6, and
expires at cleanup. Returning a card creates a new object. Rules contracts
are CR 400.7, 601.2b–c, 608.2b, 611.2, 613, and 701.7.

`TestHasToughnessGTE` first failed to compile because the constructor was
absent. It now covers the boundary, changing current toughness through
continuous effects, negative thresholds, and typed modal composition.
Existing `TestValorousStance_Indestructible` and
`TestValorousStance_DestroyTough` pass before and after migration. Shared
`TestTypedModal*`, `TestModalCast*`, `TestTargetIncarnationAfterReturn`,
`TestComposableKeyword*`, and `TestComposableContinuousCapture` pass.

This completes one slice of the outstanding legacy modal and action migrations.
The card definition uses the author DSL; unrelated definitions retain the file's
engine import. Other cards still use the legacy helpers, so no export is retired.

Validation: focused adapter and shared contract tests; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).

## Completed Slice: Dauntless Onslaught and Tandem Tactics Target Collections

- [x] Migrate both complete definitions to `TargetPermanents`, `WithTargets`,
  `Spell`, and `Until(EndOfTurn, ModifyEachPT(...))` through `pkg/mage/dsl`.
- [x] Preserve Tandem Tactics' ordered life gain with `Sequence` and
  `GainLife(You, Fixed(2))`.
- [x] Verify complete Oracle comments, instant types, and mana costs against
  `data/JMP.json`.
- [x] Audit target counts, partial legality, returned incarnations, resolving
  controller, layers, and duration. Existing adapters need no engine fix.

Each spell can target zero, one, or two creatures controlled by either player.
At resolution, only legal target incarnations receive the layer 7c boost, which
expires at cleanup. A returned target is a new object. Tandem Tactics gains life
after installing the boost, including when cast with zero targets or when only
one of two targets remains legal. If all chosen targets become illegal, neither
spell resolves and no life is gained. Rules contracts are CR 400.7, 608.2b–c,
611.2, and 613.

`TestTargetedBoostsPartialLegality` passes before and after migration. It checks
both spells with one returned target and with all targets returned, including
an opponent's creature as the remaining legal target. Existing
`TestDauntlessOnslaught` and `TestTandemTactics` cover zero, one, and two targets
and life gain. Shared typed target, copy, clone, continuous capture, duration,
and life-gain contracts pass.

This completes one slice of the outstanding action migrations. Neither spell
uses positional `Boost` or `ToAllTargets` now. Other cards retain the legacy
helpers, so no export is retired or engine blocker marked resolved.

Validation: focused card and shared contract tests; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).

## Completed Slice: Aerial Assault and Typed Permanent Counts

- [x] Add `CountPermanents(PermanentSet)` as a typed resolution value and expose
  the collection to definition inspection and binding validation.
- [x] Migrate Aerial Assault to `Spell`, `WithTargets`, `Sequence`, `Destroy`,
  and `GainLife` with a typed controlled flying-creature collection.
- [x] Verify the complete Oracle text, sorcery type, and `{2}{W}` cost against
  `data/JMP.json`. No engine behavior gap was found or closed in this slice.

The count reads the collection when the value is used. It excludes phased-out
permanents, uses the resolving controller, and reads the current game's state.
`SnapshotNumber` preserves an earlier count. Inspection checks collection
bindings, including nested references, and rejects a nil collection.

Aerial Assault can target either player's tapped creature. Its target must
remain legal at resolution; an illegal sole target prevents both instructions.
Destruction permits regeneration and respects indestructible. Life gain counts
flying creatures after destruction, even when the destroyed creature was your
own flyer. The existing target, destruction, and life-gain services retain
incarnation checks and replacement handling (CR 400.7, 608.2b–c, 701.7).

`TestCountPermanents*` first failed because the constructor was absent. Coverage
includes empty and filtered collections, phasing, return to the battlefield,
clone isolation, saved counts, nested binding validation, and copied triggers
with a different controller and later battlefield entries. The author-surface
composition test checks inspection. The existing Aerial Assault test and the
new own-flyer sequencing test pass.

This completes one slice of the remaining action and spell migrations. Other
callers still use the legacy destruction and count helpers; no export is retired.

Validation: engine tests with `-short -count=1`; Jumpstart package tests;
all card tests with `-short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures.


## Completed Slice: Launch Party Typed Destruction and Life Loss

- [x] Migrate Launch Party to `Spell`, `TargetPermanent`, `WithTargets`,
  `Sequence`, `SnapshotPlayer`, `Destroy`, and `LoseLife` through the author DSL.
- [x] Preserve the creature sacrifice additional cost and complete Oracle text,
  instant type, and `{3}{B}` mana cost from `data/JMP.json`.
- [x] Correct the card callback's missing life-loss event by using the existing
  typed action. No engine behavior fix or new constructor is required.
- [x] Verify payment, target identity, controller capture, failed destruction,
  and life-loss triggers with a regression test written before migration.

The spell chooses its creature target before sacrifice payment. The target may
be either player's creature, including the creature sacrificed as payment.
An illegal sole target prevents both instructions. Otherwise, capture its
controller at resolution before destruction; that player loses 2 life even if
regeneration or indestructible prevents destruction. A later control change
before resolution changes the player who loses life. A returned target is a
new object. Rules contracts are CR 119.3, 400.7, 601.2c, 601.2h, and 608.2b–c.

`TestLaunchPartyOutcomes` first failed for destruction, indestructible, and
regeneration: Exquisite Blood did not trigger because the old callback changed
life without an event. All eight scenarios now pass, including target departure
and return, changed control, sacrificing the target, and failed sacrifice
payment with unchanged hand, mana, and stack. The existing `TestLaunchParty`
also passes. Shared player-snapshot, target-incarnation, payment, destruction,
and opponent-life-event tests pass.

This completes one slice of the outstanding action migrations. Other card
callbacks and legacy helpers remain; no export is retired or engine blocker
marked resolved.

Validation: focused regressions and shared contracts; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).

## Completed Slice: Assassin's Strike Typed Destruction and Discard

- [x] Migrate Assassin's Strike to `Spell`, `TargetPermanent`, `WithTargets`,
  `Sequence`, `SnapshotPlayer`, `Destroy`, and `Discard` through the author DSL.
- [x] Preserve the complete Oracle text, sorcery type, and `{4}{B}{B}` mana
  cost from `data/JMP.json`.
- [x] Replace direct hand mutation with the existing typed discard action to
  preserve discard replacements and events. No engine fix or new constructor
  is required.
- [x] Audit target identity, controller capture, failed destruction, discard
  choice, replacement handling, and instruction order with a regression
  written before migration.

The creature's controller is captured at resolution before destruction. That
player chooses and discards one card even if regeneration or indestructible
prevents destruction. A control change before resolution changes who discards.
An illegal sole target prevents both instructions; a returned card is a new
object. An empty hand does not prevent destruction. Rules contracts are
CR 400.7, 608.2b–c, 608.2h, 701.8, and 701.9.

`TestAssassinsStrikeOutcomes` first failed because direct hand mutation emitted
no discard event and bypassed Library of Leng. All nine scenarios now pass:
destruction, indestructible, regeneration, absent target, returned target,
changed control, own creature, empty hand, and discard replacement. The tests
check the chosen card, event player and card identity, and destruction before
discard. The existing `TestAssassinsStrike` also passes.

This completes one slice of the remaining action and spell migrations. Other
card callbacks remain; no legacy export is retired or engine blocker marked
resolved.

Validation: focused regressions and shared discard, player-snapshot, and target
identity contracts; Jumpstart package tests; all card tests with
`-short -count=1`; `go build ./...`; `go vet ./cards/jumpstart`; `make test`;
and `make lint` pass with zero failures.

## Completed Slice: Take Heart Typed Boost and Attacker Count

- [x] Migrate Take Heart to `Spell`, `TargetPermanent`, `WithTargets`,
  `Sequence`, `Until`, `ModifyPT`, and `CountPermanents` through the author DSL.
- [x] Replace its legacy controller selector and battlefield count with
  `ControlledPermanents(And(IsCreature, IsAttacking))`.
- [x] Preserve the complete Oracle text, instant type, and `{W}` mana cost
  from `data/JMP.json`.
- [x] Audit target legality, current combat membership, resolving controller,
  and temporary boost lifetime. Existing typed services need no engine fix.

Either player's creature may be targeted, whether or not it is attacking.
The +2/+2 boost applies before life gain and expires at cleanup. Count only
creatures currently attacking under the resolving controller's control when
life gain executes. A creature removed from combat before resolution does not
count. An illegal sole target prevents both instructions; a returned card is
a new object. Rules contracts are CR 400.7, 506.4, 608.2b–c, and 611.2.

`TestTakeHeartResolution` covers two attackers, removal from combat after
casting, opponent attackers, and a target that leaves and returns. These
characterization tests and the existing `TestTakeHeart` pass before and after
migration. Shared `TestCountPermanents*`, `TestTargetIncarnationAfterReturn`,
`TestComposableContinuousCapture`, and `TestComposableSelfPumpLifetime` pass.

This completes one slice of the remaining spell and action migrations. No
engine blocker is marked resolved, and other callers still need the legacy
builders and count helper, so no export is retired.

Validation: focused card and shared contract tests; Jumpstart package tests;
all card tests with `-short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Goblin Lore Typed Draw and Random Discard

- [x] Migrate Goblin Lore to `Spell`, `Sequence`, `Draw`, and
  `DiscardAtRandom` through the author DSL, removing its engine callback.
- [x] Preserve the complete Oracle text, sorcery type, and `{1}{R}` mana
  cost from `data/JMP.json`.
- [x] Audit instruction order, resolving controller, random selection,
  discard replacements and events, and short hands. Existing typed services
  need no engine fix; no engine blocker is marked resolved.

The resolving controller draws four cards before discarding three cards at
random from their entire current hand. Both actions use the shared engine
services. Random selection uses the game's random state and permits no card
choice; discard replacements still apply. If fewer than three cards remain,
discard as many as possible. The typed action passes the spell source to the
discard replacement pipeline. Rules contracts are CR 121.2, 608.2c, 609.3,
and 701.9.

`TestGoblinLore_DrawFourDiscardThreeAtRandom` passes before and after
migration. Shared `TestTypedRandomDiscard*` and `TestTypedDrawAmount*` pass,
covering controller and copy context, clone isolation, random consumption,
replacement handling, events, short hands, ordering, and validation. These
existing contracts and the card's sequencing test cover this declarative
migration without duplicate simulation tests.

This completes Goblin Lore's surrounding definition migration. Other card
callbacks and legacy spell builders remain, so no export is retired.

Validation: focused card and shared draw/discard tests; Jumpstart package tests;
all card tests with `-short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).

## Completed Slice: Ramses Overdark Typed Enchanted Target

- [x] Migrate Ramses Overdark to `TargetPermanent`, `And(IsCreature,
  IsEnchanted)`, `Activate`, `Costs`, `Destroy`, and `WithTargets`.
- [x] Remove the card-local attachment callback and positional destruction.
- [x] Verify the full Oracle comment, legendary supertype, creature types,
  power/toughness, and mana cost against `data/LEG.json`.
- [x] Audit enchanted status, target lifetime, and destruction with the existing
  shared adapters. No engine gap or new DSL export is required.

Either player's enchanted creature can be targeted, including the source.
The Aura's controller is independent of the creature's controller. An attached
Aura enchantment is required; Equipment does not qualify. Check the target at
activation and resolution. Removing or phasing out the only Aura makes the
creature illegal; another Aura can keep it legal. A returned creature is a new
object. Destruction allows regeneration and respects indestructible. Rules
contracts are CR 303.4b–e, 400.7, 602.2, 608.2b, and 701.7.

`TestEnchantedTargetResolution` covers either Aura controller, Aura removal,
Aura phasing, a remaining second Aura, returned target identity, and clone
isolation. It passes with the existing engine before card migration.
`TestIsEnchanted*` covers attached Auras and Equipment exclusion.
`TestRamsesOverdark` passes before and after migration. The definition uses
only author DSL constructs; unrelated cards retain the file's engine import.
Other positional destruction callers remain, so no legacy export is retired.

Validation: focused tests; Legends package tests; card registration and engine
regressions with `-short -count=1`; `go build ./...`;
`go vet ./cards/legends`; `make test`; and `make lint` pass with zero failures.


## Completed Slice: Jumpstart Sacrifice Debuff Activations

- [x] Migrate Phyrexian Debaser and Plagued Rusalka to typed creature targets,
  `Activate`, grouped `Costs`, and `Until(EndOfTurn, ModifyPT(...))`.
- [x] Correct Rusalka's sacrifice cost with the existing
  `SacrificeMatchingIncludingSourceCost` adapter. It can sacrifice itself.
- [x] Verify full Oracle comments, costs, types, base P/T, and flying against
  `data/JMP.json`. No new adapter or engine fix is required.

Both abilities can target either player's creature. Select the target before
payment and recheck its legality and incarnation at resolution. Debaser pays
by tapping and sacrificing itself; Rusalka pays `{B}` and sacrifices a creature
it controls, including itself. The debuff resolves after source departure,
uses layer 7c, and expires at cleanup. A returned target is a new object.
Rules contracts are CR 400.7, 602.2, 608.2b, 611.2, and 613.

`TestPlaguedRusalkaSacrificesItself` failed before migration because the ability
could not activate with Rusalka as its controller's only creature. It now passes
and verifies source sacrifice and the opponent's -1/-1 debuff. Existing
`TestPhyrexianDebaser_SacToDebuff` and `TestPlaguedRusalka_SacToDebuff` pass.
Shared payment, target incarnation, continuous capture, temporary P/T lifetime,
and phased sacrifice-candidate tests pass.

This completes one slice of the outstanding action migrations. Other callers
still need the legacy helpers, so no export is retired. No engine blocker is
marked resolved.

Validation: focused card and shared contract tests; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Legends Fixed Mana Creatures

- [x] Migrate Princess Lucrezia, Riven Turnbull, and Sunastian Falconer to
  `Activate(Costs(TapCost(This)), AddManaAmount(color, Fixed(amount)))`
  through the author DSL.
- [x] Preserve one blue mana, one black mana, and two colorless mana,
  respectively, with no targets or additional costs.
- [x] Verify full Oracle comments, costs, legendary supertypes, creature
  subtypes, and power/toughness against `data/LEG.json`.
- [x] Audit immediate resolution, tap payment, summoning sickness, fixed
  production planning, and clone isolation. No engine fix is required.

These abilities add mana to the activating controller's pool and resolve
without using the stack. The shared tap-cost and mana services enforce
summoning sickness and haste, require an untapped source, and support automatic
mana payment. Fixed typed amounts expose the same production to the planner.
Rules contracts are CR 302.6, 602, and 605.

Shared `TestComposableMana*` and `TestTypedManaAmount*` pass. They cover
immediate resolution, tap events, payment failure, planning, clone isolation,
and definition validation. The Legends package and its existing creature
checks pass. These are declarative single-action abilities, so no additional
card simulation test is needed.

This completes one slice of the remaining fixed mana-ability migrations.
Other cards still use `WithManaAbility`, `WithActivatedAbility`, and `AddMana`,
so no helper is retired. Dynamic planning and the other mana-family work
remain open; no engine blocker is marked resolved.

Validation: focused shared contracts; Legends package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/legends`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Minotaur Skullcleaver Typed Entry Boost

- [x] Migrate Minotaur Skullcleaver to `WhenEvent(Enters(This), ...)` and
  `Until(EndOfTurn, ModifyPT(This, Fixed(2), Fixed(0)))` through the author DSL.
- [x] Preserve the full Oracle text, haste, creature types, base P/T, and
  `{2}{R}` mana cost from `data/JMP.json`.
- [x] Audit entry timing, source incarnation, and boost lifetime. Existing
  adapters need no engine fix or new export; no blocker is marked resolved.

The untargeted ability triggers when the creature enters, including entry
without casting. It gives the source incarnation +2/+0 in layer 7c until
cleanup. An old trigger cannot boost a returned incarnation. Phasing preserves
identity, and a resolved boost does not follow a card through a zone change.
Rules contracts are CR 400.7, 603, 611.2, and 613.

`TestMinotaurSkullcleaver_BoostsOnETB` passes before and after migration.
Shared `TestComposableEntryTriggerHasNoImplicitTarget`,
`TestComposableSourceLifetimes`, `TestComposableSourceReentry`,
`TestComposableSequenceSourceIdentity`, `TestComposableContinuousCapture`,
and `TestComposableSelfPumpLifetime` pass. They cover untargeted entry,
source identity, phasing, clone isolation, and temporary-effect expiration.
This declarative migration needs no duplicate card simulation test.

This completes one slice of the remaining trigger and action migrations.
Other cards retain the legacy entry trigger and positional boost helpers,
so no export is retired.

Validation: focused card and shared contract tests; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).

## Completed Slice: Hammerheim and Typed Landwalk Removal

- [x] Add `RemoveLandwalk(PermanentRef)` as a typed layer 6 modification,
  with target validation, duration installation, and detrimental metadata.
- [x] Migrate Hammerheim's removal activation to `TargetPermanent`, `Activate`,
  `Costs`, and `Until(EndOfTurn, ...)`. Verify its complete Oracle comment and
  legendary land type against `data/LEG.json`.
- [x] Resolve the landwalk-removal engine gap: the subtype map excludes
  legendary landwalk. `TestRevokeLandwalkIncludesLegendary` failed before the
  fix. Both legacy and typed removal now use the complete shared keyword list.
- [x] Retire `RevokeLandwalk` from the author DSL after migrating its only
  production caller. The engine helper remains for legacy engine tests.

Either player's creature can be targeted. The activation removes every instance
of all seven supported landwalk attributes, including Desertwalk and legendary
landwalk, while preserving unrelated abilities. Later grants apply in timestamp
order. The resolved effect survives source departure and recipient phasing;
cleanup ends it, and a returned recipient is a new object. Shared continuous
services supply capture, cloning, and duration behavior without a new executor.
Rules contracts are CR 400.7, 608.2b, 611.2, 613.1f, 613.7, and 702.14.

`TestComposableLandwalkRemoval` covers every supported variant with printed and
repeated granted instances, later grants and removals, and cleanup.
`TestComposableLandwalkRemovalLifetime` covers source departure, phasing,
recipient return, and independent clone cleanup. Metadata and author-composition
tests cover target binding validation and spell, activation, trigger, and static
contexts. Existing Hammerheim tests cover normal activation and removal of
mountainwalk supplied by multiple Goblin Kings.

This completes one slice of the remaining continuous-effect and legacy action
migrations. Hammerheim's separate legacy mana declaration and other card
migrations remain open.

Validation: focused regressions and Hammerheim tests; engine short suite; Legends
package tests; card regression and registration validation; `go build ./...`;
`go vet ./...`; `make test`; and `make lint` all pass with zero failures
(lint: zero issues).

## Completed Slice: Typed Mana for Hammerheim, Karakas, and Pendelhaven

- [x] Replace the three fixed `WithManaAbility` declarations with `Activate`,
  `Costs(TapCost(This))`, and `AddManaAmount(color, Fixed(1))`.
- [x] Verify full Oracle comments and legendary land types against `data/LEG.json`.
- [x] Resolve the harness selection blocker: use the existing engine mana
  classification instead of recognizing only the legacy concrete mana type.
  `TestActivationSelectionPrefersNonMana` failed first for fixed and dynamic
  typed mana, selecting index 0 instead of the utility ability at index 1.
- [x] Preserve explicit mana choices in seven existing card scenarios through
  `ActivateAbilityIndex`, with their behavioral assertions unchanged.

`IsActivatedManaAbility` exposes the existing immediate-resolution classification.
It recognizes legacy, fixed, dynamic, and mixed mana abilities, excludes spells
and targeted activations, and does not evaluate dynamic amounts. Harness fallback
checks activation eligibility for composed mana abilities. Tests cover utility
preference, mana fallback, and an unavailable tap cost. Existing shared mana
contracts cover immediate resolution, events, payment, planning, and clone isolation.
The three lands' utility tests pass before and after migration. Rules contract:
CR 605.1a and 605.3b. No game-rules executor change was needed.

This closes Hammerheim's remaining mana declaration and one slice of the fixed
mana caller migrations. Other legacy mana callers and the broader mana-family
blockers remain open; no author export is retired.

Validation: focused regression and classification tests; engine short suite;
Legends package tests; card regression and registration validation; `go build ./...`;
`go vet ./...`; `make test`; and `make lint` pass with zero failures.


## Completed Slice: Dreamstone Hedron and Hedron Archive Typed Activations

- [x] Migrate both complete definitions to the author DSL with `Activate`,
  grouped `Costs`, `AddManaAmount`, and `Draw`.
- [x] Preserve three colorless mana and three cards for Dreamstone Hedron,
  and two colorless mana and two cards for Hedron Archive.
- [x] Verify full Oracle comments, artifact types, and mana costs against
  `data/JMP.json`.
- [x] Audit mana classification, payment, source sacrifice, resolving controller,
  and draw handling. Existing adapters need no engine fix or new export.

The untargeted mana abilities pay a tap cost and resolve immediately. Their
fixed production remains available to automatic mana planning. The draw
abilities pay generic mana, tap, and sacrifice the source before using the
stack. Draw uses the stored controller after the source leaves and the shared
replacement and event services. Rules contracts are CR 121, 602, 605, and 608.

`TestDreamstoneHedron` passes before and after migration. Shared
`TestComposableMana*`, `TestTypedManaAmount*`, `TestTypedDrawAmount*`,
`TestComposablePayment*`, and `TestActivationSelectionPrefersNonMana` pass.
They cover immediate mana resolution, tap events, fixed production planning,
clone isolation, payment failure, draw context and replacement handling, and
selection of the utility activation beside a typed mana ability. These
single-action declarations need no duplicate card simulation tests.

This completes one slice of the remaining fixed mana and activation migrations.
Other cards retain the legacy mana, draw, and activation helpers, so no export
is retired. No outstanding engine blocker is marked resolved. Unrelated card
definitions still require this file's engine import.

Validation: focused shared contracts and Dreamstone Hedron test; Jumpstart
package tests; `go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).

## Completed Slice: Arcane Encyclopedia and Jayemdae Tome Typed Draw Activations

- [x] Migrate both complete definitions to the author DSL with `Activate`,
  grouped `Costs`, `Mana`, `TapCost(This)`, and `Draw(You, Fixed(1))`.
- [x] Preserve the activation costs of `{3}` and `{4}`, respectively, plus
  tapping the source. Both abilities draw exactly one card and have no target.
- [x] Verify full Oracle comments, artifact types, and casting costs against
  `data/JMP.json` and `data/LEA.json`. Preserve Encyclopedia's Book subtype.
- [x] Audit payment, resolving controller, and draw handling. Existing adapters
  need no engine fix or new export; no engine blocker is marked resolved.

Both abilities use the stack after payment. Draw uses the stored resolving
controller and the shared draw replacement and event services. Source departure
does not prevent resolution. The tap cost prevents repeated activation while
tapped. Rules contracts are CR 121, 602, and 608.

`TestArcaneEncyclopedia` passes before and after migration. Shared
`TestTypedDrawAmount*` and `TestComposablePayment*` also pass before and after
migration. They cover draw context, copy controller, clone isolation, replacement
handling, events, empty libraries, and payment atomicity. These single-action
abilities use proven primitives and need no duplicate card simulation tests.

This completes one slice of the remaining activation and draw migrations.
Other cards still use the legacy helpers, so no export is retired. Unrelated
card definitions retain the engine imports in these files.

Validation: focused shared contracts; Jumpstart and Limited package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart ./cards/limited`; `make test`; and `make lint`
pass with zero failures (lint: zero issues).


## Completed Slice: Sol Ring and Unstable Obelisk Typed Mana

- [x] Migrate both fixed colorless mana abilities from `WithMultiManaAbility`
  to `Activate`, `Costs(TapCost(This))`, and `AddManaAmount`.
- [x] Preserve two colorless mana for Sol Ring and one for Unstable Obelisk.
  Both complete definitions now use the author DSL.
- [x] Add Sol Ring's full Oracle comment and verify both cards' Oracle text,
  artifact types, and casting costs against `data/LEA.json` and `data/JMP.json`.
- [x] Audit immediate resolution, mana planning, tap payment, and ability
  selection. Existing adapters need no engine fix or new export.

The untargeted mana abilities resolve immediately after the tap cost is paid
and remain available to automatic mana planning. Unstable Obelisk's separate
seven-mana destruction activation retains its target, tap and sacrifice costs,
and stack resolution. Rules contracts are CR 602 and 605.1a–605.3b.

Shared `TestComposableMana*`, `TestTypedManaAmount*`, and
`TestComposablePayment*` pass before and after migration. They cover immediate
resolution, production amounts, tap events, fixed production planning, clone
isolation, and failed payments. `TestUnstableObelisk` also passes before and
after migration. These declarations use proven primitives and need no duplicate
card simulation tests.

This completes one slice of the remaining fixed mana caller migrations.
`WithMultiManaAbility` remains in Guardian Idol, Mana Crypt, and Mishra's
Workshop, so its export is retained. No outstanding engine blocker is marked
resolved. Unrelated definitions retain engine imports in these files.

Validation: focused shared contracts; Limited and Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/limited ./cards/jumpstart`; `make test`; and `make lint`
pass with zero failures (lint: zero issues).

## Completed Slice: Brightmare Optional Entry Target and Life Gain

- [x] Migrate Brightmare's complete definition to `WhenEvent(Enters(This))`,
  `TargetPermanents`, `WithTargets`, `ForEach`, `Sequence`, `Tap`, `PowerOf`,
  and `GainLife` through the author DSL.
- [x] Select zero or one creature when the entry trigger goes on the stack,
  rather than when casting the creature spell. Entry without casting and
  targeting Brightmare itself now work.
- [x] Verify the complete Oracle comment, creature types, stats, and mana cost
  against `data/JMP.json`.
- [x] Audit target identity, current power, instruction order, and life-gain
  events. Existing typed adapters need no engine fix or new constructor.

The bounded collection and explicit loop member preserve singular semantics
without an implicit first-member conversion. Either player's creature can be
chosen. Tap first, then read its current power for the trigger controller's
life gain, even if the target was already tapped. Zero chosen targets produce
no actions; an illegal or returned target prevents resolution. Negative power
produces no life gain. Source departure does not end the trigger. Rules
contracts are CR 107.1b, 400.7, 603.3d, and 608.2b–h.

`TestBrightmareEntryTrigger/self` failed before migration: entry without casting
produced no trigger on the stack. The regression now covers self and opposing
targets, zero targets, an already tapped target, source departure, target
return, power changes before resolution, and negative power. The existing cast
test now chooses the target for the trigger. The zero-target test remains and
has an accurate name and comment.

This completes one slice of the remaining action, trigger, and card callback
life-gain migrations. General event capture and life-collection blockers remain
open; no legacy export is retired while other callers remain.

Validation: focused card and shared target, loop, tap, and life tests;
Jumpstart package tests; `go test ./cards/... -short -count=1`;
`go build ./...`; `go vet ./cards/jumpstart`; `make test`; and `make lint`
pass with zero failures (lint: zero issues).


## Completed Slice: Leaf Gilder and Llanowar Elves Typed Mana

- [x] Migrate both complete definitions from `WithManaAbility` to `Activate`,
  `Costs(TapCost(This))`, and `AddManaAmount(Green, Fixed(1))` through the
  author DSL.
- [x] Preserve one green mana, the tap cost, and untargeted activation.
- [x] Verify Oracle text, casting costs, creature types, and power/toughness
  against `data/JMP.json` and `data/LEA.json`. Add Llanowar Elves' full Oracle
  comment above registration.
- [x] Audit immediate resolution, fixed production planning, and tap payment.
  Existing adapters need no engine fix or new export.

Both mana abilities resolve immediately and add mana to the activating
controller's pool. The shared tap-cost and mana services enforce summoning
sickness and haste, require an untapped source, and support automatic mana
payment. Rules contracts are CR 302.6, 602, and 605.

Shared `TestComposableMana*`, `TestTypedManaAmount*`, and
`TestComposablePayment*` pass before and after migration. They cover immediate
resolution, tap events, fixed production planning, clone isolation, and failed
payment. The card definitions are declarative uses of these proven services;
no duplicate simulation tests are added. The existing Leaf Gilder test only
checks battlefield setup and is not evidence of mana production.

This completes one slice of the remaining fixed mana caller migrations.
Other cards still use `WithManaAbility`, so its export is retained. No engine
blocker is marked resolved. Unrelated definitions retain the engine imports.

Validation: focused shared contracts; Jumpstart and Limited package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart ./cards/limited`; `make test`; and `make lint`
pass with zero failures (lint: zero issues).


## Completed Slice: Mox Cycle Typed Mana

- [x] Migrate Mox Pearl, Mox Sapphire, Mox Jet, Mox Ruby, and Mox Emerald
  from `WithManaAbility` to `Activate`, `Costs(TapCost(This))`, and
  `AddManaAmount(color, Fixed(1))` through the author DSL.
- [x] Preserve one white, blue, black, red, or green mana, respectively,
  with a tap cost and no targets.
- [x] Add each full Oracle comment directly above its registration and verify
  its text, artifact type, and `{0}` cost against `data/LEA.json`.
- [x] Audit immediate resolution, fixed mana planning, payment, and cloning.
  Existing adapters need no engine fix or new export.

Each ability adds one mana to the activating controller's pool immediately,
without using the stack. The shared tap cost requires an untapped source;
fixed production remains available to automatic mana payment. The shared
mana service emits the tapped-for-mana event. Rules contracts are CR 602
and 605.1a–605.3b.

Shared `TestComposableMana*`, `TestTypedManaAmount*`, and
`TestComposablePayment*` pass before and after migration. They cover immediate
resolution, production amounts, tap events, planning, clone isolation, and
failed payment. These single-action declarations use proven primitives and
need no duplicate card simulation tests.

This completes one slice of the remaining fixed mana caller migrations.
Other cards retain `WithManaAbility`, so its export remains. No engine blocker
is marked resolved. Unrelated definitions retain this file's engine import.

Validation: focused shared contracts; Limited package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/limited`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Tolaria and Urborg Typed Mana

- [x] Migrate the last two Legends `WithManaAbility` declarations to
  `Activate`, `Costs(TapCost(This))`, and `AddManaAmount(color, Fixed(1))`.
- [x] Preserve one blue mana for Tolaria and one black mana for Urborg,
  with a tap cost and no targets.
- [x] Verify both full Oracle comments and legendary land types against
  `data/LEG.json`.
- [x] Audit immediate resolution, fixed mana planning, tap payment, and
  utility-ability selection. Existing adapters need no engine fix or new export.

Both mana abilities add mana to the activating controller's pool immediately,
without using the stack. Fixed production remains available to automatic mana
payment, and the shared mana service emits the tapped-for-mana event. The
separate utility abilities retain their existing declarations and timing.
Rules contracts are CR 602 and 605.1a–605.3b.

Shared `TestComposableMana*`, `TestTypedManaAmount*`,
`TestComposablePayment*`, and `TestActivationSelectionPrefersNonMana` pass
before and after migration. They cover immediate resolution, production amounts,
tap events, planning, clone isolation, failed payment, and utility selection.
Existing `TestTolaria` and `TestUrborg` utility scenarios also pass before and
after migration. Their mana-named scenarios only check battlefield setup;
shared engine contracts provide the mana evidence. These declarative changes
need no duplicate card simulation tests.

This completes the remaining fixed mana declarations in Legends. The broader
mana family and both utility migrations remain open. In particular, this slice
does not claim to audit Tolaria's removal of all “bands with other” abilities.
Other sets retain `WithManaAbility`, so its export remains. No outstanding
engine blocker is marked resolved.

Validation: focused shared contracts and card tests; Legends package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/legends`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Arabian Nights Fixed Land Mana

- [x] Migrate Desert, Elephant Graveyard, and Library of Alexandria from
  `WithManaAbility` to `Activate`, `Costs(TapCost(This))`, and
  `AddManaAmount(Colorless, Fixed(1))` through the author DSL.
- [x] Preserve one colorless mana, a tap cost, and no targets for each ability.
- [x] Match the full Oracle comments to `data/ARN.json`, including Desert's
  current self-reference, and preserve the land types and Desert subtype.
- [x] Audit immediate resolution, fixed mana planning, tap payment, and
  utility-ability selection. Existing adapters need no engine fix or new export.

The mana abilities add one colorless mana to the activating controller's pool
without using the stack. Automatic payment can use the fixed production, and
the shared mana service emits the tapped-for-mana event. The mana abilities
have no step or hand-size restriction. Desert's end-of-combat restriction and
Library's seven-card condition apply only to their separate utility abilities.
Rules contracts are CR 602 and 605.1a–605.3b.

Shared `TestComposableMana*`, `TestTypedManaAmount*`,
`TestComposablePayment*`, and `TestActivationSelectionPrefersNonMana` pass
before and after migration. They cover immediate resolution, tap events,
planning, clone isolation, failed payment, and utility selection. Existing
`TestDesert`, `TestElephantGraveyard`, and `TestLibraryOfAlexandria` also pass
before and after migration. These single-action mana declarations use proven
primitives and need no duplicate card simulation tests.

This completes the three remaining fixed `WithManaAbility` declarations in
Arabian Nights. City of Brass's any-color mana and the remaining legacy utility
abilities are separate work. Other sets retain `WithManaAbility`, so its export
remains. No outstanding engine blocker is marked resolved.

Validation: focused shared contracts and card tests; Arabian Nights package
tests; `go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/arabian`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Jumpstart Fixed Colorless Land Mana

- [x] Migrate the fixed colorless mana abilities on Buried Ruin, Mirrodin's
  Core, Phyrexian Tower, and Riptide Laboratory from `WithManaAbility` to
  `Activate`, `Costs(TapCost(This))`, and `AddManaAmount(Colorless, Fixed(1))`.
- [x] Preserve one colorless mana, a tap cost, and no targets for each ability.
- [x] Verify full Oracle comments and land types against `data/JMP.json`,
  including Phyrexian Tower's legendary supertype.
- [x] Select ability index 0 explicitly in the four existing colorless mana
  scenarios so utility-ability preference cannot select a different ability.
- [x] Audit immediate resolution, fixed mana planning, payment, and ability
  selection. Existing adapters need no engine fix or new export.

Each migrated ability adds one colorless mana to the activating controller's
pool without using the stack. Automatic payment can use the fixed production,
and the shared mana service emits the tapped-for-mana event. The separate
utility costs do not apply to these mana abilities. Rules contracts are
CR 602 and 605.1a–605.3b.

Shared `TestComposableMana*`, `TestTypedManaAmount*`,
`TestComposablePayment*`, and `TestActivationSelectionPrefersNonMana` pass
before and after migration. They cover immediate resolution, production,
tap events, automatic payment, clone isolation, failed payment, and ability
selection. Existing `TestBuriedRuin`, `TestMirrodinsCore`,
`TestPhyrexianTower`, and `TestRiptideLaboratory` also pass. These declarations
use proven primitives and need no duplicate card simulation tests.

This completes one slice of the remaining fixed mana caller migrations.
Riptide Laboratory's complete definition now uses typed abilities. The other
utility declarations and Thriving lands remain separate work. Other callers
retain `WithManaAbility`, so its export remains. No outstanding engine blocker
is marked resolved; unrelated definitions retain this file's engine import.

Validation: focused shared contracts and card tests; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Sisters of the Flame Typed Mana

- [x] Migrate Fourth Edition's last production `WithManaAbility` declaration
  to `Activate`, `Costs(TapCost(This))`, and `AddManaAmount(Red, Fixed(1))`.
- [x] Preserve one red mana, a tap cost, and no targets. The complete card
  definition uses the author DSL.
- [x] Verify the full Oracle comment, `{1}{R}{R}` casting cost, Human Shaman
  subtypes, and 2/2 characteristics against `data/4ED.json`.
- [x] Audit immediate resolution, fixed mana planning, and tap payment.
  Existing adapters need no engine fix or new export.

The ability adds one red mana to the activating controller's pool without
using the stack. Fixed production remains available to automatic payment,
and the shared mana service emits the tapped-for-mana event. The shared tap
cost requires an untapped source and enforces summoning sickness and haste.
Rules contracts are CR 302.6, 602, and 605.1a–605.3b.

`TestSistersOfTheFlame` passes before and after migration and checks tapping
and red mana production. Shared `TestComposableMana*`, `TestTypedManaAmount*`,
and `TestComposablePayment*` also pass before and after migration. They cover
immediate resolution, amounts, tap events, automatic payment, clone isolation,
and failed payment. This single-action declaration uses proven primitives
and registration validation; no duplicate card simulation test is required.

This completes the remaining fixed mana declaration in Fourth Edition.
Other sets retain `WithManaAbility`, so its export remains. The broader mana
family remains open; no outstanding engine blocker is marked resolved.

Validation: focused shared contracts; Fourth Edition package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/fourthedition`; `make test`; and `make lint` pass with zero
failures (lint: zero issues).

## Completed Slice: Pendelhaven Exact Current-Stat Filters

- [x] Add `HasPowerEqual(n)` and `HasToughnessEqual(n)` to the author DSL,
  using the existing current-characteristic services.
- [x] Replace Pendelhaven's mutable engine callback with
  `And(IsCreature, HasPowerEqual(1), HasToughnessEqual(1))`.
- [x] Verify the complete Oracle comment and legendary land type against
  `data/LEG.json`; preserve both typed activations and their costs.
- [x] Audit current characteristics, target rechecks, incarnation identity,
  clone isolation, and temporary boost lifetime. No engine fix is required.

Either player's creature is eligible if its current power and toughness are
both exactly 1. Check this restriction at activation and resolution. The
resolved +1/+2 boost lasts until cleanup; it does not end when the creature
ceases to be 1/1. A returned card is a new object. Rules contracts are CR 400.7,
602.2, 608.2b, 611.2, and 613.

`TestExactStatsFilters` first failed to compile because the constructors were
absent. It now covers each characteristic independently, continuous changes,
negative values, and typed activation composition. `TestExactStatsTargetResolution`
covers power and toughness changes before resolution, a returned target, and
clone isolation. Existing `TestPendelhaven`, `TestTargetIncarnationAfterReturn`,
`TestComposableContinuousCapture`, and `TestComposableSelfPumpLifetime` pass.
The card's existing boost test passes before and after migration.

This completes one slice of the remaining author-boundary and filter work.
Pendelhaven now uses only author DSL declarations. Other definitions in the
file retain mutable callbacks; the wider author boundary remains open. No
legacy export is retired or outstanding engine blocker marked resolved.

Validation: focused adapter and shared contract tests; Legends package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/legends`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Bazaar of Baghdad Typed Draw and Discard

- [x] Migrate Bazaar of Baghdad's complete definition to `Activate`, grouped
  `Costs(TapCost(This))`, and `Sequence(Draw(You, Fixed(2)), Discard(You, Fixed(3)))`.
- [x] Preserve the full Oracle text and land type, verified against
  `data/ARN.json`. The activation has no mana cost or target.
- [x] Audit instruction order, resolving controller, payment, and partial
  discard. Existing typed adapters need no engine fix or new export.

The activation uses the stack after paying its tap cost. The stored controller
first draws two cards, then chooses three cards from the resulting hand to
discard. If fewer than three cards remain, discard as many as possible. The
shared draw and discard services apply replacements and emit events. Source
departure does not stop resolution. Rules contracts are CR 121, 602, 608.2c,
609.3, and 701.8.

`TestBazaarOfBaghdad` and shared `TestTypedDrawAmount*`,
`TestTypedDiscardAmount*`, `TestComposablePayment*`, and
`TestComposableSequenceSourceIdentity` pass before and after migration.
The shared tests cover controller context, clone isolation, copy context,
replacement handling, discard choice, short hands, and payment atomicity.
The existing card scenario covers both draw and discard counts. This migration
composes proven primitives and needs no duplicate simulation tests.

This completes one slice of the remaining activation and string-bound action
migrations. Other callers retain the legacy helpers, so no export is retired.
No outstanding engine blocker is marked resolved. Unrelated definitions retain
this file's engine imports.

Validation: focused shared contracts; Arabian Nights package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/arabian`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Mishra's Factory Typed Mana

- [x] Migrate Antiquities' last `WithManaAbility` declaration to `Activate`,
  `Costs(TapCost(This))`, and `AddManaAmount(Colorless, Fixed(1))`.
- [x] Preserve one colorless mana, the tap cost, and no targets. Verify the
  full Oracle comment and land type against `data/ATQ.json`.
- [x] Correct the existing mana scenario to select ability index 0 explicitly
  and assert exact production and tapping. The previous default selected the
  animation ability, and the loose assertion passed on harness-supplied mana.
- [x] Audit immediate resolution, automatic mana planning, payment, and
  utility-ability selection. Existing adapters need no engine fix or new export.

The mana ability adds one colorless mana to the activating controller's pool
without using the stack. Fixed production remains available to automatic
payment, and the shared mana service emits the tapped-for-mana event. The tap
cost enforces summoning sickness if the land is a creature. Rules contracts
are CR 302.6, 602, and 605.1a–605.3b.

Shared `TestComposableMana*`, `TestTypedManaAmount*`,
`TestComposablePayment*`, and `TestActivationSelectionPrefersNonMana` pass.
They cover immediate resolution, production, events, planning, clone isolation,
failed payment, and utility selection. `TestMishrasFactory` and the Antiquities
package tests pass. No duplicate simulation test is needed for this declaration.

This completes one slice of the remaining fixed mana caller migrations.
Factory's animation callback, Workshop's restricted mana, and the Urza lands'
conditional mana remain separate work. Other sets retain `WithManaAbility`,
so its export remains. No outstanding engine blocker is marked resolved.

Validation: focused shared contracts; Antiquities package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/antiquities`; `make test`; and `make lint` pass with zero
failures (lint: zero issues).

## Completed Slice: Basalt Monolith Typed Activations

- [x] Migrate both activated abilities to `Activate` with explicit `Costs`.
  Use `AddManaAmount(Colorless, Fixed(3))` for mana and `Untap(This)` for untap.
- [x] Add the full Oracle comment and verify the artifact type and `{3}`
  casting cost against `data/LEA.json`.
- [x] Preserve the untap-step restriction, three colorless mana production,
  tap cost, and separate `{3}` untap cost. Neither ability has targets.
- [x] Audit mana classification, automatic payment, source identity, and
  clone isolation. Existing adapters need no engine fix or new export.

The mana ability resolves immediately and supports automatic mana payment.
The untap ability uses the stack and affects only its captured source
incarnation. Its payment does not require tapping, so it can be activated
while the Monolith is tapped. The untap-step restriction does not prevent
this effect from untapping the artifact. Rules contracts are CR 400.7, 502,
602, 605, and 608.

`TestBasaltMonolith` passes before and after migration and checks the next
untap step. Shared `TestComposableMana*`, `TestTypedManaAmount*`,
`TestComposablePayment*`, `TestComposableSourceUntap*`, and
`TestActivationSelectionPrefersNonMana` pass before and after migration.
They cover production, immediate resolution, planning, payment failures,
source departure and return, phasing, control changes, and clone isolation.
These declarations use proven primitives; no duplicate simulation test is needed.

This completes one slice of the remaining mana and activation migrations.
Other cards retain the legacy helpers, so no export is retired. No engine
blocker is marked resolved. Unrelated definitions retain the engine import.

Validation: focused shared contracts; Limited package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/limited`; `make test`; and `make lint` pass with zero
failures (lint: zero issues).


## Completed Slice: Fallen Empires Sacrifice-Land Typed Mana

- [x] Migrate both mana abilities on Dwarven Ruins, Ebon Stronghold,
  Havenwood Battleground, Ruins of Trokair, and Svyelunite Temple to
  `Activate`, explicit `Costs`, and `AddManaAmount(color, Fixed(n))`.
- [x] Preserve entry tapped, one mana for tapping, and two mana for tapping
  and sacrificing the source. Both abilities have no targets.
- [x] Verify all five complete Oracle comments and land types against
  `data/FEM.json`.
- [x] Strengthen the existing scenarios to check exact mana production and
  sacrifice to the graveyard. Select Dwarven Ruins' first ability explicitly
  and assert that it taps.
- [x] Audit mana classification, source sacrifice, and payment. Existing
  adapters need no engine fix or new export.

Both abilities resolve immediately without using the stack. The second pays
both tap and sacrifice costs before adding two mana to the activating
controller's pool; the departed source is not needed for the fixed amount.
The tap cost requires an untapped source and enforces summoning sickness if
the land is a creature. Rules contracts are CR 302.6, 602, and 605.1a–605.3b.

The strengthened five card tests pass before and after migration. Their exact
production assertions include the five mana of each color supplied by the
activation script. Shared `TestComposableMana*`, `TestTypedManaAmount*`,
`TestComposablePayment*`, and `TestActivationSelectionPrefersNonMana` pass
before and after migration. These cover immediate resolution, fixed production
planning, events, controller context, clone isolation, and payment.

The one-mana abilities remain available to automatic payment. The planner's
existing exclusion of sacrifice costs remains; this slice does not extend
automatic planning to the two-mana abilities. Explicit activation is supported.
This completes Fallen Empires' remaining `WithManaAbility` caller migration.
Other sets retain the helper, so its export remains. Storage lands and the
broader mana family remain open; no engine blocker is marked resolved.

Validation: focused shared contracts; Fallen Empires package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/fallen_empires`; `make test`; and `make lint` pass with zero
failures (lint: zero issues).


## Completed Slice: Conservator Typed Prevention

- [x] Migrate Conservator to `Activate`, `Costs(Mana("{3}"), TapCost(This))`,
  and `PreventNextDamage(You, Fixed(2))` through the author DSL.
- [x] Verify the full Oracle comment, artifact type, and `{4}` casting cost
  against `data/LEA.json`. The ability has no target or timing restriction.
- [x] Retire `AddPreventionShieldToControllerStep`, its executor record, and
  its author DSL export after migrating its sole caller.
- [x] Audit payment, recipient identity, shield consumption, and expiration.
  Existing typed adapters need no engine fix or new export.

The ability uses the stack after paying three generic mana and tapping the
source. It prevents the next two damage to the stored resolving controller
this turn, including damage across multiple events. Source departure does
not end the shield. Unused capacity expires at cleanup. Rules contracts are
CR 602, 608, and 615.

`TestConservator` and shared `TestTypedPrevention*` and
`TestComposablePayment*` pass before and after migration. The card scenario
checks untargeted prevention during the opponent's turn and the tap cost.
Shared tests cover capacity consumption, expiration, source departure,
clone isolation, and payment atomicity. This single-action declaration uses
proven primitives and needs no duplicate simulation test.

This completes one slice of the remaining activation and prevention caller
migrations. The broader replacement family remains open; no outstanding
engine blocker is marked resolved. Unrelated definitions retain this file's
engine imports.

Validation: focused shared contracts; Limited package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/limited`; `make test`; and `make lint` pass with zero
failures (lint: zero issues).


## Completed Slice: Ashnod's Altar and Phyrexian Tower Sacrifice Mana

- [x] Migrate both sacrifice-for-mana abilities to `Activate`, explicit
  `Costs`, and `AddManaAmount(color, Fixed(2))`.
- [x] Correct both sacrifice costs to permit the source when it is a creature,
  using `SacrificeMatchingIncludingSourceCost(IsCreature, ...)`.
  Oracle says “a creature,” not “another creature.”
- [x] Verify the complete Oracle comments, types, and casting costs against
  `data/ATQ.json` and `data/JMP.json`.
- [x] Add failing card regressions for sacrificing each creature source;
  verify departure to the graveyard and immediate, exact mana production.
- [x] Audit mana classification and payment. Existing engine services and
  DSL adapters support the required behavior; no engine fix is needed.

Ashnod's Altar sacrifices a creature its controller controls for two colorless
mana, with no tap cost. Phyrexian Tower requires both tapping and sacrificing
a creature its controller controls for two black mana. Either source can pay
its own sacrifice cost if it is a creature. The Tower must also be able to pay
its tap cost. The regressions grant haste to the creature sources. Both abilities
have no targets and resolve immediately after payment, even if their source
has left. Rules contracts are CR 302.6, 602, 605.1a–605.3b, and 701.21.

`TestAshnodsAltarCanSacrificeCreatureSource` and
`TestPhyrexianTowerCanSacrificeCreatureSource` each failed with
“cannot activate ability” before migration and pass afterward. Existing card
scenarios cover sacrificing a separate creature and the Tower's colorless
ability. Shared `TestComposableMana*`, `TestTypedManaAmount*`, and
`TestMatchingSacrificeCost*` pass, covering immediate resolution, controller
context, clone isolation, failed payment, and phased-out sacrifice candidates.

This completes one slice of the remaining mana and activation migrations.
The planner still excludes sacrifice costs; explicit activation supports these
abilities. The Tower's separate colorless ability remains available for automatic
payment. Other legacy callers remain, so no export is retired or broader engine
blocker marked resolved. Unrelated definitions retain each file's engine import.

Validation: focused card regressions and shared contracts; Antiquities and
Jumpstart package tests; `go test ./cards/... -short -count=1`;
`go build ./...`; `go vet ./cards/antiquities ./cards/jumpstart`;
`make test`; and `make lint` pass with zero failures (lint: zero issues).


## Completed Slice: Limited Basic and Dual Land Typed Mana

- [x] Migrate all five basic lands and ten dual lands to `Activate`,
  `Costs(TapCost(This))`, and `AddManaAmount(color, Fixed(1))`.
- [x] Preserve basic supertypes, land subtypes, and separate intrinsic mana
  abilities. Add full Oracle reminder text from `data/LEA.json` and, for
  Volcanic Island, `data/catalog/LEB.json`.
- [x] Resolve the explicit typed mana bonus gap after a failing engine
  regression: retain actual produced colors and call the shared bonus service
  once when a tap-cost mana ability produces a positive amount.
- [x] Update test-harness mana supply and attachment ability inspection to
  recognize typed production through the existing shared APIs.
- [x] Extend the dual-land automatic-payment regression to cover typed
  abilities as well as legacy abilities.
- [x] Resolve ordinary-action enumeration after a failing regression: exclude
  typed tap-only mana production handled by automatic payment, while keeping
  utility abilities and sacrifice mana available as explicit actions.

The abilities produce one mana immediately without using the stack. Each
basic land type provides its own tap ability; dual lands do not produce both
colors from one tap. The existing tap payment enforces summoning sickness
when a land is a creature. Rules contracts are CR 305.6, 302.6, and 605.

`TestComposableManaBonuses` failed before the engine fix: explicit activation
produced one or three red mana instead of three or five with two bonuses.
It now checks stacked bonuses, zero production, no tap cost, immediate
resolution, and clone isolation. `TestComposableManaBonusUsesProducedColor`
checks one matching bonus across repeated production instructions, existing
pool colors, explicit and automatic payment, one tapped-for-mana event, and
failed repeat activation. Nested resolution tests verify color and amount
isolation. Existing Mana Flare and Gauntlet of Might scenarios also pass.

`TestComposableManaOrdinaryActivationChoices` failed with three choices
instead of two before the enumeration fix. The full search parity test had
timed out while exploring redundant basic-land mana actions; it passes after
the fix.

The production total in the event excludes bonus mana. Automatic payment
keeps its existing bonus path and receives no duplicate bonus. No legacy
export is retired: other sets and engine fixtures retain `WithManaAbility`.
The broader mana and event-capture audits remain open.

Validation: focused regressions; Limited package tests; card regression and
registration validation; `go build ./...`; `go vet ./...`; `make test`; and
`make lint` pass with zero failures (lint: zero issues).


## Completed Slice: Fire Sprites Typed Mana Conversion

- [x] Migrate Fire Sprites to `Activate`, `Costs(Mana("{G}"), TapCost(This))`,
  and `AddManaAmount(Red, Fixed(1))`.
- [x] Verify the full Oracle comment, `{1}{G}` casting cost, Faerie subtype,
  1/1 stats, and flying against `data/LEG.json`.
- [x] Strengthen the existing card scenario to activate the ability and check
  tapping and exact red mana production. The assertion includes the five red
  mana supplied by the activation script.
- [x] Audit mana classification and costed automatic planning. Existing DSL
  adapters and engine services support the complete ability; no engine fix
  or new export is required.

The controller pays one green mana and taps Fire Sprites to add one red mana.
The ability has no targets and resolves immediately without using the stack.
The tap cost enforces summoning sickness. Automatic payment retains the green
activation cost. Rules contracts are CR 302.6, 602, and 605.1a–605.3b.

The strengthened `TestFireSprites` passes before and after migration. Shared
`TestComposableMana*`, `TestTypedManaAmount*`, and `TestComposablePayment*`
pass, including immediate resolution, failed payment, planning, and clone
isolation. This declaration uses proven primitives and needs no duplicate
simulation test.

This completes one slice of the remaining mana and activation caller migrations.
Other cards retain the legacy helpers, so no export is retired. The broader mana
family and outstanding engine blockers remain open.

Validation: focused shared contracts; Legends package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/legends`; `make test`; and `make lint` pass with zero
failures (lint: zero issues).


## Completed Slice: Vampire Neonate Typed Drain Activation

- [x] Migrate the complete ability to `Activate`, explicit mana and tap
  `Costs`, and `Sequence(LoseLifeEach(OpponentsOf(You), Fixed(1)),
  GainLife(You, Fixed(1)))`.
- [x] Replace the direct life mutations and event-free `PlayerGainLife`
  callback with the existing typed actions. No engine fix is required.
- [x] Verify the full Oracle comment, `{B}` casting cost, Vampire subtype,
  and 0/3 stats against `data/JMP.json`.
- [x] Resolve this card's missing life-gain event after a failing regression.

The controller pays two generic mana and taps the source. The untargeted
ability uses the stack. At resolution, each opponent loses 1 life, then the
stored controller gains 1 life. Source departure does not stop resolution.
The gain event reports the actual positive gain after modifiers. Rules
contracts are CR 302.6, 602, and 608.2c.

`TestVampireNeonateResolution` failed before migration because normal,
doubled, and post-departure gains emitted no life-gain event. It now covers
those cases, suppressed gain, loss-before-gain order, exact mana and tap
payment, untargeted stack placement, and failed payment with no mana or an
already tapped source. The existing activation scenario and shared payment,
singular and collection life, life-outcome, and opponent-life-event tests pass.

This completes one card in the life-gain callback audit and remaining
activation migrations. Other callbacks and replacement-order blockers remain
open. No legacy export is retired.

Validation: focused card and shared contract tests; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Noxious Newt Typed Mana

- [x] Migrate the complete mana ability to `Activate`,
  `Costs(TapCost(This))`, and `AddManaAmount(Green, Fixed(1))`.
- [x] Verify the full Oracle comment, `{1}{G}` casting cost, Salamander
  subtype, 1/2 stats, and deathtouch against `data/SOS.json`.
- [x] Strengthen the existing card scenario to check tapping and exact green
  mana production. The total includes five green mana from the activation script.
- [x] Audit mana classification and payment. Existing typed adapters support
  this ability; no engine fix or new export is required.

The untargeted ability adds one green mana to the activating controller's pool
immediately, without using the stack. The tap cost requires an untapped source
and enforces summoning sickness. Fixed production remains available to automatic
payment. Rules contracts are CR 302.6, 602, and 605.1a–605.3b.

`TestNoxiousNewt_DeathtouchAndMana` passes before and after migration with
its stronger assertions. Shared `TestComposableMana*`, `TestTypedManaAmount*`,
and `TestComposablePayment*` also pass before and after migration. They cover
immediate resolution, production, events, automatic planning, clone isolation,
and failed payment. This declaration uses proven primitives and needs no
additional simulation test.

This completes one slice of the remaining fixed-mana caller migrations.
Other cards retain `WithManaAbility`, so its export remains. The broader mana
family and outstanding engine blockers remain open.

Validation: focused card and shared contract tests; Secrets of Strixhaven
package tests; `go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/secretsofstrixhaven`; `make test`; and `make lint` pass
with zero failures (lint: zero issues).


## Completed Slice: Guardian Idol and Mana Crypt Typed Mana

- [x] Migrate both fixed colorless mana abilities to `Activate`,
  `Costs(TapCost(This))`, and `AddManaAmount`: one mana for Guardian Idol
  and two for Mana Crypt.
- [x] Verify the full Oracle comments, artifact types, and casting costs
  against `data/JMP.json` and `data/PHPR.json`.
- [x] Audit immediate resolution, tap costs, and automatic payment against
  the shared typed mana contracts. No engine fix or new export is required.

Both untargeted abilities add mana to the activating controller's pool without
using the stack. Guardian Idol enters tapped and must untap before it can pay
the tap cost. Its animation ability and Mana Crypt's upkeep coin-flip ability
remain separate. This slice migrates only the two mana abilities.

`TestGuardianIdol`, `TestManaCrypt*`, `TestComposableMana*`,
`TestTypedManaAmount*`, and `TestComposablePayment*` pass before and after
migration. Shared tests cover immediate resolution, fixed production, events,
automatic planning, clone isolation, and payment. These declarations use proven
primitives and need no additional simulation tests.

This completes one slice of the remaining fixed-mana caller migrations.
Mishra's Workshop is the sole remaining production `WithMultiManaAbility`
caller; its artifact-spell spending restriction needs a typed adapter before
that helper can be retired. No outstanding engine blocker is marked resolved.

Validation: focused card and shared contract tests; Jumpstart and Promo package
tests; `go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart ./cards/promo`; `make test`; and `make lint` pass
with zero failures (lint: zero issues).


## Completed Slice: Mirrodin's Core Typed Charge Activation

- [x] Migrate the charge-counter ability to `Activate`, `Costs(TapCost(This))`,
  and `AddCounters(This, Charge, Fixed(1))` through the author DSL.
- [x] Verify the full Oracle comment and land type against `data/JMP.json`.
- [x] Remove the land file's last direct engine import.
- [x] Strengthen `TestMirrodinsCore` to select the charge ability explicitly
  and assert both the tap cost and exactly one charge counter.
- [x] Audit the counter and payment adapters. Existing engine services support
  the ability; no engine fix or new export is required.

The ability has no targets and uses the stack after tapping the land. It puts
one charge counter on the source incarnation at resolution through the shared
counter action. It does not produce mana. Rules contracts are CR 602 and 608.
The separate colorless mana ability is already typed; the any-color mana
activation remains a legacy definition outside this slice.

The strengthened card test and shared `TestTypedCounterAmount*`,
`TestComposablePayment*`, and `TestCounterValueIdentity` pass before and after
migration. Shared contracts cover replacement amounts, absent subjects, source
identity, clone isolation, and payment atomicity. This single-action declaration
uses proven primitives and needs no additional simulation test.

This completes one slice of the remaining activation and counter-action caller
migrations. Other callers retain legacy helpers, so no export is retired and
no outstanding engine blocker is marked resolved.

Validation: focused card and shared contract tests; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Amulet of Kroog Typed Targeted Prevention

- [x] Migrate the complete ability to `Activate`,
  `Costs(Mana("{2}"), TapCost(This))`, `TargetDamageRecipient`,
  `WithTargets`, and `PreventNextDamage(recipient, Fixed(1))`.
- [x] Verify the full Oracle comment, artifact type, and `{2}` casting cost
  against `data/ATQ.json`.
- [x] Strengthen the existing creature scenario so prevention determines
  survival: Hill Giant survives Lightning Bolt with exactly two damage.
  Both creature and player scenarios now assert the tap cost.
- [x] Audit target selection, payment, recipient identity, shield consumption,
  and expiration. Existing typed adapters need no engine fix or new export.

The ability uses the stack after paying two generic mana and tapping the
source. It prevents the next one damage to the selected recipient this turn.
The shared target supports either player's creatures, planeswalkers, and
players. Target legality is checked again at resolution. A permanent shield
stays with its captured incarnation; source departure does not end it. Unused
capacity expires at cleanup.

The strengthened `TestAmuletOfKroog`, `TestRakalite*`,
`TestTypedPrevention*`, `TestPreventionShieldNewIncarnation`, and
`TestComposablePayment*` pass before and after migration. Shared contracts
cover target eligibility, failed payment, source departure, recipient lifetime,
capacity, expiration, and clone isolation. This single-action declaration uses
proven primitives and needs no additional simulation test.

This completes one slice of the remaining activation and prevention caller
migrations. Other cards retain `PreventDamageToTarget`, so its export remains.
The broader replacement and damage-outcome blockers remain open.

Validation: focused card and shared contracts; Antiquities package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/antiquities`; `make test`; and `make lint` pass with zero
failures (lint: zero issues).


## Completed Slice: Jumpstart Typed Entry Life-Gain Triggers

- [x] Migrate the entry abilities of Angel of Mercy, Bulwark Giant, Sylvan
  Brushstrider, and Thragtusk to `WhenEvent(Enters(This), GainLife(You, Fixed(n)))`.
  Preserve their respective life amounts of 3, 5, 2, and 5.
- [x] Verify full Oracle comments, costs, types, stats, and Angel of Mercy's
  flying against `data/JMP.json`.
- [x] Audit entry detection, controller context, and life-gain outcomes.
  Existing typed adapters support these abilities; no engine fix is needed.

These mandatory abilities trigger on entry, including entry without casting,
and have no targets. They gain life for the stored resolving controller even
if the source leaves before resolution. The shared life action applies existing
replacement handling and emits the actual positive life-gain outcome.
Thragtusk's separate leave trigger remains outside this slice.

All four existing card scenarios and shared `TestComposableEntry*`,
`TestComposableLife*`, `TestTypedSingularLife*`, `TestLifeGainOutcome*`, and
`TestComposableSourceLifetimes` pass before and after migration. Shared contracts
cover absent and phased sources, copied triggers, controller context, clone
isolation, and replaced life gain. These single-action declarations use proven
primitives and need no additional simulation tests.

This completes the four remaining direct `EntersBattlefieldTrigger(GainLife(...))`
callers in production cards. Other entry triggers retain the legacy helper, so
no export is retired. General event capture and life replacement ordering remain
open; no outstanding engine blocker is marked resolved.

Validation: focused card and shared contracts; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Nocturnal Feeder Typed Death Drain

- [x] Migrate the complete death ability to `WhenEvent(Dies(This), ...)`,
  `Sequence`, `LoseLifeEach(OpponentsOf(You), Fixed(2))`, and
  `GainLife(You, Fixed(2))` through the author DSL.
- [x] Remove the callback's direct life mutations and implicit trigger target.
  Existing typed actions and death-event capture support the complete ability;
  no engine fix or new constructor is required.
- [x] Verify the full Oracle comment, `{2}{B}` cost, Vampire Rogue subtypes,
  2/1 characteristics, and flying against `data/JMP.json`.
- [x] Resolve this card's missing life-loss and life-gain events with a
  regression that failed before migration.

`TestNocturnalFeederDeathResolution` covers an untargeted death trigger,
controller identity distinct from ownership, source return under the opponent's
control before resolution, life loss before gain, and normal, doubled, or
suppressed life gain. The opponent loses two life, and the stored controller
gains two life before replacements. A life-loss watcher triggers exactly once;
the life-gain event reports the actual positive gain. The existing death test
and shared death, life-event, replacement, and sequence contracts pass.

This completes one card in the life-gain callback audit and remaining trigger
and action migrations. Other callbacks and general replacement-order blockers
remain open. No legacy export is retired.

Validation: focused regressions and shared contracts; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Cauldron Familiar Typed Entry Drain

- [x] Migrate the complete entry ability to `WhenEvent(Enters(This), ...)`,
  `Sequence`, `LoseLifeEach(OpponentsOf(You), Fixed(1))`, and
  `GainLife(You, Fixed(1))` through the author DSL.
- [x] Remove the callback's direct life mutations and implicit trigger target.
  Existing typed adapters support the complete entry ability; no engine fix
  or new constructor is required.
- [x] Verify the full Oracle comment, `{B}` cost, Cat subtype, and 1/1
  characteristics against `data/JMP.json`.
- [x] Resolve this ability's missing life events and failure to resolve after
  source departure with a regression that failed before migration.

`TestCauldronFamiliarEntryResolution` covers an untargeted entry trigger,
controller identity distinct from ownership, source departure before resolution,
life loss before gain, and normal, doubled, or suppressed life gain. Each
opponent loses one life; the stored controller gains one life before
replacements. The life-loss watcher triggers once, and the life-gain event
reports the actual positive gain. Existing entry and Food-sacrifice return
tests also pass.

This completes one entry ability in the life-gain callback audit. The separate
graveyard activation remains for the graveyard-reference migration. General
life replacement ordering and other callback audits remain open. No legacy
export is retired and no outstanding engine blocker is marked resolved.

Validation: focused regressions; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Wall of Blossoms and Phyrexian Rager Typed Entry Draws

- [x] Migrate both complete entry abilities to `WhenEvent(Enters(This), ...)`
  and `Draw(You, Fixed(1))` through the author DSL.
- [x] Use `Sequence` for Phyrexian Rager's draw followed by
  `LoseLife(You, Fixed(1))`.
- [x] Verify full Oracle comments, mana costs, subtypes, base power/toughness,
  and Wall of Blossoms's defender against `data/JMP.json`.
- [x] Remove the implicit source targets that stopped both entry abilities
  from resolving after source departure. The new regression failed before
  migration. Existing typed adapters need no engine fix or new constructor.

`TestEntryDrawResolution` covers both cards with the source present or departed,
an empty library, and a controller different from the card's owner. Entry creates
one untargeted trigger; its stored controller draws on resolution. Draw events
occur before Phyrexian Rager's life loss. An empty-library draw still attempts the
draw and does not skip the life loss; the controller loses the game after
resolution. The owner and opponent remain unaffected.

Existing cast-path tests and shared typed draw, entry-trigger, sequence, and
life contracts pass. Shared contracts cover draw replacements, copies, clones,
source absence, and binding validation. This completes two callers in the
remaining trigger and action migrations. Other callers retain the legacy
helpers, so no export is retired. No outstanding engine blocker is marked
resolved.

Validation: focused regressions and shared contracts; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Living Armor Typed Counter Activation

- [x] Migrate the complete ability to `Activate`, explicit tap and sacrifice
  `Costs`, `TargetPermanent`, `WithTargets`, and
  `AddCounters(creature, P0P1, ManaValueOf(creature))`.
- [x] Verify the full Oracle comment, artifact type, and `{4}` casting cost
  against `data/DRK.json`.
- [x] Strengthen `TestLivingArmor` to assert exactly four +0/+1 counters,
  in addition to the resulting toughness and source sacrifice.
- [x] Audit target selection, payment, resolution-time values, and counter
  replacement handling. Existing typed adapters need no engine fix.

The activation selects either player's creature before tapping and sacrificing
Living Armor. It uses the stack and reads the legal target's mana value at
resolution. The shared counter action applies counter replacements. The amount
is not an X mana payment. Source sacrifice does not stop the ability.

The strengthened card test passes before and after migration. Shared
`TestTypedCounterAmount*`, `TestManaValue*`, `TestTypedTarget*`, and
`TestComposablePayment*` pass. They cover dynamic amounts, nonpositive amounts,
replacements, target lifetime, binding validation, and clone isolation. This
single-action declaration uses proven primitives and needs no new simulation
test.

This completes one slice of the remaining activation and counter-action caller
migrations and the last production `TargetPermanentManaValue` call. Engine
fixtures still use the legacy helper, so its retirement remains separate.
The shared monocolored-hybrid mana-cost parser blocker remains open; this slice
uses the existing mana-value service and does not claim to resolve that gap.

Validation: focused card and shared contracts; The Dark package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/thedark`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).

## Completed Slice: Spectral Sailor and Feral Prowler Typed Draw Abilities

- [x] Migrate Spectral Sailor's complete activation to `Activate`,
  `Costs(Mana("{3}{U}"))`, and `Draw(You, Fixed(1))`.
- [x] Migrate Feral Prowler's complete death trigger to
  `WhenEvent(Dies(This), Draw(You, Fixed(1)))`.
- [x] Verify both full Oracle comments, mana costs, creature types, power,
  toughness, and printed keywords against `data/JMP.json`.
- [x] Strengthen the existing card tests to assert the drawn card. The Prowler
  test now causes its death and also asserts its graveyard destination.
- [x] Audit the draw, activation, and death-trigger adapters. Existing engine
  services support both abilities; no engine fix is required.

Both abilities have no targets and draw for their stored controller. The Sailor
pays its mana cost before its ability goes on the stack. It has no tap cost,
sorcery restriction, or activation limit. The Prowler's mandatory trigger
resolves after its source leaves the battlefield. Both use the shared draw
replacement and event services.

The existing card tests and shared `TestTypedDrawAmount*`,
`TestComposableDeathTriggerHasNoImplicitTarget`, and `TestComposablePayment*`
pass. Shared contracts cover controller context, source absence, amounts,
replacements, copies, binding validation, and payment rollback. These
single-action abilities need no additional simulation tests.

This completes one slice of the remaining trigger, activation, and action
caller migrations. Other callers still use the legacy helpers, so no export
is retired. No outstanding engine blocker is marked resolved.

Validation: focused card and shared contract tests; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Tithebearer Giant Typed Entry Draw and Life Loss

- [x] Migrate the complete entry ability to `WhenEvent(Enters(This), ...)`,
  `Sequence`, `Draw(You, Fixed(1))`, and `LoseLife(You, Fixed(1))`.
- [x] Verify the full Oracle comment, `{5}{B}` cost, Giant Warrior subtypes,
  and 4/5 characteristics against `data/JMP.json`.
- [x] Remove the implicit source target that stopped the ability from resolving
  after source departure. The extended regression failed before migration.
- [x] Audit the existing entry, draw, sequence, and life adapters. No engine
  fix or new constructor is required.

`TestEntryDrawResolution` now includes Tithebearer Giant with its source present
or departed, an empty library, and a controller different from its owner.
The entry trigger has no targets. Its stored controller draws before losing
one life. An empty-library draw does not skip life loss, and the controller
loses the game after resolution. The owner remains unaffected.

The existing cast-path scenario and shared typed draw, entry, sequence, and
life contracts pass. This completes one slice of the remaining trigger and
action migrations. Other callers retain the legacy helpers, so no export is
retired. General engine blockers remain open.

Validation: focused regressions and shared contracts; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Phyrexian Gargantua Typed Entry Draw and Life Loss

- [x] Migrate the complete entry ability to `WhenEvent(Enters(This), ...)`,
  `Sequence`, `Draw(You, Fixed(2))`, and `LoseLife(You, Fixed(2))`.
- [x] Verify the full Oracle comment, `{4}{B}{B}` cost, Phyrexian Horror
  subtypes, and 4/4 characteristics against `data/JMP.json`.
- [x] Remove the implicit source target that stopped the ability from resolving
  after source departure. The extended regression failed before migration.
- [x] Audit entry, draw, sequence, and life adapters. Existing engine services
  support the ability; no engine fix or new constructor is required.

`TestEntryDrawResolution` now includes Gargantua with its source present or
departed, an empty library, and a one-card library. The trigger has no targets
and draws for its stored controller, who differs from the owner in the test.
Both draw events occur before the two-life loss. A short or empty library does
not skip life loss; a failed draw causes game loss after resolution. The owner
remains unaffected. The existing cast-path test also passes.

This completes one slice of the remaining trigger and action migrations.
Other callers retain the legacy helpers, so no export is retired. The recorded
engine blockers remain open.

Validation: focused regressions and shared contracts; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Gargoyle Sentinel Typed Keyword Activation

- [x] Migrate the complete activation to `Activate`, `Costs(Mana("{3}"))`,
  and `Until(EndOfTurn, RemoveKeyword(This, Defender), GrantKeyword(This, Flying))`.
- [x] Remove the custom layer callback and direct keyword mutations.
- [x] Verify the full Oracle comment, casting cost, artifact creature type,
  Gargoyle subtype, 3/3 stats, and printed defender against `data/JMP.json`.
- [x] Audit payment, source identity, layer ordering, and duration. Existing
  typed adapters support the ability; no engine fix or new constructor is needed.

The ability has no targets, tap cost, or activation limit. It uses the stack
after payment. Both keyword changes belong to one layer-6 effect on the
captured source incarnation and expire at cleanup. They do not affect a
returned incarnation. The printed defender ability remains on the card.

`TestGargoyleSentinel_GainsFlying` checks both flying and defender after
activation. Shared `TestComposableKeyword*` and `TestComposablePayment*`
contracts cover keyword ordering, source lifetime, cleanup, clone isolation,
and payment rollback. These proven primitives need no duplicate simulation
test. This completes one slice of the remaining activation and continuous-effect
caller migrations. Other callers retain legacy helpers; no export is retired
and no outstanding engine blocker is marked resolved.

Validation: focused card and shared contracts; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).


## Completed Slice: Jumpstart Artifact Library Triggers

- [x] Migrate Mana Geode's entry scry to
  `WhenEvent(Enters(This), Scry(Fixed(1)))`.
- [x] Migrate Prophetic Prism's entry draw to
  `WhenEvent(Enters(This), Draw(You, Fixed(1)))`.
- [x] Migrate Terrarion's battlefield-to-graveyard draw to
  `WhenEvent(Dies(This), Draw(You, Fixed(1)))`.
- [x] Verify all three complete Oracle comments, artifact types, and mana
  costs against `data/JMP.json`.
- [x] Audit event matching and library actions. Existing typed adapters use
  the shared trigger, scry, and draw services; no engine fix is required.

All three triggers are mandatory and have no targets. The library action
uses the stored trigger controller and does not require the source to remain
on the battlefield. `Dies(This)` matches the source's battlefield-to-graveyard
transition for noncreature artifacts too, including sacrifice and destruction.
Terrarion's draw remains a separate triggered ability from its mana ability.

`TestManaGeode`, `TestProphethicPrism`, `TestTerrarion`, and
`TestTerrarion_SacrificeDrawsCard` pass before and after migration. Shared
trigger, draw, and library-choice contracts cover source departure, controller
context, copies, clone isolation, replacements, and scry placement. Universal
registration validation checks these declarations. These single-action
triggers need no duplicate simulation tests.

This completes one slice of the remaining trigger and library-action caller
migrations. The three artifacts' mana abilities remain separate migration
work. Other cards still use the legacy trigger and action helpers, so no
export is retired and no outstanding engine blocker is marked resolved.

Validation: focused card and shared contract tests; Jumpstart package tests;
`go test ./cards/... -short -count=1`; `go build ./...`;
`go vet ./cards/jumpstart`; `make test`; and `make lint` pass with zero failures
(lint: zero issues).
