# Source counter payment results and repeated mana

Slice P4-COUNTER-PAYMENT, 2026-10-04. Completed compiler adapters; no engine
behavior change. Compilation evidence and gameplay evidence remain separate.

## Fresh selection evidence

Baseline focused tests, short engine tests and build passed. Strict loading and
semantic inspection both exited 1 with 1,035 compiled and 1,410 rejected records.
There are 2,445 records and 986 distinct names. The semantic AST inventory includes
all nested expressions, even under unsupported parents. All complete records
listed below were inspected. Reprints count once in this comparison.

| Candidate | Distinct affected names | Estimated complete unlocks | Other blockers and downstream use | Effort |
| --- | --- | ---: | --- | --- |
| Source counter payment and repeated mana, selected | Black Mana Battery, Blue Mana Battery, Green Mana Battery, Red Mana Battery, White Mana Battery; Bottomless Vault, Dwarven Hold, Hollow Trees, Icatian Store, Sand Silos; Tetravus | 5 | Storage lands still need entry-tapped, optional untap and intervening-if triggers. Tetravus needs entry counters, optional resolution payment, linked tokens, token rules and exile results. Shared typed counter values support later storage and optional-payment work. | Moderate: existing engine transaction, number binding and mana services; adapters plus real-card, binding and payment tests |
| Saved-sacrifice repeated mana | Priest of Yawgmoth, Sacrifice; Metamorphosis for the related modifier form | 1 | Priest has no other blocker. Sacrifice needs additional casting costs. Metamorphosis also needs arithmetic and restricted mana. A general mana-value consumer helps both sacrifice and counter producers. | Small to moderate: saved sacrifice binding exists; mana timing and payment/solver tests still required |
| End-step and delayed triggers | Breeding Pit, Rukh Egg, Hazezon Tamar | 1 for controller end-step alone; 2 with next-end-step delayed support | Rukh Egg needs a delayed trigger and source-independent context. Hazezon needs next-upkeep delay, controlled-land counting, leave trigger and batch exile. Shared step events and captured delayed contexts serve all three. | Larger: step-event adapters and delayed context lifetime, APNAP, copies and source departure tests |

Five immediate unique-card gains from one payment shape have the best credible
benefit for this run. Merely adding a mana-value consumer gives one direct gain;
delayed triggers need more lifetime work. The ledger's previous suggestion was
reassessed. The selected contract and acceptance cases were stated before edits.

## Supported declarations and reuse boundary

Supported in permanent-source `Activated/2` and `ActivatedWithModifiers/3`:

- `RemoveCounters([AnyNumberOfCountersOfTypeFromPermanent(counter, ThisPermanent)])`.
  There must be exactly one counter-removal instruction and cost in the activation.
  Existing `And([costs...])` can combine it with other supported costs.
- `counter` uses the existing permanent counter validator, including charge,
  storage and supported P/T counter forms. `ThisPermanent` must be a symbol.
- Symbol `TheNumberOfCountersRemovedThisWay` reads that activation's chosen,
  successfully paid count. The result does not read current counters.
- `AddManaRepeated(production, TheNumberOfCountersRemovedThisWay)` uses concrete
  mana symbols, `AnyManaColor`, or existing `And([production...])` forms. Each
  listed production gets the saved count. Any-color production chooses one color
  for the full amount. Zero adds no mana.
- Existing instance-value consumers also accept this binding: `GainLife/1`,
  `LoseLife/1`, `DrawNumberCards/1` and `NumberTokens/2` in basic token creation.
  An unused payment result is permitted; the engine still locks the choice.

Reject unbound and tuple-shaped reads, multiple counter payments, mixed fixed
and variable counter removals, non-source subjects, invalid counter types and
cross-activation references. Fixed counter costs keep their previous behavior.
Optional resolution counter payments, trigger results, spell costs, X counter
costs, sacrifice-value mana and general dynamic mana values remain unsupported.
Storage lands and Tetravus still reject their complete records.

Private interfaces:

- `compileCounterPaymentPlan(Node) (*counterPaymentPlan, error)` validates the
  producer and stores only its counter type.
- `compileActivationCosts(Node, *sacrificeValuePlan, *counterPaymentPlan)` returns
  `([]boundCostFactory, bool, error)` and creates costs with the instance binding.
- `compilePaidCounterMana(Node, scope) ([]effectFactory, error)` lowers the saved
  amount; `compileInstanceValue(Node, scope) (valueFactory, error)` rejects reads
  without a producer in this activation.
- `actionPlan.buildWithBindings(instanceBindings) Actions` keeps payment bindings
  while creating fresh target declarations. Previous build callers remain valid.

RON owns syntax, scope, diagnostics and per-factory binding creation. Immutable
plans have no chosen amount or game state. Costs and effects share a fresh typed
`NumberBinding` in each factory instance. The engine owns choice and total-cost
transactions (`RemoveAnyCountersCost`), saved values (`BindNumber`, `FromNumber`),
clones/copies, target incarnations, layers, token batches and mana production
(`AddManaAmount`, `AddManaOfAnyOneColor`). No global last-action slot, card-name
branch, Go-card fallback, declaration panic or partial production registry exists.

Mana abilities resolve immediately under CR 605.1a, 605.2 and 605.3b, including
zero choices. A targeted mana-producing ability uses the stack (605.5a); spells
remain spells (605.5b). The engine's mana planner excludes abilities with counter
payments instead of inventing a player choice. Production queries do not choose,
pay or mutate state. Manual activation uses the existing shared transaction.
This slice does not extend automatic payment planning for choice-based costs.
CR 602.2/2b provides payment rollback; CR 400.7 supplies incarnation identity.
No new replacement-order or simultaneous-operation service is introduced. Existing
replacement and token-entry services remain in use.

## Gameplay and declaration evidence

| Card or contract | Test |
| --- | --- |
| All five Mana Batteries, each complete record | `TestRONCounterPaymentBatteries`: definition validation, mana/tap charging costs, stacked charge placement, counter removal, zero/partial/all choices, color, immediate mana, controller, clone isolation and pure solver exclusions |
| Invalid payments and legality | `TestRONCounterPaymentInvalidChoicesAndRollback`: negative/excess counts, unavailable combined mana, unauthorized player, phased or tapped source; costs and mana remain unchanged |
| Saved values and source lifetime | `TestRONCounterPaymentSavedInstances`: two pending counts, ability copy with a different controller, source departure/return/control/phasing and independent clone resolution |
| Target identity and timing | `TestRONCounterPaymentTargets`: invalid announcement before a payment choice, targeted ability on the stack, departed/returned or phased target fails resolution |
| Fresh factories and consumer contexts | `TestRONCounterPaymentFactoriesAndConsumers`: same compiled factory makes independent bindings; any-color and multiple-color mana, life loss, draw and token counts |
| Zero counters, other counter types, unused result | `TestRONCounterPaymentZeroAndUnused`: zero removal from a source without counters still permits mana; P/T counter declaration with no result consumer |
| Declaration boundaries | `TestRONCounterPaymentInvalidDeclarations`: no producer, tuple result, duplicate or mixed removals, wrong subject/type and cross-rule references |
| Semantic inventory | `TestRONCounterPaymentInventory`: removal operands, source, mana production and saved-value roles |

Tests were written first. After fixture API errors were corrected, gameplay tests
failed on the expected unsupported counter instruction. The inventory test then
failed on missing child roles before the reporting adapter was added. The old
fixed-cost invalid fixture now rejects a host subject rather than the newly
supported source form. Repository inventory assertions now use measured counts.
No gameplay expectation was weakened. A phased-source test reads the object after
phasing in because normal battlefield lookup excludes phased objects.

## Final measurements and limits

Actual gain equals estimate: five distinct names and five records. Loader counts
are **1,040 compiled / 1,405 rejected**, from **1,035 / 1,410**. Accepted names
increase **333 -> 338**; rejected names decrease **653 -> 648**. No previously
accepted record regressed. Strict and inspection diagnostics are identical.
AST nodes/signatures remain **72,938 / 8,528**. Unknown-role occurrences decrease
**16,921 -> 16,599**. Collection-wide engine-validated and behavior-verified counts
remain null; the five new cards have explicit definition and gameplay evidence.

No RON source changed; all twelve hashes match the vocabulary snapshot. No source
defect or environment blocker was found. Gaea's Liege and Fastbond concerns remain
open. Startup, Go definitions, generators and DSL exports retain their callers.
Removal gates remain closed; no cutover or unused-DSL deletion is claimed.

Validation command outcomes are recorded in the progress ledger. Remaining large
families include rule effects, entry operations, casting rules, conditions,
modal targets, prevention and event-player references.
