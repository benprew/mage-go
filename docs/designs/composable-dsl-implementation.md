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
- [ ] **Surrounding Card Definitions**: Migrate remaining surrounding definitions for Blindblast, Exclude, Goblin Lore, Thirst for Knowledge, and Rhystic Study. Damage-source LKI, stack references, hidden-zone choices, and event participants still need exact adapters.
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
