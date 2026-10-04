# RON activation limits

Capability P4-ACTIVATION-LIMITS, completed 2026-10-04.

## Selection and contract

Fresh baseline: RON tests and `make build` pass. Strict loading rejects the
collection with 1,003 compiled records and 1,442 issues. The full AST inventory
has 2,445 records, 986 names, 72,938 nodes and 8,528 contextual signatures.
First errors alone were not used to estimate unlocks. Complete records and
nested forms were read for the following candidates; reprints count once.

| Candidate | Distinct affected names | Estimated full-card unlocks | Other blockers | Downstream benefit | Effort |
| --- | --- | ---: | --- | --- | --- |
| Activation limits and modifier conjunctions | Fire Drake, Vampire Bats, Dream Coat, Instill Energy, Ebon Praetor, Gaea's Touch, Gate to Phyrexia | 2 | Choices/color layers; host attack permission; counters/payment results; sorcery timing/hand placement; matching sacrifice costs | Shared per-ability limit and composed timing for seven names | Small compiler adapter plus medium identity/clone/payment tests; engine isolation repair became necessary |
| Hand-count activation condition | Library of Alexandria | 1 | None for the exact seven-card condition | Pure player-state condition groundwork | Small adapter, medium state/timing/controller tests |
| Variable optional mana payment | In the Eye of Chaos, Invoke Prejudice | 1 | Invoke needs SharesAColorWithAPermanent/ExceptFor spell filters | Captured spell mana value and variable optional payment for two cast triggers | Medium payment/value adapter and substantial rollback/choice tests |

Limits give the largest credible direct multi-card gain for this comparison.
The selected boundary was stated before editing: permanent-source
ActivatedWithModifiers, fixed positive limits, and conjunctions with supported
own-turn/upkeep timing. Reuse engine limits, payment and object identity;
keep spelling, arity, lexical context and rejection in ron. Acceptance cases:
real-card payment and resolution, rejected declarations, independent instances,
control/phasing retention, leave-and-return reset, copies, clones, cleanup and
mana queries. No additional card family was begun.

## Supported declarations

* `ActivatedWithModifiers(cost, actions, ActivateOnlyOnceEachTurn)`.
* `ActivatedWithModifiers(cost, actions,
  ActivateNoMoreThanNumberTimesEachTurn(Integer(n)))`, integer literal n > 0.
* `And([modifier, ...])`, nonempty, recursively containing these limits and
  `ActivateOnlyIf(PlayerPassesFilter(You, IsTheirTurn|IsTheirUpkeep))`.

The once-only form is a symbol, not a zero-argument tuple. Maximum/Integer
arities are 1/1. There can be at most one numbered maximum per activation,
including nested conjunctions: reject duplicate maxima rather than let one
option overwrite another. Once-only and numbered limits can coexist and both
apply. Unknown modifiers, dynamic/zero/negative maxima, wrong arity/kind,
empty conjunctions, wrong player references and instant/sorcery rule contexts
remain declaration errors. Existing cost and action limitations remain errors.

The compiler holds immutable options and creates fresh actions, costs, targets
and effects per factory. It introduces no target or result binding. Inventory
reports conjunction children as activation_modifier, the maximum's operand as
activation_limit_value, and its literal as integer_literal. Inventory roles are
not support claims.

## Engine prerequisite and reuse

The first real RON tests exposed two missing engine contracts: successful
activation mutated an action shared with another game clone and the card
itself. A returned card consequently retained the old object's count. Synthetic
engine tests independently failed for clone sharing, returned-object limits and
card declaration mutation before the engine edit.

`ActionDefinition.markActivatedInGame` now obtains the game's mutable source,
copies its runtime abilities, finds the matching action ID and marks that action
only after successful payment. Both stack activation and mana production use
this shared service. An absent source needs no lasting runtime count. Granted
ability tracker handling is retained. Public signatures and current callers
are preserved. Unlimited actions need no count copy.

Reuse: WithOncePerTurn, WithMaxActivationsPerTurn, CanActivate, transactional
payment, the shared mana solver, incarnation-bound This, Until/ModifyPT,
control and phasing services, stack copying, game cloning, and cleanup resets.
Local rules: CR 602.1b/602.2/602.5b/c, 400.7 and 702.26; existing cleanup and
continuous-effect contracts remain. Control changes and phase cycles keep the
count; a returned object starts unused. A stack copy is not an activation.
Failed payment/legality checks consume no count. Cleanup resets phased sources
and detaches shared runtime actions. No new replacement ordering, LKI query,
simultaneous operation, target binding or rules implementation in ron is added.

Requirements came from complete RON records. Local data confirms Fire Drake's
once-per-turn and Vampire Bats' twice-per-turn Oracle instructions
(`data/DRK.json`, `data/LEG.json`). No source defect was corrected.

## Gameplay evidence

| Card or context | Tests |
| --- | --- |
| Fire Drake (DRK record 38), Vampire Bats (LEG record 285) | TestRONActivationLimitRealCards: full definitions and validation, flying/base P/T, no tap/sickness restriction, unpaid activation, exactly one/two uses, stack timing, mana retention on rejection, control change, phase cycle, departure/return, incarnation bonus, pending clones, cleanup and opponent-turn reuse |
| Vampire Bats | TestRONActivationLimitInstancesCopiesAndClone: fresh factories, independent permanents, clone use counts/effects, copy before second activation, correct copied source bonus |
| Fire Drake | TestRONActivationLimitPhasedReset: phased-source rejection, cleanup reset while phased, clone isolation |
| Shared timing/target/payment context | TestRONActivationLimitConjunctionAndRollback: both supported timing filters, typed permanent target, invalid-target/turn rejection before mana/tap payment, successful payment, untap retains count, resolution after timing change |
| Shared mana and inventory contexts | TestRONActivationLimitManaAndInventory: pure repeat affordability queries, immediate production, solver exclusion after use, clone isolation, cleanup reuse, exact nested roles |
| Invalid declarations | TestRONActivationLimitInvalid; retained TestRONActivationTimingInvalid and RepeatAndInventory rejection cases |
| Synthetic engine contract | TestActivationLimitIsolation: maximum one/two, clones, card declaration immutability, return, controller change, phasing and independent resolution |

Previously rejected valid once-only/conjunction entries in the timing test were
replaced by positive gameplay tests. Remaining rejection expectations remain.
The targeted rollback fixture initially used unsupported PlayerAction/GainLife;
it was corrected to the supported typed permanent tap action before asserting
payment and resolution. No supported gameplay expectation was weakened.

## Measurement and remaining work

Estimate and actual gain agree: two names, two records. Strict loader:
1,003 -> 1,005 compiled, 1,442 -> 1,440 rejected. Accepted names: 308 -> 310;
rejected names: 678 -> 676. No previously accepted record regressed. Strict and
inventory diagnostics match. Unknown-role occurrences: 17,943 -> 17,852;
nodes/signatures unchanged. Global engine_validated and behavior_verified stay
null; two additional names now have complete-record RON gameplay evidence.

Remaining first diagnostics include PermanentRuleEffect (110), AsPermanentEnters
(90), non-spell CastEffect (53), If rule (47), ChooseADamageSource (46),
Targeted_Modal (42), HostController upkeep reference (35) and PutCounters (29).
Gate to Phyrexia now reaches its missing SacrificeAPermanent cost; the other
five dependent names retain the blockers above. These gains are prerequisites,
not complete gameplay coverage for those names.

All 12 RON hashes match the vocabulary snapshot; no source refresh is required.
Gaea's Liege/Fastbond concerns remain unresolved. Production cutover, complete
loading/gameplay, Go-card/generator deletion and unused DSL cleanup remain open.
No exports were removed and the removal gate is closed.

Validation results and ranked next candidates are recorded in the progress
ledger. User changes to .gitignore and the execution design are excluded.
