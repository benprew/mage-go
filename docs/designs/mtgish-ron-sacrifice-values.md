# Saved sacrifice characteristics

Capability: P4-SACRIFICE-VALUES. Date: 2026-10-04. One completed slice.

## Selection evidence

Fresh strict and semantic loaders reported 2,445 records, 986 unique names,
1,032 compiled records and 1,413 rejected records (656 rejected names).
Baseline compiler tests, short engine tests and build passed. Collection rejection
was the expected loader exit 1, not a build or environment failure. The working
tree contained user changes in `.gitignore` and the execution design. Both remain
outside this slice. There were no unrelated untracked files.

All complete records and nested expressions for the following candidates were
read directly from RON. Names count once; none of these candidate cards has a
reprint in the collection.

| Candidate | Distinct names | Estimated complete unlocks | Remaining blockers and downstream benefit | Effort |
| --- | --- | ---: | --- | --- |
| Saved sacrifice characteristics | Diamond Valley, Life Chisel, Homarid Spawning Bed, Priest of Yawgmoth | 3 | Priest still needs dynamic AddManaRepeated; private paid numbers provide its prerequisite | Small compiler adapter, medium identity/payment tests |
| Any-number counter payments and dynamic mana | Black Mana Battery, Blue Mana Battery, Green Mana Battery, Red Mana Battery, White Mana Battery | 5 | All need variable counter choice, saved removed amount, and dynamic mana; supports additional dynamic mana sources | Medium adapters, substantial choice/payment/mana solver tests |
| Variable optional mana payments | In the Eye of Chaos, Invoke Prejudice | 1 | Invoke also needs Opponent and SharesAColorWithAPermanent spell filters; enables event mana-value payments | Medium adapter, substantial event/payment tests |

Sacrifice values won because three complete cards use already established typed
engine payment services. Batteries offer more unlocks but require several
contracts together. The optional-payment candidate offers only one direct gain.
The selection, scope, reuse boundary and acceptance cases were stated before edits.
No missing engine prerequisite or source defect was found.

The complete records show: Diamond Valley combines tap and creature sacrifice
with life gain; Life Chisel combines the same sacrifice/value with controller
upkeep timing; Homarid Spawning Bed combines {1}{U}{U} and a blue creature
sacrifice with a mana-value count of blue 1/1 Camarids. Priest combines tap and
artifact sacrifice with repeated black mana; that consumer remains unsupported.

## Exact implementation

Supported declaration contexts: permanent-source `Activated/2` and
`ActivatedWithModifiers/3`, with existing cost conjunctions, target declarations
and activation modifiers. There must be exactly one `SacrificeAPermanent/1`
cost and no additional `SacrificePermanent` cost when a paid value is read.

Supported saved references are symbols, not tuple constructors:

- `ToughnessOfTheSacrificedCreature`, with the exact sacrifice filter
  `IsCardtype(Creature)`.
- `ManaValueOfTheSacrificedPermanent`, with any already supported matching
  sacrifice filter, including Homarid Spawning Bed's blue-creature conjunction.

One declaration can use only one characteristic kind. Consumers are
`GainLife/1`, `LoseLife/1`, `DrawNumberCards/1` and the count of `NumberTokens/2`
inside the existing basic `CreateTokens/2` forms. Fixed values and in-scope spell
X retain their earlier behavior. Unsupported consumers remain errors; this does
not add dynamic mana, damage amounts, optional sacrifice payments, spell
additional costs, static values, or filtered toughness costs. Independent
activations, rule scopes and factories cannot provide one another's results.
Unbound, tuple-shaped, mixed-kind and ambiguous references fail compilation.

Private signatures:

- `compileSacrificeValuePlan(Node, Node) (*sacrificeValuePlan, error)`
- `compileBoundCosts(Node, *sacrificeValuePlan) ([]boundCostFactory, bool, error)`
- `compileInstanceValue(Node, scope) (valueFactory, error)`
- `boundCostFactory`: `func(instanceBindings) mage.Cost`
- `valueFactory`: `func(instanceBindings) mage.ResolutionValue`
- `actionPlan.buildWithPayments(map[*castEventPlan]castEventBindings,
  map[*sacrificeValuePlan]sacrificeValueBindings) Actions`

Plans hold no game state. Each factory creates typed `NumberBinding` instances
and passes the same bindings to costs and consumers. The engine owns paid-number
storage on each activation and copy; no global last-action slot is used.

Reuse: `SacrificeCreatureIncludingSourceCostWithStats`,
`SacrificeMatchingIncludingSourceCostWithManaValue`, `BindNumber`, `FromNumber`,
total-cost preparation/commit, current characteristics, owner graveyards, target
incarnations, ability copies, game clones, life/draw actions and simultaneous
`CreateTokenCopies`. The stats service also captures power because its existing
contract requires distinct power/toughness bindings. RON spelling and scope stay
in the compiler. No engine API was added or changed.

Relevant contracts: CR 602.1a/1b, 602.2/2b, 400.7, 608.2h, and existing phasing,
copy, cost transaction and token-entry contracts. No new replacement ordering or
simultaneous-operation service is claimed. Existing token batch tests continue
to cover the shared entry service; this slice supplies its paid count.

## Gameplay and declaration evidence

| Card or contract | Test mapping |
| --- | --- |
| Diamond Valley, Life Chisel, Homarid Spawning Bed | TestRONSacrificeValueRealCards: complete records and definition validation, current counters, exact mana/tap costs, payment before resolution, owners, returned payment object, source departure/return/control/phasing, pending copies, clone isolation, correct life or Camarid characteristics/count |
| Diamond Valley | TestRONSacrificeValueCopyController: opponent copy receives the saved number without another payment |
| Life Chisel | TestRONSacrificeValueRealRestrictions: step, turn, permission and phased-source rejection without sacrifice; real-card test changes timing after payment |
| Pending results and transactions | TestRONSacrificeValuePendingAndRollback: two pending payments with different values; insufficient mana, phased/opponent victim and invalid choice restore costs |
| Factory identity and source payment | TestRONSacrificeValueFactoriesAndSelfPayment: same factory creates independent sources, self-sacrifice allowed, nonpositive saved toughness produces no life gain |
| Supported consumers | TestRONSacrificeValueConsumers: both characteristic kinds used by each supported consumer |
| Target declarations | TestRONSacrificeValueTargetLegality: announcement rejection before costs; departed, returned and phased targets prevent resolution |
| Invalid scopes and shapes | TestRONSacrificeValueInvalid: absent producer, incompatible filter, multiple sacrifices, mixed characteristic kinds, tuple reference, spell context and cross-activation reads |

After fixture API errors were corrected, new gameplay tests failed with the
expected unsupported-value errors for both saved references. Implementation then
passed focused tests. Broad tests exposed the previous sacrifice-result rejection
fixture and two stale inventory counts. The fixture now rejects a noncreature
cost with a creature-toughness read; all previous gameplay checks remain. Inventory
assertions now use fresh measured counts. This updates support boundaries without
weakening a rules test.

## Measured outcome and remaining work

Estimate and actual gain agree: three unique cards and three records.
Compiled records: **1,032 -> 1,035**. Rejected records: **1,413 -> 1,410**.
Accepted names: **330 -> 333**. Rejected names: **656 -> 653**.
Zero accepted-record regressions. Strict and inventory diagnostics match.
Nodes/signatures remain 72,938/8,528; unknown-role occurrences remain 16,921.
The existing inventory already classifies these operands as values. Global
engine_validated and behavior_verified remain null; the three new cards have
explicit gameplay and definition-validation evidence, not a collection-wide claim.

All 12 RON hashes match the vocabulary snapshot. No source changed, so the
snapshot needs no source refresh. No source defect was corrected. Gaea's Liege
and Fastbond concerns remain open. No environment failure occurred.

Largest remaining first diagnostics: PermanentRuleEffect (110),
AsPermanentEnters (90), instant/sorcery CastEffect (53), If rule (48),
ChooseADamageSource (46), Targeted_Modal (42), HostController upkeep (35),
TriggerI (28), ControlledByAPlayer filter (28), TargetGraveyardCard (26).
These counts do not estimate full-card unlocks.

Ranked next candidates to reassess: (1) dynamic repeated mana using typed payment
values, with Priest of Yawgmoth directly enabled and five Batteries downstream;
(2) any-number counter payments/results for the Batteries; (3) shared end-step
and delayed-trigger prerequisites for Breeding Pit, Rukh Egg and Hazezon Tamar.
The next run must compare fresh multi-card candidates before selection.
Runtime loading, complete gameplay, Go-card/generator deletion and DSL removal
remain unfinished. Existing callers and production registry behavior are preserved.
No second slice began.

## Validation

Baseline compiler/short-engine tests and build passed. Focused
`go test ./pkg/mage/dsl/ron -run TestRONSacrificeValue -count=1` passed after the
expected TDD failure. On final code, `go test ./pkg/mage/dsl/ron ./cmd/loadcards
-count=1`, `go test ./pkg/mage/... -short -count=1`, `make test`, `make build`,
`go vet ./...`, `make lint` (zero issues), and `git diff --check` all passed.
Automatic lint changes were reviewed; none changed semantics. Final strict
loader exited 1 as required, with 1,035 compiled and 1,410 rejected records.
Final diagnostics equal the post-implementation semantic loader's issues.
No check was blocked. Tests and API/rules docs are committed with the adapter;
user files remain unstaged.
