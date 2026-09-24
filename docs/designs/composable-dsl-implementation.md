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
- [ ] **Positional Return Helpers**: Continue positional return helper migrations. Witch Hunter's separate damage ability awaits damage-source LKI. Audit Rakalite's delayed source identity and Prismari Charm's modes before retiring `ReturnToHandTarget`.
- [x] **Divine Offering**: Add `ManaValueOf`, capture with `SnapshotNumber`, and migrate destruction and controller life gain to typed actions. Retire the sole-caller helpers `DestroyTargetStep` and `GainLifeControllerFromVar`.
- [x] **Blight Attachment-Tap Capture**: Add `AttachedBecomesTapped`, capture the event land before trigger placement, migrate Blight, and retire `DestroyAttachedStep` and `DestroyAttachedData`. Correct repeated tap-event emission.
- [x] **Targeted Permanent Destruction**: Migrate all six production `DestroyTargetPermanent` callers to typed `Destroy` targets and retire the export. See the completed slice below.
- [x] **Targeted Land Destruction**: Migrate Sinkhole, Stone Rain, and Ice Storm to typed land targets and `Destroy`; retire `NewLandDestruction` and `DestroyTargetLand`. No engine blocker. See the completed slice below.
- [x] **Land-Wide Destruction**: Migrate Armageddon and Tsunami to typed `DestroyEach` collections; retire `DestroyAllLands`. No engine blocker. See the completed slice below.
- [x] **Collection Destruction Without Regeneration**: Add `DestroyEachWithoutRegeneration`, migrate Wrath of God and Shatterstorm, and retire both collection no-regeneration helpers. Resolve the action-scoping engine gap with a regression test first. See the completed slice below.
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
- [ ] Retain source incarnation and last-known characteristics through the existing damage and replacement pipeline, including death, return, and copying. `DamageSystem.ExecuteDamageToPlayer` and `ExecuteDamageToCreature` use `FindPermanent(sourceID)` for lifelink and deathtouch. Protection in `DealDamageToPermanent` also looks up the physical card ID; `LookupObject` prefers a live object with that ID. Add regressions for returned sources and departed sources with granted keywords before adding the typed adapter.

### Destruction Regeneration Scope
- [x] Keep the collection no-regeneration prohibition on each destruction action; preserve permanent characteristics and simultaneous death snapshots.
- [ ] Audit singular positional no-regeneration executors, which still grant `CantRegenerate`, before claiming action-scoped semantics for that family.

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
