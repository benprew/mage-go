# Own-turn and upkeep activation timing

Slice P4-ACTIVATION-TIMING, 2026-10-04. Complete for the shapes below.

## Candidate comparison

Fresh baseline: RON/CLI tests, engine short tests and make build pass. Strict
loading exits 1 with 996 compiled records and 1,449 rejected records; 306 unique
names compile and 680 reject. Full AST inventory: 72,938 nodes, 8,528 signatures.
The inventory includes descendants beneath rejected parents. Estimates use
complete records and nested forms, with each reprinted name counted once.

| Candidate | Distinct names inspected | Full-card unlock estimate | Other blockers and downstream capability | Effort |
| --- | --- | ---: | --- | --- |
| Own-turn/upkeep activation timing | Disrupting Scepter, Svyelunite Priest; related names below | 2 | Other names need choices, counters, results, entry operations, limits or events. Timing is a shared prerequisite for their activations. | Small compiler adapter; moderate gameplay, timing, rollback and identity matrix |
| Temporary collection keyword grants | Spitting Slug, Icatian Skirmishers, Tidal Flats, Part Water | 0 | Slug needs combat events/filters and unpaid branches; Skirmishers needs attack events and band membership; Flats needs loops and member/controller payment bindings; Part Water needs X target collections and landwalk grants. Collection grants enable all four after those prerequisites. | Moderate adapter; substantial combat/filter/target prerequisites and testing |
| Variable optional mana payment | In the Eye of Chaos, Invoke Prejudice | 1 | Invoke still needs opponent cast events and color-sharing spell filters. Dynamic payment enables both spell taxes. | Moderate value/payment adapter and event, absence, copy and payment tests |

Timing wins by credible complete-card gain per effort. The exact scope, engine
boundary and acceptance cases were stated before implementation. No first-error
frequency is treated as an unlock estimate.

All complete distinct records with ActivatedWithModifiers were read, including
nested forms. The two selected shapes also occur in Clockwork Beast, Clockwork
Avian, Cyclopean Tomb, Dwarven Weaponsmith, Rock Hydra, Rag Man, Wand of Ith,
Gwendlyn Di Corci, Hell's Caretaker, Life Chisel, Life Matrix, Mirror Universe,
Nebuchadnezzar, Icatian Moneychanger and Colossus of Sardia. Their other rules
remain unsupported: entry counters, counter actions, event/result scopes, random
choices, graveyard targets, exchanges, static rules or X costs. Instill Energy,
Ebon Praetor and Gate to Phyrexia combine the timing shape with use limits;
those conjunctions remain rejected. Library of Alexandria needs a hand-count
condition; Fire Drake and Vampire Bats need limits. No dependent card is counted
as a complete unlock.

## Exact contract and reuse

Supported top-level permanent rule:

```
ActivatedWithModifiers(cost, actions,
    ActivateOnlyIf(PlayerPassesFilter(You, IsTheirTurn)))
ActivatedWithModifiers(cost, actions,
    ActivateOnlyIf(PlayerPassesFilter(You, IsTheirUpkeep)))
```

The arities are 3, 1 and 2. You and the two filters must be symbols.
Costs and action bodies use their existing supported shapes. The compiler
retains their declaration checks, typed bindings and target legality. Bare
Activated/2 uses the same builder and preserves its existing callers.

`compileActivatedRule(Node, scope) (optionFactory, error)` creates fresh costs,
actions, targets and abilities per factory. `compileActivationTiming(Node)
([]mage.ActionOption, error)` stores immutable options. IsTheirTurn maps to
WithYourTurnOnly. IsTheirUpkeep adds a pure WithActivationCondition step query;
the engine checks both through CanActivate before payment. There is no game
state in the plan and no extra action at resolution. You means the current
activating controller; owner and later source controller do not select timing.
Restrictions do not grant priority or change the engine's normal activation
permission. An ability already on the stack does not repeat this timing check.

Unknown modifiers, And combinations, other players/filters, malformed arities,
tuple forms of symbols, missing bindings, invalid costs and activations on
instants/sorceries reject before factory publication. Limits, other-player
permissions, general conditions and other timing forms remain separate slices.
Inventory now assigns cost/actions/activation_modifier to ActivatedWithModifiers
operands and activation_condition, player_reference and activation_player_filter
to the relevant descendants. Role reporting does not assert support.

Engine reuse: NewAction/WithCost, WithYourTurnOnly, WithActivationCondition,
CanActivate, transactional activation and rollback, typed player/permanent
targets, captured source/target incarnations, ability copying/retargeting,
phasing, game clones, mana solver, layers, discard decisions and cleanup.
No engine implementation or public DSL API changes. No new LKI query,
replacement order or simultaneous operation is established. Source departure
uses the existing pending ability capture; a returned target has a new identity.
Local rules read: CR 117.1b/d, 602.1a/b, 602.2a/b and 602.5/5a. Existing
identity and phasing contracts continue to supply target behavior.

## Tests and coverage

Gameplay and invalid tests were written first. An incorrect test-only DrawStep
name was corrected to Draw. The real-card matrix then failed with unsupported
ActivatedWithModifiers, as expected. After implementation it passes. A later
mana fixture used a nonexistent pool getter; its assertion now uses TotalMana.
No gameplay assertion was weakened.

| Card or context | Evidence |
| --- | --- |
| Disrupting Scepter | TestRONActivationTimingLegality and TestRONActivationTimingScepter: complete RON record, definition validation, own turn in combat and draw steps, opponent-turn rejection, current controller, costs/tap/rollback, self/opponent discard decisions at resolution, source departure/return/control/phasing, pending clone isolation |
| Svyelunite Priest | TestRONActivationTimingLegality, TestRONActivationTimingPriest and TestRONActivationTimingInstancesAndCopy: complete record, definition validation, own upkeep versus opponent upkeep/draw, summoning sickness, wrong controller, mana/tap costs and rollback, self/opponent creature targets, shroud and retained flying/P/T, source departure/return/control/phasing, target departure/return/phasing/control/noncreature legality, pending clones, retargeted copy and separate original, cleanup isolation |
| Shared declarations and repeated activations | TestRONActivationTimingInvalid and TestRONActivationTimingRepeatAndInventory: invalid player/arity/symbol/modifier/cost/scope declarations, no unintended activation limit, no timing recheck at resolution, exact inventory roles |
| Mana ability context | TestRONActivationTimingManaPlanning: both timing shapes, pure affordability queries, rejected off-turn/step planning, immediate mana resolution |

Two additional unique names now have gameplay evidence for all rules in their
complete RON records. Collection-wide engine_validated and behavior_verified
remain null. Compilation does not prove complete collection gameplay.

## Measurement and remaining work

Estimate equals actual gain: two unique names, seven records (six Scepter
reprints and one Priest). Loader 996 -> 1,003/2,445 compiled, 1,449 -> 1,442
rejected. Names 306 -> 308 accepted, 680 -> 678 rejected. No accepted-record
regression. Inventory and strict-loader issues match. Nodes/signatures remain
72,938/8,528; unknown-role occurrences fall 19,278 -> 17,943 through operand
reporting, including still-rejected declarations. Source hashes are unchanged.
No source-data defect was corrected; known Gaea's Liege/Fastbond concerns remain.

The largest first diagnostics are PermanentRuleEffect (110), AsPermanentEnters
(90), non-spell CastEffect (53), rule If (47), ChooseADamageSource (46),
Targeted_Modal (42), and HostController upkeep references (35). Modifier gaps
now have specific diagnostics rather than a single unsupported-rule diagnostic.
They remain rejection errors, not implementation or environment failures.

Ranked next candidates: (1) activation limits, Fire Drake and Vampire Bats
directly, plus prerequisites for Instill Energy, Ebon Praetor and Gate to
Phyrexia; (2) general activation hand-count conditions, Library directly with
other state conditions requiring fresh comparison; (3) variable optional mana
payment, Chaos directly and Invoke after cast/filter prerequisites. Reassess
complete records and shared multi-card benefit next run.

Production startup, Go-card/generator removal and DSL deletion gates remain
closed. No DSL caller audit or cutover is claimed. This run stops after timing.
