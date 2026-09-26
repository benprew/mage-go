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
* **Sacrifice Action Result**: A typed `Sacrifice` instruction writes 1 only when its controller sacrifices the referenced, present, unphased permanent. It writes 0 for an absent, returned, phased-out, or differently controlled permanent. The `NumberBinding` belongs to the resolving stack frame and is copied with that frame. This result does not describe a sacrifice paid as a cost (CR 701.21a, 118.10).
* **Leave-and-Return**: When a permanent leaves the battlefield (to exile, graveyard, or hand) and returns, it becomes an entirely new object with a new `incarnationID`. A delayed trigger, continuous effect, or spell targeting the old incarnation will not affect the returned card.
* **Phasing**: Phasing out does **not** change `incarnationID` and does not cause leave-the-battlefield or enters-the-battlefield triggers. Phased-out permanents are ignored for game actions, but their incarnation is preserved and resumes when they phase in.
* **Last-Known Information (LKI)**: When an object moves zones or dies, its characteristics immediately prior to the event are recorded. Effects reading source power, attachment status, or colors read this LKI snapshot (CR 608.2h).
* **Stack References**: Typed spell targets store the stack object's ID. Target legality is checked again at resolution. Removing a spell and casting the same card again does not preserve the old target. `CounterStackObject` counters a spell by this ID; copied spells leave the stack without entering a graveyard. The source-ID counter entry point also handles abilities, including ward.
* **Spell Controller Reference**: `ControllerOfSpell` resolves the current controller of the referenced spell stack object, independent of the card owner and the resolving ability controller (CR 112.2). An absent or illegal spell reference yields no player. Each stack copy reads its own target frame; an old stack ID cannot refer to a later cast of the same card.
* **Spell Mana Value Reference**: `ManaValueOfSpell` reads the referenced spell on the stack, using its card's mana cost and its announced X (CR 202.3e). Each `{X}` contributes that value. Additional costs and cost changes do not change mana value (CR 202.4). An absent or illegal spell reference supplies zero. The read follows the target stack ID and each copy's target frame.
* **Graveyard References**: Each graveyard entry records a visit ID on the owning player. The stack captures that ID when a card is targeted. Legality checks and typed card references compare the captured visit with the current visit, so moving a card away and back does not revive its old target. Game clones share visit maps until the next graveyard entry and then copy them.
* **Death-Event Card References**: A death pattern with `WithCard` captures the destination graveyard visit when the trigger is detected. Simultaneous deaths use the completed destination state for the visit and the prior battlefield state for event characteristics. `FromCard` resolves only that visit. A token or a death redirected away from the graveyard has no card reference.
* **Token Entry**: `NewToken` marks the card as a token, and `NewPermanent` copies this status before battlefield entry processing. Token effect constructors and `NewToken` copy characteristic slices so later caller changes cannot alter created tokens. Token effects clone each printed ability with a new ability ID before entry setup assigns its source and controller. If an entry rule rejects the token, creation ends without a battlefield object; attacking and blocking entry paths also return without combat assignment. A multi-token effect prepares all accepted entries before it registers them. It registers every token before it runs entry effects and sends entry events, so entry triggers see the full group. Entry restrictions and as-enters choices are still checked for each token before the group enters.
* **Graveyard Return to Battlefield**: A typed return checks the graveyard visit before moving the card. The shared graveyard movement service records its departure, and normal battlefield entry creates a new permanent incarnation under the explicitly selected controller. The card keeps its owner and uses normal entry abilities and triggers.

---

## 3. Controllers and Layer 2 (CR 613.1b, 613.7)

* A `Permanent` has a `defaultController` and a computed controller. `NewPermanent` initializes both from the controller specified when put onto the battlefield (CR 110.2a).
* `Permanent.ControllerID()` always returns the computed controller. Card code must **never** mutate controller fields directly.
* `EffectManager.Apply` resets computed controllers to their default and applies continuous control-changing effects in **Layer 2** in timestamp order.
* A true control change occurs only when the Layer 2 computed controller differs from the previous controller. Real changes update ability context, attack permissions, and summoning sickness (CR 302.6).
* Triggers retain the controller recorded at trigger creation (CR 603.3a). Triggers on leaving the battlefield use LKI controller (CR 603.6c).
* `LeavesBattlefield(This)` matches the source's battlefield-to-any-zone event. Detection uses the pre-departure ability and controller, including a simultaneous departure. Phasing is not a zone change. The pending trigger keeps the old source incarnation if the card returns before resolution (CR 400.7, 603.6c, 603.10a).
* `CreatureYouSacrifice()` matches `EvtSacrifice` after a successful creature sacrifice. The event keeps the sacrificer's player ID and the creature-type result from before departure. A source sacrificed this way can trigger from its departure LKI. The pending trigger keeps its event-time controller and source incarnation through return, copying, and cloning. Destruction and phasing do not emit this event (CR 603.2g, 603.3a, 701.21a).
* `OpponentLandBecomesTapped().OfSubtype(subtype)` uses the normal tap-transition event. It checks the current land type, subtype, and opponent relationship when the land becomes tapped. A second tap of an already tapped land produces no event. Later control, subtype, or zone changes do not undo the pending trigger; it keeps its triggering controller (CR 603.2, 603.3a).
* `EvtSpellCast.TargetID` identifies the spell stack object that was cast; `SourceID` remains the source card ID for older event consumers. This distinction matters when another stack copy has the same source card ID. `SpellCast().OfColor(color)` checks that exact object's effective stack color after cast costs are paid (CR 601.2i). A stack color change or printed color override counts. The queued trigger keeps its event-time controller after the source leaves or returns. A copied stack object does not itself cause a cast event.
* `SpellCast().ByOpponent()` compares the cast event's player with the source controller at detection, not with its controller at resolution. `WithCaster(binding)` saves that player in the pending trigger's resolution instance. Later source control changes, game clones, and stack copies do not change the saved player (CR 603.3a, 603.5). A failed cast emits no cast event.
* `SpellCast().OfType(cardType)` checks the card types of the spell stack object identified by the cast event, after casting completes (CR 601.2i). A stack copy with the same source card ID cannot supply the type. A copied spell does not emit a new cast event. The filter uses the existing stack and trigger services.
* `OpponentDrawsCard(binding)` matches each completed card draw by a player other than the source controller at detection. Each draw has its own pending trigger and saved player. A replaced draw or an empty-library attempt emits no draw event. The binding stays with its trigger through control changes, source departure and return, copying, and game cloning (CR 121.2, 603.2g, 603.3a).
* Typed `Attacks(This)` listens to the declared-attacker event. Typed `Blocks(This)` listens to the first declared-blocker event for the source in that combat. A second attacker blocked by the same creature does not create a second source-blocks trigger. Entering attacking or blocking does not match these events. Trigger resolution keeps the source incarnation and triggering controller; a new incarnation cannot receive the old trigger's source modification.
* `EachCreatureYouControlAttacks()` reads each completed `EvtDeclaredAttacker` for a creature controlled by the battlefield source's controller at detection. Each event makes one trigger. `WithAttacker` saves the attacker's battlefield incarnation, and `WithDefendingPlayer` saves the player in the event's `TargetID`. Source and attacker phasing suppress new triggers. A saved attacker does not follow a card that leaves and returns. The current combat model records only player defenders (CR 508.1m, 508.2a).
* The engine emits declared-blocker pair events after minimum-blocker and required-block checks finish. A removed pair does not emit a blocks event. Each new legal pair emits one event. The first event for a blocker in combat has `Flag=true`; later pair events for that blocker have `Flag=false`. Normal combat and combat simulation use the same rule (CR 509.1i, 509.3a–b).
* Typed `BecomesBlocked(This)` listens to the first blocked-state transition for the source attacker (CR 509.3c). It uses the existing event after legal blocker declaration, or when a creature enters blocking an unblocked attacker. The pattern has no blocker reference; blocker-specific events require a separate pattern. The pending trigger retains its controller and source incarnation through departure, phasing, copying, and cloning.
* `EvtAttacksUnblocked` reports each live attacker with no legal blocker after blocker declaration (CR 509.3g). It includes a creature that entered attacking. A blocked attacker does not match when its blockers later leave combat. The event records the attacker, its controller, and its defending player. `AttacksUnblocked(This)` uses the shared pending-trigger path and saves the defending player when requested.
* `BlocksOrBecomesBlockedBy(This).OfSubtype(subtype)` listens to `EvtBlockerGroup` after legal blocker pairs settle. One event lists the new opposite creatures for each source and direction. The pattern requires at least one current creature of the subtype in that list and creates one trigger for the group. It tests characteristics when the event occurs; later changes do not undo the trigger. A creature entering blocking supplies only the attacker's event, since the entrant did not block (CR 509.3e–f, 509.4).
* `AnotherCreatureYouControlDies()` matches a battlefield-to-graveyard zone change for a creature other than the trigger source. It uses the dying creature's pre-departure type and controller. In a simultaneous death batch, a source that dies with another controlled creature triggers once for that other creature; the pending trigger keeps its original controller and source incarnation.
  * `AnotherCreatureDies().Nontoken()` uses the same death event and source-exclusion check for either controller. The nontoken filter reads the dying permanent's pre-departure token status, including when a token ceases to exist after the zone change (CR 111.7, 603.10a). Simultaneous source and subject deaths each use their original battlefield identities.
  * `.WithoutColor(color)` on a creature-death pattern uses the same pre-departure object view. A creature with the excluded color does not match even if it also has another color. Color changes already applied before departure count; a later card incarnation cannot change the queued trigger (CR 105.2, 603.10a).
* `AnotherCreatureYouControlEnters().Nontoken()` checks each battlefield entry event against the live entering permanent after entry effects and continuous effects apply (CR 603.6a–b). The creature type, token status, and controller are read at event detection. It excludes the trigger source and does not target the entrant. A queued trigger keeps its source incarnation and controller through source departure, return, and stack copying.
* `.OfSubtype(subtype)` on that entry pattern checks the entering permanent's current creature subtype at event detection. Later subtype changes do not change whether the trigger was queued. Registration rejects an empty subtype or a known subtype from another family.
* `.WithPermanent(binding)` on this entry pattern saves the entering battlefield incarnation when the event is detected. The pending trigger carries that value through stack copying and game cloning. `From(binding)` affects only that incarnation while it remains present and phased in; a returned card is a new object (CR 400.7). The binding is not a target.
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
  * Typed `RemoveAbilities` and `RemoveAbilitiesEach` use the same Layer 6 removal helper. `Until` captures permanent incarnations; `While` evaluates its set each time. Source departure does not end an end-of-turn removal, and phasing preserves its captured identity. See `TestRemoveAbilitiesTemporaryCapturesSetAndLayerOrder` and `TestRemoveAbilitiesStaticSetUpdates`.
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
* **Typed Control Change**: `ChangeController(PermanentRef, PlayerRef)` installs only through `Until(EndOfTurn, ...)`. It captures the target incarnation and player at resolution, then applies in Layer 2 timestamp order. Source departure does not end it; a returned target is a new object. The control prohibition and expiry reconciliation use the shared layer service.
* **Static Effects (`While`, `Static`)**: Static continuous effects continuously recalculate their matching set (CR 611.3a). Permanents entering later receive the boost or ability dynamically.

---

## 5. State-Triggered Abilities and Simultaneous Deaths (CR 603.8, 603.6c)

* **State Triggers (`WhenState`)**:
  * Trigger immediately when their pure `StateCondition` evaluates to true, including between instructions during resolution.
  * `SourceTappedCondition` and `SourceUntappedCondition` read the current battlefield incarnation. An absent or phased-out source satisfies neither condition. Resolution and static checks reject a different incarnation of the same card when the source incarnation is captured.
  * `PermanentIsTapped(This)` and `PermanentIsUntapped(This)` use that source contract. The `AttachedPermanent` forms follow the source's current attachment. They require the current source and attached host incarnations, and treat an absent or phased-out object as neither tapped nor untapped. Reattachment changes the subject of the next check. Static layer checks use the same predicate.
  * `SourceBlockingCondition` reads current combat membership for the battlefield source incarnation. An absent or phased-out source does not match. Phasing removes combat membership. A returned card cannot inherit the old block because combat records the blocking incarnation. Departure keeps the old relationship available to leave-trigger resolution, even if that card returns before the trigger resolves. Removing a blocker preserves the attacking creature's blocked status.
  * `PermanentIsAttacking(This)` and `PermanentIsAttacking(AttachedPermanent)` read current attacker membership for the selected battlefield incarnation. The combat service records attacker incarnations for declared attackers and creatures that enter attacking. An absent, phased-out, removed, or returned object does not match. Clones copy attacker identity independently. The older card-ID combat queries remain available to existing callers.
  * `PermanentIsBlocking(This)` and `PermanentIsBlocking(AttachedPermanent)` read current blocker membership for the selected battlefield incarnation. The combat service records blocker incarnations for declared blockers and creatures that enter blocking. An absent, phased-out, removed, or returned object does not match. The Aura form also requires its current host incarnation. Clones copy combat membership independently. A removed blocker does not make its attacker unblocked.
  * `SourceAttackedThisTurnCondition` reads the turn tracker's declared-attack history for the current battlefield incarnation. It is false before the attack, while the source is absent or phased out, after leave and return, and after turn reset. Game clones keep separate history. The older card-ID attack query remains available to existing callers.
  * A trigger stays **suppressed** while its original instance is pending, on the stack, or resolving.
  * Once that instance leaves the stack (resolved, countered, or removed), the ability rearms and triggers again if the condition is still met.
  * A permanent that leaves and returns has a new instance and can trigger independently.
* **Simultaneous Batch Observation**:
  * `EvtAttackersDeclared` is sent once after each legal declaration group that contains at least one newly declared attacker. `Amount` is the number newly declared, excluding creatures already attacking or put onto the battlefield attacking. The normal turn path and simulated attack path send the same event. A zero declaration sends no event. The typed group pattern checks the event's attacking player against its source controller at detection and saves any count in the pending trigger instance.
  * `EvtCreatureBecomesBlocked` reports the attacker when its combat group changes from unblocked to blocked (CR 509.3c). Declared blockers are checked after restrictions and required blocks settle, so two blockers in one declaration produce one event and a removed illegal block produces none. A creature entering blocking also causes the transition if the attacker was unblocked. An already blocked attacker does not produce a second event. `EvtDeclaredBlocker` remains the per blocker-and-attacker event; its flag still describes the first event for that blocker.
  * `EvtCreatureBlockedBy` reports each new attacker and blocker pair (CR 509.3d). It names the attacker in `SourceID` and the blocker in `TargetID`. Declarations send it only after restrictions and required blocks settle; a removed illegal block sends no pair event. A creature entering blocking sends a pair event even when the attacker was already blocked. The event uses the attacker's event-time controller. `EvtDeclaredBlocker` and the once-per-attacker `EvtCreatureBecomesBlocked` keep their separate contracts.
  * A union of destruction filters uses one `DestroyEach(MatchingPermanents(Or(...)))` action. Each matching permanent is considered once, so one regeneration shield suffices even if multiple types match. State triggers cannot observe separate type batches.
  * Mass destruction (`DestroyPermanents`) preserves the pre-destruction battlefield snapshot for leave-the-battlefield triggers.
  * Singular `DestroyWithoutRegeneration` and legacy positional/string-bound no-regeneration executors use a one-permanent batch with the same action-scoped prohibition.
  * `DestroyEachWithoutRegeneration` uses the same batch with an action-scoped regeneration prohibition (CR 701.8, 701.19c). It does not add keywords to permanents or their last-known characteristics. Indestructible and other destruction replacements still apply.
  * Zero-toughness state-based deaths share the same observation batch.
  * Conditions evaluate against the pre-batch state; trigger effects resolve against the current game state.
* **Creature-Death Controller Capture**:
  * `PermanentDies().OfType(type)` uses the same battlefield-to-graveyard event and departure LKI as creature-death patterns. The type filter checks the object before it leaves. `WithController` saves the former controller from that event, including when the source leaves in the same batch. A replacement that sends the object to exile produces no death match. The pattern observes one event per departing permanent (CR 603.2c, 603.10a).
  * `CreatureDies().WithController(binding)` and the controlled-creature variants save the dying creature's controller from the battlefield-to-graveyard event. The event records that player before the permanent departs (CR 603.10a).
  * `CreatureDies().WithOwner(binding)` and the other creature-death variants save the dying creature's owner from the event-time object or departure LKI. The pending trigger keeps the player through simultaneous death, source departure, stack copying, and game cloning. The saved owner is independent of the trigger controller and the dying creature's controller (CR 400.3, 603.10a).
  * Each pending trigger carries its own event. The bound player is installed in that trigger's resolution frame before its body runs. Stack copies keep the captured player even if their controller changes. The participant is not a target.
  * Phased-out sources do not observe the event. A source that dies in the same destruction batch can observe the other deaths through the existing pre-batch trigger snapshot.
* **Creature-Death Toughness Capture**:
  * `CreatureDies().WithToughness(binding)` and the controlled-creature variants save the dying creature's toughness when the trigger is detected. Detection uses the pre-batch permanent or its departure LKI, including continuous effects (CR 603.10, 608.2h).
  * The pending trigger stores the number before stack placement. Resolution reads its own number binding, so cleanup, a later incarnation of the same card, stack copies, and game clones do not change the saved value. This capture also applies to a creature's own death trigger.

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

The `RemoveAnyCountersCost` activation cost chooses a count during payment preparation and locks that count and the source battlefield incarnation. Validation pays the same count on an isolated game. Commit rejects a departed, returned, phased-out, or counter-short source. The paid-number result becomes available only after the chosen removal succeeds; stack copies and game clones retain separate captured values. Zero is a legal choice when the source is present. This cost uses the existing counter-removal payment path and does not add a resolution-time removal action.

* **Nonpositive Damage Cutoff (CR 120.8)**:
  * If a damage action's final amount is reduced to zero or less by prevention or modifiers, the damage action terminates before further replacement checks or execution.
  * Zero/negative damage produces **no damage event, damage history, callback, or marked damage**.
  * Later replacements cannot resurrect or increase an absent damage event.
* **Lifelink Outcome Routing**:
  * Damage from a permanent with lifelink causes its controller to gain that much life simultaneously with damage execution.
  * Life gained from lifelink routes through the event-aware life gain executor, emitting an `EvtLifeGained` with the actual positive amount after life-gain replacements.
* **Damage Source Identity**:
  * A damage instruction captures source characteristics and controller before replacement and prevention. If a permanent leaves and the same card returns, its new incarnation does not change the old instruction's protection, lifelink, or deathtouch result.
  * A resolving ability reads its captured source incarnation from durable departure LKI. A resolving spell uses the stack object's controller. An unrelated source ID does not inherit the resolving source's LKI. Phased-out permanents do not supply a new source capture.
  * `CaptureDamageSource` supplies an immutable source value to `DealDamageToPlayerFromSource` and `DealDamageToPermanentFromSource`. The UUID-based entry points capture the source when called. Combat damage and fight keep the existing shared damage pipeline.
  * The typed `DealDamageFrom(This, recipient, amount)` instruction reads the captured source incarnation from its resolution frame. It uses the same damage pipeline for player and permanent recipients. An absent recipient or a missing source snapshot causes no damage. A declared target keeps its recorded battlefield incarnation when resolution builds effect bindings, even if the card has returned.
  * `DamageSourceOf(permanent)` reads a typed permanent reference. A target or saved permanent binding retains its captured incarnation. A departed incarnation supplies damage-time source characteristics from durable departure LKI, even if the same card returns under another controller. Phasing alone makes the source absent. The adapter uses the shared damage executor and does not read a global last-action slot.
  * `DamageAmountInto(binding)` records the final amount dealt by one typed damage instruction in its resolution frame. Prevention, replacement, redirection, and minimum-life restrictions apply before this value is written. An absent source or recipient and a zero action write zero. A redirected action keeps the captured damage source. The result does not report life lost, counters removed, or a group total.
  * `DamagePlayerInto(binding)` records the final player recipient only when positive damage is dealt to a player. The shared replacement pipeline can redirect permanent damage to a player or player damage to a permanent; the binding follows the final action. A prevented or absent action clears any prior value in that resolution frame.
  * `DamagePermanentInto(binding)` records the final permanent recipient's battlefield incarnation only when positive damage is dealt to that permanent. The snapshot is taken from the final action after replacement and before execution, so a later zone change cannot make it refer to a returned card. A player result, prevention, or absent action clears any prior value in that resolution frame.
  * `DamageSourceInto(binding)` records the source on the final executed damage action only when positive damage is dealt. `FromDamageSource(binding)` can use that source in a later instruction. The saved value keeps its controller, incarnation, and damage-time characteristics after departure or return. Prevention, zero damage, or an absent source or recipient clears the binding. Each resolution and game clone owns its saved value.
* **Planeswalker Damage Result**:
  * `Game.DealDamageToPermanent` runs damage replacement and prevention before it applies results. Actual damage removes up to that many loyalty counters from a planeswalker. It does not mark damage unless that permanent is also a creature. Fully prevented damage changes no loyalty and creates no damage history.
  * Damage history still uses physical card IDs. Recipient history and damage-condition bindings remain separate work.
* **Combat Damage Assignment**:
  * Attacking creatures assign lethal damage to each blocker in damage assignment order before assigning damage to the next blocker.
  * Trample assigns excess damage beyond lethal toughness to the defending player/planeswalker.
* **Fight Damage (CR 701.14)**:
  * The engine prepares both damage actions through the normal replacement and prevention pipeline, then applies both before it sends damage events and callbacks.
  * Damage observers see both results of the fight. A creature that fights itself receives one damage action equal to twice its power.
* **Mixed Damage Batch (CR 120.4)**:
  * `Game.DealSimultaneousDamage` accepts typed player and permanent assignments with captured damage sources. It prepares each assignment through the same replacement and prevention service as singular damage, then applies all final actions before it sends events and damage callbacks.
  * The internal `Game.dealSimultaneousDamageOutcomes` returns one result for each final action that deals positive damage. Each result contains the amount, final player or permanent incarnation, and captured source. Full prevention and absent recipients produce no result. The total adapter sums these results, so redirection changes recipient identity without losing or counting damage twice. Event and callback dispatch still follows the complete batch.
  * A permanent assignment keeps the recipient's battlefield incarnation. It skips a returned or phased-out recipient. A player assignment resolves the player in the executing game, so a game clone does not change the original player.
  * `DealDamageEach(This, DamageRecipients(players, permanents), amount)` captures its source and amount once. It builds all assignments from the selected players and battlefield permanent incarnations, then uses the mixed batch. A declared permanent target set can contribute one legal, present target. Larger target groups remain blocked on recorded selection boundaries. `DamageTotalInto(binding)` stores the sum of amounts actually dealt by the final actions, after replacement and prevention, in the resolution frame. It counts redirected damage once and writes zero for an absent source or an empty batch.
  * `DamageResultsInto(binding)` stores every positive final action with its actual amount, final recipient, and captured source in the resolution frame. `ForEachDamageResult` reads these results after the complete batch and gives each iteration a separate binding scope. Its body runs sequentially. A recipient permanent reference retains its damage-time incarnation and becomes absent after departure, return, or phasing. A saved source retains its damage-time characteristics. An empty or prevented batch clears any earlier result in the same binding. Game clones copy saved results independently. Production card wiring stays on its existing path.
* **Attached Creature Death (CR 603.6c, 603.6e)**:
  * `AttachedCreatureDies()` checks the attachment relationship and both battlefield incarnations at the death event. It reads the pre-departure state for a simultaneous source and creature departure. The trigger can capture the dying creature's former controller and toughness before either object leaves; copies and game clones retain independent event values.

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

For matching sacrifice costs, preparation asks the payer for each permanent
once and validates those choices on the isolated game. The transaction stores
their battlefield incarnations. Commit sacrifices those exact objects; an
absent, returned, phased-out, or no longer eligible choice makes payment fail.
For a multi-permanent sacrifice, commit checks all choices before moving any
of them. The cost declarations remain reusable across payment transactions.
`SacrificeCreatureCostWithStats` reads the selected creature's current power
and toughness from the live game immediately before that cost sacrifices it.
After successful payment, the values enter that activation's or spell's
stack frame, including spells cast with an alternative cost. A failed payment
creates no frame or paid-stat result. A stack
copy keeps the same paid values, and a game clone copies them independently.
`SacrificeMatchingCostWithManaValue` uses the same locked choice and payment
frame. Commit reads the chosen permanent's mana value before sacrifice, then
writes it to that spell or activation's number binding only after the cost
succeeds. A returned or phased-out choice does not produce a value. The
resolution-time `IfPaid` and `UnlessPaid` paths do not carry this output.

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
* **Typed One-Color Choice**:
  * `AddManaOfAnyOneColor` evaluates its amount once, then uses one resolution-time color choice for all produced mana. A nonpositive amount produces no mana and makes no choice. Fixed positive amounts appear as one `AnyColor` production with `AnyCombination` false. Dynamic amounts still identify a mana ability but are not advertised as fixed planner output. The typed action uses the existing mana production, tap event, bonus, and payment paths.
  * Evidence: `TestAddManaOfAnyOneColorChoiceAndEvent`, `TestAddManaOfAnyOneColorPlanningAndRollback`, and `TestAddManaOfAnyOneColorDynamicAndCopy`.
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

An `EvtDeclaredAttacker` records its defending player in `TargetID` when the attack is declared. Both the turn path and simulated declaration path supply this value. A typed attack trigger captures it before the stack entry is made, so later combat changes do not change the saved player.

`EvtCreatureBlockedBy` occurs once for each new attacker and blocker pair after legal blocker declarations settle. `BecomesBlockedBy(This)` uses this event and checks the blocker subtype when the event occurs. `WithBlocker` saves that blocker's battlefield incarnation in the pending trigger. A copied trigger and a cloned game keep independent captures. The binding does not follow a blocker that leaves and returns, and a phased-out blocker is absent until it phases in.

`EvtBlockerGroup` reports the new legal combat pairs once per creature after blocker restrictions and requirements settle. `SourceID` identifies that creature; `RelatedIDs` contains the new opposing creatures in combat order. `Flag` is true when the source blocks and false when it becomes blocked. A creature put onto the battlefield blocking produces only the attacker-side event. Existing pairs and pairs removed by restrictions produce no group event. Trigger detection checks the opposing creatures' characteristics at event time; the pending event and its member list are copied for a cloned game. `BlocksOrBecomesBlockedBy(This).OfSubtype(subtype)` uses this event for a filtered "one or more" group (CR 509.3, 509.4).

`Combat.RemoveFromCombat` removes attacking or blocking membership without changing an attacker's blocked status. Removing the last blocker does not make that attacker unblocked. Typed `RemovePermanentFromCombat` resolves one battlefield incarnation; absent and phased-out references do nothing. Clone combat state is independent.

Legacy False Orders and Ydwen Efreet effects retain a private explicit-unblocking compatibility path. Historical sole-blocker tracking and the remaining card clauses require separate audits; that path is not the general combat-removal operation.

Regression evidence: `TestCombatRemovalKeepsAttackerBlocked`, `TestTypedCombatRemovalLifetime`, and existing `TestFalseOrders` / `TestYdwenEfreet`. Rules revision: February 27, 2026.
