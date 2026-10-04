# RON static landwalk grants

Capability: P7-STATIC-LANDWALK. Complete for the contexts below; the wider
landwalk family remains partial. This run implements one slice and stops.

## Selection and dependencies

Initial user changes: `.gitignore` and `mtgish-ron-execution.md`. Both remain
outside this slice. No unrelated untracked files existed. Fresh compiler tests
and `make build` passed. Strict loader and nested AST inventory reported 2,445
records, 986 unique names, 884 compiled records, 1,561 rejected records,
261 compiled unique names and 725 rejected unique names. Exit 1 is collection
rejection, not a build failure. The vocabulary is syntax evidence only.
All 12 source hashes match the snapshot.

Complete records, including nested forms, were read for all names below.
Reprints count once. The static candidate includes Aura grants, which the
previous ledger's lord-only suggestion did not include.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers | Downstream capability | Effort |
| --- | --- | --- | --- | --- | --- |
| Static landwalk grants, selected | Burrowing, Fishliver Oil, Goblin King, Hidden Path, Lord of Atlantis, Zombie Master | 3: Burrowing, Fishliver Oil, Hidden Path | Goblin King/Lord of Atlantis lack a safe subtype-only global subject contract; Zombie Master also needs granted activations | Shared validated printed/static landwalk lowering; grants for later subtype lords | Small adapter, medium attachment/static/combat lifecycle tests |
| Optional fixed counter payments | Force Spike, Power Sink, Thrull Wizard, Vodalian Mage | 2: Force Spike, Vodalian Mage | Power Sink needs X payments, player actions, mana loss and collections; Thrull Wizard needs alternatives | Typed payer decisions and payment-result scopes | Medium/large scope and payment work |
| Numeric branches and characteristic reads | Berserk, Blood Lust, Kry Shield, Subdue | 1: Blood Lust if both branches and arithmetic are implemented | Berserk also needs casting timing and delayed triggers; Kry Shield/Subdue need prevention, Kry Shield also controlled target filters | Resolution characteristic reads, arithmetic and typed conditions | Medium/large value/branch and LKI contracts |

Static landwalk yields three credible full-card unlocks with the least new
execution machinery. It reuses existing typed static plans and engine services.
The scope, reuse boundary and acceptance cases were stated before editing.
No first-error frequency was used as an unlock estimate.

A subtype-only global filter is not equivalent to an explicit creature filter:
CR 109.2 denotes a permanent of that subtype, and CR 205.3m includes kindreds.
CR 208.3a retains P/T modifications for a noncreature that later becomes a
creature. The existing global services enforce creature membership. The guard
therefore remains; this run does not silently add a creature restriction to
Goblin King or Lord of Atlantis. Complete records of Thrull Champion, Goblin
Caves, Goblin Shrine and Orc General were also inspected to check downstream
subtype contexts; control-duration, condition/event and cost/collection gaps
remain. No source defect was corrected or new data defect claimed.

## Contract and reuse

Supported exact shapes:

- `PermanentLayerEffect(HostPermanent, modifications)/2` on an Aura.
- `EachPermanentLayerEffect(filter, modifications)/2`, with the existing
  explicit creature requirement. Existing recursive conjunctions, static
  `Other(ThisPermanent)` and `ControlledByAPlayer(SinglePlayer(You))` retain
  their support and rejection boundaries.
- `AddAbility(list)/1` containing `Landwalk(IsLandType(symbol))/1`; landwalk
  and IsLandType each have exactly one positional argument. Accepted symbols:
  Plains, Island, Swamp, Mountain, Forest and Desert.
- Existing fixed `AdjustPT/2` and basic keywords can share the modification list.

Wrong kind/arity, unsupported land types, wrong-role filters and unbound host
contexts fail during compilation. Legendary/supertype landwalk, temporary
landwalk grants and landwalk rule restrictions remain unsupported here.
The existing printed landwalk parser is extracted into a private helper;
printed behavior and diagnostics are retained. Static lowering adds no public
DSL export. Immutable keyword values build fresh effects per factory. Inventory
now reports static AddAbility operands as keywords and landwalk operands as
landwalk_filter/land_subtype, including descendants of rejected records; these
roles do not assert support.

Engine reuse: GrantAbilityToAttached, GrantKeywordToAll/IncludingSource,
GrantKeywordToControlled/OtherControlled, existing static filter plans,
printed-static registration, shared layers/removal dependencies, attachment
and phasing services, control effects, clone ownership and HasLandwalkEvasion.
Rules: CR 611.3a/b/c, 613.1b/d/e/f/g, 613.4c, 613.8a/b, 702.14a–e and
702.26b/d. Membership is current, not captured at source entry. The Aura grant
follows its current host. Source departure or ability removal disables the
grant. Phasing suspends it, and phase-in restores it.

TDD exposed one engine prerequisite: HasLandwalkEvasion scanned phased defending
lands. A separate engine test failed for all six subtype attributes and legendary
landwalk before the two query branches gained phased-land exclusions. The
implement-engine skill supplied the contract-test workflow. No new engine API,
LKI reader, cost, target binding, replacement order or simultaneous operation
was needed. Existing contracts retain their prior evidence.

## Card-to-test mapping

| Test | Card/rules and acceptance evidence |
| --- | --- |
| TestRONStaticLandwalkAuras/Burrowing | Complete LEB record: creature enchant legality, payment state after invalid target, timing, host binding, opposing controller, returned/phased target rejection, defender land identity/control, phased land/phase-in, clone isolation, reattachment, source removal/restoration, indirect host/Aura phasing and source departure |
| TestRONStaticLandwalkAuras/Fishliver_Oil | Complete ARN record, same cases with islandwalk and its full mana cost |
| TestRONStaticLandwalkHiddenPath | Complete DRK record: both controllers, green creature membership, nongreen/noncreature exclusion, color-layer changes, late entry, defender Forest, source control change, source/recipient phasing, clone isolation, source removal/restoration and departure/return |
| TestRONStaticLandwalkShapesAndInstances | All six subtype forms, same-factory products, mixed P/T/basic-keyword/landwalk modifications, source inclusion and accumulated boosts, independent source departure; engine definition validation |
| TestRONStaticLandwalkControlledExclusion | Synthetic explicit creature global filter, source exclusion, controlled membership and changing source controller |
| TestRONStaticLandwalkInvalidAndInventory | Invalid kind/arity/role/type; nested static keyword, landwalk-filter and subtype inventory roles |
| TestLandwalkEvasionPhasingAndController | Engine prerequisite: six subtype attributes plus legendary; owner/control distinction, phased land, phase-in, clone isolation, control change, departure and return |

Real RON fixtures: Llanowar Elves, Merfolk of the Pearl Trident, Mountain, Island
and Forest. No Go card definition supplies a requirement or gameplay fixture.
The first test attempt had two test API errors (Stack size and void resolution
method); correcting those exposed the intended unsupported static grants and
inventory-role failures. No production code was changed before that confirmed
failure. The engine regression then confirmed phased-land failures before its
fix. Focused tests pass with `-count=1`.

## Measurement and next work

Final strict loading: 892 compiled records, 1,553 rejected records, 264 compiled
unique names and 722 rejected unique names. Gain: eight records, three unique
cards, exactly the estimate. Burrowing has six records; Fishliver Oil and Hidden
Path have one each. All three gain full-record RON gameplay evidence. No accepted
card regressed. Six Zombie Master reprints advance from the first landwalk grant
to the second subtype-only global selector; all other remaining first diagnostics
are unchanged. Compilation and
behavior coverage are separate; no global gameplay total is claimed.

Leading first gaps: PermanentRuleEffect 110 records; AsPermanentEnters 90;
If 63 across contexts; ActivatedWithModifiers 61; CastEffect 54 across contexts;
ChooseADamageSource 46; MayCost and WhenAPlayerCastsASpell 43 each;
Targeted_Modal 42; HostController 35; And 31 and ControlledByAPlayer 29 across
contexts. The remaining rejection publishes no partial production registry.
RON source is unchanged, so the vocabulary snapshot needs no refresh.
Gaea's Liege/Fastbond observations remain open.

Ranked next candidates, to reassess with fresh evidence:

1. General static subtype subjects: exact noncreature/grant/P/T contracts before
   Goblin King/Lord of Atlantis adapters; landwalk lowering is now ready.
2. Optional fixed counter payments: Force Spike/Vodalian Mage; typed payer,
   decision and paid/unpaid scope prerequisites.
3. Numeric characteristic reads, arithmetic and branches: Blood Lust, then
   timing/delayed triggers for Berserk and prevention for Kry Shield/Subdue.

Required command outcomes are in the progress ledger. Full loading/gameplay,
embedded runtime startup, native validation, Go-card/generator deletion and
unused-DSL removal gates remain unmet. No deletion or caller audit is claimed.
