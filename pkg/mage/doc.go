// Package mage implements the two-player Magic: The Gathering rules engine.
//
// # Subsystem Architecture and Ownership
//
// [Game] serves as the aggregate root and coordinator of the engine. Rather than
// implementing every domain rule directly, [Game] composes cohesive, stateful
// subsystems that each own their state invariants and cloning behavior:
//
//   - [ZoneSystem]: Owns battlefield storage, exile visibility,
//     entering-permanent tracking,
//     and Last-Known Information (LKI) snapshots.
//   - [TurnSystem]: Owns turn counters, current phase and step, active player
//     index, extra turns, [TurnSchedule] lifecycle, and one-shot untap skip counters.
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
// # Card Definitions and Instances
//
// [CardDefinition] owns one complete printed face. Its data includes the name,
// type line, mana cost, colors, power, toughness, loyalty, Oracle text, abilities,
// casting costs, cast targets, Aura metadata, and optional flip face.
// [CardDefinitionSpec] is the input to [NewCardDefinition]. The constructor
// copies input containers. Read methods return private container copies.
// Abilities, effects, costs, targets, filters, and callbacks are specifications.
// Finish their construction before publication. Do not change them after that.
// Callbacks must capture only immutable data.
//
// Every [Card] uses this representation. [Card.Definition] retains the original
// printed face. The embedded [CardDefinition] supplies the current copiable face.
// Construction options and characteristic setters create a private definition.
// They do not change another card or a published definition. Factory registration
// finalizes the printed face after the factory completes.
// [NewCard], [NewCardFromDefinition], and [CardDefinition.NewInstance] create
// identities from shared definitions. The last two also assign an owner.
// [Registry.RegisterDefinition] and [Registry.RegisterDefinitions] publish
// definitions. Batch registration checks all conflicts before publication.
// The pkg/mage/mtgish package owns its JSON schema, parser, and compiler.
//
// [Card.Abilities] returns a private slice. [Card.AllAbilities] permits iteration
// without a slice copy. Both include keyword and capability specifications.
// Use [StaticKeyword] and [StaticCapability] to declare these attributes.
// Ward supplies its attribute through its triggered ability.
//
// [AbilityContext] supplies source, controller, and ability index for evaluation.
// Definitions do not store these bindings. [Game] owns activation counts. Counts
// distinguish source incarnations, repeated abilities, and separate grants.
// Cleanup clears turn limits. A source that leaves and returns gets a new count.
// Each active continuous effect owns its source binding and execution state.
// Static definitions can be shared by cards, permanents, and simulation branches.
// Layer application keeps registration order within each layer.
//
// [Card.NewCopy] creates a new identity and retains the copiable definition.
// [Card.CloneInstance] retains identity for a simulation branch. [Game.Clone]
// copies runtime state and preserves object references within the branch.
// Definitions are shared. Battlefield counters, status, and granted-ability
// lists remain on [Permanent]. Activation counts and active effect state remain
// on [Game]. Last-known information has a private card instance.
//
// [Card.CopiableValues] captures the current definition. [Card.ApplyCopiableValues]
// applies that definition without changing identity, owner, or printed face.
// Execution state is not copied. [Card.CloneFrom] and [Card.Copy] retain their
// compatibility behavior. [CopySpellOnStack] copies announcement choices as a
// separate operation. [Permanent.CopiableValues] accounts for face-down values.
// [EffectManager.AddCopyEffect] and [EffectManager.UpdateCopyEffect] retain
// snapshots. Reapplication preserves activation state. [CopyOptions] can keep
// source colors or add an ability to the copied values. [CopyCreatureOnETB]
// permits an optional choice before entry replacements.
//
// [WithFlipFace] supplies the alternative definition. Both faces retain the
// normal mana cost and color. Copies acquire both faces. [FlipSource] and
// [Game.FlipPermanent] change flipped status without a zone change.
//
// # Effect References and Announcements
//
// [EffectContext] carries source snapshots, controller, X, chosen modes, target
// groups, the triggering event, and execution variables. [ValueSource.Resolve],
// [PlayerSelector.Select], and [ObjectSelector.Select] read this context.
// [VarInt], [VarPlayer], and [VarObject] read variables without retaining context.
// [TargetGroup] and [FirstTarget] read declared targets. [EventSource],
// [EventPlayer], [EventTarget], and [EventAmount] read the triggering event.
// [SourceObject] identifies the resolving source.
//
// [StackObject.Targets] contains only declared targets. [StackObject.TriggerEvent]
// retains a separate snapshot. [StackObject.TargetGroups] retains group boundaries,
// including empty optional groups. Illegal targets become nil identifiers at
// resolution without changing those boundaries. Target specifications store no
// announcement choices. Stack targets also retain permanent incarnations.
//
// [Game.CastSpellWithChoices] and [Game.ActivateAbilityWithTargetGroups] accept
// explicit groups. Use these APIs when more than one group has a variable count.
// Existing flat-target APIs retain their count-allocation convention. Prompts
// and spell copies retain the groups chosen for each declaration.
// [CopySpellAndMayChooseNewTargetsWithEffects] lets the copy's controller
// choose targets, then applies the supplied effects with the copy as target.
//
// Existing effect helpers and [FuncEffect] retain event-reference adapters for
// old card code. These adapters do not add event objects to stack targets.
// New effects must use explicit selectors. Use [DealDamage], [Destroy],
// [GainLife], [LoseLife], [DrawForPlayers], [Discard], [MoveObjects],
// [AddCounters], and [RemoveCounters] for the common operations. Scalar forms
// of existing constructors remain available. DrawForPlayers avoids a name
// conflict with the Draw step constant. [ObjectFilter] evaluates shared card
// and permanent characteristics. Its adapters support existing zone queries.
// [ManaProducer] and [ManaProductionEffect] supply common mana output metadata.
//
// Effects that change their source must retain its incarnation. Use
// [Game.ResolvingSourcePermanent] for legacy callbacks. Explicit object effects
// check the source snapshot before they change a permanent.
//
// # Controllers and Layer 2
//
// A [Permanent] has a default controller and a computed controller.
// [NewPermanent] initializes both values from the controller given to
// [PutOnBattlefield]. This behavior supports effects that put an opponent card
// onto the battlefield under your control (CR 110.2a). [Permanent.ControllerID]
// returns the computed controller.
//
// [EffectManager.Apply] records the current controller. It resets the
// computed controller to the default value. It then applies control-changing
// effects in Layer 2 by timestamp order (CR 613.1b, 613.7). A real control
// change occurs only when the final controller differs from the recorded
// controller. A real change updates ability context and timing restrictions
// for attacks and tap or untap symbols (CR 302.6). Triggers keep the
// controller recorded at trigger creation (CR 603.3a). Triggers that trigger
// when a permanent leaves the battlefield use last-known information (CR 603.6c).
//
// Card code must read [Permanent.ControllerID]. Card code must not change
// controller fields directly. Use [GainControl] to build resolving control
// effects:
//
//	GainControl().Targeting(ToTarget()).Until(EndOfTurn)
//
// For conditional control, use While with [ControlConditionData] values. For
// effects maintained while a source remains tapped, use:
//
//	GainControl().While(
//		ControlSourceControlledByEffectController{},
//		ControlSourceTapped{},
//	).TapMaintained()
//
// If an effect determines targets at resolution, call [Game.AddControlEffect]
// with a [ControlEffectSpec]. [ControlAttached] provides the static effect for
// control Auras. [ControlChangeContinuous] is an alias for backward compatibility.
//
// [ExchangeControl] records the current controllers of two permanents. It then
// creates indefinite Layer 2 effects for both permanents. If either permanent
// is missing, if both permanents share a controller, or if either permanent
// cannot change control, the exchange does not occur.
// [ExchangeControlOfTargets] is the resolving effect for two targets.
// [ExchangeControlOfTargetsSharingPermanentType] also verifies that both
// permanents share an artifact, creature, or land type. The target selector
// [TargetRandomActivePlayerExchangePair] selects a pair of permanents when an
// upkeep trigger goes on the stack.
//
// # One-Shot Untap Skipping
//
// [Game.SkipNextUntap] marks a permanent to skip its next untap step. The
// permanent skips the untap step of its current controller. The marker
// persists through control changes and turn cycles. The marker does not
// clear while the permanent is untapped or is not chosen to untap. The engine
// removes the marker when the permanent leaves the battlefield.
// [SkipNextUntapTarget] is the resolving effect. [StunCreature] is an alias
// for backward compatibility.
//
// # Library Removal
//
// [Player.RemoveFromLibrary] removes one card by ID. It returns the stored card
// and true on success, or nil and false if the card is absent. The remaining
// cards keep their order. Removal keeps shared slice storage unchanged.
// Callers must check success before they move the card to another zone.
//
// # Graveyard Reanimation
//
// [MoveFromAnyGraveyard] returns the player who owned the graveyard that
// contained the card. This behavior ensures accurate zone-change events.
// [ReturnTargetFromAnyGraveyardToBattlefield] puts that card onto the
// battlefield. The controller of the resolving effect becomes the default
// controller of the card. When temporary control effects expire, control
// returns to that player instead of the card owner. [EventSourceWasColor] and
// [EventSourceWasNotColor] check the colors of an event source. They check
// active battlefield permanents or last-known colors before zone change.
//
// # Object Colors and Protection
//
// [ChangeObjectColor] sets a new color on a permanent or a spell on the stack.
// [ChangeColorEffect] provides this behavior for resolving card effects. A
// permanent keeps this color only while it remains on the battlefield. A
// permanent spell keeps this color when it enters the battlefield (CR 400.7a).
// A copy of that spell does not copy the color change. [EffectiveColors]
// accepts an identifier for a permanent, stack object, or spell source. It
// returns the current layer-5 colors of the object. Cloned game states retain
// independent color states.
//
// [ProtectionFromColor] and [ProtectionFromColors] use [EffectiveColors] to
// check legality for targeting, damage, blocking, fighting, and attachment.
// Protection from card types, subtypes, or all characteristics uses
// [CardFilter].
//
// # State Triggers and Simultaneous Deaths
//
// [NewStateTriggered] records a trigger when its condition becomes true,
// including between resolving instructions. [FireEvent], [ApplyEffect],
// [CompositeEffects], and [Pipeline] check these conditions. A trigger stays
// suppressed while its original instance is pending, on the stack, or resolving.
// It can trigger again after that instance leaves the stack, even if its
// condition stays true. A permanent that leaves and returns has a new instance.
//
// [DestroyPermanents] preserves the state before a simultaneous destruction
// for leave-the-battlefield triggers. Mass destruction effects and lethal-damage
// state-based actions use this path. Zero-toughness and zero-loyalty deaths
// share the same observation batch. Conditions use the state before the batch;
// trigger effects resolve against the current game state.
//
// # Replacement Choices
//
// When multiple replacements apply to an existing action, the affected player
// chooses one with [Player.ChooseMode]. For a permanent, its controller chooses.
// The pipeline rechecks applicability after each choice and uses each effect
// at most once. Prevention has no automatic precedence over other replacements.
// These actions do not include entering the battlefield; the special ordering
// of entry replacements is outside this pipeline.
//
// # Sacrifice
//
// [Sacrifice] executes a sacrifice action. It emits a zone-change event and an
// [EvtSacrifice] event. [SacrificeSource], [SacrificeTarget], and
// [SacrificeGathered] provide resolving effects. [SacrificeSourceCost] and
// [SacrificeMatchingCost] provide activation costs. Sacrifice costs record the
// identifier of the sacrificed object. Resolving effects can read the
// last-known information of the object with [LastSacrificed].
//
// [SacrificeSource] and [SacrificeSourceStep] require the resolving controller
// to control the source when the effect applies.
//
// # Damage-Source Choices
//
// Effects that state "a source of your choice" do not target. The player
// selects the source when the effect resolves. [ChooseDamageSource] prompts the
// player to choose a permanent or a spell on the stack.
// [AddReverseDamageShield] prevents damage from that source and adds life to
// the player until end of turn. [AddReverseDamageShieldStep] combines these
// actions in card pipelines.
//
// # Player Selectors
//
// [SelectTargetPlayer] selects a declared player target. Existing card helpers
// can also read their event player through the compatibility adapter. New
// triggers must use [EventPlayer]. [SelectTargetPermanentController] reads
// the controller of a selected permanent, including last-known information.
//
// # Combat Damage Assignment
//
// If a player does not assign combat damage manually, a blocked attacker
// assigns lethal damage to each blocker in damage assignment order. The
// attacker assigns all remaining damage to the last blocker. If the attacker
// has trample, it assigns remaining damage to the defending player or
// planeswalker. Damage events record the total damage assigned and dealt,
// including excess damage above lethal toughness.
// [PreventCombatDamageToAndBy] installs an end-of-turn replacement effect
// preventing all combat damage dealt to and dealt by a specified creature;
// [PreventCombatDamageToAndByTarget] provides the resolving card effect.
//
// # Targeting Helpers
//
// Compose characteristic restrictions with [ObjectAnd], [IsCardType], and
// [ManaValueIs]. For example, an artifact with mana value X uses:
//
//	TargetPermanent(IsCardType(TypeArtifact), ManaValueIs(EqualTo(XValue())))
//
// [ObjectFilter.Kind], [ObjectFilter.Children], and
// [ObjectFilter.ManaValueComparison] retain the structure for inspection.
// [PossibleTargets] accepts a [TargetingContext]. Set XSpecified for an exact
// announcement, including X=0. An unspecified X leaves dependent comparisons
// open for discovery. Queries do not change resolution state.
// [Game.SpellTargetOptions] pairs affordable X values with legal candidates for
// each target group. Required equality filters supply candidate X values from
// battlefield mana values. Casting and resolution check the announced X.
//
// [TargetAuraAttachedToCreatureYouControl] targets an Aura attached to a creature
// controlled by the ability's controller.
//
// # Mana-Production Metadata
//
// [ManaProductionsForAbility] provides the output profile of tap-for-mana
// abilities to systems such as AI evaluation. It identifies [ManaAbility]
// values created by [WithManaAbility] or [WithMultiManaAbility]. It also
// identifies activated abilities that combine a tap cost with only [AddMana]
// or [AddAnyMana] effects.
//
// [SolveMana] and [CanSolveMana] search a unified model of all mana abilities
// on permanents. This model includes targetless mana abilities with both a mana
// cost and a tap cost. The planner handles choices for any color and any
// combination, spending restrictions, conversions, bonus mana, and ordered
// activation costs. It keeps separate abilities distinct.
//
// [CanAfford], [MaxXValue], and automatic mana payments use this planner.
// Automatic payment executes the planned ability and choice. It pays the
// activation cost before it uses the produced mana. Set
// [ManaProduction.Restriction] to define spending restrictions. Manual and
// planned activations keep this restriction on each produced mana unit.
//
// The solver creates combination choices on demand. It groups equivalent
// surplus distributions to simplify calculation. The solver also groups
// equivalent static sources and rejects impossible costs early.
//
// [ManaPool.CanPay] and [ManaPool.Pay] use the same logic for colored and
// hybrid symbols, including one-way conversions. This shared logic guarantees
// consistent results between payment checks and actual payments.
//
// [EvtTappedForMana] occurs after a mana ability with a tap cost resolves and
// produces mana (CR 106.12a). The event records the permanent in SourceID, the
// player in PlayerID, and the produced mana in Amount. Normal tap actions emit
// only [EvtTapped]. Cards such as Manabarbs listen for [EvtTappedForMana] and
// use [SelectTargetPlayer] to identify the activating player. Resolving effects
// read the mana source from [EventSourceID]. If another cost moved the source
// off the battlefield, trigger conditions can read its last-known information.
//
// # Total-Cost Payment Transactions
//
// Spell casting, activated abilities, attack costs, resolving [TryPayMana]
// payments, and standalone [ManaCostPayment] operations use a single
// total-cost transaction. The transaction combines all mana components, locks
// alternative choices, plans mana abilities, and verifies all payments on an
// isolated game state. [Commit] updates the active game only after verification
// succeeds. A canceled transaction leaves mana pools, life totals, tapped
// states, cards, counters, and other resources unchanged.
//
// [AutoTapHint.ReservedSources] excludes specified permanents from mana
// planning. Attack-cost payments reserve the attacking creature. An activated
// ability with a tap cost reserves its source permanent. This reservation
// prevents automatic mana abilities from invalidating the primary action.
// Mana spent to activate a mana ability does not count toward
// [CastContext.ColorsSpent]. Only the final mana payment of a spell contributes
// to cast metadata.
//
// [WithDynamicManaAbility] and [NewDynamicManaAbility] calculate mana
// production from the source permanent when queried or activated. Solvers use
// [ManaProductionsForAbilityInGame] to find the current production of the
// source. Post-production effects execute immediately after mana is added,
// without using the stack. [ChosenColorManaProductions] resolves mana for
// sources that produce mana of their current [ChosenColor]. Resolver callbacks
// must be pure functions without side effects. Affordability checks and search
// algorithms can call them multiple times.
//
// # Ante and Ownership
//
// [NewGame] disables ante. [NewGameWithAnte] enables ante. It validates the
// selected nontoken cards and moves them from libraries into the public ante
// zone. [AnteCards], [AnteCardsOwnedBy], [MoveToAnte], and [RemoveFromAnte]
// provide ante zone operations. Cards can use [TargetCardYouOwnInAnte],
// [DiscardHand], [AnteLibraryTop], and [ExchangeTargetAnteCardWithLibraryTop]
// without modifying player zones directly. [AnteLibraryTop] accepts a
// [PlayerSelector] to support single-player or all-player effects.
//
// [ChangeOwner] changes the owner of a card without moving the card or
// changing its controller. It operates in all card zones. It updates the card in place.
// Each cloned game has private card state. [OwnedBy] and
// [NotOwnedBy] filter permanents by ownership rather than control. When a game
// concludes with one winner, [AnteResult] gives the ante cards to that winner.
// It returns all ownership changes.
//
// Pipelines can record a permanent owner with [SnapshotPermanent]. They can
// change ownership with [ChangeOwnerGathered]. They can move an exiled card to
// the graveyard of its current owner with [MoveExiledGatheredToGraveyard].
// [IfPlayerPays] provides branches for paid and unpaid outcomes. [EffectIfPaid]
// and [UnlessTargetPays] provide simplified wrappers for this check.
//
// # Variable Target Counts
//
// [TargetXCreatures] and [TargetXPermanents] require exact counts of distinct
// targets equal to the declared value of X. [TargetBounds] provides the
// target count bounds to callers that prompt for targets.
// [TargetUpToXCreatures] allows zero through X targets for random-count
// effects. [TargetOneToXCreatures] requires one through X targets, or zero
// targets when X is zero.
//
// # Random Selection
//
// [RandIntn] is the core function for random integers in card effects. It
// returns zero when given a nonpositive upper bound. [SetRandomResults] sets
// deterministic values for testing. The engine reads these values in order and
// scales them to the requested range. Cloned games keep separate copies of the
// remaining values. Random selection functions for graveyards, hands, players,
// colors, permanents, spells, damage targets, and coin flips use this
// function.
//
// [RandomPlayer] and [RandomPermanent] return nil if no valid choices exist.
// [RandomSpellOrPermanent] and [RandomDamageTarget] return [uuid.Nil] if no
// valid choices exist. An empty choice does not use a test value.
// [RandomPermanent] accepts a [PermanentFilter]. The zero value of the filter
// matches all permanents on the battlefield. [RandomDamageTarget] selects
// uniformly among all creatures and players. [RandomSpellOrPermanent] selects
// among battlefield permanents and stack spells.
//
// [ApplyToRandomPermanent], [ApplyToRandomSpellOrPermanent],
// [ApplyToRandomPlayer], and [ApplyToRandomDamageTarget] wrap an [Effect].
// They choose the target during effect resolution.
// [SetSourceChosenColorAtRandom] and [ChangeSourceToRandomColor] set a random
// color on the source object.
// [ChooseRandomCreatureSubtypeFromTargetLibrary] selects a creature subtype
// from a library and stores it in the ChosenSubtype field.
// [TargetCreatureOfSourceChosenSubtype] targets creatures that match that
// subtype. [IsEnchanted] matches permanents that have an attached Aura.
//
// [TargetRandom] wraps a [Target]. It selects valid targets at random when
// the spell or ability goes on the stack. Fixed target counts do not change.
// For variable bounds, the player chooses only the target count.
// [TargetRandomCount] also chooses the count at random within legal bounds.
// Both functions preserve standard target validation for protection, zone
// changes, and resolution legality. Random targeting supports spells,
// activated abilities, triggered abilities, and alternate casting costs.
// [TargetUpToNCreaturesOpponentControls] defines optional targets controlled
// by an opponent.
//
// [ManaCostPerTarget] adds an extra cost for each chosen target, such as {R}
// for each target. The engine selects random targets before it evaluates total
// costs. It combines all mana costs to evaluate affordability and auto-tap.
// This prevents generic costs from consuming mana needed for colored target
// costs. If the total cost cannot be paid, the action stops without spending
// resources.
//
// AttachedCantAttackUnlessPays adds an Aura-defined attack cost to the
// enchanted creature. SourceHasManaAbilitiesOpponentLandsCouldProduce derives
// Fellwar Stone-style colored mana abilities from opposing lands. A negative
// MaximumHandSize means no maximum; SetNoMaximumHandSize installs that rule in
// a continuous-effect cycle. AddSourcePreventionShield prevents a bounded
// amount of damage from one source to one player. TryPayMana pays resolving
// effect costs from floating mana and activatable mana sources.
// AddDamagePreventionRule accepts WithToPlayer to scope a source/filter rule to
// damage dealt to one specific player.
//
// # Reusable Classic Card Effects
//
// The file effect_classic_cards.go defines constructor functions for shared
// card effects. Both original cards and cards that copy their behavior use
// these constructors:
//   - [BerserkEffect], [BloodLustEffect], [FlyingCarpetEffect], [GiantGrowthEffect]
//   - [HelmOfChatzukEffect], [HurrJackalEffect], [LaceEffect], [LesserWerewolfCounterEffect]
//   - [LightningBoltEffect], [ProdigalSorcererEffect], [SorceressQueenEffect], [StaffOfZegonEffect]
//   - [SwordsToPlowsharesEffect], [TawnosWandEffect], [TwiddleEffect], [UnsummonEffect]
//
// The calling card selects the target. This separation allows standard spells
// and random-selection effects to apply the same effect logic.
//
// The card Whimsy and related cards share these effect constructors:
//   - [AladdinsRingEffect], [AncestralRecallEffect], [BoomerangEffect], [BottleOfSuleimanEffect]
//   - [CrumbleEffect], [DisenchantEffect], [DisruptingScepterEffect], [FissureEffect]
//   - [FogEffect], [HealingSalveGainEffect], [HealingSalvePreventionEffect], [MillstoneEffect]
//   - [NevinyrralsDiskEffect], [PandorasBoxEffect], [SindbadEffect], [TheHiveEffect]
//
// Whimsy handles only the random selection and the execution sequence.
//
// # Discard Replacement
//
// [PlayerDiscardByEffect] processes discards caused by card effects through the
// replacement pipeline. [PlayerDiscard] handles discards caused by costs,
// turn-based actions, or rules. [DiscardToLibraryReplacement] provides the
// optional replacement effect for Library of Leng.
//
// # Continuous Rules Helpers
//
// [TargetOpponentChoice] wraps a [Target] that an opponent chooses. The engine
// sends the prompt to the opponent. It evaluates targeting legality by using
// the controller of the action. [SnapshotTarget] saves a target by index for
// later pipeline steps.
//
// [FightGathered] causes two creatures to fight only if both remain creatures
// on the battlefield (CR 701.14). [NewModalActivated] creates an activated
// ability with modes. Each mode has distinct targets and effects. The player
// selects the mode before selecting targets or paying costs.
//
// [AttachedCantAttackUnlessPays] adds an attack cost to an enchanted creature.
// [SourceHasManaAbilitiesOpponentLandsCouldProduce] adds mana abilities based
// on lands controlled by opponents. A negative [MaximumHandSize] indicates no
// hand size limit. [SetNoMaximumHandSize] applies this rule as a continuous
// effect. [AddSourcePreventionShield] prevents a specified amount of damage
// from one source to one player. [AddHalfDamageFromSourcePreventionShield]
// prevents half the damage (rounded down) from one source to one player.
// [TargetPermanentManaValue] provides a [ValueSource] resolving to the target
// permanent's mana value. [TryPayMana] pays effect costs from floating
// mana and available mana sources.
//
// # Basic Land Type Changes
//
// [BecomesBasicLandTargetEffect] changes a land into a Plains, Island, Swamp,
// Mountain, or Forest. In layer 4, it replaces the existing land subtypes.
// It removes printed abilities, copied abilities, and keyword abilities. It
// grants the intrinsic mana ability of the new basic land type. Abilities
// granted by other effects remain. Printed characteristics return after the
// effect ends.
//
// For effects that last until the source leaves the battlefield, use
// [BecomesBasicLandTargetUntilSourceLeaves]. This effect persists if the source
// phases out. [BecomesBasicLandAttachedEffect] and
// [BecomesChosenBasicLandAttachedEffect] apply to land Auras.
// [BecomesBasicLandsEffect] applies this layer-4 change to all permanents that
// match a filter, such as for Blood Moon.
//
// # Subtype Changes
//
// [SetSubtypes] and [AddSubtypes] modify subtypes in layer 4 by subtype family.
// A set operation replaces only the specified subtype family. An add operation
// preserves all existing subtypes. For example, changing the creature subtype
// of a Land Creature preserves its land subtypes:
//
//	SetSubtypes(targetID, SubtypeCreature, EndOfTurn, "Human")
//	AddSubtypes(targetID, SubtypeCreature, EndOfTurn, "Dinosaur")
//
// [SubtypeArtifact], [SubtypeCreature], [SubtypeEnchantment], [SubtypeLand],
// [SubtypePlaneswalker], and [SubtypeSpell] define the subtype families
// (CR 205.3). The target must have the matching card type when the effect
// applies. Setting a land subtype to a basic land type removes existing
// abilities and adds intrinsic mana abilities (CR 305.7).
// [BecomesBasicLandTargetEffect] and its variants provide wrappers for this
// rule.
//
// # Removing Abilities
//
// RemoveAllAbilities applies "loses all abilities" in layer 6. The operation
// removes abilities computed up to its timestamp, including keyword attrs and
// runtime activated, triggered, static, and mana abilities. A later layer-6
// effect can grant an ability back. RemoveAllAbilitiesFromAll applies the same
// operation to a filter for Humility-style effects. Basic land subtype changes
// reuse the same removal operation in layer 4, so ordinary layer-6 grants and
// keyword counters are applied afterward and remain as required by CR 305.7.
//
// RevealTopCardOfLibrary exposes the top card of the source permanent's
// controller's library. RevealTopCardsOfAllLibraries exposes every player's
// library top for symmetric effects such as Field of Dreams. Both permissions
// are recomputed with continuous effects and disappear when their source
// leaves the battlefield.
//
// # Import Compiler Primitives
//
// ObjectFilter uses ObjectView for cards, live permanents, and LKI snapshots.
// LKIView includes ObjectView. ObjectFilter.Match accepts an optional
// AbilityContext after the game reader. ObjectAnd, ObjectOr, and ObjectNot
// preserve this context. Use OtherThan(SourceObject()), ObjectControlledBy,
// and ObjectOwnedBy for relative references. ObjectHasSubtype and
// ObjectHasSuperType test subtypes and supertypes. Cards and Permanents
// adapters accept the same optional context. Card views have no controller;
// spell trigger views use the caster as controller. Match rejects nil objects,
// including typed nil pointers. ObjectControlledBy and ObjectOwnedBy require
// a nonzero player identifier on the object. Live object views include current
// power and toughness bonuses even without a game reader.
//
// TriggerSpec separates event checks from ability effects. NewTriggeredAbility
// accepts a TriggerSpec, effects, and WithTarget options. Use TriggerOr for
// alternative events. The permanent entry, death, departure, and attack
// constructors test ObjectFilter with the ability context. LookupObject
// supplies live state or LKI. Player event constructors use PlayerSelector.
// Existing NewTriggered helpers remain available.
//
// TargetPermanent accepts ObjectFilter and PermanentFilter values. TargetCreature
// and its variants use the same permanent targeting implementation. Use
// PossibleTargets to check target legality with source, controller, and X.
// PermanentTarget.Filter supplies only checks without an ability context. The
// CreatureTarget wrapper preserves type checks in existing AI code. TargetGroup,
// FirstTarget, and VarObject return Selector, which supports both object effects
// and player effects. TargetsOnlyCreatures supplies a conservative static
// restriction for AI inspection. ObjectAnd and ObjectOr preserve this restriction
// when their child filters guarantee it.
//
// StaticInt reads Fixed, Add, and Mul expressions without an execution context.
// It returns false for dynamic values. Boost uses this check for text and AI
// properties. It does not resolve dynamic values during static inspection.
//
// ActionDefinition.Type applies CR 605.1a to activated actions. A targetless
// action with a ManaProductionEffect is a mana ability, even with other effects.
// WithLoyaltyAbility excludes a loyalty ability from this classification; it
// does not implement loyalty costs or timing. Mana actions resolve immediately.
// ManaProfile exposes only effects that the automatic payment planner can
// execute in full. Other mana actions remain available for manual activation.
// TapForMana checks activation restrictions and records activation counts and
// events for ActionDefinition mana abilities. Production metadata describes
// potential output; it does not grant permission to activate an ability.
package mage
