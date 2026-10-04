# Temporary collection P/T adjustments

Slice P7-TEMPORARY-COLLECTION, 2026-10-04. Complete for the signatures below.
Other collection modifications remain unsupported.

## Selection and evidence

Baseline focused RON/CLI tests, engine short tests and `make build` pass.
Strict loading rejects the collection with 988 compiled and 1,457 rejected
records. There are 298 accepted and 688 rejected unique names. The inventory
walks rejected descendants: 72,938 nodes, 8,528 context/shape signatures.
First-error frequency was not used to estimate complete-card unlocks.

| Candidate | Distinct affected names inspected | Estimated full-card unlocks | Other blockers and downstream benefit | Effort |
| --- | --- | ---: | --- | --- |
| Fixed temporary collection P/T | Army of Allah, Piety, Bone Flute, Holy Light, Marsh Gas, Morale, Hell Swarm, Shield Wall; other collection records listed below | 8 | Other records need target sets, collection colors/keywords/animation, event filters, sacrifice choices or payment branches. Shared collection lowering supplies their capture prerequisite. | Small compiler/inventory adapters; moderate capture, copy, lifecycle and real-card test matrix |
| Own-turn/upkeep activation modifiers | Disrupting Scepter, Svyelunite Priest; related Library of Alexandria and Vampire Bats records | 2 | Library requires hand-count conditions; Bats requires per-turn limits. Timing enables later restricted activations. | Small timing adapters; moderate legality, rollback and controller matrix; other conditions/limits are separate contracts |
| Variable optional mana payment | In the Eye of Chaos, Invoke Prejudice | 1 | Invoke needs opponent cast scope and color-sharing filters. Variable values/payments enable other spell taxes. | Moderate value/payment adapter and event, absent-object, copy and transactional payment matrix |

Collection P/T wins by credible complete-card gain per effort. The exact contract,
engine boundary and acceptance cases were stated before editing. Reprints count
once. The collection constructor has nine nested context/shape signatures and
20 unique card names. Every complete distinct collection record was inspected,
including forms beneath unsupported parents. Additional names are Orc General,
Spitting Slug, Icatian Skirmishers, Thelonite Druid, Tidal Flats, Tourach's Gate,
Dwarven Song, Heaven's Gate, Part Water, Sea Kings' Blessing, Sylvan Paradise,
and Touch of Darkness. Their missing costs, events, branches, target collections
and modifications are not hidden by this adapter. The eight selected cards each
have one source record; all other identical reprints retain their definitions.

A subsequent timing audit also read complete Clockwork Beast, Clockwork Avian,
Cyclopean Tomb, Instill Energy, Rock Hydra, Rag Man, Wand of Ith, Dwarven
Weaponsmith, Ebon Praetor, Icatian Moneychanger, Gwendlyn Di Corci, Hell's
Caretaker, Life Chisel, Life Matrix, Mirror Universe, Nebuchadnezzar, Colossus
of Sardia and Gate to Phyrexia records. These need independent entry, counters,
results, random choices, graveyard targets, rule effects or limits. They do not
increase the two direct timing unlocks and are useful next-run dependency evidence.

## Exact contract and reuse

`CreateEachPermanentLayerEffectUntil(filter, modifications, UntilEndOfTurn)/3`
is a resolving action in existing spell, activation, trigger and supported
optional-payment continuation contexts. `modifications` is a nonempty list of
`AdjustPT(integer_literal, integer_literal)/2`. Signed literals, including -0,
use existing integer parsing. Multiple adjustments install together in one Until.

Filters reuse `compileStaticPermanentFilter` and its immutable permanent-set
plan. Existing generic permanent filters, nonempty And/Or and the controller
selector `ControlledByAPlayer(SinglePlayer(You))/1` are supported. The controller
selector can occur alone or under conjunction, but not under disjunction.
You is the resolving controller, not the source's later controller or owner.
Other/source exclusions are explicitly rejected in this collection context.
No implicit creature restriction is added; card records supply IsCardtype(Creature).

Typed target collections, unbound references, unknown player scopes, malformed
arguments/lists/durations, empty modifications, dynamic adjustments, base values,
colors, keyword grants, animation and other durations remain errors before a
factory is published. Generic filter spelling support is unchanged. Inventory
assigns permanent_filter, temporary_modification and duration operand roles to
this exact constructor; support status remains unverified as elsewhere.

`compileTemporaryCollection(Node) (effectFactory, error)` stores only immutable
filter and integer plans. Each build creates fresh modifications and Until.
There are no new target or result bindings. Existing targets/events from outer
supported contexts retain their per-instance lowering and validation.

Engine reuse: MatchingPermanents/ControlledPermanents, ModifyEachPT, Fixed,
Until(EndOfTurn), incarnation capture, layer 7c application, source-independent
installed effects, source/controller capture, game/stack cloning and copies,
transactional activation/casting costs, phasing, cleanup and state-based actions.
No engine implementation change is needed. RON has no parallel rules executor.
The selection is captured at resolution, includes both players where the filter
permits, excludes phased objects and does not gain new members later. Control,
color or combat changes after capture do not remove the modification. Returned
objects have new incarnations. Relevant rules read locally: CR 611.2a/c,
613.4b/c, with retained identity/phasing contracts CR 400.7 and 702.26b/d.
This establishes no new LKI query, replacement ordering, damage batch or
simultaneous zone-operation service. One Until installs all adjustments before
state-based actions use the resulting characteristics.

## TDD and card-to-test mappings

Real-card and invalid-declaration tests were written first. Test-only API names
were corrected before confirming the intended failures: all eight real cards
failed on unsupported CreateEachPermanentLayerEffectUntil. After implementation
all pass. The optional-continuation fixture initially declined its payment with
BasePlayer; it now uses the existing accepting ronPaymentPlayer fixture. No
expectation was weakened to conceal missing behavior. Lint reordered imports only.

| Card/context | Tests and evidence |
| --- | --- |
| All eight newly accepted names | TestRONTemporaryCollectionRealCards: full RON records and engine validation, casting or activation, payment, resolution timing, global/color/combat/controller selection, phased exclusion, before/after-resolution entries, retained membership through control/combat changes, phase cycles, incarnation return, source departure, pending clones, expiry isolation and final spell graveyard |
| Army of Allah, Morale | TestRONTemporaryCollectionOpponentCombat: opponent's attacking creature receives and retains the bonus on their turn |
| Shield Wall | TestRONTemporaryCollectionCopyController: no target declarations; copy uses its controller; original excludes phased creature; copy leaves no graveyard card |
| Bone Flute | TestRONTemporaryCollectionBoneFluteRollback: insufficient mana preserves mana/tap/stack; successful payment consumes two mana and taps; tapped source cannot activate again |
| Holy Light | TestRONTemporaryCollectionHolyLightDeaths: both players' nonwhite 1/1 creatures die through state-based actions; white creature survives unchanged |
| Synthetic contexts | TestRONTemporaryCollectionSharedContexts: fresh factories, engine validation, activation/entry-trigger/paid-continuation reuse, combined adjustments, resolving controller preserved after source control change, exact inventory roles |
| Invalid declarations | TestRONTemporaryCollectionInvalid: empty/malformed/dynamic modifications, unsupported types and durations, missing collections/player scopes, empty filters and unsupported controller disjunction/source exclusion |

Each new unique card has gameplay evidence for every rule in its complete record.
This is separate from complete collection-wide gameplay verification.

## Measurements, validation and next work

Estimate and actual gain agree: eight unique names, eight records. Strict loader
988 -> 996/2,445 compiled; rejection 1,457 -> 1,449. Unique names 298 -> 306
accepted, 688 -> 680 rejected. No previously accepted record regressed. Full
inventory issues equal strict-loader issues. Nodes/signatures stay 72,938/8,528;
unknown-role occurrences fall 19,408 -> 19,278. Global engine_validated and
behavior_verified remain null. Largest first diagnostics remain PermanentRuleEffect
(110), AsPermanentEnters (90), ActivatedWithModifiers (61), non-spell CastEffect
(53), rule If (47), ChooseADamageSource (46), Targeted_Modal (42), HostController
upkeep reference (35). These are diagnostics, not semantic unlock estimates.

Focused tests with -count=1, RON/CLI package tests, engine short tests with
-count=1, make test, make build, go vet, make lint and git diff --check pass.
Checks are repeated after lint. Both strict/full-inventory loaders exit 1 for
remaining unsupported records. No environment or pre-existing check failure
blocks this slice. All source hashes match the vocabulary snapshot; no RON
source changed and no vocabulary refresh is required. No source defect was
corrected. Gaea's Liege/Fastbond concerns remain unresolved.

Ranked next candidates: (1) own-turn/upkeep activation modifiers, two direct
unlocks; (2) temporary collection keywords/colors with a fresh target-set and
combat-filter dependency audit; (3) variable optional mana payment, Chaos directly,
Invoke after filters. Reassess these estimates next run. Production registry,
Go definitions/generators and DSL exports retain their callers. Removal gates
remain closed; no deletion audit or startup cutover is claimed. This run stops
after this one slice.
