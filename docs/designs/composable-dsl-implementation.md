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
- [ ] **Surrounding Card Definitions**: Migrate remaining surrounding definitions for Blindblast, Exclude, Explore, Goblin Lore, Thirst for Knowledge, and Rhystic Study. Damage-source LKI, stack references, land allowances, hidden-zone choices, and event participants still need exact adapters.
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
- [ ] **String-Bound Action Migrations**: Continue destruction, return, tap-or-untap, attachment, and other string-bound action migrations where exact typed forms exist.

## Engine Blockers

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
- [x] Resolve repeated destruction and intermediate state observations in the Disk helper. One union collection now uses the existing simultaneous destruction executor.
- [x] Keep the collection no-regeneration prohibition on each destruction action; preserve permanent characteristics and simultaneous death snapshots.
- [x] Correct singular typed, positional, step, and string-bound no-regeneration execution to use the action-scoped prohibition. Migrate Terror; see the completed slice below.
- [ ] Migrate remaining no-regeneration callers before retiring positional exports: Abu Ja'far needs combat-opponent capture and simultaneous destruction; Detonate needs X-constrained targets and damage-source LKI; Aswan Jaguar needs a chosen-subtype target adapter. Crumble's saved-value migration is complete and the string-bound no-regeneration constructor is retired.

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
