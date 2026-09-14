# Composable DSL: Existing Engine and Follow-up Audit

This audit separates existing engine services from DSL adapters and engine
issues. It is not an engine replacement plan. Findings describe the working
tree inspected during this review; passing focused tests does not establish
complete rules coverage.

## Existing Services to Reuse

| Service | Implementation | Evidence inspected and run | DSL work |
| --- | --- | --- | --- |
| Payment transactions | `pkg/mage/payment_transaction.go` | `TestPayAttackCosts_AggregatePaymentIsAtomic`, `TestActivateAbility_TotalCostFailureIsAtomic`, `TestCostPaymentValidationDoesNotConsumeManaAbilityActivationLimit` | Adapt typed costs and payment bindings to the existing transaction. |
| Replacement and prevention | `EffectManager.ApplyReplacements` in `pkg/mage/effect_manager.go`, `pkg/mage/replacement.go` | `TestReplacementPipelineDeduplication`, `TestMultipleReplacementsInteract`, prevention shield partial/consumption/expiry tests | Wrap existing actions and replacement effects; preserve rechecks, deduplication, and shields. |
| Delayed triggers and pending storage | `pkg/mage/trigger_system.go`, `Game.RegisterDelayedTrigger` | `TestTriggerSystem_DelayedTriggers`, `TestTriggerSystem_CloneIsolation`, `TestCR603_7a_TriggeredDelayedCreatedOnResolution` | Provide typed constructors and captured bindings. |
| Spell copies and retargeting | `pkg/mage/spell_copy.go` | `TestCopySpellOnStack_SameTargets`, `TestCopySpellOnStack_NewTargets`, copy resolution and fizzle tests | Adapt binding frames to existing copy paths. |
| Layers and durations | `pkg/mage/effect_manager.go`, `pkg/mage/continuous_effects.go` | `TestControlEffectsUseTimestampOrder`, `TestConditionalControlExpiresPermanently`, `TestCR613_*` | Map modifications to existing layer effects and duration handling. |
| Non-target choices and library operations | `pkg/mage/reveal.go` and player choice methods | `TestRevealAndPickFromTop_MayDeclineHonored`, `TestRevealAndPickFromTop_MayDeclineFalseForcesPick`, reveal/pick/order tests | Replace string or positional plumbing with typed choice results. |
| Clone isolation | `pkg/mage/clone.go`, trigger/effect clone methods | `TestTriggerSystem_CloneIsolation`, `TestCloneWithReplacementEffects` | Extend existing clone hooks for new mutable binding storage. |

The focused command passed for both packages:

```sh
go test ./pkg/mage ./pkg/mage/gametest -run 'Test(TriggerSystem_|DiscardHand|PayAttackCosts_|ActivateAbility_TotalCostFailureIsAtomic|CostPaymentValidation|CopySpellOnStack_|Reveal|PickFromHand|PutOnTopInChosenOrder|ControlEffectsUseTimestampOrder|ConditionalControlExpiresPermanently|CloneWithReplacementEffects|ReplacementPipelineDeduplication|MultipleReplacementsInteract|PreventionShield|CR603_7a|CR603_10a|CR613_)' -count=1
```

## Confirmed Transient State-Trigger Gap

`Game.FireEvent` explicitly skips abilities for which `IsStateTrigger()` is
true. The only production caller of `CheckStateTriggers` is the priority loop
in `pkg/mage/priority.go`. Discarding and drawing emit events but do not check
state triggers between instructions.

A temporary test used the existing engine APIs:

1. Give A one card in hand and one in the library.
2. Put a creature with `NewStateTriggered` on the battlefield. Its condition
   checks that A's hand is empty.
3. Run `CheckStateTriggers` while the hand is nonempty.
4. Resolve `CompositeEffects` containing `DiscardHand()` and
   `DrawCards(Fixed(1))` through `ApplyEffect`.
5. Run `CheckStateTriggers` again. The hand has one card; expect one pending
   trigger for the transient empty hand.
6. Repeat with `NewTriggered(EvtDiscard, ...)` and the same condition as a
   control case.

Observed: the state case failed with zero pending triggers; the event case
passed with one. The probe was removed after the audit. The event case proves
that event dispatch during resolution works; it is not a replacement for a
state trigger, which must also detect non-discard changes.

Fix this as a focused engine change to existing observation and pending-trigger
paths. Add a permanent failing regression before implementation. Do not add a
second trigger system or infer that every action needs a new executor.

## Other Findings and Limits

- **State-trigger rearming: confirmed by code inspection.**
  `Game.CheckStateTriggers` arms until the condition becomes false. The existing
  `TestTriggerSystem_StateTriggers` tests the bookkeeping methods, not rearming
  after resolution or countering. Correct this in the existing trigger system.
- **Sacrifice actor: confirmed by code inspection.** `SacrificeSource.Apply`
  passes the found permanent to `DoSacrifice` without checking the resolving
  controller. `DoSacrifice` uses the permanent's current controller. Add the
  control-change regression and correct the existing effect path.
- **Identity: infrastructure already exists.** `NewPermanent` assigns a new
  `incarnationID`; color effects already use it. `Permanent.ID()` still returns
  the card ID. Typed references can capture and validate the existing incarnation
  internally. Renaming public IDs and introducing zone-wide object records are
  separate architecture proposals, not prerequisites for every DSL slice.
- **Replacement ordering: partial implementation.** `ApplyReplacements` loops,
  rechecks applicability, and tracks applied instances. Its inspected selection
  path uses list order, with non-prevention effects first; it does not ask the
  affected player to select between applicable rules. Full CR 616 ordering
  needs a focused scenario audit and regression, not another pipeline.
- **LKI and simultaneous actions: partial evidence.** Existing zone-change and
  sacrifice paths capture LKI and dispatch self-triggers. The passing
  `TestCR603_10a_TriggeredLooksBackInTimeLTB` only asserts that a permanent reached
  the graveyard; it does not assert the trigger's result. Do not use that test
  as proof of general simultaneous-death behavior. Strengthen relevant existing
  tests before deciding whether the current action paths need batching changes.
- **Continuous effects: existing layer behavior is tested.** Full dependency
  loops and every static-versus-resolving capture case were not verified here.
  Extend existing effect-manager tests for a concrete failing case before
  scheduling new engine machinery.
- **Other mechanics: not exhaustively audited.** Visibility, reflexive triggers,
  linked state, all-zone identity, and every casting/copy variant require
  caller-specific inspection. A missing DSL constructor does not prove missing
  rules support.

## Follow-up Policy

For each migrating family, identify the existing implementation and tests,
then classify the change as an adapter, a missing test, or a reproduced engine
bug. Reuse passing tests. Extend weak tests. Add new tests only for uncovered
behavior or adapter-specific risks. Engine fixes should have separate changes
and acceptance criteria; block only the DSL callers that depend on the fix.

## Engine Fixes

The follow-up implementation fixes the reproduced engine failures:

- `TestAuditStateTriggerTransient` preserves the empty-hand trigger between
  discard and draw. Event dispatch and effect-sequence boundaries observe state
  without placing triggers on the stack during resolution.
- `TestAuditStateTriggerLifecycle` verifies suppression until resolution,
  countering, or removal from the stack. `TestAuditStateTriggerCloneAndReentry`
  verifies independent cloned state and a new battlefield incarnation.
  `TestAuditStateTriggerRearmsBeforeStateBasedActions` verifies rearming before
  lethal damage removes the source.
- `TestAuditSacrificeSourceController` verifies that both source-sacrifice
  effects reject a sacrifice by the former controller.
- `TestAuditReplacementAffectedPlayerChooses` and the expanded
  `TestCR616_2_ReplacementInteractionBecomesApplicable` verify player-selected
  replacement order, including prevention before or after damage doubling.
  The existing action pipeline rechecks applicability and deduplicates effects.
  Entry replacements and self-replacement priority classes are not represented
  by its current action types; this change does not claim support for them.
- The strengthened `TestCR603_10a_TriggeredLooksBackInTimeLTB` asserts life gained
  for both deaths. Its creature fixture now has positive toughness so it survives
  until the board wipe. `TestAuditSimultaneousDeathTriggers` also covers lethal
  damage and zero toughness. `DestroyPermanents` and the state-based death group
  preserve pre-event information and defer zone-change trigger dispatch until
  all members have moved.

The identity, dependency-loop, visibility, reflexive-trigger, linked-state, and
casting/copy observations above remain audit limits, not reproduced engine bugs.
Jihad now uses `NewStateTriggered` for its existing state condition. The test
harness adds starting permanents without observing incomplete setup states.
No other card implementation or DSL migration is included in these fixes.

Verification passed after the fixes:

```sh
go test ./pkg/mage/... -short -count=1
go test ./... -short -count=1
go build ./...
go vet ./...
make lint
```

Lint reported zero issues. Long simulations were not run.
