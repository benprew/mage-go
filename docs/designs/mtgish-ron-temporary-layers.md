# RON typed temporary layers

Capability: P0-SCOPE-PERMANENT-LAYERS. Complete for the shapes below.

## Candidate comparison and selection

Fresh baseline: 2,445 records, 986 unique names, 859 compiled records,
1,586 rejected records, 251 compiled unique names, 735 rejected unique names.
Focused compiler tests and `make build` passed. Both strict loading and the
nested AST inventory exited 1 for collection rejection. The vocabulary file is
only a syntax snapshot; all 12 current source hashes match it.

| Candidate | Distinct affected names | Estimated new full-card compilations | Other blockers | Downstream capabilities | Implementation/test effort |
| --- | --- | --- | --- | --- | --- |
| Typed temporary-layer references, selected | 107 in family; 35 already compiled, listed below | 0 | Dynamic P/T, other durations, removal/type changes, events, costs, decisions, target filters | Typed subjects for Howl from Beyond and Great Defender; keyword removal for Radjan Spirit and Transmutation's switch adapter; identity prerequisite for all 35 accepted cards | Small adapter; medium identity, attachment, copy, clone, cleanup tests |
| Source-aware controller filters for static creature layers | 15, listed below | 4: Castle, Fortified Area, Goblin War Drums, Jacques le Vert | Other(ThisPermanent) for three Kobold lords; IsSupertype and BandsWithOthers for five lands; payments/conditions for Arcades and Rohgahh | Controlled static layers, then controlled targets/collections and Other | Medium typed filter/static-subject plan change; large current-controller, source-departure, layer-membership and clone tests |
| Optional fixed mana payments on spell counters | 4: Force Spike, Thrull Wizard, Vodalian Mage, Power Sink | 2 for fixed single-cost core: Force Spike and Vodalian Mage; 3 with alternative-cost support including Thrull Wizard | Power Sink needs X payment, PlayerActions, land collection and mana loss; Thrull Wizard needs alternative payments | Payer and paid-path result scopes for conditional counters, then upkeep payments | Medium/large result-scope and payer adapters; large decline, rollback, alternative and independent payment tests |

Controller-filter candidate names: Adventurers' Guildhouse, Arcades Sabboth, Castle, Cathedral of Serra, Fortified Area, Goblin War Drums, Jacques le Vert, Kobold Drill Sergeant, Kobold Overlord, Kobold Taskmaster, Mountain Stronghold, Orcish Oriflamme, Rohgahh of Kher Keep, Seafarer's Quay, Unholy Citadel.

The comparison uses complete records and nested forms, with reprints counted
once. All 107 complete temporary-layer records were inspected, including forms
beneath rejected parents. The static comparison uses the 15 complete records
with a controller filter in a top-level EachPermanentLayerEffect context;
the payment comparison uses the four complete spell-counter records above.
Other contexts in those broader families are not claimed as unlocks.

The selected slice has no immediate compilation gain. It removes a legacy
reference path in 35 accepted cards using existing engine contracts, before
adding new modifications to that path. Its concrete downstream benefit spans
power readers, variable boosts, removal and switching. The other candidates
require broader scope work. This is a bounded shared prerequisite with lower
implementation risk and effort, not a single-card feature or phase-order choice.

## Exact contract and reuse boundary

`CreatePermanentLayerEffectUntil/3` takes a symbol reference, a list of
modifications, and symbol `UntilEndOfTurn`. Supported modifications are
`AdjustPT(integer, integer)/2` and `AddAbility([keyword symbols])/1`, including
multiple modifications and multiple known basic keyword symbols: Flying,
FirstStrike, Trample, Vigilance, Haste, Reach, Banding, Defender, Fear,
Deathtouch, Lifelink, Shroud, Hexproof, Menace and Indestructible. The list must
produce at least one modification. Empty lists have a positioned compile error.
Other durations, variable P/T, rule grants and modification shapes remain rejected.

References are `ThisPermanent` for a permanent source, `HostPermanent` for an
Aura source, and `Ref_TargetPermanent` within its single permanent-target
declaration. Contexts are existing SpellActions, Activated and supported
TriggerA action lists. Reference plans remain immutable; each card factory
builds its own target declaration and binding. Unbound, wrong-type, tuple,
event, numbered, cross-rule, non-Aura host and instant permanent-source reads
remain rejected. Full engine definition validation sees every modification read.

Dependencies: P0-SCOPE-PLAYER/PERMANENT, compilePermanentRef, instanceBindings,
DeclarePermanentTarget and existing action-list/target normalization.
RON owns spelling, arity, lexical scope, plan identity and lowering. Engine reuse:
`This`, `AttachedPermanent`, `PermanentTargetBinding`, `Fixed`, `ModifyPT`,
`GrantKeywordModification`, and `Until(EndOfTurn, modifications...)`.
All modifications in one RON layer action use one Until installation. Engine
layer execution captures them before applying the installed layers; no new
parallel layer service or intermediate per-modification application is added.

Targets are selected at announcement/trigger placement and checked again at
resolution. Source and attachment subjects are read at resolution. A pending
source ability remains independent after departure. A departed Aura uses its
retained original attachment; a returned Aura cannot redirect the pending read.
After installation, subjects stay captured even if an Aura attaches elsewhere.
Leaving and returning does not preserve the modification. A subject phased
out at resolution is absent; phasing an already modified subject retains its
incarnation. End-of-turn removal expires P/T and keyword grants. Copies and
clones use their own frames and layer state. Shared transactional activation
payment remains responsible for rollback and source-sacrifice costs.

Rules: CR 113.7a, 400.7, 608.2b/c/h, 611.2a/c, 613.1f, 613.4c,
702.26b/d/e/f. No characteristic LKI value reader, replacement operation,
plural target group, cost-result binding or damage batch is introduced.
Replacement ordering stays in the existing engine. No engine contract or
implementation was missing for this slice. This resolves compiler adapters.

The inventory now assigns permanent_reference, temporary_modification,
duration, value and keyword roles within the supported action context. Children
of unknown parents remain unknown. Inventory support remains unverified;
role reporting is not gameplay certification.

## Affected records and remaining blockers

The 35 accepted unique names are:

Blessing, Carrion Ants, Dragon Engine, Dwarven Lieutenant, Elven Fortress, Elven Lyre, Emerald Dragonfly, Firebreathing, Flying Carpet, Frozen Shade, Ghosts of the Damned, Giant Growth, Goblin Balloon Brigade, Granite Gargoyle, Helm of Chatzuk, Holy Armor, Icatian Infantry, Icatian Lieutenant, Icatian Priest, Icatian Scout, Jump, Killer Bees, Order of Leitbur, Order of the Ebon Hand, Pavel Maliki, Pixie Queen, Pradesh Gypsies, Righteousness, Shivan Dragon, Staff of Zegon, Tuknir Deathlock, Wall of Fire, Wall of Opposition, Wall of Water, Wyluli Wolf.

All retain compilation and engine validation. There are no newly compiled names
or advanced first errors. The other family records remain rejected as follows;
these are first diagnostics, not an exhaustive list of later gaps. Their nested
modifications and surrounding rules were inspected before selection.

| Unique card | Remaining first diagnostic |
| --- | --- |
| Arcades Sabboth | `unsupported action (MayCost)` |
| Ashnod's Battle Gear | `unsupported rule (PermanentRuleEffect)` |
| Atog | `unsupported cost (SacrificeAPermanent)` |
| Battering Ram | `unsupported trigger (AtTheBeginningOfCombatDuringAPlayersTurn)` |
| Berserk | `unsupported non-spell rule on an instant or sorcery (CastEffect)` |
| Blood Lust | `unsupported action (IfElse)` |
| Cave People | `unsupported trigger (WhenACreatureAttacks)` |
| Coral Helm | `unsupported cost (DiscardACardAtRandom)` |
| Cyclopean Tomb | `unsupported rule (ActivatedWithModifiers)` |
| Deep Spawn | `unsupported action (MayCost)` |
| Dragon Whelp | `unsupported action (If)` |
| Dwarven Soldier | `unsupported trigger (Or)` |
| Electric Eel | `unsupported damage source (ThatEnteringPermanent)` |
| Erhnam Djinn | `unsupported permanent filter (ControlledByAPlayer)` |
| Fallen Angel | `unsupported cost (SacrificeAPermanent)` |
| Fire Drake | `unsupported rule (ActivatedWithModifiers)` |
| Gabriel Angelfire | `unsupported action (ChooseAnAbility)` |
| Gaea's Liege | `unsupported rule (If)` |
| Giant Shark | `unsupported rule (PermanentRuleEffect)` |
| Giant Slug | `unsupported action (CreateFutureTrigger)` |
| Glyph of Delusion | `multiple target bindings are not supported (Targeted)` |
| Glyph of Destruction | `unsupported permanent filter (ControlledByAPlayer)` |
| Goblin Flotilla | `unsupported trigger (AtTheBeginningOfCombatDuringAPlayersTurn)` |
| Goblin Kites | `unsupported permanent filter (ControlledByAPlayer)` |
| Goblin Wizard | `unsupported action (MayAction)` |
| Great Defender | `unsupported temporary layer action (AdjustPTX)` |
| Halfdane | `unsupported permanent filter (Other)` |
| Hammerheim | `unsupported temporary layer action (LosesAbility)` |
| Homarid Warrior | `unsupported action (PermanentDoesntUntapDuringControllersNextUntap)` |
| Howl from Beyond | `unsupported temporary layer action (AdjustPTX)` |
| Island of Wak-Wak | `unsupported temporary layer action (SetPower)` |
| Jade Statue | `unsupported rule (ActivatedWithModifiers)` |
| Johan | `unsupported trigger (AtTheBeginningOfCombatDuringAPlayersTurn)` |
| Kry Shield | `unsupported permanent filter (ControlledByAPlayer)` |
| Lesser Werewolf | `unsupported rule (ActivatedWithModifiers)` |
| Mishra's Factory | `unsupported temporary layer action (SetPT)` |
| Murk Dwellers | `unsupported trigger (WhenACreatureAttacksAndIsntBlocked)` |
| Nalathni Dragon | `unsupported action (If)` |
| Orcish Captain | `unsupported action (FlipACoin_OnWinAndLose)` |
| Orcish Veteran | `unsupported rule (PermanentRuleEffect)` |
| Osai Vultures | `unsupported rule (TriggerI)` |
| Pendelhaven | `unsupported permanent filter (PowerAndToughnessIsEqualTo)` |
| Radjan Spirit | `unsupported temporary layer action (LosesAbility)` |
| Rapid Fire | `unsupported non-spell rule on an instant or sorcery (CastEffect)` |
| River Merfolk | `unsupported temporary ability grant (Landwalk)` |
| Runesword | `unsupported action (CreateTriggerUntil)` |
| Sandals of Abdallah | `unsupported temporary ability grant (Landwalk)` |
| Scarwood Hag | `unsupported temporary ability grant (Landwalk)` |
| Shelkin Brownie | `unsupported temporary layer action (LosesAbility)` |
| Singing Tree | `unsupported temporary layer action (SetPower)` |
| Sorceress Queen | `unsupported permanent filter (Other)` |
| Spirit Shield | `unsupported rule (PermanentRuleEffect)` |
| Spitting Slug | `unsupported trigger (WhenACreatureBlocks)` |
| Stone Giant | `unsupported permanent filter (ControlledByAPlayer)` |
| Subdue | `unsupported action (CreatePreventDamageUntil)` |
| Svyelunite Priest | `unsupported rule (ActivatedWithModifiers)` |
| Tawnos's Weaponry | `unsupported rule (PermanentRuleEffect)` |
| Thallid Devourer | `unsupported action (PutCounters)` |
| Tolaria | `unsupported rule (ActivatedWithModifiers)` |
| Transmutation | `unsupported temporary layer action (SwitchPT)` |
| Urborg | `unsupported action (ChooseACheckableAbility)` |
| Urza's Avenger | `unsupported action (ChooseAnAbility)` |
| Vaevictis Asmadi | `unsupported action (MayCost)` |
| Vampire Bats | `unsupported rule (ActivatedWithModifiers)` |
| Vodalian Knights | `unsupported rule (PermanentRuleEffect)` |
| Vodalian War Machine | `unsupported cost (TapAPermanent)` |
| Wall of Caltrops | `unsupported rule (TriggerI)` |
| Wall of Wonder | `unsupported action (CreatePermanentRuleEffectUntil)` |
| War Barge | `unsupported temporary ability grant (Landwalk)` |
| Wormwood Treefolk | `unsupported temporary ability grant (Landwalk)` |
| Xenic Poltergeist | `unsupported duration (UntilNextUpkeep)` |
| Zelyon Sword | `unsupported rule (PermanentRuleEffect)` |

## TDD and card-to-test mapping

Tests are in `pkg/mage/dsl/ron/temporary_layer_test.go`.

| Test | Card/context and evidence |
| --- | --- |
| TestRONTemporaryLayersDeclareTypedReads | P/T, keyword and mixed synthetic declarations; one installation, fresh target declarations, typed reads, missing declaration rejected by engine validation |
| TestRONTemporaryLayersRejectInvalidDeclarations | Empty modification list, unbound/wrong-type/malformed/event/host refs, invalid P/T and keyword shape, unsupported duration |
| TestRONTemporaryLayerCrossRuleAndSpellSourceRejected | Instant source and cross-rule target scope |
| TestRONTemporaryLayerSpellCopiesAndLegality | Giant Growth, Jump, Llanowar Elves; announcement/resolution timing, copy retargeting, clone isolation, returned/absent/phased-in targets and cleanup |
| TestRONTemporaryLayerActivationDepartureAndRollback | Elven Lyre, Flying Carpet, Llanowar Elves; failed mana payment rollback, sacrifice payment, departed source, P/T versus keyword grants, clone and expiry |
| TestRONTemporaryLayerSourceIdentity | Shivan Dragon; two independent activations accumulate, returned source before/after installation, phasing before/after resolution, clone cleanup |
| TestRONTemporaryLayerHostResolutionAndCapture | Blessing, Llanowar Elves; opposing host, attachment read at resolution, original source LKI after departure/return, returned/phased host, frozen installed subject after reattachment, expiry |
| TestRONTemporaryLayerTriggerAndMixedSubjects | Synthetic RON entry trigger; mixed P/T and flying modifications, departed source, target chosen at placement; activation with distinct source and target reads |
| TestRONTemporaryLayerInventoryRoles | Nested typed role reporting, without support claims |

The executable tests failed before implementation because all tested temporary
layers lacked typed subject reads. Gameplay setup corrections selected ARN for
Flying Carpet and read live permanents after transactional payment replaces
state; neither is an engine failure. Trigger stack inspection was replaced with
an effect-timing assertion because entry triggers are pending until placement.
The inventory test separately failed before its role adapter was added.

Six real cards gain direct temporary-layer contract evidence: Giant Growth,
Jump, Elven Lyre, Flying Carpet, Shivan Dragon and Blessing. Llanowar Elves is
the shared real target fixture. This is increased behavior evidence for existing
accepted cards, separate from zero compilation gain. The 35-name inventory does
not claim full gameplay verification for those cards or the collection. Existing
RON gameplay tests remain passing.

## Measurements, source findings and next candidates

Final strict loader: 859 compiled of 2,445 records, 1,586 rejected records,
251 compiled unique names and 735 rejected unique names. All first diagnostics
are byte-for-byte unchanged. Estimated gain 0 equals actual gain 0. Loading
continues to reject the collection without publishing a partial registry.

Leading first gaps: PermanentRuleEffect 110 records, AsPermanentEnters 90,
ActivatedWithModifiers 61, instant/sorcery CastEffect 53, rule If 47,
ChooseADamageSource 46, ControlledByAPlayer 43, MayCost 43,
WhenAPlayerCastsASpell 43. These are compiler gaps, separate from build failures.

No RON source defect was found or corrected in this slice. All 12 source hashes
match the vocabulary snapshot, so no snapshot refresh is needed. Gaea's Liege
and Fastbond concerns remain open. No environment failure occurred.

Next ranked candidates:

1. Source-aware static controller filters: four concrete full-card unlocks,
   with reusable source/player context for controlled targets and collections.
2. Typed numeric/variable temporary modifications: Howl from Beyond and Great
   Defender, then shared power/value readers for further boosts.
3. Optional fixed mana payment scopes: Force Spike and Vodalian Mage, then
   alternative payment for Thrull Wizard and X/collection work for Power Sink.

Runtime loading, complete gameplay, Go-card deletion, obsolete Go-set generator
retirement and unused-DSL removal gates remain unmet. No DSL deletion or caller
audit is claimed. No asset/platform/startup changes require WASM validation.
This run implements only P0-SCOPE-PERMANENT-LAYERS and stops.
