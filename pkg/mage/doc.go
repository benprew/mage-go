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
// Use [WithCastingCondition] for a pure restriction before casting choices
// and payment. Conditions read the caster. [IsDuringSteps] reads the current
// step through [TurnReader]. [AfterFirstUpkeep] permits later upkeeps after
// the first upkeep ends. A restriction does not grant casting permission.
//
// Use [TargetToughnessLessThanSourcePower] on typed permanent targets for
// a strict current toughness comparison. Source departure uses retained power;
// a returned source cannot replace that information. Combine [HasPowerEqual]
// and [HasToughnessEqual] for exact current P/T targets.
//
// Use [SetBasePowerToughness] with [Until] to set captured base values.
// Combine [AddCardType] and [AddSubtypeFamily] with these base values for
// temporary animation. The added types permit the subtype checks in one effect.
// [ValidateSubtypeFamily] permits declaration checks before factory construction.
// [SetBasePower] changes only base power. Bonuses and counters still apply. [TargetOtherThanSource] excludes the
// source from a typed permanent target.
//
// Use [Activate] with [Mana] costs and [Until] with [ModifyPT] of [This]
// for a temporary source bonus. Each resolution adds a separate bonus to
// the original permanent. A returned card does not receive that bonus.
// Jumpstart and Legends source bonuses use this form with explicit mana costs.
// Osai Vultures uses [RemoveCountersCost] to pay two carrion counters
// before its temporary source bonus resolves.
// Fallen Angel uses [SacrificeMatchingIncludingSourceCost] with [IsCreature].
// The source can pay this cost. The bonus uses its original incarnation.
// Use [OpponentControls] with [While] for a static condition that reads
// the current opponent's permanents. Combine [Not] and [IsToken] to exclude
// tokens. Control changes and phasing change this condition immediately.
// Use [PermanentMatchesFilter] with [This] or [AttachedPermanent] for a
// live static condition. A returned permanent cannot satisfy an old reference.
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
//
// [PreventNextDamage] permits allocation across simultaneous damage sources.
// The affected player or recipient controller chooses within legal bounds.
// Native combat saves all assignments before the damage batch starts.
// First-strike and normal damage steps use separate batches.
// [BandingDamageAssigner] lets a player assign each blocking source separately
// to an attacking band. Each source keeps its own damage information.
// The old [BandingDamageDistributor] can still set the total for each recipient.
//
// Use [MultiplyValues] to multiply two resolution values. Procrastinate uses
// twice the announced X with [AddCountersTo] after [TapSubject].
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
