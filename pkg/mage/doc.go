// Package mage implements the two-player Magic: The Gathering rules engine in Go.
//
// Use [Game.WaitingPreventionPermanentSources] for exact permanent references
// retained by finite recipient, unlimited recipient and next-source prevention.
// Resolved combat prevention supplies its exact source or to-and-by subject.
// Collection prevention supplies each member saved at resolution.
// Typed static prevention supplies its current subject. The live rule must
// remain active; a former attachment cannot supply that subject.
// [PreventDamageFromMatchingPermanents] supplies its current exact subject.
// Its permanent filter uses current or retained source information.
// [ChooseDamageSourceFromWaitingPreventionReferences] uses this category only.
// [Game.WaitingRedirectionPermanentSources] reads live destinations of typed
// static damage redirection. It uses the same subject check as damage execution.
// Legacy cycle redirection also supplies its exact live destination.
// Each registration keeps the destination incarnation. A returned card needs
// a new registration. Cycle owners still control registration removal.
// [ChooseDamageSourceFromWaitingRedirectionReferences] uses this category only.
// Use [Game.WaitingPreventionSpellSources] for exact spell references in
// active source shields. Live or retained spell data keeps separate casts apart.
// [ChooseDamageSourceFromWaitingPreventionSpellReferences] uses this category only.
// Consumption and cleanup end its availability. A continued permanent keeps
// its separate incarnation reference.
// [Game.WaitingDrawStepLifePermanentSources] reads exact sources of active
// draw-step life replacements. Departure, phasing and ability removal exclude
// the source. A pending life gain does not keep a permanent reference.
// [ChooseDamageSourceFromWaitingDrawStepLifeReferences] uses this category only.
// [Game.WaitingRegenerationPermanentSources] reads exact subjects of active
// regeneration shields. Departure and phasing keep their saved references.
// [ChooseDamageSourceFromWaitingRegenerationReferences] uses this category only.
// [PreventDamageToSourceByRemovingCounters] supplies its exact live subject to
// [Game.WaitingPreventionPermanentSources], even with no counters. A saved rule
// cannot follow a returned source or an ability that was removed.
// [Game.AddSourcePreventionShield] and [Game.AddHalfDamageFromSourcePreventionShield]
// retain a live permanent source's exact incarnation. Both supply that source
// to [Game.WaitingPreventionPermanentSources] until consumption or cleanup.
// Departure and phasing keep the reference. A returned source cannot inherit
// the shield. Registrations without a live permanent keep their legacy matching
// behavior and cannot supply an exact waiting reference.
// [Game.AddReverseDamageShield] retains the same exact permanent source and
// supplies it to the waiting prevention query until consumption or cleanup.
// Positive damage from that incarnation consumes the shield before life gain.
// [RedirectDamageFromPermanentThisTurn] saves a permanent source, player and
// destination until cleanup. Its exact source and destination supply waiting
// redirection references, even after departure or phasing. Damage requires the
// original destination to be present as a creature or planeswalker.
// [RedirectNextDamageFromSelectedSource] saves an exact source, subject and player.
// It redirects one damage event to the saved player until cleanup. All matching
// simultaneous packets use it together. A selected permanent spell continues
// to the exact permanent it becomes before entry effects run.
// Its source filter checks current or retained information before damage.
// [Game.WaitingRedirectionPermanentSources] includes its permanent references.
// [Game.WaitingRedirectionSpellSources] includes its exact selected spell.
// [ChooseDamageSourceFromWaitingRedirectionReferences] reads both categories.
// [Game.WaitingPreventionExileSources] and
// [Game.WaitingRedirectionExileSources] read exact visits in active damage rules.
// Departure keeps public facts; face-down cards keep no characteristics.
// A returned card cannot replace the saved visit. Consumption and cleanup end
// availability. Cycle registrations retain their existing cycle lifetime.
// [ChooseDamageSourceFromWaitingPrecastExileReferences] uses these two categories.
// [Game.WaitingPreventionHandSourcesFor] and
// [Game.WaitingRedirectionHandSourcesFor] read exact hand visits in active rules.
// The hand player can inspect private facts. Other viewers receive public facts.
// Departure keeps separate private facts and public visibility for that visit.
// A waiting reference cannot extend a cost reveal or follow a returned card.
// [ChooseDamageSourceFromWaitingPrecastHandReferences] uses the choosing player.
// Consumption and cleanup end availability. Queries and clones keep separate data.
// [Game.WaitingPreventionCommandSources] and
// [Game.WaitingRedirectionCommandSources] keep exact command visits in active rules.
// [ChooseDamageSourceFromWaitingCommandReferences] uses these two categories.
// Departure keeps the visit; a return cannot replace it. Hidden information
// stays hidden. Consumption and cleanup end availability.
// [Game.WaitingModifierPermanentSources] reads explicit sources of active
// registered mill and life-gain modifiers and their static factories.
// Queries do not call callbacks or factories. Phasing suspends availability.
// A saved registration cannot follow a returned source. Static factories use
// current abilities; ability removal stops their references.
// [ChooseDamageSourceFromWaitingModifierReferences] uses this category only.
// [Game.WaitingEmptyLibraryPermanentSources] reads exact sources of active
// empty-library draw replacements without calling their callbacks.
// [Game.AddEmptyLibraryDrawReplacement] saves the live source incarnation.
// Phasing suspends the rule. Departure ends it; a return needs a new registration.
// [ChooseDamageSourceFromWaitingEmptyLibraryReferences] uses this category only.
// [Game.WaitingPersistentReplacementPermanentSources] reads exact sources of
// registered life-gain, sanctuary draw and counter-doubling replacements.
// Their registrations save the live incarnation. Phasing suspends them;
// departure ends them. A return needs a new registration.
// [ChooseDamageSourceFromWaitingPersistentReplacementReferences] uses this category only.
// [Game.WaitingUntapSkipPermanentSources] reads exact sources of active
// typed untap-step replacements. Queries use the replacement's live ability
// and incarnation checks. Phasing, departure and ability removal stop them.
// [ChooseDamageSourceFromWaitingUntapSkipReferences] uses this category only.
// [Game.WaitingDiscardPermanentSources] reads exact sources of active typed
// discard-to-library replacements. The saved incarnation and live ability
// control execution and enumeration. Phasing and ability removal stop the rule.
// Each discard reads the current controller. A return needs a new registration.
// [ChooseDamageSourceFromWaitingDiscardReferences] uses this category only.
// [Game.WaitingExileSources] reads exact spell and permanent subjects of active
// exile-instead rules. A permanent spell keeps its exact continuation before
// entry effects run. Phasing and departure keep the saved references until
// cleanup. Returned permanents and later casts cannot inherit a saved rule.
// [ChooseDamageSourceFromWaitingExileReferences] uses this category only.
// Legacy registrations before casting supply no exact waiting reference.
// Other waiting replacement references need separate contracts.
//
// [Game.SetCreatureDamageRedirect] saves a live subject until use or cleanup.
// [Game.SetAttackerDamageRedirect] saves live source and destination identities
// until cleanup. Both supply exact waiting redirection references after departure
// or phasing. Damage still requires the original live destination. A returned
// source cannot inherit the attacker redirect. An absent installation adds no rule.
//
// Use [PreventNextDamageFromSelectedSource] for a chosen-source shield.
// A shield on a permanent spell continues to the exact permanent it becomes.
// Entry effects run after the link. Separate copies and returned objects cannot
// inherit it. [PreventNextDamageFromExactSource] keeps exact-object behavior.
// Both services require a typed source; source enumeration is separate.
//
// Use [Game.WaitingCounterPlacementPermanentSources] for explicit source
// exclusions in active entry-counter replacements. General filters do not
// supply references. A saved registration cannot follow a returned source.
// [ChooseDamageSourceFromWaitingCounterPlacementReferences] uses this category only.
//
// Use [Game.AddCommandObject] to install an object with a fresh command visit.
// The caller must remove the object from its former zone first.
// [Game.CommandDamageSources] supplies face-up objects to source choices.
// [ChooseDamageSourceFromCommandObjects] uses this category only.
// [DamageSource.CommandVisit] keeps a return separate from its former visit.
// Removal retains departure information. Hidden objects supply no public
// characteristics. Queries and game clones keep separate object snapshots.
// This service does not grant casting permission or install command abilities.
//
// [Game.CaptureHandDamageSource] reads one live hand visit.
// [DamageSource.HandVisit] keeps returned cards separate from saved sources.
// Hidden hands supply identity and owner only. Public hands supply current facts.
// [Game.StackHandSources] reads exact hand targets and payment card bindings
// and nested event card captures on stack objects. Departure keeps the facts
// available at that visit's departure; final stack departure ends this category.
// [ChooseDamageSourceFromStackHandReferences] uses this category only.
// A cost reveal keeps its exact card public until that stack object leaves.
// Copies keep the reference without extending the original reveal.
// A selected hand source cannot follow a later spell or permanent.
// Hand targets save their exact visit at announcement. Copies keep that visit.
// A new target choice saves the current visit, even with the same card ID.
// Returned targets are illegal for the original spell or ability.
// Waiting card frames use their separate reference contracts.
// [Game.CaptureHandDamageSourceFor] and [Game.StackHandSourcesFor] add the
// hand player's private inspection information. Other viewers keep public facts.
// [ChooseDamageSourceFromInspectableStackHandReferences] uses the choosing player.
// Inspection does not reveal a card. Departure saves separate public visibility;
// returned visits cannot inherit it. Damage rechecks use actual current or
// departure facts, independent of the information shown to a player.
//
// [Game.CaptureGraveyardDamageSource] reads one live graveyard visit.
// [DamageSource.GraveyardVisit] keeps returned cards separate from saved sources.
// [Game.StackGraveyardSources] reads exact targets, payment card bindings
// and nested event card captures on stack objects. Departure keeps public facts;
// the final referring stack departure ends this category's availability.
// Resolving frames retain these references until body completion.
// Queries and game clones keep separate mutable card facts.
// [ChooseDamageSourceFromStackGraveyardReferences] uses this category only.
// A selected graveyard source cannot follow a later spell or permanent.
// Waiting and delayed card frames need separate reference contracts.
//
// [Game.CaptureExiledDamageSource] reads one live exile visit.
// [DamageSource.ExileVisit] keeps each visit separate from later objects.
// Face-down cards supply no characteristics, even with inspection permission.
// [Game.StackBoundExileSources] reads exact exile targets, card payment bindings
// and event card captures on stack objects. Departure keeps exact information.
// [ChooseDamageSourceFromStackExileBindings] uses only this category.
// Exile targets save their visit before payment. Copies keep the visit until
// a legal new target choice saves the current visit. Returned targets are illegal.
// The resolving stack frame keeps references until resolution ends.
// A selected exile source cannot follow a later cast or battlefield entry.
// [Game.GrantCastFromExile] saves the current exile visit. Departure ends
// its permission. A failed payment keeps the visit and its permission.
// Re-exile creates a new visit. Clones keep separate exile card information.
//
// [Game.WaitingDelayedGraveyardSources] reads exact visits in waiting card frames.
// Current or departure facts keep returned visits separate. Singular and plural
// bindings share the existing delayed capture, stack transfer and clone services.
// [ChooseDamageSourceFromWaitingDelayedGraveyardBindings] uses this category only.
//
// [Game.StackAbilityPermanentSources], [Game.StackTargetPermanentSources],
// [Game.StackBoundPermanentSources], [Game.StackTargetSpellSources],
// [Game.StackBoundSpellSources] and [Game.StackBoundDamageSources] include
// resolving frames until body completion. Nested resolution and clones keep
// each frame. Spell departure updates retained target information. Saved
// incarnations and spell identities cannot follow returned objects.
// These categories do not enumerate references hidden in effect callbacks.
//
// [Game.WaitingDelayedPermanentSources] includes the saved source when
// [DealDamageFrom] or a packet in [DealDamagePackets] uses [This] as recipient.
// Queries do not read amounts or resolve recipients. A zero amount keeps the
// reference. Departure and phasing keep its saved incarnation; a return cannot
// replace it. Bound and attached recipients do not declare a source reference.
// Saved sequences and modes keep the declaration until the delay triggers.
//
// [WhileWithSharedSubjects] reads its condition and exact subjects at its first layer.
// Later layers keep those objects for the same pass, even after ability removal.
// Each new pass reads current state. Values are read in their applicable layer.
// Type dependencies compare selections in separate game probes.
//
// Use [GrantEachBandsWithOther] for a static combat ability with a quality filter.
// Current subjects and qualities control formation and damage assignment.
// [Game.FormAttackingBand] checks distinct attackers with one defender before
// blocker declaration. Ability loss keeps the band; combat removal removes
// its member. Losing banding also removes all bands-with-other grants.
// A matching holder and another matching creature transfer the damage choice.
// The choice can assign damage to any eligible combat opponent.
// [RemoveBandsWithOther] removes all quality grants in layer 6.
// Ordinary banding stays available. Later grants can restore a quality.
// With [Until], the saved incarnation keeps removal through a phase cycle.
// A returned permanent cannot inherit it. Removal keeps a formed band.
// Combat damage choices use the abilities that remain at assignment.
// A blocker across several groups divides one amount among its opponents.
// The shared damage batch keeps each source and its own keywords separate.
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
// Use [Game.CardPowerToughness] for current card P/T with count CDAs.
// Cards without a controller use their owner. Spells use their current
// controller. Battlefield reads use the existing layer results.
// [PTEqualsCount], [PTEqualsControlledCount] and [ToughnessEqualsControlledCount]
// keep their count declarations outside the battlefield. Copies keep those
// declarations; each battlefield instance has a separate source binding.
// Saved object information must use the existing LKI services.
// Use [CantBeTargetedBySpellsIf] for a live conditional spell target restriction.
// The condition reads the current source controller at each target query.
// Ability targets use their existing rules. Source departure, phasing and
// ability removal stop the restriction through the shared effect lifecycle.
//
// Spell definitions can use OnlyDuring to restrict casting to one or more turn steps.
// Game checks the restriction before choices and payment. Copies of stack spells
// and spell resolution do not repeat the casting check.
// Use [ReduceSpellCostStatic] for a generic cost reduction on matching spells
// cast by the current controller. A phased or departed source cannot supply
// the reduction. The saved source incarnation prevents a returned card from
// restoring an old entry before the next layer pass.
// Use [IncreaseActivationCostStatic] for fixed mana increases on abilities
// of matching live permanents. Use [ReduceActivationCostStatic] with
// [AttachedPermanent] for a generic reduction with a minimum mana payment.
// The payment service combines mana costs and announced X before modifiers.
// Increases precede reductions. Mana planning uses the same total.
// Phasing, source departure and ability removal stop the modifier.
// Aura reattachment changes the subject; a returned host cannot restore it.
//
// Use [IncreaseSpellCostStatic] for fixed generic and colored spell taxes.
// Increases apply before reductions in the shared total spell cost.
// With controllerOnly, the source's current controller must cast the spell.
// Source departure, phasing and ability removal stop the tax.
// Use [DamageTakenThisTurn] or [ArtifactDamageTakenThisTurn] for a typed
// player damage total at resolution. The total uses actual damage this turn
// and the source types at the time of damage. Later type changes do not change it.
// Use [PayOrElse] for one optional payment with two resolution branches.
// The payment result stays local to the operation.
// Use [PayActionOrElse] with [GainLifePayment], [DamagePayment], or
// [DestroyPayment] for one optional action paid during resolution.
// The accepted payment selects the branch even if prevention or replacement
// changes its outcome. An absent subject or an indestructible destruction
// subject cannot pay. Each payment keeps its own references and amount.
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
// Use [BlockAnyNumber] and [BlockEachAttacker] with [Until] for captured
// blocking permission and requirements. These rules survive ability removal.
// The shared blocker planner still checks evasion and minimum blocker counts.
//
// Use [CantAttackMatchingThisTurn] for a live attack restriction until cleanup.
// Use [MustAttackControlledThisTurn] for a saved player's live attack requirement.
// These rules remain separate from creature abilities. Later entries and control
// changes can change membership. Source departure does not end the rules.
// Attack requirements do not force a player to pay an attack cost.
//
// Use [AssignsNoCombatDamage] with [Until] to stop combat assignment for
// the resolved permanent. The rule survives ability removal and source departure.
// A phase cycle keeps its identity; a returned card cannot inherit it.
// Noncombat damage and incoming damage still use the damage services.
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
// Use [BlockedByThisTurn] for live members of an exact blocker's turn history.
// Saved blocker references keep their identity after departure and phasing.
// [ControllerWhenLastBlockedThisTurn] reads the attacker controller at the last
// matching block. Returned objects cannot use an earlier object's history.
// Both services use turn history that remains until final cleanup.
//
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
// Use [UnlessPermanentCameUnderControlThisTurn] for a resolving player control history
// exception. Each incarnation retains acquisition turns for all players. Later
// control changes cannot erase an earlier acquisition in the same turn.
// Use [IfPermanentMatchesElse] to select one of two resolution bodies.
// Use [UnlessPermanentMatches] for a body on a known nonmatching object.
// Both read the condition once. Missing information runs neither body.
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
// Delayed bodies keep spell-event bindings for the original stack identity.
// Use [Game.WaitingDelayedSpellSources] and
// [ChooseDamageSourceFromWaitingDelayedSpellBindings] for saved waiting spells.
// Queries use current or exact departure information. A later cast cannot
// replace a saved spell. These services supply one source category.
//
// Delayed bodies keep singular and plural card bindings for their saved visits.
// [Game.WaitingDelayedHandSourcesFor] uses the viewer's hand inspection rights.
// [Game.WaitingDelayedExileSources] keeps face-down characteristics hidden.
// [ChooseDamageSourceFromWaitingDelayedCardBindings] uses these two categories.
// Trigger placement transfers the references to the existing stack queries.
// Each execution restores the outer card frame, including after an error.
// A saved reference cannot extend a reveal or follow a returned card.
//
// Delayed bodies also keep typed damage-source bindings and their saved facts.
// Use [Game.WaitingDelayedDamageSources] and
// [ChooseDamageSourceFromWaitingDelayedDamageBindings] for these waiting frames.
// Queries read current or exact departure information. Body execution restores
// the saved event information without changing the outer resolution frame.
// Placed delayed bodies use [Game.StackBoundDamageSources].
//
// Use [ChooseDamageSourceFrom] for a non-targeted choice from explicit legal
// source references. [DamageSourceChooser] permits typed player decisions.
// The result uses [DamageSourceBinding] in the current resolution frame.
// The caller must supply the complete legal source list.
// [Game.LiveDamageSources] reads current unphased permanents and live spells.
// [ChooseDamageSourceFromLiveObjects] makes a filtered choice from those objects.
// These live-object services exclude referred and command-zone objects.
// [Game.StackAbilityPermanentSources] reads original permanent sources of
// abilities still on the stack. It uses current or retained information for
// each exact incarnation, including information saved before phasing.
// [ChooseDamageSourceFromStackAbilities] chooses only from this category.
// [Game.StackTargetPermanentSources] reads exact permanent target incarnations
// referred to by spells and abilities still on the stack. Target legality
// does not remove the saved reference. Copies keep their own target list.
// [ChooseDamageSourceFromStackTargets] chooses only from this category.
// [Game.StackTargetSpellSources] reads exact spell targets referred to by
// objects still on the stack. Departure information remains available until
// the final referring object leaves. Recasts keep separate stack identities.
// [ChooseDamageSourceFromStackSpellTargets] chooses only from this category.
// [Game.StackBoundPermanentSources] reads saved event and payment incarnations
// referred to by objects still on the stack. Announcement captures also qualify.
// Immutable event captures supply references without running their effects.
// Delayed bodies on the stack also supply saved permanent, target and token
// identities. Waiting and pending delayed triggers do not enter this category.
// [ChooseDamageSourceFromStackBindings] chooses only from these binding contexts.
// [Game.WaitingDelayedPermanentSources] reads saved permanent, target and token
// frames and exact event subjects while delayed abilities wait to trigger.
// Direct [This] reads in destruction, sacrifice and return bodies also qualify.
// Single-recipient [DealDamageFrom] bodies qualify with [This] or
// [DamageSourceOf] of [This], even when the damage amount is zero.
// [DealDamageEach], [DealDamageEachPlayer] and [DealDamagePackets] bodies
// also declare these source reads without evaluating recipients or amounts.
// Empty collections and zero amounts keep the source reference.
// [IfPermanentMatches], [UnlessPermanentMatches] and [IfPermanentMatchesElse]
// declare self references in the condition and both bodies without checking
// the condition. A false condition does not remove a declared reference.
// Saved sequences and modes retain the original source incarnation.
// One-shot detection ends this category; duration-based delays can keep waiting.
// [ChooseDamageSourceFromWaitingDelayedBindings] uses only these references.
// [Game.StackBoundDamageSources] reads permanent and spell source identities
// saved by event captures and delayed frames on stack objects. Queries stay separate from
// the event information used by the trigger body. Spell departure information
// uses the exact stack identity and remains separate in cloned games.
// [ChooseDamageSourceFromStackDamageBindings] chooses only from this category.
// [Game.StackBoundSpellSources] reads spell identities saved by cast events.
// Detection saves information before a pending trigger reaches the stack.
// Current queries and departure snapshots keep the exact spell identity.
// [ChooseDamageSourceFromStackSpellBindings] chooses only from this category.
// Other saved binding types need separate source contracts.
// General source choices also need the other referred and command-zone objects.
// A general source choice needs all categories before it can use the complete list.
// Use [ChooseDamageSourceMatchingFrom] for a property filter on that list.
// [DamageSourceHasColor] and [DamageSourceHasType] read the supplied information.
// [AllDamageSources] and [AnyOfDamageSources] combine those filters.
// Live references read current properties. Saved references read saved properties.
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
// Use [RedirectDamageToPermanentIf] for conditional player damage redirection
// from matching permanent sources. [DamageSourceIsUnblocked] reads the status
// set after blocker declaration, or the saved departure or phasing status.
// Use [RedirectDamageFromCardTypeToPermanentIf] to include matching sources
// outside the battlefield. Both rules use live player and destination references.
// Redirection preserves the source, amount and combat status for later replacements.
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
// Use [PreventNextDamageFromExactSource] for one damage event from an exact
// permanent, spell or command reference to a player this turn. Property checks use
// current or retained information for that object. A returned permanent or
// another cast cannot consume the shield. Bindings can change after installation;
// each shield keeps its own source. [PreventNextDamageFromSelectedSource] adds
// permanent-spell continuity for a source chosen before that spell resolves.
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
// Use [PlayerControls] with [FromPlayer] for a saved event player condition.
// Detection and resolution keep separate player bindings for each event.
// [OpponentControlsMoreThanYou] compares current matching permanent counts.
// Phased permanents and equal counts cannot satisfy that comparison.
//
// Use [TargetControlledBy] with [You] or [FromPlayer] of an event binding.
// [CombatDefendingPlayer] selects the nonactive player during two-player combat.
// Target control checks use this player even when the defending player casts
// the spell. An attack event can keep its separate saved defending player.
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
// Use [AddRestrictedManaOfAnyOneColor] for a resolution amount with one
// spending rule. The controller chooses one color for the full amount.
// The mana pool and payment planner keep the rule on each mana unit.
// A nonpositive amount adds no mana and requires no color choice.
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
// Use [SkipUntapSteps] with [Static] to replace each player's whole untap step.
// Source departure, phasing and ability removal stop the replacement.
// Turn-start control checks still permit creatures controlled from earlier turns.
// Use [LimitUntapsDuringStepIf] to check a condition before untap selection.
// Phasing precedes this check. Untapping cannot change the saved selection.
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
// [ChosenPlayerOf] reads a live permanent's entry choice without LKI.
// Use [SetMaximumHandSizeStatic] or [NoMaximumHandSizeStatic] with a typed
// player reference. [Game.MaximumHandSize] and cleanup read active rules in
// timestamp order after the object layers. Phasing and source departure stop
// the rule. An entry choice stays fixed when the source changes controller.
// Use [ChooseBasicLandTypeAsEnters] for a land choice before battlefield entry.
// [Permanent.ChosenBasicLandType] keeps the typed choice separate from color.
// [BecomesChosenBasicLandTypeAttachedEffect] applies it to the current host.
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
// Use [WhenAttacksUnblockedThisTurn] to watch one incarnation this turn.
// Each legal blocker declaration can trigger it. Source departure and control
// changes do not change the saved controller. Cleanup ends the subscription;
// pending triggers retain their saved body and references.
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
// Use [SourceAttackHistoryThisTurn] for the original source's current-turn
// attack history. A captured incarnation keeps that history after departure.
// Use [CreatureDiedThisTurnCondition] for a creature death under either player.
// Both conditions read current history at detection and resolution.
// Cleanup clears this history. Information reads do not permit absent actions.
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
