# RON general static permanent subjects

Capability: P7-STATIC-PERMANENTS. Complete for the shapes below. The broader
static layer family remains partial. This run stops after this slice.

## Selection and dependencies

Initial user changes: `.gitignore` and `mtgish-ron-execution.md`; both are
preserved and excluded. No unrelated untracked files existed. Fresh compiler
tests and `make build` passed. Strict loading and full nested AST inventory
reported 2,445 records, 986 unique names, 892 compiled records, 1,553 rejected
records, 264 compiled unique names and 722 rejected unique names. Exit 1 was
collection rejection, not a build failure. All 12 source hashes match the
vocabulary snapshot. Its relevant signatures were inspected as syntax evidence.

Complete RON records and nested forms were read for every name in this table,
including identical reprints. Reprints count once. The selected capability
includes a shared selector prerequisite for modifiers that remain unsupported.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers | Downstream capability | Effort |
| --- | --- | --- | --- | --- | --- |
| General static subjects, selected | Goblin King, Lord of Atlantis, Zombie Master, Energy Flux, Kormus Bell, Living Lands, Living Plane, Conversion, Blood Moon, Guardian Beast | 2: Goblin King, Lord of Atlantis | Zombie Master needs granted activations; Energy Flux needs granted triggers/payments; land animation needs SetPT/AddCardtype and Kormus Bell also SetColor; Conversion needs payments/SetLandType; Blood Moon needs nonbasic filters/SetLandType; Guardian Beast needs conditional and rule effects | General static permanent sets for the listed grants, animations and land changes | Small adapter; medium lifecycle and noncreature tests; existing engine primitives |
| Optional fixed counter payments | Force Spike, Vodalian Mage, Power Sink, Thrull Wizard | 2: Force Spike, Vodalian Mage | Power Sink needs X payments, player actions, mana loss and collections; Thrull Wizard needs payment alternatives | Typed payer decisions and payment-result dominance | Medium/large scope/payment implementation and tests |
| Numeric branches and characteristic reads | Blood Lust, Berserk, Kry Shield, Subdue | 1: Blood Lust with both branches and arithmetic | Berserk needs timing/delayed triggers; Kry Shield/Subdue need prevention; Kry Shield also controlled targets | Characteristic readers, arithmetic, conditions and LKI contracts | Medium/large value/branch implementation and tests |

General subjects offer two credible unlocks with the smallest change, because
the existing composed-layer services already select noncreatures. This corrects
the compiler's unnecessary creature guard without introducing a new engine
service. Optional payments have the same unlock estimate but require more
binding/decision work. Numeric branches have lower immediate gain and broader
contracts. The previous ledger suggestion was reassessed. First-error frequency
was not used as a full-card estimate. The contract, reuse boundary and acceptance
cases were stated before editing.

## Exact contract and reuse

Supported rule: `EachPermanentLayerEffect(filter, modifications)/2` in permanent
card rules. The modification list must produce at least one modification in
the new general context. Supported modifications:

- `AdjustPT(signed_integer, signed_integer)/2`.
- `AddAbility(list)/1` of the existing basic keyword symbols: Flying,
  FirstStrike, Trample, Vigilance, Haste, Reach, Banding, Defender, Fear,
  Deathtouch, Lifelink, Shroud, Hexproof, Menace and Indestructible.
- The existing `Landwalk(IsLandType(symbol))/1` form for Plains, Island, Swamp,
  Mountain, Forest and Desert can share the list.

The existing ordinary permanent filter forms retain their meanings:
AnyPermanent, IsTapped, IsUntapped, IsAttacking, IsBlocking; unary IsCardtype,
IsNonCardtype, IsCreatureType, IsLandType, IsEnchantmentType, IsNonCreatureType,
IsNonLandType, IsColor, IsNonColor, HasAbility and DoesntHaveAbility; recursive
`And(list)/1` and `Or(list)/1`. Static logical lists are nonempty at every level.
`Other(ThisPermanent)/1` and
`ControlledByAPlayer(SinglePlayer(You))/1` are supported at the root and through
conjunctions. Neither may occur under a disjunction, even through a nested
conjunction. Wrong kind/arity, malformed references, other source/controller
references and unsupported modifications fail during compilation.

RON owns the immutable filter plan and instance construction. General filters
build `MatchingPermanents` or `ControlledPermanents`, optionally wrapped in
`WithoutPermanent(..., This)`. Fixed values and keyword grants build fresh
`ModifyEachPT` and `GrantEachKeyword` modifications in `Static(While(...))`.
The constant condition uses the existing pure `StateConditionFunc` with a
declared battlefield dependency. The engine supplies source lifetime, printed
ability activity, incarnation capture, layer ordering and clone state. This
callback performs no query, decision, mutation or random operation.

Explicit-creature filters keep the existing creature services; Aura-host
lowering is unchanged. No public DSL export, mutable compiler binding,
last-action slot, RON executor or alternate registry was added. Engine behavior
required no repair. Static numeric P/T, base P/T/type/color changes, granted
activations/triggers, conditional rules and other compiler contexts remain
unsupported. No Go definition, generator or DSL function was removed; the
collection does not meet the cutover/removal gates. No caller audit is claimed.

Rules evidence: CR 109.2, 205.3m, 208.3a, 400.7, 611.3a–c, 613.1d/f/g,
613.4b/c, 702.14c and 702.26b/d. Membership is current, not captured at source
entry. Keywords apply to matching noncreatures. P/T modifications apply if the
recipient becomes a creature. Tests use synthetic noncreature subtype fixtures
to distinguish the selector contract; they do not claim new Kindred metadata
support or a noncreature P/T reader. No LKI, replacement order or simultaneous
zone action was added.

Inventory now gives unary IsCreatureType/IsNonCreatureType operands the
`creature_subtype` role under permanent filters, including rejected ancestors.
Reporting still marks every signature unverified and leaves global engine and
behavior verification totals null. Role recognition does not prove support.

## Card-to-test mapping

| Test | Rules and acceptance evidence |
| --- | --- |
| TestRONStaticPermanentLords/Goblin_King | Complete real LEB record: other Goblins get +1/+1 and mountainwalk, both controllers, non-Goblin exclusion, owner/control distinction, source exclusion, same-factory lords and accumulation, source/recipient return, phasing, source removal/restoration, defender land phasing, clone isolation |
| TestRONStaticPermanentLords/Lord_of_Atlantis | Complete real LEB record: same cases for other Merfolk and islandwalk |
| TestRONStaticPermanentCastTiming | Both real lords: printed mana cost, rejected unpaid cast, unchanged hand/stack/mana, no effect on stack, cloned resolution isolation, effect after resolution |
| TestRONStaticPermanentNoncreaturesAndInstances | Fresh validated synthetic factories; noncreature keyword recipients, fixed bonuses after animation, global and controlled membership, same-factory sources, source exclusion, changed controller, source phasing, later entry, departure, clone isolation |
| TestRONStaticPermanentCurrentMembership | Ordinary nested disjunction, subtype change in layer 4, later layer membership, subtype expiration in clone, source departure |
| TestRONStaticPermanentRootExclusionAndCreatureBoundary | Root Other excludes only source, includes land recipients, explicit creature filter excludes noncreatures until animation |
| TestRONStaticPermanentInvalid | Malformed/unbound/wrong-role filters, unsupported controller forms, dependent disjunctions, nested empty logical lists |
| TestRONStaticPermanentInventory | General shape compiles; subtype role and unverified support reporting |
| Existing static tests and collection/registry tests | Accepted Aura, creature-global, controlled and source-excluded effects retained; unsupported static numeric modifiers remain errors; real Lightning Bolt and registry behavior retained |

Real fixtures: Mons's Goblin Raiders, Merfolk of the Pearl Trident, Llanowar
Elves, Forest, Mountain and Island, all from RON. No Go card definition supplies
requirements or fixtures. Tests first failed on the two real lords and both
synthetic noncreature contexts with the expected unsupported static-subject
diagnostic. The later inventory-role test failed with `unknown` before its
adapter. Two guessed cleanup methods were corrected to the existing effect
expiry service before the intended inventory failure. No production change or
test weakening concealed those fixture errors.

The old declaration test for root Other plus fixed P/T is now a supported
declaration; its rejection case now uses unsupported static AdjustPTX, while
the new root-exclusion test verifies the newly accepted fixed form. Two exact
inventory count assertions were updated. Other gameplay/registry assertions
were retained.

## Measurements and next work

Final loading: **904 compiled records**, **1,541 rejected records**,
**266 compiled unique names**, **720 rejected unique names**. Gain: **12
records and two unique cards**, exactly the estimate. Goblin King and Lord of
Atlantis each have six identical records and now have full-record RON gameplay
tests. Comparing before/after record diagnostics found no regression among
accepted records. Compilation and gameplay coverage remain separate; no global
behavior total is claimed. Strict loading exits 1 and installs no registry.

Five rejected names advance to later gaps: Zombie Master to granted Activated;
Energy Flux to granted TriggerA; Kormus Bell, Living Lands and Living Plane to
SetPT. Conversion, Blood Moon and Guardian Beast retain earlier blockers.
Leading first gaps remain PermanentRuleEffect 110 records; AsPermanentEnters
90; If 63; ActivatedWithModifiers 61; CastEffect 54; ChooseADamageSource 46;
MayCost and WhenAPlayerCastsASpell 43 each; Targeted_Modal 42; HostController
35; ControlledByAPlayer 29; TriggerI 27. These are compiler/context gaps, not
environment or build failures.

No RON source changed or source defect was corrected. All source hashes match
the vocabulary snapshot, which therefore needs no refresh. Gaea's Liege and
Fastbond observations remain open. Required command outcomes are recorded in
the progress ledger.

Ranked next candidates, to reassess from fresh evidence:

1. Shared static land animation: Living Lands, Living Plane and Kormus Bell;
   establish exact multi-layer subject/type/base-P/T behavior, plus color for
   Kormus Bell. Three potential full-card unlocks.
2. Optional fixed counter payments: Force Spike and Vodalian Mage; typed payer,
   decisions and payment-result dominance. Two potential full-card unlocks.
3. Numeric characteristic reads/arithmetic/branches: Blood Lust and concrete
   downstream work for Berserk, Kry Shield and Subdue. One immediate unlock;
   keep it behind useful multi-card slices unless needed as a prerequisite.

Zombie Master's granted activation remains a useful shared future adapter, but
its one immediate unlock does not take priority over the multi-card work above.
Complete RON gameplay/loading, runtime cutover, native startup, Go-card and
generator deletion, and unused-DSL removal gates remain unmet.
