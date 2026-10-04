# RON static permanent animation

Capability: P7-STATIC-ANIMATION. Complete for the combined declaration below.
Other type, color and base P/T declarations remain separate compiler contexts.

## Selection

The initial working tree had user changes to `.gitignore` and the migration
execution design. Both are preserved and excluded from this slice. Fresh RON
and CLI tests and `make build` passed. The strict loader and nested AST inventory
reported 904 compiled records, 1,541 rejected records, 266 accepted unique names
and 720 rejected unique names, from 2,445 records and 986 unique names. Their
exit 1 denotes collection rejection, not build or environment failure.

Complete records and nested forms were read for the names below, including
identical reprints. First diagnostics alone did not determine the estimates.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers | Downstream use | Effort |
| --- | --- | --- | --- | --- | --- |
| Static animation, selected | Living Lands, Living Plane, Kormus Bell | 3 | Missing exact layer contract, SetPT, AddCardtype and SetColor adapters | Layer-separated animation, retained subjects, dependency and timestamp contracts for later temporary animation | Medium engine prerequisite, small compiler adapter, medium contract tests |
| Optional fixed counter payments | Force Spike, Vodalian Mage, Power Sink, Thrull Wizard | 2: Force Spike, Vodalian Mage | Power Sink needs X payment, player loops and mana loss; Thrull Wizard needs alternative payments | Typed payer decisions and payment-result dominance | Medium/large scope and payment work |
| Numeric branches and characteristic reads | Blood Lust, Berserk, Kry Shield, Subdue | 1: Blood Lust | Berserk needs casting timing and delayed destruction; Kry Shield/Subdue need prevention; Kry Shield also needs controlled targets | Characteristic readers, arithmetic, branch scopes | Medium/large value and branch work |

Static animation offers the largest credible immediate gain. The existing
`AnimateLands` writes base P/T in layer 4, so it cannot supply the required
layer 7b timestamp behavior. The slice establishes that missing reusable engine
contract before adding adapters. The existing API remains available to its
callers. The contract, reuse boundary and acceptance cases were stated before
implementation. The implement-engine skill supplied the engine test workflow.

## Exact supported contract

In a permanent card's rules, `EachPermanentLayerEffect(filter, list)/2` accepts
this complete combination, in any list order:

- Exactly one `SetPT(PT(integer, integer))/1`; both values are signed literals.
- Exactly one `AddCardtype(Creature)/1`; Creature is a symbol.
- Zero or one `SetColor(SimpleColorList(list))/1`; elements are White, Blue,
  Black, Red or Green symbols. An empty list makes the recipients colorless.
  Omission preserves their current colors.

The list cannot contain other modifications. Standalone SetPT, standalone
AddCardtype, duplicate declarations, other types, dynamic values, malformed
operands and chosen colors remain compilation errors. Aura-host and temporary
contexts are unchanged. There is no partial execution of a mixed list.

Filters use the existing `staticPermanentFilterPlan`: ordinary permanent
filters and nonempty recursive And/Or, plus root/conjunctive
`ControlledByAPlayer(SinglePlayer(You))` and `Other(ThisPermanent)`.
Source-dependent disjunctions and unbound references remain errors. Each factory
builds a fresh set and rule. No target, result or event binding is introduced.

Engine API:
`AnimateEach(PermanentSet, int, int, ...core.Color) StaticRule`.
Install it with `Static`. Nil colors preserve colors; a non-nil empty variadic
slice makes the recipients colorless. Colors are copied by the service.
Definition inspection exposes the set and rejects a nil or invalid set.

The service adds Creature in layer 4, optionally sets colors in layer 5, and
sets base P/T in layer 7b. It retains types, subtypes, supertypes and abilities.
At the start of the animation in layer 4, it saves incarnation references for
that layer cycle. The effect manager owns this per-game state, keyed by rule
and source identity, and resets it for each application. Later layers use the
same selection, even if adding Creature changes membership or the source loses
its ability in layer 6 (CR 613.6). Every later application selects current
members again. Clone effect managers have independent cycle state.

For layer-4 dependencies, the service compares its selection before and after
applying the other type effect on an isolated game clone. The trial effect is
also cloned when it carries state. This integrates with the existing dependency
sorter without changing live gameplay or consuming choices. It establishes
membership-changing type dependencies; it does not establish a general new
contract for every engine dependency or every possible dependency graph.

Printed static wrappers now forward P/T sublayers. Without that prerequisite,
a static base-setting effect ran in 7c and overrode a later 7b set. Animated
noncreatures receive ordinary combat eligibility from their current creature
type, without a grant that bypasses Defender. Existing printed-creature base
restrictions and explicit attack/block prohibitions retain their behavior.
Summoning sickness and intrinsic land mana abilities use existing services.

Reuse: typed permanent sets, captured incarnation references, effect manager
reset/lifecycle, printed static registration, dependency sorting, copy-on-write
clones, current control, phasing, intrinsic mana, costs and stack resolution.
RON contains spelling and declaration validation; the engine contains reusable
animation rules. No Go-card fallback, card-name branch, global action slot,
production registry, Go definition deletion or DSL removal was added.

Rules evidence: local CR 205.1b, 302.6, 305.6–7, 400.7, 611.3a–c,
613.1d/e/g, 613.4b–d, 613.6–8 and 702.26b/d. Local `data/LEA.json` and
`data/LEG.json` Oracle records agree with the three RON declarations. No source
defect was corrected. Gaea's Liege and Fastbond observations remain unresolved.

## Test mapping

| Card or contract | Tests and evidence |
| --- | --- |
| Living Lands, Living Plane, Kormus Bell | `TestRONStaticAnimationRealCards`: complete real records, costs/rollback, resolution timing, clone stack resolution, both players' lands, matching boundaries, late entry, summoning sickness, type/subtype retention, black color, source/target phasing, source ability loss, source departure/return and preserved mana |
| Synthetic declarations | `TestRONStaticAnimationInvalid`: required combination, duplicates, wrong arity/kind/type, dynamic base values, wrong color syntax and colorless declaration |
| Synthetic instances | `TestRONStaticAnimationInstancesAndControl`: fresh factories, independent sources, exclusion/controller filters, current controller rather than owner, source removal and clone isolation |
| Synthetic colors and syntax | `TestRONStaticAnimationColorsAndInventory`: preserve/replace/clear colors, artifact-land type retention, color expiry, operand inventory roles with support unverified |
| Engine retained subjects | `TestStaticAnimationLayerContinuation`: noncreature-only selection survives type addition, source ability removal, repeated layer passes, departure and clones |
| Engine layers | `TestStaticAnimationLayerOrder`: both timestamp orders for competing base sets, layer 7c boosts/counters and layer 7d switching |
| Engine dependencies | `TestStaticAnimationTypeDependency`: older animation waits for later land-type change, dependency expiry and clone isolation |
| Engine legality | `TestStaticAnimationCombatAndMana`: Defender, attack restriction, attack/block eligibility, mana preservation and summoning sickness |
| Engine entry/identity | `TestStaticAnimationEntryAndIncarnations`: creature entry detection with animated power, retained event value after departure, new returned incarnations and returned source selection |
| Engine declarations | `TestStaticAnimationMetadataAndInvalidSet`: nil-set rejection, static/mass metadata and set sublayer |

The first engine test failed because AnimateEach was absent. Real RON tests
failed with unsupported SetPT. Additional tests failed on 7b timestamp order,
Defender bypass and type dependency order before those repairs. The inventory
operand test failed with unknown roles before its adapter was restored. Fixture
API errors (a nonexistent attack helper and a mana cost on a synthetic land)
were corrected without changing engine requirements. The first broad check
exposed preserved printed-creature attack restrictions; the combat capability
change was narrowed and those regressions passed. Two exact inventory counts
were updated to measured totals; gameplay assertions were retained.

## Measurements and remaining work

Final loader: 917 compiled records, 1,528 rejected records, 269 accepted unique
names and 717 rejected unique names. Actual gain: 13 records and three names,
exactly the estimate. Living Lands and Kormus Bell each have six identical
records; Living Plane has one. No accepted record regressed and the remaining
record diagnostics are unchanged. Loading still exits 1 and publishes no
registry. Each unlocked card has direct complete-record gameplay evidence;
no collection-wide behavior or engine-validation total is claimed.

Inventory now reports fixed_pt, card_type, color_list and color operands for
these static forms. All signature support stays unverified. No RON file changed;
all 12 source hashes match the vocabulary snapshot, so no source refresh is
required. Required command outcomes are recorded in the progress ledger.

Next priorities: optional fixed counter payments (two immediate unlocks), then
shared temporary animation prerequisites for Mishra's Factory, Jade Statue and
Thelonite Druid, then characteristic readers/branches for Blood Lust and its
concrete downstream cards. Reassess complete records and fresh loader evidence
before selecting. Temporary animation still needs its own duration, timing,
cost and collection contracts; this static service does not claim that support.
Full gameplay/loading, embedded startup, native smoke tests, Go-card/generator
deletion and DSL cleanup remain incomplete. This run stops after this slice.
