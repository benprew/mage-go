// Package mage implements the two-player Magic: The Gathering rules engine.
//
// # Controllers and Layer 2
//
// A Permanent has a default controller and a computed controller. NewPermanent
// initializes both from the controller supplied to PutOnBattlefield; this is
// important for effects that put an opponent-owned card onto the battlefield
// under your control (CR 110.2a). ControllerID returns the computed value.
//
// EffectManager.Apply snapshots the current controller, resets the computed
// value to the default, and applies control-changing effects in Layer 2 and
// timestamp order (CR 613.1b, 613.7). Only a difference between the snapshot
// and the final result is a real control change. A real change updates ability
// context and continuous-control timing for attack and {T}/{Q} restrictions
// (CR 302.6). Trigger controllers remain snapshots taken when the trigger is
// created (CR 603.3a); leaves-the-battlefield triggers use LKI (CR 603.6c).
//
// Card code reads Permanent.ControllerID and must not directly mutate control.
// GainControl is the reusable resolving-effect builder:
//
//	GainControl().Targeting(ToTarget()).Until(EndOfTurn)
//
// Conditional control uses While with ControlConditionData values. For
// tap-maintained effects such as Willow Satyr, use:
//
//	GainControl().While(
//		ControlSourceControlledByEffectController{},
//		ControlSourceTapped{},
//	).TapMaintained()
//
// Code that discovers targets dynamically at resolution uses
// Game.AddControlEffect with a ControlEffectSpec. ControlAttached is the
// canonical Control Magic-style static effect. ControlChangeContinuous is its
// compatibility name.
//
// ExchangeControl atomically snapshots two battlefield permanents' current
// controllers and installs simultaneous indefinite Layer 2 effects. If either
// object is missing, both already have the same controller, or either can't
// change control, no part occurs. ExchangeControlOfTargets is the resolving
// two-target effect. ExchangeControlOfTargetsSharingPermanentType additionally
// revalidates a shared artifact, creature, or land type. The dependent
// TargetRandomActivePlayerExchangePair target chooses a Power Struggle-style
// pair when an upkeep trigger goes on the stack.
//
// # One-shot untap skipping
//
// Game.SkipNextUntap marks a specific battlefield object to skip its next
// attempted untap during its then-controller's untap step. The marker follows
// control changes and turn ordering, is not consumed while the object is
// untapped or otherwise isn't selected to untap, and is removed when the
// object leaves the battlefield. SkipNextUntapTarget is the resolving effect;
// StunCreature remains its compatibility name.
//
// # Graveyard reanimation
//
// MoveFromAnyGraveyard reports the player whose graveyard actually contained
// the card so zone-change events are attributed correctly.
// ReturnTargetFromAnyGraveyardToBattlefield puts that card onto the battlefield
// with the resolving effect's controller as its default controller. Later
// control effects therefore expire back to that player rather than the owner.
// EventSourceWasColor and EventSourceWasNotColor inspect either a live event
// source or its last-known battlefield colors, including color-changing
// effects that applied before it left the battlefield.
//
// # Object colors and protection
//
// ChangeObjectColor assigns a new color to either a permanent or a spell on
// the stack; ChangeColorEffect exposes the same behavior to resolving card
// effects. A changed permanent keeps that color only for its current
// battlefield incarnation. A changed permanent spell keeps the change as it
// becomes a permanent, per CR 400.7a, but a copy of that spell does not copy
// the color-changing effect. EffectiveColors accepts a permanent ID, stack
// object ID, or spell source ID and reports the object's current layer-5
// colors. Game clones own independent spell-color state.
//
// ProtectionFromColor and ProtectionFromColors consult EffectiveColors for
// targeting, damage, blocking, fighting, and attachment legality. Protection
// from card types, subtypes, or everything continues to use its CardFilter.
//
// # Sacrifice
//
// Sacrifice performs the named action and fires both its battlefield-to-
// graveyard zone-change event and EvtSacrifice. SacrificeSource,
// SacrificeTarget, and SacrificeGathered provide resolving effects;
// SacrificeSourceCost and SacrificeMatchingCost provide costs. Sacrifice costs
// record the sacrificed object's ID; resolving effects can read its generic
// PermanentLKI, including power and toughness, with LastSacrificed.
//
// # Damage-source choices
//
// Effects that say "a source of your choice" make a non-targeting choice when
// they resolve. ChooseDamageSource prompts for a permanent or spell on the
// stack, and AddReverseDamageShield binds its one-shot prevention and life gain
// to that source for the rest of the turn. AddReverseDamageShieldStep combines
// those operations for card pipelines.
//
// # Player selectors
//
// SelectTargetPlayer reads the first resolving target and returns it only when
// it identifies a player. Trigger helpers for draw, discard, and sacrifice
// events bind the event's player in that position. Zone-change triggers bind
// the moved object instead; use SelectTargetPermanentController when an effect
// refers to that object's controller.
//
// # Combat damage assignment
//
// A blocked attacker without an explicit player-supplied assignment assigns
// lethal damage to each blocker in order, then assigns all remaining damage to
// the final blocker. With trample, remaining damage is dealt to the defender
// instead. Damage events report the full amount assigned and dealt, including
// damage beyond a blocker's lethal toughness.
//
// # Mana-production metadata
//
// ManaProductionsForAbility exposes the production profile of tap-for-mana
// abilities to engine consumers such as AI evaluation. It recognizes both
// ManaAbility values created by WithManaAbility or WithMultiManaAbility and
// equivalent activated abilities built from Tap with only AddMana or
// AddAnyMana effects. SolveMana and CanSolveMana search one unified source
// model containing every exact mana ability on each permanent, including
// targetless mana-producing abilities with both a mana cost and Tap. The
// planner preserves AnyColor and AnyCombination choices, mana restrictions,
// conversions, bonus mana, and ordered activation costs without merging the
// amounts of separate abilities. CanAfford, MaxXValue, and automatic mana
// payment use the same planning behavior. Automatic payment executes the exact
// planned ability and production choice, paying its activation cost before
// using the produced mana. Set ManaProduction.Restriction to declare spending
// restrictions on a mana ability; both prompted and planned activation retain
// that restriction on every produced mana unit. AnyCombination choices are
// generated lazily and collapse surplus distributions that are equivalent for
// the requested cost and any mana-source activation costs. Search likewise
// canonicalizes interchangeable static sources and rejects costs whose total
// or colored requirements cannot be reached before enumerating source subsets.
// Dynamic and post-production abilities remain distinct. ManaPool.CanPay and
// ManaPool.Pay share one exact allocation path for colored and hybrid symbols,
// including optional one-way conversions, so overlapping hybrid choices cannot
// make affordability and final payment disagree.
//
// EvtTappedForMana fires after a mana ability with {T} in its activation cost
// resolves and actually produces mana (CR 106.12a). Its SourceID is the tapped
// permanent, PlayerID is the activating player, and Amount is the mana produced
// by that ability. Ordinary taps continue to fire only EvtTapped. Card triggers
// such as Manabarbs should listen for EvtTappedForMana and use SelectTargetPlayer
// for the activating player. Resolving effects can read the mana source through
// EventSourceID; trigger conditions can use LKI when another activation cost
// moved it off the battlefield.
//
// # Total-cost payment transactions
//
// Spell casting, battlefield and graveyard activated abilities, attack costs,
// resolving TryPayMana payments, and standalone ManaCostPayment values share
// one total-cost transaction. It combines all mana components, locks optional
// and either choices, plans exact mana abilities, and proves every payment in
// order on isolated game state before Commit changes the live game. A rejected
// transaction leaves mana pools, life totals, tapped state, cards, counters,
// and other cost resources unchanged.
//
// AutoTapHint.ReservedSources removes permanents needed by the surrounding
// action from mana planning. Attack-cost payment reserves the attacker, and an
// activated ability with {T} reserves its source. This prevents automatic mana
// production from making the rest of the locked total cost or action illegal.
// Mana spent activating a mana ability is excluded from
// CastContext.ColorsSpent; only a spell's locked total-mana payment contributes
// to cast-time mana metadata.
//
// WithDynamicManaAbility and NewDynamicManaAbility derive productions from the
// source permanent at query and activation time. Solvers use
// ManaProductionsForAbilityInGame, so they advertise only the source's current
// production. Post-production effects run immediately after the mana is added,
// without using the stack. ChosenColorManaProductions is the reusable resolver
// for sources that produce one mana of their current ChosenColor. Resolver
// callbacks must be pure because affordability and search may query them more
// than once; cloned games resolve against their independent permanent state.
//
// # Ante and ownership
//
// NewGame keeps ante disabled. NewGameWithAnte opts in even when its ante
// lists are empty, validates the selected nontoken cards, and moves them from
// their owners' libraries into the shared public ante zone. AnteCards,
// AnteCardsOwnedBy, MoveToAnte, and RemoveFromAnte are the reusable zone APIs.
// Card definitions can use TargetCardYouOwnInAnte, DiscardHand,
// AnteLibraryTop, and ExchangeTargetAnteCardWithLibraryTop to express ante
// spells without directly manipulating player zones. AnteLibraryTop accepts a
// PlayerSelector, so the same effect covers "your" and "each player."
//
// ChangeOwner changes ownership without moving a card or changing its
// controller. It works in every modeled card zone and replaces the card
// copy-on-write so search clones stay isolated. OwnedBy and NotOwnedBy filter
// battlefield permanents by current ownership rather than control. After a
// game has exactly one winner, AnteResult awards the remaining ante to that
// winner and returns the net ownership changes from original to final owner.
// Pipelines can snapshot a permanent's owner with SnapshotPermanent, transfer
// it with ChangeOwnerGathered, and move an exiled card to its current owner's
// graveyard with MoveExiledGatheredToGraveyard. IfPlayerPays provides
// paid and unpaid branches; EffectIfPaid and UnlessTargetPays are convenience
// wrappers around it.
//
// # Variable target counts
//
// TargetXCreatures and TargetXPermanents require exactly the announced value
// of X in distinct targets. TargetBounds exposes the cast-time bounds to
// callers that generate or prompt for targets. TargetUpToXCreatures exposes
// zero-through-X bounds used by random-count effects. TargetOneToXCreatures
// exposes one-through-X bounds, with zero targets when X is zero.
//
// # Random selection
//
// RandIntn is the card-facing random integer primitive. It safely returns zero
// for nonpositive bounds. SetRandomResults scripts raw values for deterministic
// tests; values are consumed in order and normalized into the requested range,
// and cloned games retain independent copies of the remaining script. Random
// graveyard, hand, player, color, permanent, spell-or-permanent, damage-target,
// and discard helpers, as well as unscripted coin flips, use this shared
// primitive. RandomPlayer and RandomPermanent return nil when their candidate
// sets are empty; RandomSpellOrPermanent and RandomDamageTarget return
// uuid.Nil. Empty selections do not consume a scripted random result.
// RandomPermanent accepts a PermanentFilter, with its zero value matching
// every battlefield permanent. RandomDamageTarget chooses uniformly from all
// creatures and players. RandomSpellOrPermanent combines battlefield
// permanents with nonability stack objects.
//
// ApplyToRandomPermanent, ApplyToRandomSpellOrPermanent,
// ApplyToRandomPlayer, and ApplyToRandomDamageTarget wrap an existing Effect
// and select its target during resolution. SetSourceChosenColorAtRandom and
// ChangeSourceToRandomColor cover source-local random color effects.
// ChooseRandomCreatureSubtypeFromTargetLibrary stores a uniformly selected
// distinct creature subtype in the source's ChosenSubtype field;
// TargetCreatureOfSourceChosenSubtype reads that value for later targeting.
// IsEnchanted matches permanents with an Aura attached.
//
// TargetRandom wraps an ordinary Target and chooses distinct legal identities
// uniformly at random as the spell or ability is put on the stack. Fixed target
// counts remain fixed; for variable bounds, the controller chooses only the
// count. TargetRandomCount also chooses the count uniformly from the legal
// bounds. Both wrappers preserve the ordinary target on the stack, so filters,
// protection, zone changes, resolution-time legality, and fizzle rules apply
// normally. Random targeting is supported for spells, activated abilities,
// triggered abilities, and prompted alternate/copy casting paths.
// TargetUpToNCreaturesOpponentControls expresses optional opponent-creature
// targets while preserving the controller restriction.
//
// ManaCostPerTarget creates a contextual action cost such as “{R} for each
// target.” Random targets are acquired before the cost is evaluated. All mana
// portions of an action cost are combined for affordability, autotap, and
// payment, preventing generic mana from consuming colors needed by a later
// per-target component. The shared total-cost transaction checks sequential
// resource use before mana sources or nonmana costs are changed, so an
// unaffordable action fails atomically rather than paying an affordable prefix.
//
// RandomCounterDistribution assigns a ValueSource total of counters when the
// stack object is created. Each target receives one, then every remaining
// counter is assigned with RandIntn. The frozen assignment is copied with the
// stack object and is not redistributed when a target becomes illegal.
// TargetSpellOrPermanent selects one spell on the stack or permanent on the
// battlefield while excluding activated and triggered abilities. Its ordinary
// target-zone snapshot ensures that moving between those zones makes the
// original target illegal at resolution.
//
// # Reusable classic card effects
//
// Effects copied by other cards have named constructors in
// effect_classic_cards.go. BerserkEffect, BloodLustEffect, FlyingCarpetEffect,
// GiantGrowthEffect, HelmOfChatzukEffect, HurrJackalEffect, LaceEffect,
// LesserWerewolfCounterEffect, LightningBoltEffect, ProdigalSorcererEffect,
// SorceressQueenEffect, StaffOfZegonEffect, SwordsToPlowsharesEffect,
// TawnosWandEffect, TwiddleEffect, and UnsummonEffect are the canonical
// definitions used by both the original cards and cards that reproduce their
// effects. Target selection remains on the caller, allowing a normal spell to
// target conventionally and a random-effect card to select an object before
// applying the same effect.
//
// Whimsy and its source cards likewise share AladdinsRingEffect,
// AncestralRecallEffect, BoomerangEffect, BottleOfSuleimanEffect,
// CrumbleEffect, DisenchantEffect, DisruptingScepterEffect, FissureEffect,
// FogEffect, HealingSalveGainEffect, HealingSalvePreventionEffect,
// MillstoneEffect, NevinyrralsDiskEffect, PandorasBoxEffect, SindbadEffect, and
// TheHiveEffect. Whimsy owns only random selection and ordered action dispatch.
//
// # Discard replacement
//
// PlayerDiscardByEffect routes effect-caused discards through the replacement
// pipeline. PlayerDiscard is reserved for costs, turn-based actions, and other
// discards that are not caused by an effect. DiscardToLibraryReplacement
// implements Library of Leng-style optional destination replacement.
//
// # Continuous rules helpers
//
// TargetOpponentChoice wraps an individual Target whose choice is made by the
// opposing player. Spell and activated-ability acquisition route only that
// target's prompt to the opponent while evaluating legality from the action
// controller and source as usual. SnapshotTarget binds an indexed target for
// later pipeline steps. FightGathered has two variable-bound creatures fight
// only if both remain battlefield creatures (CR 701.14).
// NewModalActivated gives each mode of an activated ability its own targets
// and effects; the mode is chosen before targets and costs are processed.
//
// AttachedCantAttackUnlessPays adds an Aura-defined attack cost to the
// enchanted creature. SourceHasManaAbilitiesOpponentLandsCouldProduce derives
// Fellwar Stone-style colored mana abilities from opposing lands. A negative
// MaximumHandSize means no maximum; SetNoMaximumHandSize installs that rule in
// a continuous-effect cycle. AddSourcePreventionShield prevents a bounded
// amount of damage from one source to one player. TryPayMana pays resolving
// effect costs from floating mana and activatable mana sources.
// # Basic land type changes
//
// BecomesBasicLandTargetEffect implements effects that set a land to Plains,
// Island, Swamp, Mountain, or Forest. In layer 4 it replaces the land's old
// land subtypes and abilities generated by printed or copied rules text,
// including keyword abilities, and grants the intrinsic mana abilities of the
// new basic land types. Abilities granted by other effects are preserved, and
// printed characteristics return when the effect ends. For
// Oracle text lasting until the source leaves the battlefield, use
// BecomesBasicLandTargetUntilSourceLeaves; it correctly survives the source
// phasing out. BecomesBasicLandAttachedEffect and
// BecomesChosenBasicLandAttachedEffect cover land Auras, while
// BecomesBasicLandsEffect applies the same layer-4 operation to a filtered set
// of permanents for effects such as Conversion and Blood Moon.
//
// # Subtype changes
//
// SetSubtypes and AddSubtypes are the family-aware layer-4 primitives. A set
// operation replaces only the selected subtype family; an add operation keeps
// every existing subtype. For example, changing the creature subtypes of a
// Land Creature preserves its land subtypes:
//
//	SetSubtypes(targetID, SubtypeCreature, EndOfTurn, "Human")
//	AddSubtypes(targetID, SubtypeCreature, EndOfTurn, "Dinosaur")
//
// SubtypeArtifact, SubtypeCreature, SubtypeEnchantment, SubtypeLand,
// SubtypePlaneswalker, and SubtypeSpell identify the independent CR 205.3
// subtype sets. The target must have the corresponding card type when the
// effect is applied. Setting a land subtype to a basic land type automatically
// performs the CR 305.7 ability removal and intrinsic mana derivation;
// BecomesBasicLandTargetEffect and its variants remain convenient
// Oracle-shaped wrappers around that behavior.
//
// # Removing abilities
//
// RemoveAllAbilities applies "loses all abilities" in layer 6. The operation
// removes abilities computed up to its timestamp, including keyword attrs and
// runtime activated, triggered, static, and mana abilities. A later layer-6
// effect can grant an ability back. RemoveAllAbilitiesFromAll applies the same
// operation to a filter for Humility-style effects. Basic land subtype changes
// reuse the same removal operation in layer 4, so ordinary layer-6 grants and
// keyword counters are applied afterward and remain as required by CR 305.7.
package mage
