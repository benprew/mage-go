# RON static source exclusion

Capability: P1-STATIC-OTHER. Complete for the shape and context below.

## Selection

Fresh baseline compiler tests and build passed. Strict loader and nested AST
inventory both exited 1 for collection rejection: 874 compiled records,
1,571 rejected records, 256 accepted unique names, 730 rejected unique names.
There are 2,445 records and 986 unique names. Reprints count once.
The existing user changes to `.gitignore` and the execution design are preserved.

Complete RON records and nested signatures were read for these candidates:

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers | Downstream benefit | Effort |
| --- | --- | --- | --- | --- | --- |
| Static source exclusion, selected | Goblin King, Kobold Drill Sergeant, Kobold Overlord, Kobold Taskmaster, Lord of Atlantis, Zombie Master | 3: all three Kobold lords | Goblin King/Lord of Atlantis need subtype-only static subjects and Landwalk; Zombie Master needs Landwalk and granted activated abilities, with a subtype-only second filter | Source identity in static conjunctions; later landwalk and granted-ability lord adapters | Small compiler adapter; medium lifecycle tests |
| Variable temporary P/T | Berserk, Blood Lust, Great Defender, Howl from Beyond, Kry Shield, Subdue | 2: Great Defender, Howl from Beyond | Berserk: casting timing, delayed trigger and ActionPermanent; Blood Lust: conditions, subtraction/toughness values; Kry Shield/Subdue: prevention and action-local numeric references; Kry Shield also controlled target filters | Captured values, mana-value/power/toughness readers, later dynamic modifications | Medium adapters and capture/identity tests |
| Optional fixed counter payments | Force Spike, Power Sink, Thrull Wizard, Vodalian Mage | 2: Force Spike, Vodalian Mage | Thrull Wizard needs payment alternatives; Power Sink needs X, player actions, mana loss and collections | Payer references, decisions, separate payment-result bindings | Medium/large scope and payment tests |

The selection offers three concrete unlocks with existing engine services.
The other two candidates each unlock two cards and need broader prerequisites.
First-error frequency alone did not select the slice: the two old lords have
subtype-only filters, and Zombie Master has two distinct static declarations.
No additional slice was started.

## Supported signatures and contract

`EachPermanentLayerEffect/2` accepts `Other(ThisPermanent)/1` below nonempty
nested `And([filters])/1` in a filter that requires creatures. The source operand
must be the symbol `ThisPermanent`. The filter can also contain the established
`ControlledByAPlayer(SinglePlayer(You))/1` conjunct. Existing source-independent
filters and Or subexpressions remain available. Repeated source exclusions have
the same meaning as one exclusion. Exclusion/controller predicates below Or
remain rejected; their semantics cannot be flattened into conjunction flags.

Modifications remain fixed `AdjustPT(integer, integer)/2` and
`AddAbility([basic keyword symbols])/1`. No source exclusion was added to target,
cost, event, Aura, loop or resolving collection contexts. Subtype-only global
filters remain rejected by the existing creature guard. Malformed arities,
selector wrappers, tuple ThisPermanent(), host/target/event/player operands and
noncreature contexts fail compilation with positioned diagnostics.

RON owns syntax, role validation and immutable `staticCreatureFilterPlan`
decomposition. The plan holds an ordinary permanent filter plus current-source
controller and source-exclusion constraints. Each factory call builds fresh
continuous effects. It does not capture a controller UUID or an object selected
by another instance. Inventory assigns the operand of permanent-filter Other/1
the `permanent_reference` role without claiming support in other contexts.

Engine reuse:

- `BoostOtherControlledCreatures` and `GrantKeywordToOtherControlled` for
  controlled source exclusion.
- `BoostAllCreatures(..., WhileOnBattlefield)` and `GrantKeywordToAll` for
  source exclusion across controllers.
- Printed static registration, layer ordering, ability-removal dependencies,
  control effects, phasing, game cloning and continuous-effect retirement.

Membership uses the current source identity and controller. Other matching
instances can affect each other. Owners do not determine controlled membership.
Later entrants qualify. Source departure, phasing and ability removal suspend
its static effects; return installs effects for the new permanent. Recipient
phase-in restores current membership. Keywords use layer 6; fixed bonuses use
layer 7c after control changes in layer 2.
Rules evidence: CR 611.3a/b/c, 613.1b/f/g, 613.4c, 613.8a/b and 702.26b/d.
No engine contract was missing. No new characteristic LKI read, target legality,
announcement/resolution timing, cost transaction, replacement ordering or
simultaneous zone operation was introduced; existing engine services and tests
retain those contracts. This static form has no payment or saved-action result.

## Tests and coverage

TDD first produced expected unsupported Other compiler failures for all three
real lords and the global synthetic declaration, plus an unknown inventory role.
The synthetic test first needed CardPT in its fixture before it reached the
intended compiler failure. Declaration rejection tests were included before
implementation. After lowering, focused gameplay tests passed.

| Test | Card/rule mapping |
| --- | --- |
| TestRONOtherStaticCards/Kobold_Drill_Sergeant | Other controlled Kobold creatures get +0/+1 and trample; source excluded |
| TestRONOtherStaticCards/Kobold_Overlord | Printed first strike; other controlled Kobold creatures gain first strike |
| TestRONOtherStaticCards/Kobold_Taskmaster | Other controlled Kobold creatures get +1/+0; source excluded |
| TestRONOtherStaticCards (all subtests) | Opposing controller and non-Kobold exclusions; owner/controller distinction; later entry; source/recipient control changes; source/recipient phasing; source removal/restoration; departure/return; independent clone removal |
| TestRONOtherStaticInstances (all three lords) | Two products of the same real-record factory; each excludes itself but affects the other; bonuses accumulate on Crimson Kobolds; removal retires only that instance's effects |
| TestRONOtherStaticGlobalAndNested | Real engine execution of global exclusion, repeated/nested conjunctions, ordinary color disjunction, fixed bonus plus flying, both controllers, matching source excluded |
| TestRONOtherStaticRejectsInvalidDeclarations | Arity, role/scope, unsupported disjunction, absent creature restriction, unchanged target-filter boundary |
| TestRONOtherStaticInventoryRole | Nested source operand role, complete inspection separate from support |
| TestInventoryRepositoryCollection / TestInventoryLeavesRealRONGameplayAndRegistryUnchanged | Updated exact measurements; inspection leaves the registry and accepted Lightning Bolt gameplay unchanged |

Crimson Kobolds, Llanowar Elves and Forest are real RON fixtures. Existing engine
keyword/combat tests retain trample and first-strike execution evidence; this
slice tests RON keyword application and lifecycle rather than a new combat rule.
Three unique cards gain direct RON gameplay evidence for their full printed
rules. No collection-wide behavior total or completion certification is claimed.

## Final evidence and remaining work

Strict loading and inventory both report **877 compiled records**, **1,568
rejected records**, **259 accepted unique names**, **727 rejected unique names**.
The actual gain is **three records and three unique names**, exactly the estimate.
No accepted record regressed. Eighteen reprints of three other names advance
their first errors: Goblin King and Lord of Atlantis now report noncreature global
layer filters; Zombie Master now reports unsupported Landwalk grants. Their
remaining nested blockers are listed in the selection table.

All 12 RON source hashes match the vocabulary snapshot. No source was changed,
so no vocabulary refresh is needed. No source defect was corrected. The prior
Gaea's Liege/Fastbond observations remain open. These remaining diagnostics are
compiler/context gaps, not environment or build failures.

The first broad test run found only two stale exact inventory assertions
(874/1,571 versus 877/1,568). They were updated to the measured counts; their
registry and gameplay assertions were retained. Lint passed with zero issues;
automatic formatting only reordered imports in the new test. Final required
commands and results are recorded in the progress ledger.

Next ranked candidates:

1. Variable temporary P/T: Howl from Beyond and Great Defender (two concrete
   unlocks); later numeric readers support the four remaining affected cards.
2. Optional fixed counter payments: Force Spike and Vodalian Mage (two concrete
   unlocks); then alternative payments and X/results/collections.
3. Static landwalk plus creature-subtype subjects: Goblin King and Lord of
   Atlantis (two potential unlocks); Zombie Master also needs a granted activated
   ability contract. Reassess full records and fresh evidence before selection.

Runtime loading, complete gameplay mapping, Go-card/generator deletion and DSL
removal gates remain unmet. No production registry cutover, deletion or caller
audit is claimed. This run stops after P1-STATIC-OTHER.
