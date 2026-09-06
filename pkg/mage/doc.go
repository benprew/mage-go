// Package mage implements the two-player Magic: The Gathering rules engine.
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
// # Sacrifice
//
// [Sacrifice] executes a sacrifice action. It emits a zone-change event and an
// [EvtSacrifice] event. [SacrificeSource], [SacrificeTarget], and
// [SacrificeGathered] provide resolving effects. [SacrificeSourceCost] and
// [SacrificeMatchingCost] provide activation costs. Sacrifice costs record the
// identifier of the sacrificed object. Resolving effects can read the
// last-known information of the object with [LastSacrificed].
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
// [SelectTargetPlayer] reads the first target of a resolving effect. It
// returns the target only if the target is a player. Helper functions for
// draw, discard, and sacrifice triggers set that target to the event player.
// Zone-change triggers set that target to the moved object. Use
// [SelectTargetPermanentController] when an effect requires the controller
// of that object.
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
// changing its controller. It operates in all card zones. It replaces the card
// copy-on-write to keep cloned search states isolated. [OwnedBy] and
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
// [RandomCounterDistribution] distributes counters when a stack object is
// created. It gives one counter to each target, then distributes remaining
// counters with [RandIntn]. The assignment remains fixed on the stack object
// even if a target becomes illegal. [TargetSpellOrPermanent] targets one spell
// on the stack or one permanent on the battlefield. It excludes activated and
// triggered abilities. If the targeted object changes zones before resolution,
// the target becomes illegal.
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
// [RemoveAllAbilities] applies "loses all abilities" in layer 6. The operation
// removes abilities created before its timestamp. It removes keyword
// attributes, activated abilities, triggered abilities, static abilities,
// and mana abilities. Later layer-6 effects can grant new abilities.
// [RemoveAllAbilitiesFromAll] applies this effect to all permanents that match
// a filter, such as for Humility. Basic land subtype changes use this removal
// in layer 4. As a result, layer-6 ability grants and keyword counters apply
// afterward and persist (CR 305.7).
package mage
