# RON fixed source-counter activation costs

Run 2026-10-04: P4-COUNTER-COST. One completed adapter slice; no runtime cutover.

## Selection from fresh evidence

Baseline RON/CLI tests and build pass. Strict loader and semantic inventory both
exit 1: 1,019 of 2,445 records compile; 1,426 rejected records represent 664
unique names. There are 986 names in total. Reprints count once below.
The full AST inventory includes descendants below unsupported parents. Complete
RON records, including all identical reprints, were read for the names below.
Go card definitions were not used as requirements.

| Candidate | Distinct affected names | Estimated complete unlocks | Other blockers and downstream capabilities | Effort |
| --- | --- | ---: | --- | --- |
| Fixed source-counter activation costs | Feral Thallid, Thorn Thallid, Elvish Farmer, Thallid, Thallid Devourer, Spore Flower, Icatian Javelineers, Triskelion, Delif's Cube, Osai Vultures, Scavenging Ghoul, Rasputin Dreamweaver, Life Matrix | 2 | Token creation blocks Farmer/Thallid/Devourer; prevention blocks Flower; entry counters block Javelineers/Triskelion; delayed triggers block Cube; end-step/history/intervening conditions block Vultures/Ghoul; entry/prevention/intervening conditions/rule effects block Rasputin; permanent ability grants block Matrix | Small compiler adapter; medium payment/identity/gameplay tests |
| Sacrifice-characteristic results | Diamond Valley, Life Chisel, Priest of Yawgmoth, Homarid Spawning Bed | 2 | Valley/Chisel need captured toughness; Priest needs repeated dynamic mana, Bed needs token creation. Typed payment-result identity is a shared prerequisite for all four | Medium compiler/result-binding work; substantial LKI/payment/copy tests |
| Variable optional mana payments | In the Eye of Chaos, Invoke Prejudice | 1 | Chaos needs event-spell mana-value payment; Invoke also needs Opponent and color-sharing spell filters. Enables captured variable payments for both cast triggers | Medium adapter; substantial optional-payment/choice/event tests |

Fixed removal wins on the smallest credible effort for two complete unique cards
and concrete shared prerequisites for three token-producing cards. All three
candidates reuse existing engine contracts. The five Mana Batteries also need
any-number counter removal, typed paid-count results and repeated mana, so they
are not unlocks from fixed removal. The five storage lands need those contracts
plus entry, untap and intervening-if support. These forms remain rejected.

## Exact supported shapes and boundaries

`compileCounterRemovalCost(Node) (costFactory, error)` accepts:

- `RemoveCounters([ACounterOfTypeFromPermanent(counter, ThisPermanent)])`.
- `RemoveCounters([NumberCountersOfTypeFromPermanent(Integer(n), counter, ThisPermanent)])`,
  with a literal nonnegative integer `n`, including zero.

The list contains exactly one instruction. Counter types reuse the complete
validated set in `compilePermanentCounterType` (25 named kinds and ten literal
P/T pairs); no new counter type is introduced. Tuple arity, kind, amount, type
and symbol source are checked before factory construction.

Contexts: battlefield-source `Activated/2` and `ActivatedWithModifiers/3`,
including nested existing `And/1` cost conjunctions and supported timing/limits.
Each factory call constructs a fresh `mage.RemoveCountersCost(counter, n)`.
The source is the activating instance supplied to the engine cost transaction,
not a compiled mutable target or a global reference. No result binding is added.

Unbound/host/target/event references, arbitrary values/X, any/all-counter amounts,
multiple instructions, removal actions, entry/static rules and optional payment
remain unsupported. The existing optional-payment adapter explicitly admits only
mana costs; this change does not widen it. Granted ability contexts remain
blocked by their own adapter prerequisites.

`inventoryArgumentRole` reports `counter_removal_instruction` below a cost
`RemoveCounters/1`, then `value`, `permanent_counter_type`, and
`permanent_reference` for the two exact instruction shapes. This includes known
cost operands below optional-payment parents even though the compiler rejects
them. Unknown parent roles remain unknown; reporting does not claim execution.

## Engine and rules reuse

`RemoveCountersCost` and the total-cost transaction own affordability, locked
source incarnation, actual payment, clone isolation and rollback. The adapter
contains no reusable rules execution. Controller permission, timing and limits
use the existing activation service. Non-tap removal costs do not require an
old creature; tests activate the real cards with summoning sickness.

Existing source references/LKI, typed any-target bindings, copied stack objects,
phasing, regeneration shields and damage replacement/prevention supply the
resolution behavior. CR 602.1a/602.2/602.2b requires activation payment and illegal
activation rollback; CR 122.1/122.1a/122.2 defines counters and their zone-change
behavior; CR 400.7 and 702.26 preserve the incarnation/phasing distinction.
Existing contracts cover source departure and copy resolution (113.7a, 707.10b).
Removal is payment, and the regeneration/damage action waits for resolution.
No new simultaneous-operation or counter-removal replacement contract is added.
Damage prevention still uses the shared pipeline; it does not refund counters.

## Test mapping and TDD evidence

Tests were added before the adapter. After fixing fixture types (`uint8` counters
and the `PhaseIn` list argument), the gameplay/composition/forms tests failed on
`unsupported cost (RemoveCounters)` and inventory lacked removal operand roles.
Invalid declaration tests already passed and continue to pass.

After implementation, fixture errors were corrected: real cards have an upkeep
trigger at ability index 0 and an activation at index 1; `RunStepWithPriority`
resolves triggers, so explicit upkeep events and stack placement test the pending
state. A timing fixture uses an artifact rather than a creature without CardPT.
The expectations for payment, timing and real gameplay remain unchanged.

| Card or context | Test | Evidence |
| --- | --- | --- |
| Feral Thallid, Thorn Thallid, complete records | TestRONCounterCostRealCards | Definition validation; own versus opponent upkeep; pending placement then one spore counter; insufficient payment; exactly three removed at activation; regeneration versus damage at resolution; summoning sickness; source return/phasing/phase cycle/control; pending clones; regeneration replaces destruction |
| Thorn Thallid | TestRONCounterCostTargetsCopiesAndInstances | Opponent permission and invalid target reject without payment; phased source rejects; independent sources; copied ability pays nothing; departed source damage; returned/phased/noncreature targets; shared prevention; clone isolation |
| Shared costs | TestRONCounterCostRollbackAndComposition | Mana/tap before and after counter cost; duplicate removal costs cannot spend the same counters; atomic failure and exact success |
| Shared factory/payment | TestRONCounterCostFactoryIsolationAndManaRollback | One compiled factory yields independent sources; failure on later mana restores counter/tap costs; payment in one game leaves its clone unchanged; independent clone payment/resolution |
| Exact forms and modifiers | TestRONCounterCostFormsAndTiming | One counter, zero counters, fixed P/T-counter removal; definition validation; upkeep/once-per-turn reject without extra payment; resolution does not recheck timing |
| Invalid declarations | TestRONCounterCostInvalid | Kind/arity/list/amount/type/reference errors, variable amount, unsupported removal action and optional payment contexts |
| Semantic reporting | TestRONCounterCostInventory | Removal instruction, amount, type and source operand roles |

## Fresh results and remaining work

Estimate and result agree: two complete names, two records. Final loader accepts
1,021/2,445 records and rejects 1,424 (662 unique names); 324 unique names compile.
No previously accepted record became rejected. Strict and semantic-inventory
issues match exactly. AST nodes/signatures remain 72,938/8,528; unknown-role
occurrences fall from 17,378 to 17,251. Global engine-validation and behavior
counts remain unmeasured (`null`); compilation is separate from gameplay evidence.
Two new complete records have RON gameplay tests, with the contracts above.

No RON source changed. All source hashes match the vocabulary snapshot, so no
source refresh is required. No source-data defect was corrected. Existing
Gaea's Liege/Fastbond concerns remain open. No engine prerequisite or environment
failure was found. Whole-collection rejection remains intentional and atomic.

Next candidates, to reassess with fresh complete records: (1) basic token creation
for Thallid, Thallid Devourer and Elvish Farmer; (2) typed sacrifice toughness for
Diamond Valley/Life Chisel; (3) any-number counter payment/results and dynamic
mana for the five Mana Batteries. This run does not implement those candidates.
Go definitions, startup wiring, generators and DSL exports retain their callers;
removal gates remain closed.
