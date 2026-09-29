# Rules Engine Contracts and Subsystems Architecture

This document specifies the internal architecture, subsystem invariants, and Magic: The Gathering Comprehensive Rules (CR) execution contracts for `mage-go`.

`ExileWithResult` uses the same singular battlefield exile operation as `Exile`. It saves a numeric result in the current resolution frame only after a current, unphased battlefield incarnation moves. An absent subject or a card that left and returned saves zero. The result records a completed move, including a token leaving the battlefield; it does not identify a card in exile. This keeps action results separate from linked exile records (CR 400.7, 406.2, 701.13).

`MayAttach` resolves both permanent references before it offers a choice. It offers no choice for an absent, phased-out, same-host, or illegal pair. If the controller accepts, `Game.Attach` checks legality again and performs the move. The shared move path updates the attachment timestamp, continuous effects, and attach event (CR 303.4j, 701.3, 613.7e). The action does not follow a permanent that leaves and returns.

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
* **Saved permanent subtype reads**: A resolution branch over a saved battlefield incarnation reads its current subtype while that incarnation remains present. After departure, it may read the `ZoneSystem` LKI snapshot keyed by that incarnation. A returned object with the same card ID does not replace the saved object. Phasing does not create a departure snapshot. This read follows CR 400.7 and 608.2h; a target that became illegal remains subject to the separate target legality check in CR 608.2b.
* **`TurnSystem`**: Owns turn counters, current phase and step, active player index, extra turns, `TurnSchedule` lifecycle, and one-shot untap restrictions.
* **`ResolutionState`**: Owns transient resolution scratch state including X values, chosen modes, event source/amount context, cast-time snapshots (`CastContext`), targets, damage/counter distributions, and cost payment reveals/sacrifices.
* **Cast-event X capture**: `EvtSpellCast` carries the spell's X value from its stack object at event dispatch. A battlefield `SpellCast().WithAnnouncedX` trigger saves that value in its own frame before the spell leaves the stack. The capture remains fixed if the spell or trigger is copied or the game is cloned (CR 107.3a, 603.2).
* **Cast-event spell binding**: `SpellCast().WithSpell(binding)` saves the cast stack object's ID in each pending trigger. `FromSpell(binding)` reads that exact object only while it remains on the stack. A second cast of the same card has another stack ID. The binding is not a target, so target legality does not apply. A trigger copy keeps the saved ID, and a game clone copies the active binding frame (CR 400.7, 603.2, 608.2k).
* A graveyard-exile cost that captures mana value locks the chosen card and graveyard visit during total-cost preparation. Commit checks that visit before paying and saves the card's pre-exile mana value in the spell or activation frame only after the exile succeeds. A failed payment does not publish the capture. Stack copies and game clones retain separate frames (CR 118.10, 400.7).
* **`TriggerSystem`**: Owns pending triggers waiting for stack placement, active `DelayedTrigger` instances, and armed state triggers (CR 603.8).
* **`ManaSystem`**: Owns mana source discovery, mana ability solver planning, and payment scratch buffers.
* **`DamageSystem`**: Owns damage execution, combat damage step aggregation, damage history tracking, and damage reflection.
* **Permanent damage history**: `HasDealtDamageToPlayerFromSource(DamageSource, playerID)` and `HasDealtDamageToPermanentFromSource(DamageSource, *Permanent)` read game-long history keyed by the permanent source incarnation. The permanent-recipient query also checks its battlefield incarnation. The damage system records only positive damage after replacement and prevention. Departure retains the old record, return creates a new identity, and phasing does not change identity. Game clones copy the records. The typed `PlayersDealtDamageBy` and `PermanentsDealtDamageBy` sets filter current candidates through these queries; they do not use the older card-ID history. These queries require a captured permanent source; spell sources have no battlefield incarnation. The older card-ID queries remain for existing callers but do not distinguish returned objects (CR 400.7, 120.4b, 120.8).
* **Reverse turn damage history**: `PermanentsThatDealtDamageToThisTurn(recipient, candidates)` reads the shared turn record by both battlefield incarnations. `This` uses the resolving trigger's captured incarnation, so its own death and a later return do not change the recipient. The selector filters current, unphased candidates and checks their current incarnation. Prevention and replacement determine which positive damage enters the record. Final cleanup clears it; game clones copy it (CR 400.7, 120.4b, 120.8, 608.2i).
* **Turn and combat history**: Declared attacker and first declared blocker events record the current battlefield incarnation. Pure state queries can read either this-turn or this-combat history. A new beginning-of-combat event clears only combat history; the end-of-combat step keeps it available to triggers, then clears it after priority. Cleanup clears both records. Leaving and returning does not inherit either record; phasing preserves the incarnation but a phased-out source cannot satisfy the battlefield condition. A creature put onto the battlefield attacking or blocking did not make a declaration.
* **Current blocker set**: `BlockersOf(attacker)` reads the shared combat groups when consumed. It requires the attacker's current battlefield incarnation to be attacking, and includes only current creature incarnations recorded as its blockers. A removed, phased-out, departed, or returned participant is absent. A blocked attacker with no remaining blockers yields an empty set (CR 506.4, 509.1g–h).
* **`TrackerSystem`**: Owns turn-scoped observations (`TurnTrackers`) and duel-scoped observations (`DuelTrackers`).
* **`RandomSource`**: Owns deterministic integer and coin flip queues for tests and AI simulation.

Read-only queries are organized into focused domain interfaces: `PlayerReader`, `BattlefieldReader`, `ResolutionReader`, `TurnReader`, `CombatReader`, `TrackerReader`, and `StackReader`, which compose into `GameReader`.

---

## 2. Object Identity, Incarnations, and Phasing (CR 400.7)

* **Defending-Player Attack Restriction**: `CanAttackOnlyIfDefendingPlayerControls(filter)` installs a battlefield static rule on its source. The shared attack restriction checks the nonactive player's current matching battlefield permanents in this two-player engine. It uses control, not ownership, and excludes phased-out or departed permanents. The restriction applies when attackers are declared; it does not remove an existing attacker (CR 508.1c, 508.4c, 604.1–2, 611.3a).

`Game.Attach(sourceID, targetID)` checks both battlefield objects before it changes an attachment. An Aura must satisfy its enchant restriction, protection, and enchantment prohibitions. An Equipment must attach to a creature; a Fortification must attach to a land. An ordinary permanent cannot attach. A failed move leaves the old host and both attachment lists unchanged. Attaching to the same host has no effect (CR 303.4j, 701.3a–b). Phased-out and departed objects are absent for this action. The operation records the new host incarnation when it succeeds. A successful move gives the attachment's printed static effects a new timestamp in their existing relative order. The layer service applies them after effects with older timestamps (CR 701.3c, 613.7a, 613.7e). A failed or same-host attempt does not change effect order. Game clones keep independent effect order.

* **Physical Card vs. Rules Object**: `Permanent.ID()` returns the physical card UUID, but `Permanent.incarnationID` tracks the unique existence of that permanent on the battlefield.
* **Battlefield References**: Resolving effects (such as `Sacrifice(This)` or `Untap(This)`) resolve and verify the specific captured battlefield incarnation.
* **Sacrifice Action Result**: A typed `Sacrifice` instruction writes 1 only when its controller sacrifices the referenced, present, unphased permanent. It writes 0 for an absent, returned, phased-out, or differently controlled permanent. The `NumberBinding` belongs to the resolving stack frame and is copied with that frame. This result does not describe a sacrifice paid as a cost (CR 701.21a, 118.10).
* **Simultaneous Destruction Graveyard Result**: `DestroyEach(subjects, PermanentsPutIntoGraveyardThisWayInto(binding))` saves the count of distinct selected battlefield incarnations that the shared destruction batch moves to a graveyard. Regenerated, indestructible, phased-out, absent, returned, or replacement-exiled objects do not count. Tokens moved to a graveyard count. The result is local to the resolving instance, and an empty set writes zero. `DestroyEachWithoutRegeneration` has the same result contract (CR 400.7, 609.3, 701.8).
* **Turn State Condition**: `IsYourTurn()` compares the current active player with the condition context's controller at evaluation. `IfState` therefore uses the resolving spell or ability's stored controller, even when its source is absent. Game clones read their own active player. The condition makes no choice and does not restrict casting or activation (CR 500.1).
* **Player Control State Condition**: `PlayerControls(You, filter)` reads the condition controller; `PlayerControls(ActivePlayer, filter)` reads the current active player. Both query current effective control through the battlefield filter, so ownership does not qualify, phasing excludes a permanent, and leave and return use the new controller. The condition makes no choice and does not save a player or permanent. Target and saved player references lack a pure state frame and fail registration (CR 403.3–4, 604.1, 611.3a).
* **Rounded Resolution Division**: `DivideRoundDown` and `DivideRoundUp` divide signed resolution values with mathematical rounding toward negative or positive infinity. Their operands read the same resolution frame. A dynamic zero divisor supplies zero under CR 107.2; a fixed zero divisor is rejected at registration. A downstream action applies its own rules for a negative result (CR 107.1b).
* **Controller Hand-Size Condition**: `ControllerHandSizeIs(n)` reads the current hand of the condition context's controller at each evaluation. `ActivationIf` checks it before cost payment. Owner identity and the source card's hand do not affect it. Missing controllers fail the check; negative sizes fail registration.
* **Controller Life-Gain History Condition**: `ControllerGainedLifeThisTurn()` reads positive actual gains from the existing turn tracker. Life-gain replacement determines whether the gain is recorded. The condition uses the ability controller, including after source departure; an absent controller fails. A cloned game has separate history. Final cleanup clears the record after cleanup priority rounds.
* **Player Hand-Size Value**: `HandSizeOf(player)` reads the selected player's current hand size when a resolving instruction evaluates the value. It uses the same player reference and target frame as other typed actions. It does not inspect card identities or save an earlier hand size. An absent player supplies zero, and game clones read their own hands (CR 400.1, 400.2).
* **Player Life-Total Value**: `LifeTotalOf(player)` reads the selected player's current life total when a resolving instruction evaluates the value. It uses the typed player reference and current target frame. An absent player supplies zero. It reads each game clone's own life total and does not save the value from an earlier instruction (CR 119.2–5, 608.2c, 608.2h).
* **Starting-Life State Condition**: `ControllerLifeAtLeastStartingLife()` compares the condition controller's current life total with that player's starting total at every check. The source card's owner and starting total do not affect it. A missing controller fails the check; game clones use their own current life totals. The check is pure and can govern a static ability (CR 119.1–5, 611.3a).
* **Highest-Life Player Reference**: `PlayerWithUniqueHighestLife()` selects the sole highest-life player from the current game when a resolving instruction reads the reference. Ties and a missing player yield an absent reference. Earlier life changes in the same resolution can change the selected player. A pending ability and each game clone read their own current totals. The reference is singular and does not capture a player at trigger creation (CR 119.1–5, 608.2c).
* **Leave-and-Return**: When a permanent leaves the battlefield (to exile, graveyard, or hand) and returns, it becomes an entirely new object with a new `incarnationID`. A delayed trigger, continuous effect, or spell targeting the old incarnation will not affect the returned card.
* **Phasing**: Phasing out does **not** change `incarnationID` and does not cause leave-the-battlefield or enters-the-battlefield triggers. Phased-out permanents are ignored for game actions, but their incarnation is preserved and resumes when they phase in.
* **Last-Known Information (LKI)**: When an object moves zones or dies, its characteristics immediately prior to the event are recorded. Effects reading source power, attachment status, or colors read this LKI snapshot (CR 608.2h).
* `OwnerOf(PermanentRef)` reads the current owner's player ID from a live battlefield incarnation at resolution. Control changes do not change that owner. A phased-out or departed incarnation supplies no player; `SnapshotPlayer` keeps a previously read owner through later zone changes (CR 108.3, 400.7).
* `PermanentsOwnedBy(PlayerRef, PermanentFilter)` selects current battlefield incarnations by card owner, independently of controller. The shared battlefield filter excludes phased-out permanents. An absent player yields an empty set; leaving and returning changes membership at the next read (CR 110.2, 400.7).
* **Stack References**: Typed spell targets store the stack object's ID. Target legality is checked again at resolution. Removing a spell and casting the same card again does not preserve the old target. `CounterStackObject` counters a spell by this ID; copied spells leave the stack without entering a graveyard. The source-ID counter entry point also handles abilities, including ward.
* **Untap Action Result**: `Game.UntapPermanent` reports whether a permanent actually became untapped. A stun counter replaces an untap attempt: the action removes one stun counter, leaves the permanent tapped, and sends no untap event. `UntapPermanentWithResult` stores 1 only for a completed transition and 0 for a replaced, redundant, absent, phased-out, or stale-incarnation attempt. The value belongs to the resolving frame, including stack copies and game clones (CR 122.1g, 400.7).
* **Permanent Tap Event**: `PermanentBecomesTapped()` observes each completed `EvtTapped` transition for an active battlefield source. Type and opponent filters use the tapped permanent's event-time characteristics and controller. Optional participant bindings retain that controller and battlefield incarnation in the pending trigger frame. A simultaneous tap batch makes all selected permanents tapped before it sends their individual events; each matching permanent creates one trigger. An already tapped or phased-out permanent creates no tap event, and entering tapped is not a tap transition (CR 603.2c, 603.2e).
* **Target Group Boundaries**: A stack object's target-aligned specifications record which declared group supplied each chosen target, including an empty optional group before a nonempty group. Spell and ability copies use that record when they offer new targets. A copy keeps each group's chosen count; game clones and copies preserve the record separately. Trigger placement records each chosen group's specifications directly, so `WhenEvent` and `WhenEventModes` can use independent variable-size target groups. A typed trigger rejects an illegal, repeated, or out-of-bounds choice within a group before stack placement. Spells declared with `WithTargetGroups` use `CastSpellByNameWithTargetGroups` to announce each group's count and targets before payment. The flat cast path rejects these declarations. The same object can be chosen in separate groups. Typed permanent target declarations forward announced X bounds to the shared target validator and stack specification expansion. Exact-X and up-to-X counts are checked before payment and remain fixed on copies. If a legacy stack object has no aligned record, copying retains the existing count-inference fallback. Delayed declarations still reject two variable-size typed groups until their announcement paths can record both chosen counts (CR 601.2b–c, 602.2b, 603.3d, 707.10c).
* **Pure-X Activation Cost**: `{X}` is a present cost even though its fixed mana component is zero. Action-cost preparation retains it and converts the announced X into generic mana before payment. A failed payment leaves targets, tapping, and stack state unchanged (CR 601.2b, 601.2h, 602.2b).
* **Spell Controller Reference**: `ControllerOfSpell` resolves the current controller of the referenced spell stack object, independent of the card owner and the resolving ability controller (CR 112.2). An absent or illegal spell reference yields no player. Each stack copy reads its own target frame; an old stack ID cannot refer to a later cast of the same card.
* **Spell Mana Value Reference**: `ManaValueOfSpell` reads the referenced spell on the stack, using its card's mana cost and its announced X (CR 202.3e). Each `{X}` contributes that value. Additional costs and cost changes do not change mana value (CR 202.4). An absent or illegal spell reference supplies zero. The read follows the target stack ID and each copy's target frame.
* **Graveyard References**: Each graveyard entry records a visit ID on the owning player. The stack captures that ID when a card is targeted. Legality checks and typed card references compare the captured visit with the current visit, so moving a card away and back does not revive its old target. Game clones share visit maps until the next graveyard entry and then copy them.
* **Dynamic Graveyard Target Limit**: `GraveyardCardManaValueAtMost(ControllerLifeGainedThisTurn())` reads the current turn tracker for the ability controller when target candidates are listed at announcement and when legality is checked at resolution. The target service uses the same graveyard visit and card filter as other graveyard targets. Replacement that prevents a life-gain event does not increase the limit. A later turn reset can make an announced target illegal; the target is not rescued by an old life-gain snapshot (CR 115.1d, 608.2b, 119.9).
* **Optional Graveyard Card Target**: `GraveyardUpToOne()` changes only the graveyard declaration's lower bound to zero. The shared target service accepts zero choices even when a legal card exists, and the trigger stays on the stack with no chosen target. A copy keeps the chosen count and identity; a chosen card must retain its graveyard visit at resolution. The option does not change a card action into a mandatory action (CR 115.6, 603.3d, 608.2b).
* **Hand Visits**: `BasePlayer.AddToHand` records a new visit for the card in hand. `RemoveFromHand` ends it. `SetHand` keeps visits for cards that remain and creates visits for new members. Game clones share the visit map until a hand change copies it. `DrawCardInto` saves the card and its visit after one completed draw. `DrawCardsInto` saves every completed draw in one instruction as separate hand visits. `FromCard` and `FromCards` resolve only visits that remain current. A replaced or empty-library draw saves no card (CR 400.7, 121.1–2, 121.6c).
* **Draw-Event Card References**: `PlayerDrawsCard(player).WithCard(card)` and `OpponentDrawsCard(player).WithCard(card)` capture the completed draw event's card ID and hand visit in the pending trigger. A replaced or failed draw has no draw event. Separate draws create separate trigger frames. `FromCard` is absent after that hand visit ends, including when the same card returns to hand. Source phasing suppresses detection; a pending trigger retains its capture through source departure, stack copying, and game cloning (CR 400.7, 121.1–2, 603.3a).
* **Saved-Card Discard**: `DiscardCard(FromCard(binding))` checks the current hand visit and owner, then uses the shared effect-discard replacement action. A card in another zone, a replaced draw with no card, or an old hand visit cannot be discarded by that instruction. The discard ends the hand visit and emits the normal discard event (CR 400.7, 701.9).
* **Multi-Card Discard Payment**: `DiscardCost(n)` locks each chosen hand visit during payment preparation. It prepares all cost-discard replacements before moving cards, then moves all accepted cards before dispatching discard events. Event observers see the complete payment, including replacement destinations. A prevented discard rejects the entire discard group before movement. The total-cost validation uses an isolated game; stale hand visits fail commit before other costs are spent. The effect hand-discard batch shares the movement and event executor, while retaining partial-action behavior. Random discard remains a separate path (CR 400.7, 601.2h, 602.2b, 701.9).
* **Hand-Discard Result**: `DiscardEntireHand` takes a snapshot of the selected player's hand at instruction start and attempts each discard through the shared effect-discard action. `DiscardAmountInto` counts completed discards after replacements, including discards redirected to another zone. It excludes prevented or impossible discards and writes zero for an empty hand or absent player. The result belongs to the current resolution frame; it does not read global discard history (CR 609.3, 701.9).
* **Player-Set Hand Discard**: `DiscardEach` snapshots every selected hand in active-player order. It applies the shared effect-discard replacements to all proposed cards before moving them. It moves every accepted card before any discard event is sent, so triggers observe the complete action. A prevented discard stays in hand; a discard redirected to library still sends a discard event. `DiscardTotalInto` counts completed card moves from this batch after replacements and writes zero when none complete. The count belongs to the current resolution frame. The following instruction, such as `DrawEach`, starts only after all discards finish (CR 101.4, 609.3, 701.9).
* **Player Discard Triggers**: A successful discard emits one `EvtDiscard` for that card after the card moves. `PlayerDiscardsCard(binding)` matches either player; `OpponentDiscardsCard(binding)` also compares that player with the source controller at detection. Both save the player in that trigger instance. `.WithCard(card)` also saves the card ID and its new graveyard visit from the discard event. The card reference is absent when replacement sends the card elsewhere or the card later changes zones. Each event in a simultaneous discard batch keeps its own visit. A replacement that changes the discard destination still emits the event; an impossible discard emits none. Source phasing suppresses detection, while a queued trigger keeps its saved player, card identity, and controller through source departure or return (CR 400.7, 603.2, 603.3a, 701.9).
* **Public Card Reveal**: `RevealCard(FromCard(binding))` checks the saved zone visit before it records a public snapshot. Revealing does not move the card. `Game.PublicReveals()` returns copies of reveal-time card characteristics in order, so later zone changes and caller mutations do not change what players saw. This history does not keep the card currently revealed (CR 701.20a-b).
* **Controller-bound permanent sets**: `PermanentsControlledBy(player, filter)` resolves the player and reads current battlefield membership when consumed. It uses effective controller rather than owner, excludes phased-out permanents, and follows control changes. A later battlefield visit is a new permanent and is counted only if it then matches. The player binding stays in the spell or ability frame, including copied stack objects and game clones (CR 110.2, 400.7, 702.26b).
* **Continuous-control state**: `ControlledSinceTurnStart(This)` and its `AttachedPermanent` form use the permanent's current controller and control-gain turn. The condition also requires that the current ability controller controls that permanent. A control change resets the gain turn. A new battlefield visit starts a new record. An absent or phased-out source or subject fails the check (CR 110.2, 302.6, 400.7, 702.26b).
* **Saved-Card Type Branch**: `IfCardHasType` and `IfCardLacksType` read a `CardRef` at resolution through its current zone visit. They check the card's current types. An absent reference runs neither body, including the negative branch. Graveyard targets keep their target legality and captured visit; a card that leaves and returns cannot satisfy the old reference. A branch body has local binding scope (CR 300.2, 400.7).
* **Death-Event Card References**: A death pattern with `WithCard` captures the destination graveyard visit when the trigger is detected. Simultaneous deaths use the completed destination state for the visit and the prior battlefield state for event characteristics. `FromCard` resolves only that visit. A token or a death redirected away from the graveyard has no card reference.
* **Token Entry**: `NewToken` marks the card as a token, and `NewPermanent` copies this status before battlefield entry processing. Token effect constructors and `NewToken` copy characteristic slices so later caller changes cannot alter created tokens. Token effects clone each printed ability with a new ability ID before entry setup assigns its source and controller. If an entry rule rejects the token, creation ends without a battlefield object; attacking and blocking entry paths also return without combat assignment. A multi-token effect prepares all accepted entries before it registers them. It registers every token before it runs entry effects and sends entry events, so entry triggers see the full group. Entry restrictions and as-enters choices are still checked for each token before the group enters.
  `CreateOneToken` uses that entry path and saves the accepted token's battlefield incarnation after entry. A later `From` read is absent if the token left or phased out. An entry that creates no permanent clears an earlier value of the same binding.
  `CreateTokenCopiesInto` uses the same batch and saves all accepted incarnations in an instance-local set. Later reads omit a token that left, returned, or phased out. A zero count or rejected batch clears the set. Game clones and delayed effects copy the saved identities. Iteration takes a member snapshot; it does not make the body actions simultaneous.
* **Graveyard Return to Battlefield**: A typed return checks the graveyard visit before moving the card. The shared graveyard movement service records its departure, and normal battlefield entry creates a new permanent incarnation under the explicitly selected controller. The card keeps its owner and uses normal entry abilities and triggers.
  `ReturnCardToBattlefield(card, controller, EnterTapped())` sets tapped status during shared entry preparation, before entry triggers are detected. An entry trigger observes the tapped permanent. Entering tapped does not create a tap-transition event. The instruction does not change the card's printed abilities or later returns. A stale or absent graveyard reference causes no entry (CR 400.7, 603.2e, 614.1d).

---

## 3. Controllers and Layer 2 (CR 613.1b, 613.7)

* A `Permanent` has a `defaultController` and a computed controller. `NewPermanent` initializes both from the controller specified when put onto the battlefield (CR 110.2a).
* `Permanent.ControllerID()` always returns the computed controller. Card code must **never** mutate controller fields directly.
* `EffectManager.Apply` resets computed controllers to their default and applies continuous control-changing effects in **Layer 2** in timestamp order.
* A true control change occurs only when the Layer 2 computed controller differs from the previous controller. Real changes update ability context, attack permissions, and summoning sickness (CR 302.6).
* Triggers retain the controller recorded at trigger creation (CR 603.3a). Triggers on leaving the battlefield use LKI controller (CR 603.6c).
* `SpellCast().TargetsCreature()` reads the exact cast spell stack object from the cast event. It checks each chosen target's current battlefield incarnation and creature type when the event is detected. A target that left and returned, phased out, or moved zones does not match; departure LKI does not qualify. A later target departure does not cancel a pending trigger. `ByController()` compares the caster with the battlefield trigger source's event-time controller. `OfInstantOrSorcery()` checks the cast spell's types. These filters reuse the shared cast event and stack target record (CR 115.9b, 601.2i).
* `BlockingOrBlockedBy(permanent)` reads current combat groups when the consuming instruction resolves. It returns each linked creature once and checks both sides' battlefield incarnations. A creature that leaves, returns, phases out, or leaves combat is absent from the set. This selection does not bind the creatures from the trigger event (CR 509.3, 509.4).
* `LandEnters().WithController(binding)` checks the entering object's land type after entry effects and continuous effects apply. It saves that object's computed controller at detection. The saved player survives land departure, source departure, trigger copying, and game cloning. The land's owner and the player who caused the entry can differ from that controller (CR 603.6a–b).
* `AnotherCreatureEnters().WithController(binding)` and the controlled-creature entry patterns save the entering permanent's computed controller when the entry event is detected. The saved player belongs to the pending trigger and stays fixed after control changes, departure, stack copying, or game cloning. A phased-out source cannot detect the entry, but a trigger already on the stack keeps its saved player (CR 603.3a, 603.6a–b).
* `LandEnters().OfSubtype(subtype)` checks one valid land subtype on the entering permanent when the event occurs. The queued trigger remains valid after that land leaves or changes subtype. The filter uses the existing zone-entry event and subtype condition; it does not infer the player who caused the entry (CR 305.4–5, 603.6a–b).
* `AnotherCreatureYouControlEnters().HasKeyword(keyword)` checks the entering creature's effective keyword after entry effects and continuous effects apply. The zone-entry event creates the trigger with that result; a later keyword change or zone move does not change the pending trigger. The source must be an active battlefield permanent, and an opposing or self entry does not match (CR 603.6a–b).
* `EvtZoneChange.ActorID` records a known player who caused battlefield entry. It is separate from `PlayerID`, which records the entering permanent's controller. Land play, permanent spell resolution, and engine effects that know the acting player supply the actor. `PutOnBattlefieldBy` supplies it for an explicit effect entry. The older `PutOnBattlefield` leaves it unknown. `LandEnters().WithActor(binding)` matches only an entry with a known actor and saves that player in the pending trigger. A later control change, zone change, stack copy, or game clone does not change the saved player (CR 603.2, 603.3a, 603.6a–b).
* `LeavesBattlefield(This)` matches the source's battlefield-to-any-zone event. Detection uses the pre-departure ability and controller, including a simultaneous departure. Phasing is not a zone change. The pending trigger keeps the old source incarnation if the card returns before resolution (CR 400.7, 603.6c, 603.10a).
* `CreatureDies().OfSubtype(subtype)` and related creature-death patterns use the zone-change trigger's pre-departure game view to test the creature subtype. The filter matches each qualifying death in a simultaneous batch and remains fixed after the dying card returns or LKI is cleared. A phased-out source does not observe the death (CR 400.7, 603.10a, 700.4).
* Battlefield-leave events carry the departing battlefield incarnation. `CreatureDies().DamagedByThisTurn()` compares that identity and the trigger source's incarnation with positive damage recorded after prevention and replacement. This keeps a returned card or source from inheriting an earlier damage match. A source that dies in the same batch can still observe the creature's death through the pre-event view. Phasing preserves incarnation; a phased-out source cannot observe the event (CR 400.7, 603.10a).
* `WhenDiesThisTurn(subject, body)` captures one battlefield incarnation when its instruction resolves. The delayed trigger observes a battlefield-to-graveyard event through the pre-departure batch view or departure LKI. It keeps the creating source and controller, including when that source dies with the subject. A returned card has a new incarnation and cannot match. A replaced move to exile or phasing does not match. The subscription ends during cleanup, after any matching pending trigger has been queued (CR 400.7, 514.2, 603.7, 603.10a, 700.4).
* `WhenLeavesThisTurn(subject, body)` shares that delayed-subscription and captured-body path. It matches the saved incarnation's battlefield departure to any zone, including after a destination replacement, and retains a pending trigger when source and subject leave simultaneously. Phasing is not a zone change. The subscription ends at cleanup and cannot follow a returned card (CR 400.7, 514.2, 603.7, 603.10a).
* `CreatureYouSacrifice()` matches `EvtSacrifice` after a successful creature sacrifice. The event keeps the sacrificer's player ID, the creature-type result, and the battlefield incarnation from before departure. `WithPermanent(binding)` saves that former incarnation for each pending trigger. It is absent as a battlefield object after sacrifice, but `DamageSourceOf(From(binding))` can use its departure LKI. A returned card has a new incarnation. A source sacrificed this way can trigger from its departure LKI. Trigger copies and game clones keep separate saved values. Destruction and phasing do not emit this event (CR 400.7, 603.2g, 603.3a, 701.21a).
* `OpponentLandBecomesTapped().OfSubtype(subtype)` uses the normal tap-transition event. It checks the current land type, subtype, and opponent relationship when the land becomes tapped. A second tap of an already tapped land produces no event. Later control, subtype, or zone changes do not undo the pending trigger; it keeps its triggering controller (CR 603.2, 603.3a).
* `ArtifactBecomesTapped()` uses the same tap-transition event and checks artifact type when the event occurs. Optional `ByOpponent` compares the tapped artifact's controller with the watcher controller at that event; later control changes do not change the pending trigger. `WithArtifact` saves the artifact's battlefield incarnation, and `WithController` saves its event-time controller. The pending trigger keeps both through later source or artifact movement and stack copying. A returned or phased-out artifact is absent from `From` at resolution; the saved player remains available through `FromPlayer` (CR 400.7, 603.2e, 603.3a).
* `ArtifactAbilityActivatedWithoutTapCost()` uses `EvtAbilityActivated` after successful payment. Its flag records whether `{T}` was in the activation cost, including mana activations. The pattern checks the artifact's event-time type and can restrict the artifact to the permanent attached to its source. `WithController` captures the artifact controller even when the activation sacrifices the artifact; `WithArtifact` captures a live battlefield incarnation when present. Failed activations do not emit this event (CR 602.2, 603.2g).
* `EvtAbilityActivated.SourceIncarnation` records the source's battlefield incarnation when activation begins. Ordinary, mana, and planned mana activations send the event only after successful payment. A sacrifice cost can remove the source before dispatch; the event still carries the old incarnation. A card that returns has a new incarnation and cannot replace the event source. `PlayerID` remains the activator, which can differ from a later controller (CR 400.7, 602.2, 603.2g).
* `BecomesTapped(This)` uses the same tap-transition event for the source permanent. Entering tapped, tapping an already tapped source, and a tap event for another permanent do not match. A pending trigger retains its original controller and source incarnation after departure or return (CR 603.2e, 603.3a).
* `BecomesTargetOfSpell(This)` reads `EvtBecomesTarget` after target announcement. It requires the source as target and a spell event; an activated ability event does not match. The pending trigger keeps its controller and source incarnation after the source leaves or returns. Phased-out or absent sources cannot trigger (CR 603.2, 603.3a, 603.6c).
* `BecomesTargeted(This)` reads the same event for spells and activated abilities. `ByOpponent()` compares the event controller with the source controller when the target is announced. `WithController(binding)` saves `EvtBecomesTarget.PlayerID` in the pending trigger's instance frame. A later control change, source departure or return, stack copy, or game clone does not change the saved player. A phased-out or absent source cannot trigger (CR 603.2, 603.3a).
* `EvtSpellCast.TargetID` identifies the spell stack object that was cast; `SourceID` remains the source card ID for older event consumers. This distinction matters when another stack copy has the same source card ID. `SpellCast().OfColor(color)` checks that exact object's effective stack color after cast costs are paid (CR 601.2i). A stack color change or printed color override counts. The queued trigger keeps its event-time controller after the source leaves or returns. A copied stack object does not itself cause a cast event.
* `CastContext.ControllerCreatureSubtypesAtCast` records subtypes of creatures controlled when the cast finishes, after costs are paid (CR 601.2i, 608.2i). `IfCast(ControlledCreatureSubtypeAtCast(subtype), effect)` reads that spell snapshot during resolution. A later creature departure, control change, or phase change does not change it. Stack copies and game clones hold separate snapshot maps.
* `CastContext.RevealedAtCast` records independent card snapshots from paid reveal costs when casting finishes. `IfCast(RevealedSubtypeAtCast(subtype), effect)` reads that spell snapshot during resolution. Later card movement or changes to the original card do not change it. Stack copies and game clones copy the snapshot cards so that each cast context stays independent.
* `SpellCast().ByOpponent()` compares the cast event's player with the source controller at detection, not with its controller at resolution. `WithCaster(binding)` saves that player in the pending trigger's resolution instance. Later source control changes, game clones, and stack copies do not change the saved player (CR 603.3a, 603.5). A failed cast emits no cast event.
* `SpellCast().NotOwnedByCaster()` compares the exact cast stack object's card owner with the event caster after casting completes. It requires a matching stack ID, source card ID, and caster. The owner is independent of the spell controller and the watching permanent's controller (CR 108.3, 112.2, 601.2i). Copies of spells do not produce cast events.
* `SpellCast().OfType(cardType)` checks the card types of the spell stack object identified by the cast event, after casting completes (CR 601.2i). A stack copy with the same source card ID cannot supply the type. A copied spell does not emit a new cast event. The filter uses the existing stack and trigger services.
* `SpellCast().OfEnchantmentSubtype(subtype)` checks that same cast stack object for enchantment type and its named enchantment subtype. A different stack object with the same source card ID cannot supply the subtype. The filter uses the spell card's stack characteristics and the shared subtype-family validation (CR 205.3c–d, 601.2i).
* `OpponentDrawsCard(binding)` matches each completed card draw by a player other than the source controller at detection. Each draw has its own pending trigger and saved player. A replaced draw or an empty-library attempt emits no draw event. The binding stays with its trigger through control changes, source departure and return, copying, and game cloning (CR 121.2, 603.2g, 603.3a).
* `PlayerDrawsCard(binding)` uses the same completed-draw event and pending-trigger capture for either player. It does not filter against the source controller. Each saved drawer remains with its trigger through control changes, source departure and return, stack copying, and game cloning (CR 121.2, 603.2g, 603.3a).
* A completed `EvtCardDrawn` carries the drawn card ID and its hand-visit ID. It fires once per successful draw, after the card enters hand. `ControllerDrawsCardInto(player, card)` saves both values in the pending trigger. `FromCard(card)` resolves only while that hand visit remains current; a card that leaves and returns is absent. The controller-only pattern does not give a source access to an opponent's hidden drawn card. Replaced draws and empty-library attempts do not emit this event (CR 121.2, 121.5–121.6, 603.2g).
* `ReceivesDamage(This, amount).WithSourceController(controller)` matches each positive `EvtDamageDealt` whose final permanent recipient is the battlefield source. The damage service emits the final amount after replacement and prevention and the captured damage source controller. Each event has its own pending trigger and bindings. Full prevention, zero damage, another recipient, or a phased-out source does not match. Source or recipient departure after detection does not change the saved values (CR 120.8, 603.2g, 603.3a).
* `DealsCombatDamageToPlayer(This, player).WithAmount(amount)` matches each positive combat damage event from the battlefield source to a player. The event's combat flag excludes spell and ability damage. The damaged player and final amount stay in the pending trigger after source departure or return. Full prevention sends no damage event (CR 120.2a, 120.8, 510.2, 603.2g).
* `OneOrMoreCreaturesYouControlDealCombatDamageToPlayer().WithPlayer(player)` reads the shared `EvtCombatDamageDealt` event, which is grouped by damaging-creature controller, recipient player, and combat damage step. Partial prevention leaves one event if positive damage reaches that player; full prevention leaves none. The recipient binding belongs to the pending trigger, so stack copies and game clones keep independent captures. The event is available to battlefield abilities only (CR 510.2, 603.2g).
* `.WithoutSubtype(subtype)` on that pattern tests the source snapshots saved when positive combat damage is dealt. A source that leaves, returns, phases, or changes subtype after dealing damage cannot change the pending group. One qualifying source is enough; other sources in the same group can have the excluded subtype. The shared per-source amount map remains available to legacy callers (CR 400.7, 510.2, 603.2g).
* `Draw(You, amount, DrawAmountInto(binding))` uses the shared individual draw action and its replacement pipeline. It counts only successful draws from that instruction, including a partial draw before the library empties. A replaced or empty-library attempt does not increase the count. The result is stored in the current resolution frame, and an absent player writes zero (CR 121.2, 121.4, 121.6).
* `AnnouncedTargetCount(group)` reads the original target specifications and choices on the resolving stack object. The count stays fixed when a target becomes illegal, leaves and returns, or phases out. A copy keeps the count when its controller changes targets. It reads one declared permanent target group, so another group cannot change the value. If all targets are illegal, the object does not resolve and the value is not read (CR 601.2c, 608.2b, 707.10).
* `ForEachSavedCard` iterates the original card result in saved order. The count does not change when a later instruction moves one of those cards. Each member retains its captured zone visit, so a card that leaves and returns is absent through `FromCard`. Bindings produced inside one iteration do not carry into the next or out of the loop. Each body runs as a separate instruction; the loop is not a batch action (CR 400.7, 608.2c, 121.2).
* Typed `Attacks(This)` listens to the declared-attacker event. Typed `Blocks(This)` listens to the first declared-blocker event for the source in that combat. A second attacker blocked by the same creature does not create a second source-blocks trigger. Entering attacking or blocking does not match these events. Trigger resolution keeps the source incarnation and triggering controller; a new incarnation cannot receive the old trigger's source modification.
* `OneOrMoreCreaturesAttack()` reads the completed attacker-declaration group for either player. A positive declaration makes one trigger even if several creatures attack. `WithCount` saves the number declared for that trigger instance. An empty declaration or a creature that enters attacking does not match. The battlefield source must be present and not phased out when the declaration occurs; a pending trigger keeps its controller after the source leaves or returns (CR 508.1m, 508.3a, 603.2c).
* `OneOrMoreCardsLeaveYourGraveyard()` reads the existing `EvtCardsLeftGraveyard` group for the battlefield source controller at detection. `MoveCardsFromGraveyard` emits one positive group event for cards it actually removes; separate `MoveFromGraveyard` calls emit separate events. `WithCount` saves that event's amount in the pending trigger frame. A phased-out source does not detect an event. A pending trigger keeps its original controller and amount when its source leaves or returns (CR 603.2c, 603.3a).
* `EachCreatureYouControlAttacks()` reads each completed `EvtDeclaredAttacker` for a creature controlled by the battlefield source's controller at detection. Each event makes one trigger. `WithAttacker` saves the attacker's battlefield incarnation, and `WithDefendingPlayer` saves the player in the event's `TargetID`. Source and attacker phasing suppress new triggers. A saved attacker does not follow a card that leaves and returns. The current combat model records only player defenders (CR 508.1m, 508.2a).
* The engine emits declared-blocker pair events after minimum-blocker and required-block checks finish. A removed pair does not emit a blocks event. Each new legal pair emits one event. The first event for a blocker in combat has `Flag=true`; later pair events for that blocker have `Flag=false`. Normal combat and combat simulation use the same rule (CR 509.1i, 509.3a–b).
* `BlocksCreature(This)` consumes each legal declared-blocker pair, including later pairs with `Flag=false`. Its subtype filters inspect the attacking creature at detection. `WithOpponent` saves that attacker's battlefield incarnation in the pending trigger. A returned or phased-out attacker is absent when the trigger resolves. A creature entering blocking does not emit a declared-blocker pair (CR 509.3b, 509.3f, 509.4).
* `EvtCombatPair` reports each new legal attacker/blocker pair from both creature perspectives. The attacker event has `Flag=false`; the blocker event has `Flag=true`. A creature entering blocking reports only the attacker perspective. `BlocksOrBecomesBlockedByCreature(This)` matches each event for its battlefield source, filters the opposing creature at detection, and optionally captures that creature's incarnation in the pending trigger. The capture becomes absent after departure and return or while phased out. The event adds no legacy target slot and does not change existing declared-blocker or blocked-by triggers (CR 509.3b, 509.3d, 509.3f, 509.4).
* Typed `BecomesBlocked(This)` listens to the first blocked-state transition for the source attacker (CR 509.3c). It uses the existing event after legal blocker declaration, or when a creature enters blocking an unblocked attacker. `EvtCreatureBecomesBlocked.TargetID` stores the defending player at the transition; for a permanent defender, it stores that permanent's controller at that time. `WithDefendingPlayer` saves this player in the pending trigger frame. The pattern has no blocker reference; blocker-specific events require a separate pattern. The pending trigger retains its controller, source incarnation, and saved defender through departure, phasing, copying, and cloning.
* `EvtAttacksUnblocked` reports each live attacker with no legal blocker after blocker declaration (CR 509.3g). It includes a creature that entered attacking. A blocked attacker does not match when its blockers later leave combat. The event records the attacker, its controller, and its defending player. `AttacksUnblocked(This)` uses the shared pending-trigger path and saves the defending player when requested.
* `BlocksOrBecomesBlockedBy(This).OfSubtype(subtype)` listens to `EvtBlockerGroup` after legal blocker pairs settle. One event lists the new opposite creatures for each source and direction. The pattern requires at least one current creature of the subtype in that list and creates one trigger for the group. It tests characteristics when the event occurs; later changes do not undo the trigger. A creature entering blocking supplies only the attacker's event, since the entrant did not block (CR 509.3e–f, 509.4).
* `AnotherCreatureYouControlDies()` matches a battlefield-to-graveyard zone change for a creature other than the trigger source. It uses the dying creature's pre-departure type and controller. In a simultaneous death batch, a source that dies with another controlled creature triggers once for that other creature; the pending trigger keeps its original controller and source incarnation.
* `.PowerOrToughnessAtMost(n)` on a creature-death pattern tests either characteristic in the pre-departure object view. Continuous effects apply before the snapshot. A simultaneous source death still observes each qualifying subject. Replacement to exile produces no matching death event (CR 603.2g, 603.10a).
  * `OpponentCreatureDies()` uses the same pre-departure event and checks the dying creature's controller against the battlefield source controller at detection. Ownership does not decide the match. Each matching creature, including a token in a simultaneous death batch, creates one trigger. `.WithController` saves the former controller in that trigger instance (CR 603.2c, 603.10a).
  * `AnotherCreatureDies().Nontoken()` uses the same death event and source-exclusion check for either controller. The nontoken filter reads the dying permanent's pre-departure token status, including when a token ceases to exist after the zone change (CR 111.7, 603.10a). Simultaneous source and subject deaths each use their original battlefield identities.
  * `.WithoutColor(color)` on a creature-death pattern uses the same pre-departure object view. A creature with the excluded color does not match even if it also has another color. Color changes already applied before departure count; a later card incarnation cannot change the queued trigger (CR 105.2, 603.10a).
* `CreatureYouControlEnters()` checks each battlefield entry event against the live entering permanent after entry effects and continuous effects apply (CR 603.6a–b). It includes the trigger source's own entry. `AnotherCreatureYouControlEnters()` adds source exclusion. Both check creature type and controller at detection and do not target the entrant. Entry filters and captures share the same event path. A queued trigger keeps its source incarnation and controller through source departure, return, and stack copying.
* `.OfSubtype(subtype)` on that entry pattern checks the entering permanent's current creature subtype at event detection. Later subtype changes do not change whether the trigger was queued. Registration rejects an empty subtype or a known subtype from another family.
* `.WithPermanent(binding)` on this entry pattern saves the entering battlefield incarnation when the event is detected. The pending trigger carries that value through stack copying and game cloning. `From(binding)` affects only that incarnation while it remains present and phased in; a returned card is a new object (CR 400.7). The binding is not a target.
* `EnchantmentYouControlEnters().OfSubtype(subtype)` checks the entering permanent's effective enchantment subtype when the battlefield entry event occurs (CR 205.3h, 603.6a–b). It shares the controlled-enchantment trigger and optional entering-permanent capture. A later subtype change does not cancel the queued trigger. Registration accepts only one known enchantment subtype; the filter does not test a later resolving condition on the saved permanent.
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
* `AttachedAuras(host)` reads the live attachments of the host's battlefield incarnation. It includes an attachment only when that object is an unphased Aura and its attachment still names that incarnation. It does not test the Aura's controller. A returned host cannot inherit an earlier attachment (CR 303.4b, 303.4e, 400.7). `CountPermanents` reads this set again during each static layer evaluation, so an Aura attachment or departure changes the Layer 7c bonus without a new trigger (CR 611.3a, 613.1g).
* **Temporary Effects (`Until`)**: Effects modifying characteristics or control capture affected objects **at resolution time** (CR 611.2c). Permanents entering the battlefield later are not included.
* **Typed Control Change**: `ChangeController(PermanentRef, PlayerRef)` installs only through `Until(EndOfTurn, ...)`. It captures the target incarnation and player at resolution, then applies in Layer 2 timestamp order. Source departure does not end it; a returned target is a new object. The control prohibition and expiry reconciliation use the shared layer service.
* **Static Effects (`While`, `Static`)**: Static continuous effects continuously recalculate their matching set (CR 611.3a). Permanents entering later receive the boost or ability dynamically.

---

## 5. State-Triggered Abilities and Simultaneous Deaths (CR 603.8, 603.6c)

`AnotherCreatureEnters()` uses the battlefield entry event for a creature under any controller. It excludes the watcher itself by source card ID. The event check uses the entering permanent's effective type after entry; `.WithPower` saves its power in the pending trigger before a later change can affect that value. A phased-out watcher does not detect the event. The pattern uses the existing entry trigger path and does not create an entry replacement (CR 603.6a–b).

* **Opponent Land Tap Event (CR 603.2e)**:
  * `OpponentLandBecomesTapped().WithLand(binding)` captures the tapped land's battlefield incarnation when its tap transition occurs. The pending trigger keeps that reference through source departure, controller change, stack copying, and game cloning. `From(binding)` is absent if the land leaves and returns or phases out before resolution. The event has no target or payment step.
* **State Triggers (`WhenState`)**:
  * Trigger immediately when their pure `StateCondition` evaluates to true, including between instructions during resolution.
  * `SourceTappedCondition` and `SourceUntappedCondition` read the current battlefield incarnation. An absent or phased-out source satisfies neither condition. Resolution and static checks reject a different incarnation of the same card when the source incarnation is captured.
  * `PermanentIsTapped(This)` and `PermanentIsUntapped(This)` use that source contract. The `AttachedPermanent` forms follow the source's current attachment. They require the current source and attached host incarnations, and treat an absent or phased-out object as neither tapped nor untapped. Reattachment changes the subject of the next check. Static layer checks use the same predicate.
  * `SourceBlockingCondition` reads current combat membership for the battlefield source incarnation. An absent or phased-out source does not match. Phasing removes combat membership. A returned card cannot inherit the old block because combat records the blocking incarnation. Departure keeps the old relationship available to leave-trigger resolution, even if that card returns before the trigger resolves. Removing a blocker preserves the attacking creature's blocked status.
  * `PermanentIsAttacking(This)` and `PermanentIsAttacking(AttachedPermanent)` read current attacker membership for the selected battlefield incarnation. The combat service records attacker incarnations for declared attackers and creatures that enter attacking. An absent, phased-out, removed, or returned object does not match. Clones copy attacker identity independently. The older card-ID combat queries remain available to existing callers.
  * `PermanentIsBlocking(This)` and `PermanentIsBlocking(AttachedPermanent)` read current blocker membership for the selected battlefield incarnation. The combat service records blocker incarnations for declared blockers and creatures that enter blocking. An absent, phased-out, removed, or returned object does not match. The Aura form also requires its current host incarnation. Clones copy combat membership independently. A removed blocker does not make its attacker unblocked.
  * `IfPermanentAttacking` and `IfPermanentBlocking` read the same incarnation-keyed combat service during resolution. Their singular subject can be a legal target or a saved permanent. A phased-out, absent, returned, or removed object skips the branch. Each branch checks combat membership when it runs, so earlier instructions can change the result. These branches do not change the static state-condition contract (CR 400.7, 506.4, 608.2h).
  * `SourceAttackedThisTurnCondition` reads the turn tracker's declared-attack history for the current battlefield incarnation. It is false before the attack, while the source is absent or phased out, after leave and return, and after turn reset. Game clones keep separate history. The older card-ID attack query remains available to existing callers.
  * A trigger stays **suppressed** while its original instance is pending, on the stack, or resolving.
  * Once that instance leaves the stack (resolved, countered, or removed), the ability rearms and triggers again if the condition is still met.
  * A permanent that leaves and returns has a new instance and can trigger independently.
* **Simultaneous Batch Observation**:
  * `EvtAttackersDeclared` is sent once after each legal declaration group that contains at least one newly declared attacker. `Amount` is the number newly declared, excluding creatures already attacking or put onto the battlefield attacking. The normal turn path and simulated attack path send the same event. A zero declaration sends no event. The typed group pattern checks the event's attacking player against its source controller at detection and saves any count in the pending trigger instance.
  * `EvtCreatureBecomesBlocked` reports the attacker when its combat group changes from unblocked to blocked (CR 509.3c). Declared blockers are checked after restrictions and required blocks settle, so two blockers in one declaration produce one event and a removed illegal block produces none. A creature entering blocking also causes the transition if the attacker was unblocked. An already blocked attacker does not produce a second event. `EvtDeclaredBlocker` remains the per blocker-and-attacker event; its flag still describes the first event for that blocker.
  * `EvtCreatureBlockedBy` reports each new attacker and blocker pair (CR 509.3d). It names the attacker in `SourceID` and the blocker in `TargetID`. Declarations send it only after restrictions and required blocks settle; a removed illegal block sends no pair event. A creature entering blocking sends a pair event even when the attacker was already blocked. The event uses the attacker's event-time controller. `EvtDeclaredBlocker` and the once-per-attacker `EvtCreatureBecomesBlocked` keep their separate contracts.
  * `BecomesBlockedBy(This).WithoutSubtype(subtype)` checks the blocker's current creature subtype at pair-event detection (CR 509.3f). It does not match an absent or phased-out blocker. A later subtype change does not alter the pending trigger. The optional blocker binding retains the event-time battlefield incarnation.
  * A union of destruction filters uses one `DestroyEach(MatchingPermanents(Or(...)))` action. Each matching permanent is considered once, so one regeneration shield suffices even if multiple types match. State triggers cannot observe separate type batches.
  * `BouncePermanentsToHand` uses the shared zone-change batch. It checks each selected battlefield incarnation once, returns nontoken cards to their owners' hands, and dispatches leave events after all selected objects move. State triggers cannot observe an intermediate return. `ReturnEachToHand` selects its set before this action; phasing and a different incarnation exclude an object (CR 400.3, 400.7, 603.6c).
  * `ExilePermanents` uses the same zone-change batch for one selected group. It checks battlefield incarnation and phasing for each member, then moves each accepted object through `ExilePermanent`. Leave events and state-trigger checks see the completed group. Tokens leave but do not remain in exile. `ExileEach` selects the set once before the move. This contract does not add a general zone-change replacement service (CR 400.6, 400.7, 603.6c).
  * `TapEach` selects current battlefield incarnations once, discards duplicates and permanents that are already tapped, phased out, absent, or replaced by a later incarnation, then uses `tapPermanentsTogether`. All selected permanents become tapped before any tap event fires. Each actual transition produces one event. `ForEach` remains sequential and has a different observation contract (CR 603.2e).
  * `UntapEach` selects current battlefield incarnations once, discards duplicates and permanents that are already untapped, phased out, absent, or replaced by a later incarnation, then uses `untapPermanentsTogether`. A stun counter replaces one untap attempt for its permanent. All successful untaps finish before any untap event fires; each success produces one event (CR 122.1g, 701.26b).
  * `ManaValueAtMost(set, value)` evaluates its limit in the current resolution frame when it selects the set. It checks each present member's mana cost using the existing permanent mana-value rule (CR 202.3), then passes the selected members to the consumer. With `DestroyEach`, selection occurs before one simultaneous destruction batch. A paid-life number binding remains local to its spell or ability instance; invalid or unbound reads fail registration.
  * Mass destruction (`DestroyPermanents`) preserves the pre-destruction battlefield snapshot for leave-the-battlefield triggers.
  * `DestroyEach` and `DestroyEachWithoutRegeneration` can save `CreaturesDiedThisWayInto` in the current resolution frame. The count includes only selected creatures moved directly to a graveyard by that one simultaneous destruction instruction (CR 701.8). It excludes indestructible, regenerated, absent, phased-out, and noncreature objects. Other deaths during triggered abilities or later instructions do not change the saved count. A game clone has its own result frame.
  * `DestroyWithResult` and `DestroyWithoutRegenerationWithResult` use the same destruction batch for one battlefield incarnation. `DestroyedThisWayInto` saves one only when that instruction moves its subject to a graveyard as a destruction. It saves zero for an absent, phased-out, indestructible, regenerated, or replaced destruction. The result belongs to the resolving instance; a returned incarnation cannot satisfy an old reference (CR 400.7, 701.8).
  * Singular `DestroyWithoutRegeneration` and legacy positional/string-bound no-regeneration executors use a one-permanent batch with the same action-scoped prohibition.
  * `DestroyEachWithoutRegeneration` uses the same batch with an action-scoped regeneration prohibition (CR 701.8, 701.19c). It does not add keywords to permanents or their last-known characteristics. Indestructible and other destruction replacements still apply.
  * Zero-toughness state-based deaths share the same observation batch.
  * Conditions evaluate against the pre-batch state; trigger effects resolve against the current game state.
* **Creature-Death Controller Capture**:
  * `CreatureDies().WithPermanent(binding)` and the other permanent-death patterns save the departing battlefield incarnation at event detection. The value is a former permanent reference, not the graveyard card. Battlefield actions find it absent after death; `DamageSourceOf(From(binding))` can use durable departure LKI for a later damage instruction. A returned card has a different incarnation. Simultaneous death, stack copies, and game clones keep the captured identity in each trigger frame (CR 400.7, 603.10a, 608.2h).
  * `PermanentDies().OfType(type)` uses the same battlefield-to-graveyard event and departure LKI as creature-death patterns. The type filter checks the object before it leaves. `WithController` saves the former controller from that event, including when the source leaves in the same batch. A replacement that sends the object to exile produces no death match. The pattern observes one event per departing permanent (CR 603.2c, 603.10a).
  * The damage system also records positive damage to a permanent this turn by source and recipient battlefield incarnation. Prevention and absent or phased recipients do not add a record. A source that leaves and returns has a different key; phasing keeps the same key. Clone copies do not share this record. Cleanup clears it only after all cleanup priority rounds. This internal record is a prerequisite for a future typed “damaged by this source this turn dies” event filter; the current legacy card-ID query remains available to unmigrated callers.
  * `PermanentDies().YouControlled().NotSacrificed()` checks the departing permanent's event-time controller and the completed zone-change action. The sacrifice path marks its zone-change event; destruction and state-based deaths do not. A replaced move to exile is not a death. A source that dies in the same batch observes the other matching deaths, and each pending trigger keeps its original controller. The same filters can refine creature-death patterns. Registration rejects repeated filters (CR 603.2c, 603.10a, 701.21).
  * `CreatureDies().WithController(binding)` and the controlled-creature variants save the dying creature's controller from the battlefield-to-graveyard event. The event records that player before the permanent departs (CR 603.10a).
  * `CreatureDies().WithOwner(binding)` and the other creature-death variants save the dying creature's owner from the event-time object or departure LKI. The pending trigger keeps the player through simultaneous death, source departure, stack copying, and game cloning. The saved owner is independent of the trigger controller and the dying creature's controller (CR 400.3, 603.10a).
  * Each pending trigger carries its own event. The bound player is installed in that trigger's resolution frame before its body runs. Stack copies keep the captured player even if their controller changes. The participant is not a target.
  * Phased-out sources do not observe the event. A source that dies in the same destruction batch can observe the other deaths through the existing pre-batch trigger snapshot.
* **Creature-Death Toughness Capture**:
  * `CreatureDies().WithToughness(binding)` and the controlled-creature variants save the dying creature's toughness when the trigger is detected. Detection uses the pre-batch permanent or its departure LKI, including continuous effects (CR 603.10, 608.2h).
  * `CreatureDies().WithPower(binding)` and the other creature-death variants use the same detection snapshot for power. Each pending trigger saves its own number before stack placement. Registration rejects this capture on a general permanent-death pattern.
  * The pending trigger stores the number before stack placement. Resolution reads its own number binding, so cleanup, a later incarnation of the same card, stack copies, and game clones do not change the saved value. This capture also applies to a creature's own death trigger.
* **Creature-Entry Power Capture**:
  * `AnotherCreatureYouControlEnters().WithPower(binding)` reads the entering creature's effective power when the event is detected (CR 603.2, 603.6b). Each pending trigger saves its own number. Later power changes or a new battlefield visit do not change it; stack copies and game clones copy it with the trigger frame.

---

## 6. Replacement and Prevention Pipeline (CR 614–616)

`PreventCombatDamageBy` and `PreventCombatDamageByEach` capture each selected battlefield incarnation when the instruction resolves. The replacement checks the captured source identity on combat damage to any player or permanent. It does not depend on the source's later controller or current battlefield presence. A card that leaves and returns has a new incarnation, while phasing preserves one. The replacement expires after the end-of-combat step, before a later combat; it does not prevent noncombat damage or damage to the selected source from another source (CR 400.7, 615.1–615.3).

* **Cast-payment entry counters**: The battlefield entry path passes the entering permanent spell's own distinct colored mana count to self-entry counter abilities. A direct entry passes zero, including when another spell causes it. `EntersWithCounters` places its counters before the entry event through `AddCountersWithReplacement`, so applicable counter replacements see the proposed placement. Failed spell payment cannot start this entry action.
* **Permanent-set counter placement**: `AddCountersTogether` prepares a separate counter action for each distinct, live battlefield incarnation. The replacement pipeline transforms each action before any counter count changes. The service then places all final counters through game-local mutable permanents. A stale, phased-out, or duplicate selection does not add counters, and game clones do not share the resulting counter state (CR 400.7, 122.2, 614.1c).

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

Positive damage events keep the final damage action's source snapshot through event dispatch, including a simultaneous damage batch. `ReceivesDamage(This, amount).WithSource(source)` captures that snapshot when the trigger is detected. The pending trigger can use its source after a permanent leaves and returns or a spell leaves the stack. The snapshot includes the source controller and characteristics used by later damage; it does not find a new object with the same card ID. Prevention and zero damage send no damage event (CR 400.7, 603.2g, 608.2h).

Positive damage to a permanent records `EvtDamageDealt.TargetIncarnation` after replacement and prevention. `DealsDamage(This, amount).WithPermanent(recipient)` saves that identity in the pending trigger. If the permanent leaves and returns before resolution, `From(recipient)` is absent. Damage to a player and fully prevented damage do not produce a permanent recipient binding (CR 400.7, 120.4b, 603.2g).

`DealsDamage(This, amount).WithRecipientPlayer(recipient)` saves the final player recipient in the pending trigger. A final permanent recipient saves no player, including when a player damage action is redirected to a permanent. Full prevention creates no event or pending trigger. The player ID belongs to each event's frame and survives trigger copies and game cloning (CR 120.4b, 120.8, 603.2g).

`DealsDamage(This, amount).WithSource(source)` and its `AttachedPermanent` form save the final damage source snapshot for either a player or permanent recipient. The pending trigger uses the captured incarnation and characteristics after departure or return. Full prevention and zero damage do not produce a trigger. Each simultaneous damage event keeps its own capture (CR 120.4b, 120.8, 603.2g).

`DealsDamageToPlayer(This, player).WithSource(source)` uses the same snapshot at detection. It saves the source in the pending trigger's frame after final positive damage to a player. Prevention and recipient redirection to a permanent produce no match. A returned source is a new incarnation; the saved source keeps the old controller and characteristics for subsequent damage. Stack copies and game clones retain independent frames.

The `RemoveAnyCountersCost` activation cost chooses a count during payment preparation and locks that count and the source battlefield incarnation. Validation pays the same count on an isolated game. Commit rejects a departed, returned, phased-out, or counter-short source. The paid-number result becomes available only after the chosen removal succeeds; stack copies and game clones retain separate captured values. Zero is a legal choice when the source is present. This cost uses the existing counter-removal payment path and does not add a resolution-time removal action.

Direct fixed counter-removal costs now lock the source incarnation even when no output binding is requested. Commit checks direct fixed and chosen-count costs before mana abilities, mana spending, and life payment. It checks their combined count for each counter type, so separate affordable removals cannot cause a partial total payment after a counter shortage. Failed preflight removes no counters and publishes no paid-number result. Preparation copies each cost; reused declarations and cloned games keep separate locks. A zero fixed count still requires the original present incarnation. This uses the existing payment service (CR 118.3, 400.7, 601.2h, 602.2b, 702.26b). It does not establish general payment-order choice, wrapper locking or rollback after a mutation during payment itself.

Direct uncaptured X counter payments now fix the announced amount through the existing action-cost context. Direct payment transactions fix the current X during preparation. Positive X uses the fixed-counter executor, source-incarnation lock, and combined-total commit check. Changed ambient X cannot change the prepared amount. Zero requires no counter action and remains payable without a source; this preserves the uncaptured constructor's existing behavior. Captured X and ordinary fixed-zero costs retain their present-source requirements. No output, LKI, target, replacement, or simultaneous-action contract is added. Optional/alternative wrapper locking and mutations during commit remain separate work.

Counter-removal payment declarations validate the counter enum before gameplay. Fixed, announced-X, and chosen-count forms reject counter types outside `[0, core.NumCounters)` at registration; fixed counts also reject negative values. Inspection includes uncaptured fixed and X costs, including nested optional, alternative, additional, and resolution-payment declarations. Direct payment checks reject invalid types before array access, and chosen-count preparation rejects them before asking for a number. Invalid payment preparation spends no mana, life, or counters. This is declaration safety for the supported counter names, not a new counter rule or payment-order service (CR 122.1, 601.2h, 602.2b).

`RemoveCountersCostInto` uses that payment transaction for a fixed count. It locks the source incarnation during preparation and saves the count only after commit removes the counters. A departed, returned, phased-out, or counter-short source cannot pay. Zero is valid if the source is present. The saved number belongs to the activation stack instance and follows its copies and game clones. The existing `RemoveCountersCost` signature remains available to production callers.

An exiled card has a new visit identity each time it enters exile. `ExileMatchingCardFromGraveyardCostWithCard` locks the chosen graveyard visit during payment preparation and saves the resulting exile visit only after the total cost commits. `FromCard` finds that card while the same exile visit remains present and visible to the resolving controller. Removing and exiling the same physical card again does not restore the old reference. Stack copies and game clones keep the saved visit, while failed payments keep no result (CR 400.7j, 601.2h, 602.2b).

* **Nonpositive Damage Cutoff (CR 120.8)**:
  * If a damage action's final amount is reduced to zero or less by prevention or modifiers, the damage action terminates before further replacement checks or execution.
  * Zero/negative damage produces **no damage event, damage history, callback, or marked damage**.
  * Later replacements cannot resurrect or increase an absent damage event.
* **Player Life-Loss Trigger (CR 119.2–3, 603.2g, 603.3a)**:
  * `PlayerLosesLife(player).WithAmount(amount)` matches each positive `EvtLifeLost` for either player. `OpponentLosesLife` adds the event-time opponent check. Direct life loss and damage that causes life loss use the same completed event. Zero loss and fully prevented damage create no matching trigger. Each player in a simultaneous life-loss action has an independent pending trigger with its player and amount. Phasing suppresses the battlefield source; source departure or control change after detection does not change saved event values.
* **Controller Damage Trigger (CR 120.4, 603.2g, 603.3a)**:
  * `ControllerIsDealtDamage(amount)` matches each final positive damage event whose recipient is the battlefield source's controller when damage occurs. Redirection to a permanent, full prevention, zero damage, life loss without damage, and a phased-out source produce no match. Each event in a simultaneous damage batch can create a separate trigger.
  * Each positive `EvtDamageDealt` carries the captured damage source's `SourceIncarnation` for either player or permanent recipients. A source that leaves and returns keeps its old incarnation in the event. Prevention that removes all damage sends no event. Spell sources have no battlefield incarnation, so this field is nil for them. Event consumers must keep the captured source identity when they queue a trigger (CR 400.7, 120.2b, 120.8, 603.2g).
  * The trigger saves the final damage amount in its resolution frame. A later source control change or departure does not change the trigger controller or saved amount. Stack copies and game clones keep separate frames.
  * `ControllerIsDealtDamage(amount).WithSource(source)` saves the final damage event's source snapshot in that frame. A later departure, return, or phase change does not replace the saved source. Damage from it keeps its captured controller and lifelink. Full prevention creates no source capture.
* **Attached Damage Trigger (CR 120.8, 603.2g, 603.3a)**:
  * `DealsDamage(AttachedPermanent, amount)` checks the current attachment when a positive damage event occurs. It accepts damage to a player or permanent. Prevention and replacement determine the final event amount before capture. A phased-out Aura cannot trigger. The queued trigger keeps its amount and controller if the Aura moves, returns, or changes attachment before resolution.
  * `DealsDamageToPlayer(AttachedPermanent, player).WithAmount(amount)` uses the same attachment check and also requires a player recipient. It captures the damaged player and final positive amount in each pending trigger. Damage to a permanent, full prevention, and damage while the Aura is phased out do not match. The bindings are event participants and do not declare targets.
  * `DealsDamageToPlayer(source, player).ToController().WithAmount(amount)` adds an event-time recipient check against the trigger controller. It accepts `This` or `AttachedPermanent`; a later control change does not alter the pending trigger or its saved player and amount. The filter uses the same damage event and does not match prevented damage.
  * `DealsDamageToPlayer(source, player).WithSourceController(controller)` saves `SourceControllerID` from the final positive damage event. For `AttachedPermanent`, this is the attached permanent's controller when it deals damage, which can differ from the Aura's controller. The binding survives source departure and later control changes in a pending trigger. It does not create a target or read the current controller at resolution.
* **Lifelink Outcome Routing**:
  * Damage from a permanent with lifelink causes its controller to gain that much life simultaneously with damage execution.
  * Life gained from lifelink routes through the event-aware life gain executor, emitting an `EvtLifeGained` with the actual positive amount after life-gain replacements.
  * `ControllerGainsLife(player).WithAmount(amount)` saves the positive final `EvtLifeGained` amount in the pending trigger. It does not read a later life total or the requested gain. Each trigger and stack copy keeps its own capture.
  * `PlayerGainsLife(player).WithAmount(amount)` uses the same completed gain event for either player. The saved player and amount belong to that event, regardless of later control changes or a copy's controller. A zero or consumed gain does not match (CR 119.9, 603.2g).
  * A collection gain captures its player set and amount once. It prepares each player's life gain modifier and replacement outcome before changing any life total. All positive gains change life totals before any gain event is sent, so event observers see the final totals. A zero or fully replaced gain sends no event.
  * The turn tracker totals positive `EvtLifeGained` amounts per player. `LifeGainedThisTurn(PlayerRef)` reads that total at effect resolution, including gain after replacement. Zero, consumed gain, and life loss do not add to it. Final cleanup clears the total after priority ends.
* **Damage Source Identity**:
  * A damage instruction captures source characteristics and controller before replacement and prevention. If a permanent leaves and the same card returns, its new incarnation does not change the old instruction's protection, lifelink, or deathtouch result.
  * A resolving ability reads its captured source incarnation from durable departure LKI. A resolving spell uses the stack object's controller. An unrelated source ID does not inherit the resolving source's LKI. Phased-out permanents do not supply a new source capture.
  * `CaptureDamageSource` supplies an immutable source value to `DealDamageToPlayerFromSource` and `DealDamageToPermanentFromSource`. The UUID-based entry points capture the source when called. Combat damage and fight keep the existing shared damage pipeline.
  * Each positive `EvtDamageDealt` carries `SourceControllerID` from the captured source, after damage replacement and prevention. Player and permanent recipients use the same field. It does not follow a source card that leaves and returns under another controller. A prevented or zero damage action sends no event.
  * The typed `DealDamageFrom(This, recipient, amount)` instruction reads the captured source incarnation from its resolution frame. It uses the same damage pipeline for player and permanent recipients. An absent recipient or a missing source snapshot causes no damage. A declared target keeps its recorded battlefield incarnation when resolution builds effect bindings, even if the card has returned.
  * `DamageSourceOf(permanent)` reads a typed permanent reference. A target or saved permanent binding retains its captured incarnation. A departed incarnation supplies damage-time source characteristics from durable departure LKI, even if the same card returns under another controller. Phasing alone makes the source absent. The adapter uses the shared damage executor and does not read a global last-action slot.
  * A one-permanent sacrifice cost can save its locked choice as a permanent binding during payment, including the ability source when the cost permits it. The committed spell or activation frame retains that battlefield incarnation after sacrifice. `From` remains absent because the permanent left; `DamageSourceOf(From(binding))` can use its departure LKI. A phased-out or returned source cannot satisfy a locked choice. A failed or stale payment produces no binding. Stack copies and game clones keep the saved identity in their own frames (CR 400.7, 601.2h, 608.2h).
  * `DamageAmountInto(binding)` records the final amount dealt by one typed damage instruction in its resolution frame. Prevention, replacement, redirection, and minimum-life restrictions apply before this value is written. An absent source or recipient and a zero action write zero. A redirected action keeps the captured damage source. The result does not report life lost, counters removed, or a group total.
  * `DamagePlayerInto(binding)` records the final player recipient only when positive damage is dealt to a player. The shared replacement pipeline can redirect permanent damage to a player or player damage to a permanent; the binding follows the final action. A prevented or absent action clears any prior value in that resolution frame.
  * `DamagePermanentInto(binding)` records the final permanent recipient's battlefield incarnation only when positive damage is dealt to that permanent. The snapshot is taken from the final action after replacement and before execution, so a later zone change cannot make it refer to a returned card. A player result, prevention, or absent action clears any prior value in that resolution frame.
  * `DamageSourceInto(binding)` records the source on the final executed damage action only when positive damage is dealt. `FromDamageSource(binding)` can use that source in a later instruction. The saved value keeps its controller, incarnation, and damage-time characteristics after departure or return. Prevention, zero damage, or an absent source or recipient clears the binding. Each resolution and game clone owns its saved value.
* **Planeswalker Damage Result**:
  * `Game.DealDamageToPermanent` runs damage replacement and prevention before it applies results. Actual damage removes up to that many loyalty counters from a planeswalker. It does not mark damage unless that permanent is also a creature. Fully prevented damage changes no loyalty and creates no damage history.
  * Legacy history queries still use physical card IDs. Incarnation-aware game-long player and permanent history and turn-scoped permanent history are separate records. Turn-scoped player history records each final positive damage result by source battlefield incarnation and player ID. `SourceDamagedOpponentThisTurnCondition` checks the current source incarnation against the ability controller's current opponent. `PermanentsDealtDamageByThisTurn` checks both source and current recipient incarnations against positive turn damage after replacement and prevention. It excludes phased-out or returned recipients; a departed source can still be read through its captured reference. Final cleanup clears the turn record; game clones copy it independently. General damage-condition bindings remain separate work.
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
  * `DealDamageEach(This, DamageRecipients(players, permanents), amount)` captures its source and amount once. It builds all assignments from the selected players and battlefield permanent incarnations, then uses the mixed batch. One variable-size declared permanent target group can contribute each legal, present target. Multiple variable-size groups in one ability remain blocked on recorded selection boundaries. `DamageTotalInto(binding)` stores the sum of amounts actually dealt by the final actions, after replacement and prevention, in the resolution frame. It counts redirected damage once and writes zero for an absent source or an empty batch.
  * `TargetDamageRecipients(name, minimum, maximum)` supplies a single mixed target group to `DealDamageEach`. Announcement uses the shared target-count and legality checks before payment. Resolution retains the recorded group and excludes phased, absent, or returned permanent incarnations while preserving legal player targets. The remaining assignments enter one damage batch, so observers see the complete result.
  * `TargetPlayerOrPlaneswalkerDamage(name)` uses the shared player-or-planeswalker target selector. A creature cannot be announced for this declaration. A targeted planeswalker must retain its battlefield incarnation through resolution; phasing or leave and return makes that target absent.
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
  * `LifePayCostInto` writes the committed fixed life payment to the stack instance's number bindings. The value is available to that spell or activated ability at resolution, including its copies. It is absent after failed payment. The binding does not read life loss from another action.
  * `LoseLifeEach(..., LifeLostInto(binding))` writes the total life lost by that set instruction to the current resolution frame. The shared life service changes all selected life totals before it dispatches `EvtLifeLost` events and measures the result before event handlers can change life again. An empty set or nonpositive amount writes zero. The result is separate from life payment and damage results.
  * `LoseLifeForWithResult(player, amount, LifeLostInto(binding))` uses the same life service for one selected player. It saves the actual loss in the current resolution frame before event handlers run. An absent or illegal target and a nonpositive amount save zero. The result belongs to that instruction, even when the source has another pending spell or activation.
  * `GainLifeForWithResult(..., LifeGainedInto(binding))` writes one player's actual gain to the current resolution frame. It uses the same gain preparation, replacement, and event service as `GainLifeEach`. An absent player or nonpositive amount writes zero. The frame belongs to the resolving stack instance and is copied with it. The result does not include damage, life loss, or payment.
  * `GainLifeEach(..., LifeGainedInto(binding))` writes the sum of prepared positive gain outcomes to the current resolution frame. The shared batch service applies all gains before it sends `EvtLifeGained` events. Replacement and life-gain modifiers change the saved amount; an empty set or nonpositive amount writes zero. The result is separate from life loss and payment.
  * `LifePayXCostInto` resolves the announced X to a fixed life cost before total-cost validation. The locked payment writes its committed amount to the same instance-local number-binding frame. A failed payment does not move the spell or create a stack object.

---

## 9. Total-Cost Payment Transactions (CR 601.2, 602.2)

`IfPaid` and `UnlessPaid` check whether each cost component is currently payable before they offer a resolution-time choice (CR 608.2d). If no legal sacrifice exists, the player cannot choose to sacrifice. `UnlessPaid` applies its unpaid body; `IfPaid` skips its paid body. Accepted payments still use the isolated total-cost transaction and live commit. A payment that becomes impossible during preparation or commit counts as unpaid, without a partial live payment. Each use in `ForEachSavedCard` makes a new choice against the game state left by earlier iterations.

`ExileFromGraveyardCost` locks the payer's distinct card choices and their graveyard visits during total-cost preparation. The isolated validation game pays those exact choices. Before live mana, life, or other cost payment, commit checks that every selected visit is still present. A card that leaves and returns is a new object and cannot satisfy the old choice. `EitherCost` locks its selected branch with the card choices, so the live commit does not ask again. The existing graveyard movement service moves the chosen group and emits its graveyard-leave batch event. The direct-cost `ExileFromGraveyardCostWithCards` form publishes the resulting exile visits in the paid stack instance after commit; it publishes nothing on failed payment. The original constructor does not publish a binding (CR 118.3, 400.7, 400.7j, 601.2h, 701.13a).

`DiscardCardsWithDifferentNamesCost(n)` uses the same locked hand-visit and simultaneous cost-discard service. The chosen group must contain exactly `n` different card objects with distinct names. Preparation asks the live payer once, validates those visits on an isolated game, and retains the choices in the prepared transaction. Commit checks those visits and names before mana, life, or other payment. Invalid choices fail rather than selecting substitute cards. Zero is valid; negative counts fail registration. All cost-discard replacements are prepared before movement, and prevention rejects the complete group. CR 701.9c makes payment illegal when a characteristic-constrained discard is sent to a hidden zone without revelation. The current discard executor supports graveyard and unrevealed library destinations; a name-constrained batch rejects the latter before any card moves. This does not change unfiltered discard payments, which can still pay through library redirection. Optional and alternative announcement costs flatten to the same locked payment. Declaration reuse, stack copies, and game clones do not choose or pay again. No paid-card or characteristic output is added.

`DiscardCost` and `DiscardCardCostInto` lock each chosen card's hand visit during total-cost preparation. The isolated validation game retains those visits. Commit rejects a missing or returned visit before mana, life, or other costs change live state; no paid binding or discard event is published on that failure. All chosen visits must be present for a multi-card cost. Zero cards is valid, and hand reordering preserves visits. Choices belong to the prepared transaction, so a reused cost declaration or a game clone does not change another transaction's choices. The shared discard action still applies replacement ordering and cost-versus-effect restrictions (CR 118.3, 400.7, 601.2h, 602.2b, 701.9). This contract does not add filtered discard choices. Simultaneous discard movement uses the multi-card payment contract above.

The typed single-card discard result uses `PlayerDiscardByEffect` after the affected player chooses the card. It saves the resulting graveyard visit only when the completed discard puts the card there. Replacement to the library still completes the discard but produces no graveyard `CardRef`. The saved visit is instance-local; a later zone move makes it absent, and a game clone keeps its own result.

The typed multi-card discard result uses the same affected-player choice and effect-discard replacement action for each chosen card. `DiscardCardsInto` saves only completed moves to graveyards. `FromCards` retains each new graveyard visit, and `CountCards` counts only visits still present when read. A prevented discard, a redirected destination, or a later move contributes no current card. The result is local to one resolution frame, and game clones copy its member list independently (CR 400.7j, 609.3, 701.9).

`ChooseUpToOneCard` asks the named player at resolution whether to choose one card from the current filtered members of a saved graveyard set. It uses the existing graveyard choice service. A declined, absent, or invalid choice saves no card. The selected card must remain in the same graveyard visit through the choice. The binding belongs to the current resolution frame, so a stack copy or game clone cannot change another instance's result (CR 400.7, 608.2d).

`ActivatedAbilityEvent().WithActivator(actor).WithSource(source)` watches the existing completed `EvtAbilityActivated` event (CR 602.2, 605.3). Both captures are optional. The event is published after a normal activation pays its costs or a mana ability activates. A failed activation publishes no event. The pending trigger saves `GameEvent.PlayerID` and a copy of the activated source's event-time incarnation and characteristics in its own resolution frame. Source departure, control changes, stack copies, and game clones cannot change those saved values. A phased-out battlefield watcher does not detect the event. `WithSource` supplies a damage source, not a live permanent reference. The pattern does not grant stack behavior when the activation is a mana ability. `FromArtifact()` checks that recorded incarnation against the source's current object or departure LKI, so a returned object cannot supply the type. `ByOpponent()` compares the activator with the watcher controller when the event occurs. `WithoutTapCost()` checks the completed event's activation-cost flag. It excludes a cost with `{T}` and keeps a cost that taps by another instruction. Each filter is independent and a repeated filter fails registration.

An activated cost that removes the announced X counters fixes X during cost
preparation. The captured form uses the existing locked counter-cost payment:
commit rechecks the source incarnation, phasing, and counter count before it
publishes the paid count to that activation's resolution frame. A failed commit
publishes no count. The legacy X-counter cost remains available to existing
callers.

`MillCost(count)` is a cost, including in a resolution-time optional payment. It uses the shared mill amount modifiers and card movement path. Preparation locks the modified amount and verifies that the payer has enough library cards to mill that full amount (CR 701.17b). Commit rechecks the amount and library before it changes mana or life. If a modifier changes or the library becomes too small, payment fails without changing those resources. A modifier that changes the amount to zero permits payment. The cost does not save milled-card results.

`MillCostInto(count, paid)` uses the same locked transaction and publishes its final paid count only after commit. It stores the number in that spell or activation's binding frame, including copies. A failed payment publishes no value. This contract does not capture milled card identities or add zone-change replacement handling.

`TapAttachedPermanentCost` locks the ability source incarnation, attached permanent incarnation, and attachment during payment preparation. Commit rechecks them before mana, life, or other costs change live state. It uses the shared tap transition and event path. A phased-out, absent, returned, detached, or already tapped host cannot pay. Phasing back in preserves its incarnation. This is a normal tap action cost, so the `{T}` symbol's summoning-sickness restriction does not apply.

`TapNOtherMatchingCostWithPermanent` uses the existing one-permanent matching tap payment. Preparation locks the chosen battlefield incarnation; commit rechecks its controller, filter, tapped state, phasing, and incarnation before payment. Only a successful commit publishes that incarnation to the spell or activation frame. A failed commit leaves the binding absent and does not spend later life costs. The tap uses the shared event path. Stack copies and game clones keep independent binding frames. A returned card cannot stand in for the paid permanent (CR 118.3, 118.10, 400.7).

Grouped activated abilities announce and validate one target list per declared
group before payment. The ability stack object stores the chosen target's
declaration beside each target, so an empty optional group does not take a
choice from a later group. Copying, retargeting, and resolution use that
alignment. Invalid group counts or choices leave costs unpaid; a failed
payment leaves the activation off the stack.

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
`SacrificeCreatureIncludingSourceCostWithStats` uses the same capture and can
select the ability source if it is a creature. A source sacrificed to pay its
own activation remains absent during resolution, while its paid values remain
in the ability's stack frame.
After successful payment, the values enter that activation's or spell's
stack frame, including spells cast with an alternative cost. A failed payment
creates no frame or paid-stat result. A stack
copy keeps the same paid values, and a game clone copies them independently.
`SacrificeMatchingCostWithManaValue` uses the same locked choice and payment
frame. Commit reads the chosen permanent's mana value before sacrifice, then
writes it to that spell or activation's number binding only after the cost
succeeds. A returned or phased-out choice does not produce a value. The
resolution-time `IfPaid` and `UnlessPaid` paths do not carry this output.
`SacrificeMatchingIncludingSourceCostWithManaValue` uses that same contract
and admits the ability source only when it matches the filter. Its captured
value remains available after the source is sacrificed.

`SacrificeCreatureCostWithSubtype` uses the same locked choice and frame. It
can select the source creature. Commit reads the chosen creature's current
subtypes before sacrifice and saves 1 for a match or 0 for no match. A failed
payment saves no value. A stack copy keeps its paid value.

`SacrificeMatchingCostWithCard` uses that locked choice and saves the card's
graveyard visit only after successful payment. The saved card belongs to the
spell or ability stack frame, including copies and game clones. It remains
present only while that graveyard visit is current (CR 400.7j). A token has no
graveyard card to save. Failed payment saves no card. This is a singular cost;
it does not bind a set of sacrificed objects or their former characteristics.

`DiscardCost(n)` now keeps the exact hand-card choices made during total-cost
validation. The payer chooses on the live game once. Validation pays those
card IDs on its isolated game, and commit uses the same IDs without asking
again. A choice that is absent from the hand at validation or commit fails
payment; commit checks it before mana or life payment. The reusable cost
declaration does not retain a transaction's choices. Direct legacy `Pay`
calls still choose at the time of payment. `DiscardCardCostInto(binding)`
saves the one chosen card only in its new graveyard visit. `FromCard(binding)`
becomes absent after another zone move or when a replacement sends the card
to a different zone. The committed capture is local to the spell or activation
frame and follows stack copies and game clones.

`DiscardMatchingCardCost(filter)` reuses the same locked hand-visit payment
and discard replacement service for one matching card. The live payer chooses
from matching hand cards once. Invalid choices fail without substitution.
Validation and commit recheck both the hand visit and the filter; commit
rejects a stale choice before spending mana or life. The declaration retains
no chosen card. Prevention rejects payment, and an unrevealed library
replacement cannot pay a characteristic-constrained discard (CR 701.9c).
The zero filter matches all cards. Direct activation and spell costs,
announcement optional/alternative wrappers, and resolution-time payment
groups are supported. This constructor produces no binding or characteristic
capture. Stack copies do not repay the cost; game clones pay only on their
own state. Phasing and battlefield LKI do not apply to hand-card choices.

`DiscardCostWithCards(n, binding)` saves each resulting graveyard visit recorded
by the shared discard batch. The batch records visits before it dispatches
events, so later zone moves cannot replace the saved identity. Redirected
discards can pay the unfiltered cost but supply no graveyard reference. Zero
cards supplies an empty set. Commit publishes the set only after successful
payment; a failed transaction clears all outputs. Locked cost copies retain
their own choices and results. The stack frame owns a separate result slice
and follows stack copies and game clones. A saved reference is absent after
its graveyard visit ends (CR 400.7j, 601.2h, 701.9). This output supports direct
activation costs and spell additional costs. Alternative costs, optional
wrappers, and resolution-time payment captures are rejected at registration.

`DiscardCardsWithDifferentNamesCostWithCards(n, binding)` combines the same
paid-set output with the existing distinct-name constraint. Preparation and
commit use the locked hand visits and the shared simultaneous discard batch.
The cost rejects undefined characteristics from an unrevealed hidden-zone
replacement (CR 701.9c). It has the same direct-cost scope, registration checks,
and instance-local capture lifetime as `DiscardCostWithCards`. It does not
capture names or other discarded characteristics for later evaluation.

`RevealFromHandCostWithCard` locks one matching hand card and its hand visit
during payment preparation. The same choice is checked during isolated
validation and live commit, before mana or life is paid. A card that leaves
and returns has a new hand visit and cannot pay the locked cost. Successful
payment saves that hand visit in the spell or ability frame. `FromCard` is
absent after that card leaves hand; stack copies and game clones keep separate
frames. A successful reveal payment adds one public snapshot to
`Game.PublicReveals()`; failed validation adds none. This capture applies to a direct cost component. It does not make a
resolution-time payment or an `EitherCost` branch produce a card binding
(CR 400.7, 601.2h, 701.20b).

`OptionalCostInto` locks the payer's choice during total-cost preparation and
saves a 1 or 0 outcome in the spell or activation payment frame. The live
commit does not ask again. A failed total payment publishes no result. This
instance-local outcome does not use the source-wide legacy optional-cost
tracker. The inner cost must be direct and must not produce another binding
(CR 118.8b, 601.2b, 601.2h).

`OptionalRevealFromHandCost` combines that locked optional outcome with the
existing revealed-card payment capture. If the player pays, the same selected
hand visit is checked during validation and commit and is saved in the current
payment frame. If the player declines or has no matching card, the outcome is
zero and no card is saved. Failed validation publishes neither a reveal nor a
payment result. Stack copies and game clones retain separate frame values.
This contract applies to one direct reveal cost; it does not cover other
optional or alternative payment branches (CR 118.8b, 601.2h, 701.20a-b).

`DiscardCost` uses `PlayerDiscardByCost` and the shared `DiscardAction`
replacement pipeline. The action records that the discard pays a cost. A
replacement that watches all discards can change its destination, and the
completed discard emits one event. An effect-only replacement still excludes
cost discards. Total-cost validation runs these replacements on an isolated
game; a replacement that prevents the discard makes validation fail without
changing the live game. The locked card choice is reused at commit. This
contract covers chosen `DiscardCost`, not random or name-constrained discard
costs. If a replacement sends a
discarded card to a hidden zone without revealing it, its characteristics are
undefined for the related payment (CR 701.9c).

`TapNOtherMatchingCost` locks exactly the chosen battlefield incarnations during
payment validation. It excludes the ability source, checks current control,
untapped status, phasing, and the matching filter again before live payment,
and taps the chosen group before it sends their tap events. The group can
include creatures with summoning sickness because it is a tap cost for another
object, not those creatures' own `{T}` abilities. An invalid choice or failed
payment leaves live mana, life, and tap state unchanged. The declaration does
not keep selected objects between transactions.

`Tap(subject, TapAmountInto(binding))` records an action result in the current
resolution frame. It writes 1 only when the referenced battlefield incarnation
changes from untapped to tapped; the normal tap action emits its event once.
An already tapped, phased-out, absent, or returned incarnation writes 0 and
emits no new tap event. `IfPositive(FromNumber(binding), body)` can test this
result later in the same `Sequence`. Stack copies and game clones keep their
resolution frames separate. This result is for a resolving instruction; tap
costs use the payment transaction.

`Sacrifice(subject, SacrificeSubtypeInto(subtype, binding))` uses the same
singular sacrifice action and saves a result in the current resolution frame.
It reads the permanent's current subtype immediately before that action moves
the permanent. It writes 1 only after an eligible permanent is sacrificed and
had that subtype; absent, phased-out, returned, or differently controlled
permanents write 0. The result does not use the global last-sacrificed slot.
Stack copies and game clones keep separate resolution frames. This result is
for a resolving instruction, not a cost payment (CR 400.7, 608.2h, 701.21a).

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
  * `EvtTappedForMana.SourceIncarnation` records the producing permanent's battlefield incarnation before the mana ability resolves. `LandTappedForMana().WithLand(land)` saves that identity in the pending trigger. A land that leaves and returns is a new object; `From(land)` does not select it. Phasing makes the saved land unavailable while phased out. The land filter checks the event's incarnation or its departure LKI, so a later object with the same card ID cannot change the event's type (CR 400.7, 603.10, 106.12a).
  * `LandTappedForMana().WithPlayer(player).WithAmount(amount)` matches this event when its source is a land. Either capture is optional. It saves the activating player and the event's produced-mana total before trigger placement. A nonland source or zero production does not match. The saved values stay fixed if the land leaves and returns or the trigger is copied. This trigger boundary is separate from the mana-production replacement boundary in CR 106.12b.
  * `EvtTappedForMana.ManaTypes` records each distinct type produced by the resolving tap-cost mana ability. It uses the chosen type for `AnyColor` production and excludes mana from separate bonus abilities. Direct activation, ordinary tap, typed immediate activation, and planned payment supply the same event data. `LandTappedForMana().OfManaType(type)` checks this event-time set under CR 106.12a; registration accepts only a concrete mana type and one type filter. The event owns a copy of its type slice.
  * `LandTappedForMana().WithManaTypes(binding)` copies this event-time set into each trigger instance. `AddManaOfProducedTypeToPlayer(player, binding, amount)` chooses one member for the recipient and adds the full amount through the shared mana action. A choice outside the set falls back to its first member. Empty captures and absent recipients add nothing. The captured set stays fixed after the land leaves or returns and in game clones. Targetless triggers with this action use triggered mana timing under CR 605.1b and 605.4a.
  * `LandTappedForMana().OfAttachedLand()` checks the trigger source's attachment against the producing land's event-time incarnation. It does not match another land, a new incarnation of the same card, or a phased-out source. Once the trigger exists, later attachment changes do not change it. The filter uses the existing tap-cost mana event and attachment state.
  * `AddManaToPlayer(player, type, amount)` evaluates its player and amount in the original resolution frame, then uses the shared mana action for that player's pool. An absent player or nonpositive amount adds no mana. A targetless activated ability with this action is a mana ability under CR 605.1a, but the automatic payment planner does not advertise its output because its recipient may differ from the payer.
  * A targetless `WhenEvent(LandTappedForMana(), body)` trigger with a direct or sequenced mana action resolves immediately after the producing tap-cost mana ability (CR 605.1b, 605.4a). It uses the captured event bindings and does not expose the mana trigger on the stack during resolution. Earlier and newly created ordinary triggers remain pending until priority. A trigger with targets or a body that cannot add mana uses ordinary timing. Conditional mana bodies and other triggered-mana event categories remain to be implemented.
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

`BlockedAttackersOf` reads current combat links by battlefield incarnation. It returns attackers still blocked by the referenced creature, including multiple attackers blocked by one creature. The set is empty when the blocker is absent, phased out, or no longer blocking. An attacker removed from combat, phased out, or returned as a new object is excluded. It does not use historical blocked status: an attacker can remain blocked after its last blocker leaves (CR 509.1h).

An `EvtDeclaredAttacker` records its defending player in `TargetID` when the attack is declared. Both the turn path and simulated declaration path supply this value. A typed attack trigger captures it before the stack entry is made, so later combat changes do not change the saved player.

`EvtCreatureBlockedBy` occurs once for each new attacker and blocker pair after legal blocker declarations settle. `BecomesBlockedBy(This)` uses this event and checks the blocker subtype when the event occurs. `WithBlocker` saves that blocker's battlefield incarnation in the pending trigger. A copied trigger and a cloned game keep independent captures. The binding does not follow a blocker that leaves and returns, and a phased-out blocker is absent until it phases in.

`EvtBlockerGroup` reports the new legal combat pairs once per creature after blocker restrictions and requirements settle. `SourceID` identifies that creature; `RelatedIDs` contains the new opposing creatures in combat order. `Flag` is true when the source blocks and false when it becomes blocked. A creature put onto the battlefield blocking produces only the attacker-side event. Existing pairs and pairs removed by restrictions produce no group event. Trigger detection checks the opposing creatures' characteristics at event time; the pending event and its member list are copied for a cloned game. `BlocksOrBecomesBlockedBy(This).OfSubtype(subtype)` and `.WithoutSubtype(subtype)` use this event for filtered "one or more" groups. The latter matches if at least one opposing creature lacks the subtype; a matching member does not make a separate trigger. Per-pair effects need a separate event contract (CR 509.3b, 509.3d, 509.4).

`Combat.RemoveFromCombat` removes attacking or blocking membership without changing an attacker's blocked status. Removing the last blocker does not make that attacker unblocked. Typed `RemovePermanentFromCombat` resolves one battlefield incarnation; absent and phased-out references do nothing. Clone combat state is independent.

Legacy False Orders and Ydwen Efreet effects retain a private explicit-unblocking compatibility path. Historical sole-blocker tracking and the remaining card clauses require separate audits; that path is not the general combat-removal operation.

Regression evidence: `TestCombatRemovalKeepsAttackerBlocked`, `TestTypedCombatRemovalLifetime`, and existing `TestFalseOrders` / `TestYdwenEfreet`. Rules revision: February 27, 2026.

# Cast-event mana payment capture

`EvtSpellCast` carries `CastManaSpent`, the total mana drained to pay the exact cast stack object's costs. `FireEvent` reads the stack object named by the event's `TargetID` and checks its source card ID before publishing the event. A trigger copies the number into its own resolution frame when it is put on the stack, so later spell departure, trigger copies, and game clones do not change it. A cast with no mana payment records zero.

`EvtSpellCast` also carries `CastColorsSpent`, the number of distinct colored mana colors with a positive payment for that same stack object. Colorless mana and zero entries do not count. The cast event fixes this number before dispatch; `SpellCast().WithColorsSpent` copies it into the trigger's own resolution frame. The shared `CastContext.DistinctColorsSpent` calculation supplies the count, including for existing non-DSL callers.

`ColorsSpentToCastThisSpell` reads the same count from the cast context of the spell currently resolving. The payment transaction records the colors on that stack object; stack copies and game clones keep separate cast contexts. The value is valid in a typed spell body. It does not supply a former cast payment to an activated or triggered ability (CR 601.2h, 608.2h).

When a permanent spell resolves to the battlefield, its entry event carries the distinct colored mana count from that spell's cast context. `Enters(This).WithColorsSpent` saves the event value in the pending trigger frame. Direct battlefield entry, including entry during another spell's resolution, carries zero. This preserves the payment fact through source departure and stack copying without treating the new permanent as the former spell (CR 400.7, 603.6a). "Enters with" counter clauses remain entry replacements (CR 603.6d), not users of this trigger capture.

`SacrificeNMatchingCostWithTotalPower` uses the prepared payment's locked set of exactly N other permanents. Live commit checks each captured incarnation and phasing state before it reads power. It sums current power before any of the chosen sacrifices, then writes one number to the spell or ability frame only after successful payment. Copies keep the value and cloned games isolate their frames. This fixed-count output does not implement an any-number cost, linked graveyard exile, or a simultaneous sacrifice batch.

## Singular Counter-Removal Outcome (CR 122.2, 609.3)

`RemoveCountersFromWithResult` uses the same counter-removal executor as `RemoveCountersFrom`. It caps a positive request to the counters currently on the referenced battlefield incarnation and records only that instruction's completed removal in the active resolution frame. A nonpositive request or an absent, phased-out, returned, or illegal subject records zero, clearing an earlier result in that binding. Counters that cease to exist during a zone change are not removed counters (CR 122.2). The action does not use departure LKI to remove counters from an absent source. Stack copying and game cloning retain separate frames and counter state. This establishes an action result; payment captures and choice policies remain separate.

## Filtered discard payment capture

`DiscardMatchingCardCostInto` composes the locked singular filtered discard payment with the existing paid-card frame. A successful direct activated cost or spell additional cost saves the movement-time graveyard visit, not the physical card across later visits (CR 400.7). Preparation offers matching candidates once; commit rechecks the locked hand visit and filter before spending resources. Prevention or an unrevealed hidden destination rejects characteristic-constrained payment (CR 701.9c). Failed payment publishes no capture. Stack copies retain the capture and cloned games isolate its resolution. Singular and set discard captures both reject optional, alternative, and resolution-time payment contexts at registration.
