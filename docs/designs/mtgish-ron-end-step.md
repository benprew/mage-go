# Ordinary RON end-step triggers

Capability: P6-END-STEP. Completed 2026-10-04. This run implements one
ordinary end-step event context. It does not implement delayed step triggers.

## Selection evidence

The baseline working tree had user changes to `.gitignore` and
`mtgish-ron-execution.md`. Both are preserved. Baseline short engine tests and
build passed. Strict loading and semantic inspection both exited 1 with
1,040 compiled and 1,405 rejected records, from 2,445 records / 986 names.
The existing ledger, semantic inventory and instance-scope prerequisites were
present. Inventory was refreshed through the AST-based inspection command.
The vocabulary snapshot was read as syntax evidence, not support evidence.

The following candidates were compared before editing. Full records, including
nested forms, were inspected in `cards/mtgish`; Go definitions were not used.
Counts refer to distinct names, with reprints counted once.

| Candidate | Affected unique cards | Full-card estimate | Other blockers | Downstream benefit | Effort |
| --- | --- | ---: | --- | --- | --- |
| Ordinary end-step event | Ball Lightning; Breeding Pit; Erg Raiders; Khabál Ghoul; Osai Vultures; Pestilence; Scavenging Ghoul; Season of the Witch; Voodoo Doll; Whirling Dervish | 2 | See per-card table below | Typed step-event player references; ordinary boundary support needed by later histories and intervening-if adapters | Small: one event adapter, typed scope, inventory role; broad contract tests |
| Storage-land lifecycle | Bottomless Vault; Dwarven Hold; Hollow Trees; Icatian Store; Sand Silos | 5 if all three prerequisites are implemented | Entry tapped, optional untap and TriggerI tapped-source condition; activation/counter/mana portion is supported | Reusable entry operations, untap permissions and intervening-if checks | Moderate to large: three adapter contexts and lifecycle/choice tests |
| Saved-sacrifice repeated mana | Priest of Yawgmoth; Sacrifice; Metamorphosis | 1 | Sacrifice needs spell additional-cost/result scope; Metamorphosis also needs Plus and creature-spell restricted mana | Shared paid-value mana consumer for the two spells | Small consumer adapter, moderate saved-result and solver tests; spell contexts substantially larger |

Ordinary end-step events win on two complete cards with one small adapter and
existing engine services. The five-card storage candidate requires three
contexts. Saved-sacrifice mana has only one immediate unlock. Delayed next-end-step
work, including Rukh Egg, was examined as an extension but remains outside this
ordinary-event slice. The contract, reuse boundary and acceptance cases were
stated to the user before editing.

## Exact declaration and execution boundary

Supported permanent-source rule:

```
TriggerA(AtTheBeginningOfAPlayersEndStep(AnyPlayer), actions)
TriggerA(AtTheBeginningOfAPlayersEndStep(SinglePlayer(You)), actions)
```

`TriggerA/2`, `AtTheBeginningOfAPlayersEndStep/1`, `SinglePlayer/1`, and
symbol-shaped AnyPlayer/You are validated before a factory is produced.
`actions` retains existing ActionList/1 and single-group Targeted/2 support.
Symbol `Trigger_ThatPlayer` is a typed player reference inside this event body.
It is valid in existing player consumers, including GainLife/1, LoseLife/1,
DrawACard, DrawNumberCards/1, DiscardACard, DiscardNumberCards/1 and optional
fixed-mana PlayerMayCost/2. New life consumers use the explicit step player;
other existing player consumer restrictions are preserved.

The private adapter signature is:

```
compileEndStepTrigger(event, body Node, ctx scope) (optionFactory, error)
```

`stepEventPlan{playerName string}` provides immutable scope identity. Each
factory creates a fresh `*mage.PlayerBinding`. `instanceBindings.steps` maps
that plan to the binding. `playerReferencePlan.step` reads it through FromPlayer.
Existing `compilePlayerRef`, `compilePlayerEffect` and `actionPlan.buildWithBindings`
retain their public/private caller signatures. Target declarations are built
in the same factory as effects and event bindings.

Engine reuse: EachEndStep, YourEndStep, TurnStepBegins, WhenEvent, BindPlayer,
FromPlayer, Sequence, the pending-trigger/APNAP path, typed target declarations,
stack copies, Game.Clone, sacrifice, token entry, costs, life, draw and discard.
No engine implementation or API changed. RON owns spelling and declaration
scope. No source data was changed. No global last-action slot, card-name branch,
fallback, declaration panic or partial production registry was introduced.

At event time, YourEndStep checks the current controller. The queued ability
keeps that controller after source changes. Trigger_ThatPlayer keeps the event
player even if a copy has a different controller. A phased source does not
trigger. A pending ability can resolve after departure or phasing. ThisPermanent
keeps its incarnation; it cannot follow a returned card. A player cannot
sacrifice a source now controlled by someone else. The engine selects typed
targets at placement and checks legality again at resolution.

Rules read: local CR 513.1–2, 603.1–5 and 701.21a, with the existing identity,
phasing and target contracts in engine documentation (CR 400.7, 608.2b).
No new replacement operation, LKI query or simultaneous-operation service was
needed. Existing token/zone services remain the execution boundary.

## Complete-record inventory and remaining blockers

| Card | Result or remaining blockers |
| --- | --- |
| Ball Lightning | Fully compiled and tested: Trample, Haste, each-end-step source sacrifice; printed mana and 6/1 characteristics |
| Breeding Pit | Fully compiled and tested: controller-upkeep optional BB payment/unpaid sacrifice; controller-end-step 0/1 black Thrull creation |
| Khabál Ghoul | NumCreaturesOrPlaneswalkersThatDiedThisTurn in counter amount remains unsupported |
| Scavenging Ghoul | Same death-history value remains unsupported; fixed corpse-counter regeneration cost is already supported |
| Season of the Witch | Upkeep PayLife optional payment and end-step DidntAttackThisTurn/CouldntAttackThisTurn/ExceptFor filters remain unsupported |
| Erg Raiders | TriggerI, DidntAttackThisTurn and control/turn-history condition remain unsupported |
| Osai Vultures | TriggerI and death-history condition remain unsupported; its paid temporary source boost is already supported |
| Pestilence | TriggerI/no-creatures condition and MultipleRecipients damage remain unsupported |
| Voodoo Doll | TriggerI, pin-counter totals, activation PayManaX and XIs remain unsupported |
| Whirling Dervish | TriggerI and opponent-damage history remain unsupported |

Hazezon Tamar needs future controller-upkeep, land-count/controller filters,
leave-event and global exile support. Rukh Egg needs CreateFutureTrigger at the
next end step. Neither is counted as an unlock for this slice.

## Test mapping and TDD

| Card / contract | Tests |
| --- | --- |
| Ball Lightning | TestRONEndStepBallLightning: full record and definition validation, either player's boundary, source return/control/phasing, countered trigger, owner graveyard, clone isolation |
| Breeding Pit | TestRONEndStepBreedingPit: full record and definition validation; opponent versus controller steps; upkeep pay/decline/short payment; upkeep/end-step interaction; source departure/return/control-before-placement/phasing; copied controller; exact Thrull characteristics and owner |
| Both complete cards | TestRONEndStepNativeTurnAndCombat: failed casting rollback, real mana costs/casting, haste attack eligibility, trample combat damage, normal RunStepWithPriority dispatch/resolution, late entry, cleanup distinction and next opponent boundary |
| Current controller / phasing | TestRONEndStepCurrentController: YourEndStep follows controller rather than owner and resumes after phasing |
| Typed event instances | TestRONEndStepEventBindings: separate factories, multiple events before resolution, source departure, copied controller, preserved event player, clone isolation |
| Player consumers | TestRONEndStepExplicitPlayerConsumers: opponent event player receives life/draw/discard; event player is not a target |
| Targets | TestRONEndStepTargetsAndPhaseMembership: placement choices, legal tap plus event-player effect, departed/returned/phased target rejection, phased-source exclusion |
| Invalid declarations | TestRONEndStepInvalidDeclarations: wrong arity, tuple symbols, opponent/host filters, incompatible permanent/spell read, unbound upkeep event read, cross-rule escape and unsupported TriggerI |
| Semantic reporting | TestRONEndStepInventory; updated TestInventoryRepositoryCollection/TestInventoryLeavesRealRONGameplayAndRegistryUnchanged |

The initial focused run failed on unsupported AtTheBeginningOfAPlayersEndStep
and missing inventory operand roles. The implementation then exposed fixture
assumptions: RunStepWithPriority resolves passed-priority triggers, synthetic
cards need owners for destruction, and a pending sacrifice cannot sacrifice a
source after control changes. These were corrected to the documented engine/CR
contracts. Explicit opponent-player life tests exposed a missing step-player
consumer adapter; life actions now use that typed player. No expected gameplay
requirement was removed. All focused tests pass.

## Fresh measurements and validation

Estimated gain equals actual gain: **two names and two records**.
Compiled **1,040 -> 1,042**; rejected **1,405 -> 1,403**.
Accepted unique names **338 -> 340**; rejected names **648 -> 646**.
The only newly accepted records are DRK Ball Lightning and FEM Breeding Pit.
There are zero regressions among previously accepted records. Strict and
inspection diagnostic arrays match.

Semantic signatures remain **8,528**, AST occurrences **72,938**. Unknown-role
occurrences **16,599 -> 16,588**. All twelve RON hashes match the syntax snapshot;
no vocabulary source refresh is needed. Collection-wide engine_validated and
behavior_verified remain null. Two complete cards gained RON gameplay and
explicit definition-validation evidence; compilation is not global gameplay
coverage.

Remaining leading diagnostics: PermanentRuleEffect 110; AsPermanentEnters 90;
If 65 across contexts; CastEffect 54 across contexts; ChooseADamageSource 46;
Targeted_Modal 42; HostController 35; ControlledByAPlayer 29; TriggerI 28;
TargetGraveyardCard 26. No source defect was corrected. Previously recorded
Gaea's Liege and Fastbond concerns remain open.

Final command outcomes are recorded in the progress ledger. Go-card definitions,
generators and DSL callers are preserved: the runtime/removal gates remain
closed. No DSL function was removed, so no deletion caller audit is claimed.
No second slice was started.

Next candidates to reassess: storage-land lifecycle prerequisites (five names),
shared death-history values for Khabál Ghoul/Scavenging Ghoul with intervening-if
support downstream for Osai Vultures, then delayed next-end-step capture for
Rukh Egg and the other nested future-trigger users. Compare smaller entry-only
shapes and full-record gains before committing to the whole storage lifecycle.
