# RON matching sacrifice activation costs

Capability: P4-SACRIFICE-MATCHING. Completed 2026-10-04.

## Selection evidence

Fresh baseline: RON/CLI tests, engine short tests and `make build` pass.
Strict and semantic-inventory loaders both exit 1 with 1,005 compiled and
1,440 rejected records, 310 accepted and 676 rejected unique names. This is
collection rejection, not a Go build failure. The AST inventory visits nested
expressions beneath unsupported parents. Complete records for matching
sacrifice costs were read; reprints count once.

| Candidate | Distinct affected names | Estimated full unlocks | Other blockers | Downstream benefit | Effort |
| --- | --- | ---: | --- | --- | --- |
| Matching sacrifice activation costs | Nine direct names below; dependent names in the audit below | 9 | Dependent records need result bindings, counters, tokens, filters, layers, choices or other contexts | Reusable transactional choice cost for artifact, creature, land-subtype and color/subtype combinations | Small adapter; medium gameplay/rollback/identity tests |
| Fixed counter placement | Armor Thrull, Fungal Bloom, Citanul Druid; Black/Blue/Green/Red/White Mana Battery, Dwarven Weaponsmith, Ebon Praetor | 3 | Batteries need variable removal and typed payment results; Weaponsmith needs sacrifice costs; Praetor also needs removal and paid-subtype conditions | Shared counter spelling/actions for triggers and targeted/source activations; concrete five-card battery prerequisite | Medium counter-kind adapter; medium identity/target/replacement tests; verify unusual P/T counter support |
| Variable optional mana payment | In the Eye of Chaos, Invoke Prejudice | 1 | Invoke needs ExceptFor/SharesAColorWithAPermanent and controlled creature spell-color matching | Mana-value payment captured from two cast triggers | Medium adapter; substantial optional decision/payment/clone tests |

Sacrifice costs win on nine credible direct unique unlocks for a small adapter.
Counter placement remains valuable but does not unlock the Batteries by itself.
Variable optional mana payment has one immediate unlock. The selection, contract,
reuse boundary and acceptance cases were stated before editing.

## Supported declaration contract

`SacrificeAPermanent/1` is a tuple in permanent-source `Activated/2` or
`ActivatedWithModifiers/3` costs. It composes through the existing cost `And/1`
and existing supported activation timing/limit modifiers. Its operand is a
permanent filter, not a reference or an action. Supported atoms are those of
`compileFilter`: AnyPermanent, tapped/untapped/attacking/blocking status, card
and non-card types, supported subtype forms, color/non-color and keyword forms.
Nested filter `And/1` and `Or/1` require nonempty lists at every level.

RON spelling, arity, filter validation and diagnostics remain in `ron`.
`compileSacrificeCostFilter(Node) (mage.PermanentFilter, error)` lowers immutable
filters. `compileCosts` returns a factory that makes a fresh
`mage.SacrificeMatchingIncludingSourceCost(filter, text)` for each card factory.
The engine chooses one matching permanent controlled by the activating player;
source payment is permitted when it matches. Ownership does not supply payment
permission. The chosen incarnation is locked by the shared total-cost transaction.
The cost happens during activation and is not repeated at resolution or copying.

No new result binding or engine API is introduced. `Other(ThisPermanent)` and
player/source-relative filters remain errors in this cost context. Resolution
sacrifice actions, optional/must sacrifice payments, additional casting costs,
saved sacrificed characteristics, multiple-object declarations and other unsupported
forms remain errors. No global last-sacrifice value is read. Dependent records
remain rejected in their entirety. Inventory now reports the filter operand even
when its parent context or descendants remain unsupported.

## Engine reuse and rules

Read `pkg/mage/doc.go`, the ability-instance and payment contracts in
`engine-rules-contracts.md`, the RON/activation API in `dsl-reference.md`,
`action_sacrifice.go`, `payment_transaction.go`, `costed_mana.go`,
`mana_ability.go`, `mana_system.go` and compiler/action/target construction.
Local CR 602.1a, 602.2/2b and 701.21a establish activating-player payment,
transactional legality and movement to the owner's graveyard. CR 400.7,
702.26 and existing ability-source/target contracts supply incarnation, phasing,
source LKI, copies and clones. Sacrifice is not destruction: regeneration shields
do not stop it. No new replacement ordering or simultaneous-operation contract
is introduced; this declaration sacrifices one permanent per cost through the
existing payment service. Existing multi-cost choice-lock tests remain evidence
for the shared engine service.

Ashnod's Altar is an explicitly activated mana ability and resolves immediately.
The existing automatic mana planner admits tap plus mana costs, not sacrifice
choices (`manaSourceAbilityForPlanning`). This slice preserves that documented
planning boundary: pure queries do not advertise, choose or pay sacrifice mana;
explicit activation produces two usable colorless mana. No engine rule requires
automatic sacrifice selection. No claim of new automatic sacrifice planning is made.

## Complete-record and nested audit

All names containing `SacrificeAPermanent` were inspected in full. This includes
cost, resolving action, optional/must payment, player action, replacement and
casting contexts. Their nested filter/value/action requirements differ.

| Names | Remaining complete-record blockers after this slice |
| --- | --- |
| Atog; Fallen Angel; Dark Heart of the Wood; Goblin Chirurgeon; Horror of Horrors; Gate to Phyrexia; Orcish Mechanics; Sage of Lat-Nam; Ashnod's Altar | None; full records compile and have gameplay tests |
| Dwarven Weaponsmith | PutCounters; own-upkeep/tap/sacrifice cost now supported |
| Ebon Praetor | P/T counter placement/removal and APermanentWasSacrificedThisWay subtype result |
| Diamond Valley; Life Chisel | ToughnessOfTheSacrificedCreature typed result |
| Priest of Yawgmoth | ManaValueOfTheSacrificedPermanent typed result and dynamic mana |
| Homarid Spawning Bed | Paid mana-value result and token creation |
| Elvish Farmer; Thallid Devourer | Spore counters, counter costs and tokens |
| Orc General | Other(ThisPermanent) cost and collection filters |
| Heroism | Damage-prevention replacement and paying damage-source controller |
| Raiding Party | Targeting restrictions, player loops, choices/results and batch destruction |
| Thelonite Druid | Temporary collection animation and controlled-land selection |
| Thelonite Monk | Permanent-duration land-type layer action |
| Tourach's Gate | Counter actions/queries, host cost/state condition and source exclusion/context |
| Hell's Caretaker | Graveyard target and battlefield placement |
| Lord of the Pit | MustCost and source exclusion; cannot lower to this activation-only contract |
| Elder Spawn; Yawgmoth Demon | Optional sacrifice payment; Elder Spawn also has blocking rule |
| Sacrifice; Metamorphosis; Goblin Grenade | Additional casting costs; first two also need paid result/dynamic mana; Metamorphosis has spend restriction |
| Serendib Djinn | Resolving sacrifice/result condition and state trigger |
| Mana Vortex | FromStack, optional and player resolving sacrifices, state trigger |
| Land Equilibrium | Entry replacement, affected player and relative land-count condition |
| Transmute Artifact | MustCost, paid result, library search, comparison, variable payment and zone placement |

No source data changed. Known Gaea's Liege/Fastbond concerns remain open.
The 12 RON source hashes match the vocabulary snapshot; no refresh is needed.

## Gameplay evidence

| Cards or context | Tests and coverage |
| --- | --- |
| All nine direct names, all rules | TestRONSacrificeCostRealCards: full-record compilation/definition validation, activating controller versus owner, choice, immediate cost versus delayed effects, owner graveyard, regeneration cannot prevent sacrifice, tap cost, pending clone, source departure/return/control/phasing, source bonus/cleanup, life/draw player, targeted shield/destruction/damage and immediate mana |
| Fallen Angel; Goblin Chirurgeon | TestRONSacrificeCostSourceAndTargets: self-sacrifice, self-target paid away, returned source/target, target departure/phasing/type legality |
| Fallen Angel | TestRONSacrificeCostInstancesAndCopy: independent card factories/permanents, copied pending boost pays no second cost, original source binding, clone resolution and cleanup isolation |
| Gate to Phyrexia | TestRONSacrificeCostGateTimingAndLimits: wrong turn/step/target consumes no creature/use; successful own-upkeep use; once-per-turn; resolution after timing changes |
| Orcish Mechanics | TestRONSacrificeCostDamageReplacement: source departure LKI and damage prevention pipeline |
| Ashnod's Altar | TestRONSacrificeCostManualManaAndQueries: explicit immediate mana, pure planner query boundary, usable produced mana, no reuse of paid creature, clone isolation |
| Shared cost context | TestRONSacrificeCostRollbackAndChoice: choice once, mana/tap/sacrifice composition, double-cost failure, missing mana, phased/opposing/wrong-type candidate and invalid player choice, live-state rollback and clone preservation |
| Shared filter/inventory | TestRONSacrificeCostCompositionAndInventory: nested type/color/subtype conjunction/disjunction, fresh factories, operand role |
| Invalid declarations | TestRONSacrificeCostInvalid: wrong kind/arity/reference, empty nested filters, source-relative filter, action/optional-payment contexts and unbound result rejection |

Gameplay tests initially failed on the missing SacrificeAPermanent adapter.
Test fixture fixes used ATQ for Ornithopter, accounted for the source card in
its owner's hand after bounce, used phased-inclusive existence checks, and set
owners for destruction. The new planner test initially assumed automatic
sacrifice planning; engine/API inspection showed that this is an existing
explicit planning exclusion, so the test now asserts the actual boundary and
usable mana after explicit activation. No production rule was bypassed or
fallback registry added.

## Measurement and validation

Expected and actual gain agree: nine unique names, ten records (Atog has two
reprints here). Strict loader: 1,005 -> 1,015 compiled of 2,445; rejected:
1,440 -> 1,430. Accepted unique names: 310 -> 319; rejected names: 676 -> 667.
Zero accepted-record regressions. Strict/full-inventory diagnostics match.
Nodes/signatures: 72,938/8,528 unchanged. Unknown-role occurrences:
17,852 -> 17,801. Global engine_validated and behavior_verified remain null;
nine additional names have complete-record gameplay evidence, independent of
collection compilation counts.

Validation command results are recorded in the progress ledger. Strict loading
still exits 1 and publishes no partial registry. Removal gates remain closed;
production startup, Go definitions/generators and DSL exports retain callers.
No unused-DSL deletion audit or runtime cutover is claimed.

Ranked next candidates: (1) fixed counter placement, now also directly unlocking
Dwarven Weaponsmith alongside the other counter cards, with fresh unusual-counter
engine analysis; (2) typed sacrifice characteristic results, concretely Diamond
Valley/Life Chisel and Priest of Yawgmoth after dynamic mana support; (3) variable
optional mana payment, In the Eye of Chaos directly and Invoke after spell filters.
Reassess full records next run. No second slice starts here.
