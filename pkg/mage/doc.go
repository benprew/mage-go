// Package mage implements the two-player Magic: The Gathering rules engine in Go.
//
// # Subsystem Architecture and Ownership
//
// [Game] serves as the aggregate root and coordinator of the engine. Rather than
// implementing every domain rule directly, [Game] composes cohesive, stateful
// subsystems that each own their state invariants and cloning behavior:
//
//   - [ZoneSystem]: Owns battlefield storage with copy-on-write semantics for AI
//     search, exile storage with visibility metadata, entering-permanent tracking,
//     and Last-Known Information (LKI) snapshots.
//   - [TurnSystem]: Owns turn counters, current phase and step, active player
//     index, extra turns, [TurnSchedule] lifecycle, and one-shot untap restrictions.
//   - [ResolutionState]: Owns transient resolution scratch state including X values,
//     chosen modes, event source/amount context, cast-time snapshots ([CastContext]),
//     targets, damage/counter distributions, and cost payment reveals/sacrifices.
//   - [TriggerSystem]: Owns pending triggers waiting for stack placement, active
//     [DelayedTrigger] instances, and armed state triggers (CR 603.8).
//   - [ManaSystem]: Owns mana source discovery, mana ability solver planning,
//     and payment scratch buffers.
//   - [DamageSystem]: Owns damage execution, combat damage step aggregation,
//     damage history tracking, and damage reflection.
//   - [TrackerSystem]: Owns turn-scoped observations ([TurnTrackers]) and duel-scoped
//     observations ([DuelTrackers]).
//   - [RandomSource]: Owns deterministic integer and coin flip queues for tests
//     and AI simulation.
//
// In addition to these subsystems, [Game] coordinates the [Stack], [Combat],
// [EffectManager] (continuous and replacement effects), and [Player] instances.
//
// Read-only queries are organized into focused domain interfaces:
// [PlayerReader], [BattlefieldReader], [ResolutionReader], [TurnReader],
// [CombatReader], [TrackerReader], and [StackReader], which compose into [GameReader].
//
// Spell definitions can use OnlyDuring to restrict casting to one or more turn steps.
// Game checks the restriction before choices and payment. Copies of stack spells
// and spell resolution do not repeat the casting check.
// Use [ReduceSpellCostStatic] for a generic cost reduction on matching spells
// cast by the current controller. A phased or departed source cannot supply
// the reduction. The saved source incarnation prevents a returned card from
// restoring an old entry before the next layer pass.
// Use [PayOrElse] for one optional payment with two resolution branches.
// The payment result stays local to the operation.
// Resolution counter payments use PayCountersOrElse for a fixed count and
// PayChosenCountersOrElse for an optional chosen count. Each payment requires
// the original live permanent. The chosen count is available only in its paid
// branch. Counter choices and removal use the shared payment transaction.
//
// Use [UnlessManaPaid] or [IfManaPaid] for a mana payment during resolution.
// Supply a mana cost with X and a [ResolutionValue]. The amount is read once
// before the payment choice. Activation cost reductions do not change it.
// [ManaValueOfSpellWithLKI] retains the value when a spell leaves the stack.
// A later cast of the same card cannot replace that information.
// Use [WithCastingCondition] for a pure restriction before casting choices
// and payment. Conditions read the caster. [IsDuringSteps] reads the current
// step through [TurnReader]. [AfterFirstUpkeep] permits later upkeeps after
// the first upkeep ends. A restriction does not grant casting permission.
// Use [BeforeCombatStep] for attackers, blockers, or combat damage boundaries.
// A first-combat window stays closed during later combats. Set during to true
// for a window in each combat phase.
// Skipped combat points close their windows at the applicable phase boundary.
//
// Use [AllCreaturesMustBlock] with [While] for a live filtered blocking rule.
// [AttachedPermanent] follows the current host. Separate requirements share
// block capacity and cannot bypass evasion or minimum blocker counts.
// [Game.CompleteBlockAssignments] reads a proposal without changing combat.
// Both declaration paths commit a legal maximum before they send events.
//
// Use [TargetBlocksSource], [TargetBlockedBySource], or [TargetInCombatWithSource]
// for a target in a current blocking pair with the original source. Both
// incarnations must remain in combat. LKI cannot restore the pair.
// Use [BlockersOf] or [BlockedAttackersOf] for a collection at resolution.
// [PreventCombatDamageByEach] captures that collection for this combat.
// [PreventCombatDamageByEachThisTurn] captures it until cleanup.
// Later combat changes cannot add sources to the captured collection.
//
// Use [TargetToughnessLessThanSourcePower] on typed permanent targets for
// a strict current toughness comparison. Source departure uses retained power;
// a returned source cannot replace that information. Combine [HasPowerEqual]
// and [HasToughnessEqual] for exact current P/T targets.
//
// Use [SetBasePowerToughness] with [Until] to set captured base values.
// Use [SetEachBasePowerToughness] and [AddEachCardType] for a set animation.
// Until saves the members before any type change. Later entries cannot join it.
// Combine [AddCardType] and [AddSubtypeFamily] with these base values for
// temporary animation. The added types permit the subtype checks in one effect.
// [ValidateSubtypeFamily] permits declaration checks before factory construction.
// [SetBasePower] changes only base power. Bonuses and counters still apply. [TargetOtherThanSource] excludes the
// source from a typed permanent target.
//
// Use [WithActivationStateCondition] with [PermanentMatchesFilter] for a
// live source or attachment restriction before activation payment. Mana planning
// uses the same pure check. Copies and resolution do not repeat it.
// Use [PlayerIsPermanentOwner] or [PlayerIsPermanentController] with
// [WithActivationStateCondition] for a live player restriction.
// Combine this condition with [WithAnyPlayerMay] when the permitted player
// can differ from the source controller. Use [This] or [AttachedPermanent].
//
// Use [EachBeginCombat] or [YourBeginCombat] for combat-start triggers.
// Combine [TurnStepBegins] with a player binding to save the event player.
// Extra combats generate separate events; skipped combats generate none.
//
// Use [BindActivationHistory] and [WithActivationHistory] to keep activation
// counts separate for each ability and source incarnation. [IfActivationsAtLeast]
// reads the count at resolution. Countered activations count; copies do not add
// activations. Clones keep separate history. Final cleanup clears the count.
//
// Use [Activate] with [Mana] costs and [Until] with [ModifyPT] of [This]
// for a temporary source bonus. Each resolution adds a separate bonus to
// the original permanent. A returned card does not receive that bonus.
// Jumpstart and Legends source bonuses use this form with explicit mana costs.
// Osai Vultures uses [RemoveCountersCost] to pay two carrion counters
// before its temporary source bonus resolves.
// Use [TapNMatchingIncludingSourceCost] for a fixed matching tap payment.
// The source can pay when it matches. Each choice keeps its incarnation.
// A written tap cost permits summoning sickness; a tap symbol does not.
// The shared payment transaction checks all choices before it spends resources.
// Use [SacrificeNMatchingIncludingSourceCost] for a fixed count payment that
// permits the source. All choices precede one simultaneous sacrifice batch.
// Stale count choices fail before mana and other resources are spent.
// RON Goblin Warrens and Mold Demon use this cost in activation and resolution.
// Fallen Angel uses [SacrificeMatchingIncludingSourceCost] with [IsCreature].
// The source can pay this cost. The bonus uses its original incarnation.
// Use [OpponentControls] with [While] for a static condition that reads
// the current opponent's permanents. Combine [Not] and [IsToken] to exclude
// tokens. Control changes and phasing change this condition immediately.
// Use [IfPermanentMatches] for a resolving property condition. It uses the
// current object or information saved for its exact incarnation. An information
// read does not make an absent object available for an action.
// [AttackedThisTurn] and [DidntAttackThisTurn] keep each incarnation separate.
// Entering combat without an attack declaration does not satisfy attack history.
// [TargetPlayerAttackedThisTurn] checks the player's history at target selection
// and resolution. Creature departure and control changes do not erase that history.
//
// Use [PermanentMatchesFilter] with [This] or [AttachedPermanent] for a
// live static condition. A returned permanent cannot satisfy an old reference.
// Use [HasCounter] or [CounterCountEquals] for a live counter condition.
// These filters read one counter kind. Exact counts do not use a narrow cast.
// Use [CounterCountAtLeast] for a minimum count of one counter kind.
// Resolving RON counter conditions use current or retained information for
// the original permanent. An absent permanent cannot receive an action.
// Use a typed target with [WithTargets] for a targeted bonus.
// Dwarven Lieutenant combines [IsCreature] with [HasSubType] for its Dwarf target.
// Either player's Dwarf creature can be the target, including the source.
// Angelic Page combines [IsCreature] with [IsAttacking] or [IsBlocking].
// Check combat status again at resolution. After resolution, leaving combat
// does not end the bonus.
// For an entry bonus, put [WithTargets] on [WhenEvent] with [Enters].
// Choose the target when the trigger goes on the stack. The creature spell
// has no target, and the entered creature can target itself.
// [ValueAtResolution] captures X when the bonus starts. A blocking restriction
// belongs to the target filter; it does not end a bonus after resolution.
//
// Icatian Scout and Svyelunite Priest use typed creature targets with
// [GrantKeywordModification] and [Until] for temporary keyword grants.
// Combine [OnlyDuring] with [OnlyOnYourTurn] for a controller upkeep restriction.
// Jump and Pixie Queen use the same typed flying grant in [Spell] and [Activate].
// Either player's creature can be the target. The grant expires at cleanup.
//
// Use [UntilSourceUntaps] with continuous modifications for a bonus that lasts
// while the source remains tapped. The duration starts at resolution. Untapping,
// source phasing, and source departure end it permanently. A phased recipient
// keeps its captured identity. Target departure does not remove a source's
// separate choice to remain tapped during its untap step.
//
// Use [DeclareSpellTargetTargetingControlledPermanent] for a spell that targets
// a permanent you control. It checks current control and the saved incarnation
// at announcement and resolution. A target that gained shroud can still qualify.
// Absent, returned and phased-out targets do not qualify. Copies use their own
// stack identity and controller. Combine card filters to restrict the spell type.
//
// Use [PreventDamageFromBlockedCreatures] with [Static] for damage from
// creatures that the current subject blocks. Combat and noncombat damage
// use the current pair. Retained information cannot restore a former pair.
//
// Use [ChooseDamageSourceFrom] for a non-targeted choice from explicit legal
// source references. [DamageSourceChooser] permits typed player decisions.
// The result uses [DamageSourceBinding] in the current resolution frame.
// The caller must supply the complete legal source list.
//
// Use [PreventDamageFromCardType] with [Static] for damage from sources
// with a specified card type. The current subject can be [This] or
// [AttachedPermanent]. Source types use current or retained information for
// the exact object, including spell sources. This rule has no damage limit.
// Use [DamageSourceOfSpell] for a live typed spell reference and
// [SnapshotDamageSource] to save its information before departure.
// [DamageSource.StackID] keeps separate casts and copies apart.
// [DamageSource.SameObject] compares a saved source with an exact object.
//
// Use [PreventCombatDamageFrom] and [PreventCombatDamageTo] with [Static]
// for live combat prevention. [AttachedPermanent] follows the current host.
// The rule checks source presence and incarnation for each damage event.
// [DamageSourceIsEnchanted] uses the saved Aura status for absent sources.
// Recipient ability removal does not remove a separate Aura's rule.
// Use [CantAttack] for the same live subject attack restriction.
//
// Use [MayPutFromHandOntoBattlefield] for one optional matching permanent
// card at resolution. The choice keeps its hand visit. Entry uses the selected
// player as controller and actor. This action does not cast or play the card.
//
// Use [Mill] for a player action and [MillCost] for a full payment.
// The library owner can order cards that move together to the graveyard.
// All order choices precede the moves. Payment queries make no choices.
//
// Use [ExileGraveyardCard] with a typed card target or [FromCard] for exile
// from a graveyard. A saved visit prevents the effect from moving a returned
// card. [PermanentDies] with WithCard saves the first graveyard visit before
// trigger placement. RON source-death bodies use this capture with
// [EventSourceIsSelf]; the capture stays separate for each pending event.
//
// Use [GainControlWhileSourceControlled] for control that ends when you lose
// control of the original source. Set requireTapped when untapping must also
// end the effect. A returned source or target cannot restore the old effect.
// Target phasing preserves a valid duration; source phasing ends it.
// The separate permission to remain tapped stays optional after target departure.
//
// Use [CantBeEnchantedExceptBy] with [Static] to permit one Aura on a subject.
// Use [AttachedPermanent] and [This] to exclude the granting Aura from the rule.
// The rule checks casting and attachment legality. Illegal attached Auras move
// to the graveyard as a state-based action; this action does not destroy them.
//
// Use [TargetPermanentYouControl] or [TargetPermanentOpponentControls] with
// [WithCastTarget] for a controller restriction on an Aura's enchant ability.
// Attachment and state checks read the Aura's current controller. Shroud does
// not prevent a direct attachment. A control change can make the Aura illegal.
//
// Use [DeclareSpellOrPermanentTarget] with [WithTargets] for one target across
// the stack and battlefield. Copies can change the target category.
// [SetObjectColor] uses this declaration to keep the original object identity.
// A permanent spell keeps its changed color on entry. Later zone changes end it.
//
// Use [UntapWithManaPayment] for an optional selection of controlled, tapped
// permanents followed by one total mana payment and one untap batch.
// All choices precede payment. Failed payment cannot untap part of the selection.
// The operation keeps each selected incarnation through mana abilities.
//
// Use [UnionPermanents] and [IntersectPermanents] for contextual collections.
// Each live incarnation appears once, in battlefield order. [AttachmentsOn]
// selects attachments on current host-set members with the same incarnation.
// [PermanentsControlledByPlayers] permits a [PlayerSet] controller restriction.
// Queries use the resolving context and cannot restore absent members with LKI.
// Use [SingletonPermanent] to use a typed permanent reference in a collection.
// It retains the reference's identity and legality checks. RON resolving tap,
// untap, return and destruction collections can combine these references with
// target collections and live filters through union and intersection.
//
// Mana payment plans save activation colors in [ManaTap] with
// [ManaActivationPayment]. The saved payment prevents an activation from
// spending colors required by the final cost. Execution checks the current
// cost and source incarnation before payment. A returned source cannot use
// its former plan. Payment queries do not spend mana or make choices.
//
// Use [DeclareActivatedAbilityTarget] with [WithTargets] to select an activated
// ability by its stack identity and source card type. Source types use current
// information or retained information for the original incarnation or zone visit.
// Mana abilities do not use the stack. Triggered abilities do not match.
// [ControllerOfAbility] reads the selected ability's controller, including copies.
// [CounterAbility] uses the shared counter operation and does not refund costs.
//
// Attack costs use [SourceCantAttackUnlessPays] or [AttachedCantAttackUnlessPays].
// Both attack declaration paths read the current source after payment. The
// original incarnation must remain present and under the declaring player's
// control. Payment can sacrifice the attacker when it matches the cost filter.
// An absent source cannot become an attacker or produce an attack event.
//
// # Engine Principles & Invariants
//
//   - Pure Queries: Readers and query methods never mutate game state or consume RNG.
//   - Clone Isolation: Cloned games ([Game.Clone]) must have strictly independent
//     state for tree search and AI simulation; mutations in one game never affect another.
//   - Transactional Cost Payments: Spell casting and ability activations verify and pay
//     costs atomically on an isolated clone before committing to the live game state.
//   - Durable Incarnations: Battlefield permanents track a unique incarnation ID
//     independent of their physical card UUID to prevent leave-and-return reference confusion.
//
// Draw-step replacements use [Static] with [SkipDrawStepForLife]. The rule
// follows the current controller and remains active only while the source has
// the ability. A pending gain keeps the player who skipped the step.
//
// Use [ChooseRandomPermanent] with [MatchingPermanents] to save a random
// battlefield reference. Use [TapSubject] with [From] to tap that reference.
// An empty collection does not consume a random value.
// Use [PreventDamageThisTurn] with a typed recipient for unlimited prevention.
// A zero source filter permits all sources. [DamageSourceIsAttacking] reads
// current combat status or the saved departure or phasing status of the source.
// Use [Until] with [GrantKeywordModification] and CantRegenerate for a
// prohibition that lasts until end of turn, even after its source leaves.
//
// Use [DealDamageEachPlayer] for amounts that depend on each selected player.
// Its player binding is local to the amount. All amounts are read before one
// simultaneous damage batch. Replacement effects do not recalculate them.
// Use [DivideRoundDown] and [DivideRoundUp] with [Fixed] of two for half values.
// [DeclareOpponentOrPlaneswalkerTarget] excludes the controller only from
// the player targets. A copied spell checks against its own controller.
//
// [PreventNextDamage] permits allocation across simultaneous damage sources.
// The affected player or recipient controller chooses within legal bounds.
// Native combat saves all assignments before the damage batch starts.
// First-strike and normal damage steps use separate batches.
// [BandingDamageAssigner] lets a player assign each blocking source separately
// to an attacking band. Each source keeps its own damage information.
// The old [BandingDamageDistributor] can still set the total for each recipient.
// Normal nontrample attackers use [CombatDamageAssigner] to divide their
// complete amount freely among current blockers. There is no lethal order.
// Invalid maps use a legal default. Single blockers need no choice.
// Outgoing attacking-band damage uses [CombatDamageAssigner] for each source.
// A blocking band uses [BandingDamageAssigner] or the existing distributor.
// Each source keeps its own trample and deathtouch. Shared assignments can
// supply lethal damage before a source with trample damages the defender.
//
// Use [PreventNextDamageFromPermanent] for one damage event from a typed
// permanent source to a player this turn. The filter is checked at installation
// and before damage. A returned source cannot consume the old shield.
// All matching packets in one simultaneous event use the shield together.
//
// Use [MultiplyValues] to multiply two resolution values. Procrastinate uses
// twice the announced X with [AddCountersTo] after [TapSubject].
//
// Use [IfSacrificePaid] or [UnlessSacrificePaid] for an optional payment of
// one typed permanent. The payer must control the original incarnation.
// Returned and phased permanents cannot pay. Zone replacements can change
// the destination without changing successful payment.
//
// Use [TargetControlledBy] with [You] or [FromPlayer] of an event binding.
// Target selection and later checks read the saved event player. Source
// control changes and copies do not change that player. [AttacksUnblocked]
// with WithDefendingPlayer saves the player before target selection.
//
// Use [IfRequiredPaid] or [UnlessRequiredPaid] for a mandatory matching
// sacrifice payment during resolution. The branch keeps its own result.
// Source exclusion uses the original incarnation, so a returned source can pay.
// [SacrificePermanentAction] permits exile replacements through the shared
// pipeline. A replaced sacrifice can pay without causing a death event.
//
// Use [ManaValueOf] to read a permanent through its typed reference. Use
// [SnapshotNumber] before [Destroy] when a later instruction needs that value.
// The saved number remains available after the permanent leaves.
// Swords to Plowshares saves [PowerOf] and [ControllerOf] before [Exile],
// then uses the saved values with [GainLifeFor]. Exile metadata stays distinct
// from destruction metadata when effects are combined.
// Use [SnapshotPlayer] with [ControllerOf] to save its controller before
// destruction. Crumble combines both snapshots with [DestroyWithoutRegeneration]
// and [GainLifeFor]; the saved controller gains life even if destruction fails.
//
// Use [WhenEvent] with [Enters] and [TapSubject] of [AttachedPermanent]
// to tap an enchanted permanent on entry. The attachment is read at resolution;
// a departed source uses its last attachment and cannot follow a new incarnation.
//
// [AttachedBecomesTapped] saves the event permanent before trigger placement.
// Read it with [From] so attachment changes cannot redirect the effect.
//
// Targeted destruction uses [Destroy] with a typed permanent target. Declare
// that target with [WithTargets] on the spell, mode, trigger, or activation.
// The target filter supplies the restriction; the action allows regeneration.
// Use [UnlessPaid] with [You] and [Destroy] of [This] for a granted
// upkeep payment. The trigger keeps its controller and source incarnation.
// Land destruction uses [IsLand] with the same target and action constructors.
// To destroy all lands, use [DestroyEach] with [MatchingPermanents] and [IsLand].
// Use [HasSubType] for a land subtype. Combine [IsCreature] and
// [HasColorFilter] with [And] to restrict destruction by creature color.
// Combine type filters with [Or] in one collection to destroy each matching
// permanent once. Separate destruction actions can consume extra shields and
// expose intermediate states to triggers.
// The collection is selected at resolution and does not include phased permanents.
// Use [DestroyEachWithoutRegeneration] when the instruction forbids regeneration.
// This restriction applies to the action and does not change permanent keywords.
//
// Use [CantBeBlocked] with [Until] for a temporary blocking restriction.
// It uses combat rules, so ability removal does not erase the restriction.
// The subject keeps its captured incarnation; source departure does not end it.
// Use [Activate] with [Costs] and [WithTargets] for tap-cost blocking effects.
// Put power limits in the target filter. Later power changes do not end the effect.
//
// Use [RemovePermanentFromCombat] with a typed reference to remove combat
// membership. An attacker stays blocked when its last blocker is removed.
// Use [RemovePermanentFromCombatAndUnblock] when the instruction makes
// historically sole-blocked attackers unblocked. [CantBlock] with [Until]
// restricts later blocks and keeps its captured incarnation until cleanup.
//
// Use [ManaCostPayment] for announced X activation mana. The payment
// transaction substitutes X before an activation cost reduction. The stack
// keeps the announced value for target counts and resolution.
//
// Use [GrantLandPlays] to add to a player's land allowance for this turn.
// Grants accumulate and remain separate in cloned games. The action does not
// play a land or change land timing. Explore grants the allowance before drawing.
//
// Use [Activate] with [ModifyPT] and [Until] for targeted temporary boosts.
// Put mana, tap, sacrifice, and discard costs in [Costs]. A source sacrificed
// to pay a cost does not end the boost on the target.
// Use [DiscardRandomCost] for a random hand payment after deterministic costs.
// The payment keeps hand visits and random results separate for each use.
// Failed payment checks do not spend live resources or scripted random results.
// Put subtype and current power/toughness limits in the target filter.
// These limits are checked again at resolution, not during the boost.
// Icatian Lieutenant uses a Soldier creature target; Icatian Priest uses any
// creature target. Both use mana costs and an end-of-turn duration.
// Ghosts of the Damned and Pradesh Gypsies use negative power modifiers.
// Tuknir Deathlock uses the same target and duration with a +2/+2 bonus.
//
// Use [Until] with [ModifyEachPT] for a temporary collection boost.
// [ControlledPermanents] with [IsCreature] selects your creatures at resolution.
// Shield Wall uses this collection with a +0/+2 bonus until end of turn.
// [MatchingPermanents] with [IsCreature] and [IsAttacking] or [IsBlocking]
// selects combat creatures for either player. Combat removal does not end
// the boost. Later entries and returned cards do not receive the boost.
// Use [MatchingPermanents] with [IsCreature] for global P/T reductions.
// Fixed negative P/T summaries report a detrimental outcome.
//
// Use [AnimateEach] with [Static] to add the creature type and set base values.
// Supply colors to replace current colors. The selection stays the same across
// layers in one application, even if the source loses its ability in layer 6.
// Each later application uses current membership.
//
// Use [LimitUntapsDuringStep] with [Static] for a fixed untap-step limit.
// The active player chooses a set that satisfies all limits before untap.
// Type overlap counts against each limit. Other untap effects remain legal.
// Use [PreventUntapDuringStep] with a typed subject and state condition for
// a live source or Aura-host restriction. The untap decision reads both before
// any permanent untaps. Recipient ability removal does not remove this rule.
// Other untap effects can still untap the subject.
//
// Use [PreventUntapDuringStepForMatching] for a static untap-step restriction.
// It checks final characteristics before the untap operation. Other untap
// effects remain legal. Recipient ability removal does not remove the rule.
//
// Use [PreventCombatDamageByThisTurn] with a typed permanent reference for
// combat damage from that source until cleanup. It keeps the captured
// incarnation after control changes or a phase cycle. A returned permanent
// has a new incarnation. [PreventCombatDamageBy] still ends at end of combat.
//
// Use [CantBeBlockedExceptBy] with [While] for a static blocking restriction.
// With [Until], the rule keeps the selected incarnation until cleanup.
// Each blocker must pass its filter and all other combat restrictions.
//
// Use [CantAttackEach] with [Static] for a current collection attack restriction.
// It checks final characteristics at declaration. [ControlledPermanents] reads
// the current source controller. Later rule changes do not remove attackers.
//
// Use [SourceCanBlockOnly] with [Not] to prevent the source from blocking
// matching attackers. Use [CantBeBlockedExceptBy] with [Not] for matching
// blockers. Filters read current characteristics at the block legality check.
// Clones keep both filter maps for queries before the next layer pass.
//
// Use [AttackAsThoughHaste] for an attack permission that leaves tap costs
// restricted by summoning sickness. It does not grant the haste ability.
// The permission keeps its subject incarnation with [Until] and reads the
// current subject with [While]. Use [GrantKeywordModification] with
// AttrCanAttack to permit attacks through defender.
//
// Use [UntilNextUpkeep] to save the duration player at resolution.
// The effect ends before that player's next upkeep triggers are detected.
// Source control changes, departure, and phasing do not change this duration.
// Subject references keep their captured incarnations.
// Use [UntilEndOfNextUpkeep] when the effect must last through upkeep priority.
// [ToughnessOfWithLKI] keeps the exact incarnation for a saved toughness read.
//
// Use [SourceTappedStateWithLKI] for a source tapped-state check that needs
// departure information at resolution. Live-only state conditions remain
// available for static membership checks. Draw-step events are detected before
// the normal draw; pending triggers go on the stack after that draw.
//
// Use [ControllerOfWithLKI], [PowerOfWithLKI], and [ManaValueOfWithLKI]
// when an information read must survive departure. They use the exact source,
// attachment, saved permanent, or singular target incarnation. Phase information
// remains separate from departure information. These reads do not permit actions
// on absent objects. The existing live-only queries keep their behavior.
//
// Use [BlocksCreatureMatching], [BecomesBlockedByCreatureMatching], and
// [BlocksOrBecomesBlockedByCreatureMatching] for creature pair events.
// Supply [This] or [AttachedPermanent] and a permanent filter. WithOpponent
// saves the opposing incarnation before trigger placement. Later attachment
// and characteristic changes do not change that saved object.
//
// Use [NewConditionalManaAbility] for fixed mana branches with one tap cost.
// Payment queries and activation use the same current-state condition.
// The condition must stay the same when the source taps.
//
// Use [RevealHandFor] to show a player's current hand to all players.
// [DiscardMatchingHand] moves all matching hand cards before discard events.
// [CountMatchingHand] reads the current count without reveal history or LKI.
//
// Use [LookAtLibraryTop] for private inspection and optional top ordering.
// The viewer and library player are separate typed references.
// [MayShuffleLibrary] lets the chooser decide after the inspection.
// [Game.LibraryLooks] returns copied history for the selected viewer.
// Optional [LibraryLookObserver] and [LibraryTopOrderChooser] interfaces
// supply private display and ordering without a change to [Player].
//
// Use [SearchLibraryToHand] for a required single-card search, followed by
// a shuffle. Use [SearchLibraryToBattlefield] with a card filter for a search
// that can find no card. A zero filter requires a card when one is available.
// The destination move occurs before the shuffle. A prohibited battlefield
// entry keeps the card in the library. These searches do not reveal hand cards.
//
// Use [AddPoisonFor] with a typed player reference to add poison counters.
// Nonpositive amounts make no change. The existing state check handles loss
// at ten poison counters. [TokenAbilities] gives each token its own ability
// identity; event bindings keep their values in the resolution frame.
//
// Use [ShuffleHandsAndGraveyardsIntoLibraries] before a separate draw action.
// Use [ShuffleHandsIntoLibrariesAndDraw] to draw each player's moved count.
// All moves finish before zone events, and all shuffles finish before draws.
// Counts stay local to each resolution. Empty hands still cause a shuffle.
//
// Use [LoseUnspentManaFor] to clear one selected player's mana pool.
// Use [HasManaAbility] to check current abilities without testing payment
// or current output. [ControllerOfSpellWithLKI] keeps departure control
// for instructions that run after a spell is countered.
//
// Use [TargetManaValueEqualX] for a permanent target with mana value equal to X.
// [DeclareSpellTargetManaValueX] includes the target spell's own X in its value.
// Target checks use the action's announced X before resolution begins.
//
// Use [ModifyPT] with [While] for live static P/T values.
// [CountPermanents] and [AttachedAuras] read current membership on each layer pass.
// [CopySpellOnStack] can retarget Aura casting declarations. Permanent spell
// copies enter as tokens with separate ability and static effect state.
//
// Use [FilterPermanentSet] to keep current matching members of a typed set.
// The original set keeps its target and incarnation checks.
// Use [PermanentDamageRecipients] with [DealDamageEach] for one simultaneous
// damage batch to permanents. Players are excluded from this recipient set.
//
// Use [CantBeTargetedBySpells] with [Static] for a live spell restriction.
// [SpellTargetSource] identifies spells during target queries before stack entry.
// Ability sources use their separate source information. An Aura spell targets;
// a direct attachment does not. Source ability removal ends its own rule.
//
// Use [ShuffleGraveyardFor] so graveyard moves finish before zone events.
// The library is shuffled even when that graveyard is empty.
// RON exile payments use [ExileMatchingCardFromGraveyardCostInto] to keep
// a separate paid mana value for each pending activation.
// Use [ChooseOpponentAsEnters] for an opponent choice before battlefield entry.
// Each permanent keeps its choice through control changes and phasing.
// [EventIsChosenPlayerUpkeep] checks that choice when the upkeep event occurs.
//
// Use [AddCountersUpTo] for an optional counter amount with an action limit.
// The effect controller chooses at resolution. The limit still applies after
// replacements, because this action cannot cause an excess on its subject.
// Other actions can add counters above this limit.
//
// Use [GrantTriggeredAbility] or [GrantEachTriggeredAbility] with [Static]
// for a live trigger grant. The selection uses the donor's context.
// The trigger uses each recipient as its source and reads its controller
// at detection. Bindings belong to the granted ability. A pending trigger
// keeps its source incarnation after the grant ends.
//
// Use [TriggeredAbilityModification] with a duration for a captured trigger grant.
// The recipient has its own source and binding context. A returned permanent
// cannot receive the old grant. [HasRampage] reads current rampage abilities.
// RON Gabriel Angelfire and Rapid Fire use the shared rampage effect and event.
//
// Use [PermanentAttackedDuringLastTurn] for the live source or attachment's
// attack history during one player's last completed turn. Cleanup saves exact
// incarnations. Phasing keeps history; returned objects cannot use old history.
// Use [CantAttackIf] for a live conditional attack restriction. A later condition
// change does not remove attackers already declared.
//
// Use [RevealAllHands] or [RevealAllLibraryTops] with [Static] for public
// visibility while the source ability is active. Control changes do not change
// the affected players. [Game.IsHandRevealed] reads current hand visibility.
// Clones retain separate hand and library-top visibility flags.
//
// Use [ChooseAtResolution] with [ResolutionOption] for a choice made while an
// effect resolves. Each option runs in the existing resolution frame.
// Supply a pure When condition if an option has a legality restriction.
// Costs and targets are selected before this choice. Copies choose separately.
//
// Automatic mana payment can use fixed mana-only conversions without a tap.
// The reduced cost must consume at least the produced mana count. Repeated
// conversion keeps the source available, even when it is tapped or newly entered.
// It does not receive tap bonuses or emit a tapped-for-mana event. X costs,
// activation limits and arbitrary conditions are not planned.
// Exact payments keep the final cost colors. The shared cost transaction
// keeps failed casting payments separate from live resources.
// Fixed-color mana bodies can include [IfActivationsAtLeast] with [AtNext].
// Planned payment runs the complete body after it records the activation.
// Each successful activation can install its own delay. Queries and failed
// payments do not change history or install delays.
//
// Use [AttachedPermanentDies] to capture an attached permanent's death.
// It permits any permanent type. [AttachedCreatureDies] still requires a creature.
// Controller, owner, toughness, permanent and card bindings save event facts.
// The card binding keeps the first graveyard visit. A later visit cannot replace it.
// RON Creature Bond uses these facts with [DealDamageFrom] and the Aura as source.
//
// Use [SkipNextUntaps] for a fixed count of future controller untap steps.
// Overlapping restrictions count the same steps. Other untaps remain legal.
// Use [SkipNextUntapEach] to capture the recipients of one-step restrictions.
// Control changes and phasing preserve counts; departure ends them.
//
// Use [WhenDiesThisTurn] or [WhenLeavesThisTurn] for a delayed departure.
// Each installation keeps the watched incarnation, controller and body bindings.
// Cleanup removes the subscription. A pending trigger can still resolve.
// Destruction replaced by exile sends a leave event, but no death event.
//
// Use [TargetChosenByOpponent] for an opponent choice at announcement or
// trigger placement. [TargetOpponentControls] adds a separate control condition.
// Target legality uses the action controller and source. Modal activation keeps
// the first target choice. A copy uses its controller's permission to retarget.
//
// # Further Documentation
//
// For specialized guides and comprehensive specifications, refer to:
//
//   - docs/dsl-reference.md: Complete card author reference for the composable DSL
//     (subjects, selectors, actions, triggers, spells, activations, and continuous effects).
//   - docs/engine-rules-contracts.md: Detailed Comprehensive Rules (CR) execution contracts
//     including Continuous Layers 1–7, LKI, state triggers, replacement pipelines, and cleanup.
//   - docs/architecture-4plus1.md: 4+1 architectural view of the rules engine and subsystems.
//   - pkg/mage/dsl: Public card-author export package providing the canonical authoring grammar.
package mage
