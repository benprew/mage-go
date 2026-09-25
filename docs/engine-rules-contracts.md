# Rules Engine Contracts and Subsystems Architecture

This document specifies the internal architecture, subsystem invariants, and Magic: The Gathering Comprehensive Rules (CR) execution contracts for `mage-go`.

---

## Table of Contents

1. [Subsystem Architecture and State Ownership](#1-subsystem-architecture-and-state-ownership)
2. [Object Identity, Incarnations, and Phasing (CR 400.7)](#2-object-identity-incarnations-and-phasing-cr-4007)
3. [Controllers and Layer 2 (CR 613.1b, 613.7)](#3-controllers-and-layer-2-cr-6131b-6137)
4. [Continuous Layers and Effect Evaluation (CR 611–613)](#4-continuous-layers-and-effect-evaluation-cr-611613)
5. [State-Triggered Abilities and Simultaneous Deaths (CR 603.8, 603.6c)](#5-state-triggered-abilities-and-simultaneous-deaths-cr-6038-6036c)
6. [Replacement and Prevention Pipeline (CR 614–616)](#6-replacement-and-prevention-pipeline-cr-614616)
7. [Damage Execution and Nonpositive Outcomes (CR 120, 120.8)](#7-damage-execution-and-nonpositive-outcomes-cr-120-1208)
8. [Life Payments and Payment Boundaries (CR 119.4)](#8-life-payments-and-payment-boundaries-cr-1194)
9. [Total-Cost Payment Transactions (CR 601.2, 602.2)](#9-total-cost-payment-transactions-cr-6012-6022)
10. [Turn Structure, Cleanup Priority, and History Retention (CR 514)](#10-turn-structure-cleanup-priority-and-history-retention-cr-514)
11. [Mana System and Immediate Resolution (CR 605)](#11-mana-system-and-immediate-resolution-cr-605)
12. [One-Shot Untap Skipping](#12-one-shot-untap-skipping)
13. [Ante and Ownership Operations](#13-ante-and-ownership-operations)

---

## 1. Subsystem Architecture and State Ownership

`Game` serves as the aggregate root and coordinator of the engine. Subsystems own distinct state invariants and cloning behavior:

* **`ZoneSystem`**: Owns battlefield storage with copy-on-write (CoW) semantics for AI search, exile storage with visibility metadata, entering-permanent tracking, and Last-Known Information (LKI) snapshots.
* **`TurnSystem`**: Owns turn counters, current phase and step, active player index, extra turns, `TurnSchedule` lifecycle, and one-shot untap restrictions.
* **`ResolutionState`**: Owns transient resolution scratch state including X values, chosen modes, event source/amount context, cast-time snapshots (`CastContext`), targets, damage/counter distributions, and cost payment reveals/sacrifices.
* **`TriggerSystem`**: Owns pending triggers waiting for stack placement, active `DelayedTrigger` instances, and armed state triggers (CR 603.8).
* **`ManaSystem`**: Owns mana source discovery, mana ability solver planning, and payment scratch buffers.
* **`DamageSystem`**: Owns damage execution, combat damage step aggregation, damage history tracking, and damage reflection.
* **`TrackerSystem`**: Owns turn-scoped observations (`TurnTrackers`) and duel-scoped observations (`DuelTrackers`).
* **`RandomSource`**: Owns deterministic integer and coin flip queues for tests and AI simulation.

Read-only queries are organized into focused domain interfaces: `PlayerReader`, `BattlefieldReader`, `ResolutionReader`, `TurnReader`, `CombatReader`, `TrackerReader`, and `StackReader`, which compose into `GameReader`.

---

## 2. Object Identity, Incarnations, and Phasing (CR 400.7)

* **Physical Card vs. Rules Object**: `Permanent.ID()` returns the physical card UUID, but `Permanent.incarnationID` tracks the unique existence of that permanent on the battlefield.
* **Battlefield References**: Resolving effects (such as `Sacrifice(This)` or `Untap(This)`) resolve and verify the specific captured battlefield incarnation.
* **Leave-and-Return**: When a permanent leaves the battlefield (to exile, graveyard, or hand) and returns, it becomes an entirely new object with a new `incarnationID`. A delayed trigger, continuous effect, or spell targeting the old incarnation will not affect the returned card.
* **Phasing**: Phasing out does **not** change `incarnationID` and does not cause leave-the-battlefield or enters-the-battlefield triggers. Phased-out permanents are ignored for game actions, but their incarnation is preserved and resumes when they phase in.
* **Last-Known Information (LKI)**: When an object moves zones or dies, its characteristics immediately prior to the event are recorded. Effects reading source power, attachment status, or colors read this LKI snapshot (CR 608.2h).

---

## 3. Controllers and Layer 2 (CR 613.1b, 613.7)

* A `Permanent` has a `defaultController` and a computed controller. `NewPermanent` initializes both from the controller specified when put onto the battlefield (CR 110.2a).
* `Permanent.ControllerID()` always returns the computed controller. Card code must **never** mutate controller fields directly.
* `EffectManager.Apply` resets computed controllers to their default and applies continuous control-changing effects in **Layer 2** in timestamp order.
* A true control change occurs only when the Layer 2 computed controller differs from the previous controller. Real changes update ability context, attack permissions, and summoning sickness (CR 302.6).
* Triggers retain the controller recorded at trigger creation (CR 603.3a). Triggers on leaving the battlefield use LKI controller (CR 603.6c).
* `ExchangeControl` records the current controllers of two permanents and creates indefinite Layer 2 effects for both. If either permanent leaves, shares a controller, or cannot change control, the exchange aborts.

---

## 4. Continuous Layers and Effect Evaluation (CR 611–613)

Continuous effects are evaluated by `EffectManager.Apply` across the official 7-layer system:

* **Layer 1**: Copy effects and mutable card characteristics.
* **Layer 2**: Control-changing effects (timestamp order).
* **Layer 3**: Text-changing effects.
* **Layer 4**: Type-changing effects:
  * `SetSubtypeFamily`: Replaces a single subtype family (e.g. creature types), preserving other types/subtypes.
  * Setting a land to a basic land type removes printed abilities and intrinsic mana abilities, granting the intrinsic mana ability of the new type (CR 305.7).
* **Layer 5**: Color-changing effects (`SetColors`, `SetEachColors`). Empty color list renders the object colorless.
* **Layer 6**: Ability adding and ability removing:
  * `GrantKeyword` and `GrantActivatedAbility` add abilities.
  * Typed `RemoveKeyword` uses the existing `RevokeAttr` service to remove all instances of one keyword. Later grants restore the keyword in timestamp order. Temporary removal captures the recipient incarnation, survives source departure and recipient phasing, and expires independently in cloned games. See `TestComposableKeywordRemoval` and `TestComposableKeywordRemovalLifetime`.
  * Typed `RemoveLandwalk` uses the same removal service for all registered subtype landwalk attributes and `LegendaryLandwalk`. The subtype map alone does not include legendary landwalk. Legacy `RevokeLandwalk` shares the complete list. See `TestRevokeLandwalkIncludesLegendary` and `TestComposableLandwalkRemoval`.
  * `RemoveAllAbilities` strips abilities computed up to its timestamp. Later Layer 6 timestamps can re-grant abilities.
  * `CantRegenerate` represents a regeneration prohibition, not an ability. Ability removal preserves it, regardless of timestamp. The existing attribute storage and duration manager retain the prohibition until expiry (CR 611.2a, 701.19c).
* **Layer 7**: Power and toughness modifications:
  * 7a: Characteristic-defining abilities (CDAs).
  * 7b: Setting base P/T to specific values.
  * 7c: Modifications (+N/+N, -N/-N).
  * 7d: P/T counters (+1/+1, -1/-1).
  * 7e: P/T switching.

### Blocking Restrictions

`CantBeBlocked` uses the existing `AddCantBeBlockedExceptBy` rule storage with
an empty allowed set. The composed effect captures the subject's incarnation,
then reinstalls the rule during each continuous-effect pass. Ability removal
does not erase this rule, and the rule applies to later blockers (CR 509.1b,
611.2a–c). Printed `UnblockableKW` abilities remain removable. The legacy
`MakeUnblockableUntilEndOfTurn` executor uses the same captured restriction.

### Dynamic vs. Frozen Membership
* **Temporary Effects (`Until`)**: Effects modifying characteristics or control capture affected objects **at resolution time** (CR 611.2c). Permanents entering the battlefield later are not included.
* **Static Effects (`While`, `Static`)**: Static continuous effects continuously recalculate their matching set (CR 611.3a). Permanents entering later receive the boost or ability dynamically.

---

## 5. State-Triggered Abilities and Simultaneous Deaths (CR 603.8, 603.6c)

* **State Triggers (`WhenState`)**:
  * Trigger immediately when their pure `StateCondition` evaluates to true, including between instructions during resolution.
  * A trigger stays **suppressed** while its original instance is pending, on the stack, or resolving.
  * Once that instance leaves the stack (resolved, countered, or removed), the ability rearms and triggers again if the condition is still met.
  * A permanent that leaves and returns has a new instance and can trigger independently.
* **Simultaneous Batch Observation**:
  * A union of destruction filters uses one `DestroyEach(MatchingPermanents(Or(...)))` action. Each matching permanent is considered once, so one regeneration shield suffices even if multiple types match. State triggers cannot observe separate type batches.
  * Mass destruction (`DestroyPermanents`) preserves the pre-destruction battlefield snapshot for leave-the-battlefield triggers.
  * Singular `DestroyWithoutRegeneration` and legacy positional/string-bound no-regeneration executors use a one-permanent batch with the same action-scoped prohibition.
  * `DestroyEachWithoutRegeneration` uses the same batch with an action-scoped regeneration prohibition (CR 701.8, 701.19c). It does not add keywords to permanents or their last-known characteristics. Indestructible and other destruction replacements still apply.
  * Zero-toughness state-based deaths share the same observation batch.
  * Conditions evaluate against the pre-batch state; trigger effects resolve against the current game state.

---

## 6. Replacement and Prevention Pipeline (CR 614–616)

* **Replacement Applicability**:
  * A replacement effect watches a proposed game action (not an event that already occurred) and transforms it into an altered action or prevention.
  * When multiple replacements apply to the same action, the affected player (or controller of the affected permanent) chooses which replacement applies (CR 616.1).
  * Applicability is rechecked after each transformation. Each replacement instance applies at most once to an event (CR 614.5).
* **Damage Prevention Shields**:
  * Prevention shields consume damage capacity up to their limit. Excess damage passes through.
  * Prevention removes damage from being dealt; it does not heal damage that was already dealt.
* **Static Whole-Step Replacements (CR 614.10b)**:
  * Fasting-style step replacements skip the entire draw step (suppressing normal card draw, step events, and priority).
  * The resulting action (e.g. gaining 2 life) occurs first in the next step that actually occurs.
  * `Static(SkipDrawStepForLife(amount))` follows the source's current controller and ability availability. Phasing disables the rule; return to the battlefield creates a new rule for the new incarnation.
  * Pending gains keep the player who skipped the step through control changes, source departure, and game cloning. They use the event-aware life gain executor and report actual replacement outcomes.
  * Low-level explicit replacement records remain tied to their captured incarnation. A record with no incarnation cannot become active when the card enters later. These records are executor test fixtures, not the card-author API.

---

## 7. Damage Execution and Nonpositive Outcomes (CR 120, 120.8)

* **Nonpositive Damage Cutoff (CR 120.8)**:
  * If a damage action's final amount is reduced to zero or less by prevention or modifiers, the damage action terminates before further replacement checks or execution.
  * Zero/negative damage produces **no damage event, damage history, callback, or marked damage**.
  * Later replacements cannot resurrect or increase an absent damage event.
* **Lifelink Outcome Routing**:
  * Damage from a permanent with lifelink causes its controller to gain that much life simultaneously with damage execution.
  * Life gained from lifelink routes through the event-aware life gain executor, emitting an `EvtLifeGained` with the actual positive amount after life-gain replacements.
* **Combat Damage Assignment**:
  * Attacking creatures assign lethal damage to each blocker in damage assignment order before assigning damage to the next blocker.
  * Trample assigns excess damage beyond lethal toughness to the defending player/planeswalker.

---

## 8. Life Payments and Payment Boundaries (CR 119.4)

* **Exact Life Total Payment**:
  * Under CR 119.4, a player may pay life equal to their current life total (reducing their life total to 0).
  * A player with negative or zero life may pay 0 life.
  * A player may never pay a negative amount of life, nor pay more life than their current life total.
* **Transaction Safety**:
  * Life costs are verified on an isolated clone before committing.
  * Canceled or failed costs leave live life totals and pending events untouched.
  * Positive life payments emit `EvtLifeLost` through the shared executor; zero payments emit no event.

---

## 9. Total-Cost Payment Transactions (CR 601.2, 602.2)

All casting and activations follow the locked total-cost transaction model:
1. Announce modes, targets, X, and cost choices.
2. Determine total cost: $\text{Base Cost} + \text{Additional/Alternative Costs} - \text{Reductions}$.
3. Lock total cost in a transaction context.
4. Auto-tap planner and manual mana activations produce required mana. Sources involved in the primary action (e.g. attacking creature or source with {T} cost) are reserved to prevent self-consumption.
5. Verify and pay all costs atomically on an isolated clone.
6. `Commit` updates the live game. If payment fails or is aborted, all resources rollback cleanly.

---

## 10. Turn Structure, Cleanup Priority, and History Retention (CR 514)

* **Cleanup Step Flow**:
  1. The active player discards to hand size limit (CR 514.1).
  2. All damage is removed from permanents, including phased-out permanents, and temporary "until end of turn" effects expire simultaneously (CR 514.2).
  3. State-based actions are checked. If SBAs occur or abilities trigger, priority is given to players (CR 514.3a).
* **Repeated Cleanup Priority**:
  * If priority occurs during cleanup, another cleanup step is created afterward.
  * `TurnTrackers` (including creature death counts `CreaturesDiedThisTurn`), attacker history, and activation limits **persist across repeated cleanup passes**.
  * Only the **final cleanup pass** (where no SBAs occur and the stack is empty) resets turn history and activation limits.

---

## 11. Mana System and Immediate Resolution (CR 605)

* **Definition of Mana Ability (CR 605.1a, 605.3b)**:
  * An activated ability is a mana ability if it doesn't require a target, could put mana into a player's pool, and isn't a loyalty ability.
  * A tap cost is **not** strictly required by the rules. Targetless activations with unconditional mana instructions resolve **immediately without using the stack**.
* **Tap Transitions (CR 603.2e, 701.26a)**:
  * `TapPermanent` emits `EvtTapped` only when an untapped permanent becomes tapped. Repeated calls on a tapped permanent do not trigger abilities.
  * `AttachedBecomesTapped` captures the event permanent at detection through the existing trigger capture path. Pending triggers, copies, and delayed effects keep that incarnation independently of later attachment changes.
* **Tapped-for-Mana Events**:
  * `EvtTappedForMana` fires only when a mana ability having a {T} cost produces mana (CR 106.12a). Normal tap actions without mana produce only `EvtTapped`.
* **Typed Mana Bonuses**:
  * Explicit typed mana activations retain the actual produced colors through resolution and use the shared mana-bonus service once after a tap-cost ability produces a positive amount. Zero production and abilities without a tap cost receive no tap bonus.
  * Production totals exclude bonus mana. Nested resolutions restore their parent's tracking state, and clones copy the color list. Planned activation keeps its existing bonus path.
  * Evidence: `TestComposableManaBonuses`, `TestComposableManaBonusUsesProducedColor`, and `TestTypedManaAmountNestedResolution`.
* **Automatic Solver**:
  * Plans mana abilities, handles color permutations, spending restrictions, and bonus mana distributions.
  * Excludes abilities with targets or mixed non-mana effects from automatic planning.
  * Ordinary activation choices omit typed tap-only mana production handled by automatic payment. Utility abilities and sacrifice mana remain explicit choices (`TestComposableManaOrdinaryActivationChoices`).

---

## 12. One-Shot Untap Skipping

* `SkipNextUntap(PermanentRef)`: Prevents untap during the controller's next untap step.
  * Does not tap the permanent or prevent other untap effects.
  * Repeated applications before that step expire together.
  * If the permanent changes controller, the restriction follows to the new controller's untap step.
* `SkipNextUntapOf(PermanentRef, PlayerRef)`: Restricts untapping during a specific player's untap step (e.g. "your next untap step").
  * Captures the specified player at resolution. Control changes do not alter the target untap step.
  * Skips do not consume if the step itself is skipped.

---

## 13. Ante and Ownership Operations

* **Ante Zone**: When enabled via `NewGameWithAnte`, designated nontoken cards move to the public ante zone.
* **Ownership vs. Control**:
  * `ChangeOwner` alters card ownership copy-on-write without changing control.
  * When a duel concludes with a victor, `AnteResult` transfers ownership of cards in the ante zone to the winner.

## Spell Step Restrictions (CR 601.3, 707.10)

`Spell(..., OnlyDuring(step))` uses the existing action timing fields. Normal
casting, alternate-cost and free zone casting, casts of card copies, and
`GetCastableSpells` enforce the restriction before choices or payment. A rejected
cast leaves the hand, mana, and stack unchanged. The restriction grants no
additional casting permission and does not replace normal sorcery timing.
Either player can satisfy the step restriction. Copying an existing stack spell
is not casting; neither copying nor resolution checks this restriction again.
Card copies and cloned games retain the action definition.

Rules baseline: repository Comprehensive Rules effective February 27, 2026.
Evidence: `TestSpellStepRestriction` (every step, both players, four cast paths),
`TestSpellStepCopyAndResolution`, `TestSpellStepDoesNotGrantSorceryTiming`,
`TestSpellStepOptionValidation`, and `TestTeleportCastingRestriction`.

## Combat Removal (CR 506.4, 509.1h)

`Combat.RemoveFromCombat` removes attacking or blocking membership without changing an attacker's blocked status. Removing the last blocker does not make that attacker unblocked. Typed `RemovePermanentFromCombat` resolves one battlefield incarnation; absent and phased-out references do nothing. Clone combat state is independent.

Legacy False Orders and Ydwen Efreet effects retain a private explicit-unblocking compatibility path. Historical sole-blocker tracking and the remaining card clauses require separate audits; that path is not the general combat-removal operation.

Regression evidence: `TestCombatRemovalKeepsAttackerBlocked`, `TestTypedCombatRemovalLifetime`, and existing `TestFalseOrders` / `TestYdwenEfreet`. Rules revision: February 27, 2026.
