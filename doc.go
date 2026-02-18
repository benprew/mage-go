/*
Package mage implements a Magic: The Gathering game engine. Cards are defined
by registering factory functions, and the engine handles game state, the stack,
combat, targeting, and the rules layer system.

# Defining Cards

Cards are registered with [Register] and constructed with type-specific helpers:

	mage.Register("Serra Angel", func() mage.Card {
	    c := mage.NewCreature("Serra Angel", "{3}{W}{W}", 4, 4, "Angel")
	    c.AddAbility(mage.HasKeyword(mage.Flying))
	    c.AddAbility(mage.HasKeyword(mage.Vigilance))
	    return c
	})

Card constructors: [NewCreature], [NewInstant], [NewSorcery], [NewEnchantment],
[NewAura], [NewArtifact]. Abilities are attached with AddAbility.

# Spells and Effects

Instants and sorceries use [NewSpellAbility] (no targets) or [NewTargetedSpell]
(with a target) to define what happens on resolution. Effects are the individual
actions: deal damage, destroy, draw cards, etc.

	// Lightning Bolt: deal 3 damage to any target
	c := mage.NewInstant("Lightning Bolt", "{R}")
	c.AddAbility(mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))))

	// Wrath of God: destroy all creatures (no target)
	c := mage.NewSorcery("Wrath of God", "{2}{W}{W}")
	c.AddAbility(mage.NewSpellAbility(mage.DestroyAllCreatures()))

# Targets

Target constructors define what a spell or ability can legally target.
Each returns a [Target] that the engine uses for validation and selection:

  - [TargetCreature] — a creature on the battlefield (with optional filters)
  - [TargetPlayer] — any player
  - [TargetAnyTarget] — any creature or player ("any target")
  - [TargetPermanent] — any permanent (with optional filters)
  - [TargetLand], [TargetArtifact], [TargetArtifactOrEnchantment] — type-restricted
  - [TargetSpellOnStack] — a spell on the stack (for counterspells)
  - [TargetControlledCreature] — a creature you control
  - [TargetOpponent] — an opponent
  - [TargetCreatureInYourGraveyard], [TargetCardInYourGraveyard] — graveyard targets
  - [TargetCreatureInHand] — a creature card in the controller's hand

Filters can be passed to [TargetCreature] and [TargetPermanent] to restrict
which permanents are valid targets:

	// Doom Blade: destroy target nonblack creature
	mage.TargetCreature(mage.Not(mage.HasColorFilter(mage.Black)))

	// King Suleiman: destroy target Djinn or Efreet
	mage.TargetCreature(mage.Or(mage.HasSubType("Djinn"), mage.HasSubType("Efreet")))

# Filters

[PermanentFilter] predicates are composable functions used by targets, effects,
and continuous effects to select permanents. Pre-built filters include:

  - Type filters: [IsCreature], [IsArtifact], [IsEnchantment], [IsLand]
  - Subtype: [HasSubType]("Wall"), [HasSubType]("Swamp")
  - Color: [HasColorFilter]
  - Keywords: [HasKeywordFilter], [NotHasKeywordFilter]
  - Power: [HasPowerGTE]
  - Controller: [ControlledBy], [NotControlledBy]
  - Identity: [NotID]

Combine with [And], [Or], and [Not]:

	// Non-Wall creatures
	mage.And(mage.IsCreature, mage.Not(mage.HasSubType("Wall")))

	// Forests that are lands
	mage.And(mage.IsLand, mage.HasSubType("Forest"))

Filters are used in targeting, mass-destruction effects like [DestroyAllMatching],
damage effects like [DealDamageToAllCreatures], and continuous effects.

# ValueSource

Many effects accept a [ValueSource] instead of a raw int, allowing dynamic values:

  - [Fixed](n) — a constant value
  - [XValue]() — reads the X value from the spell being cast (g.CurrentX)

This lets the same effect constructors work for both fixed and X-cost spells:

	// Lightning Bolt: 3 damage
	mage.DealDamage(mage.Fixed(3))

	// Fireball: X damage
	mage.DealDamage(mage.XValue())

	// Giant Growth: +3/+3
	mage.BoostUntilEndOfTurn(mage.Fixed(3), mage.Fixed(3), mage.SelectTarget)

# PermanentSelector

Effects that can apply to either the source or a target use [PermanentSelector]:

  - [SelectTarget] — apply to the first resolved target
  - [SelectSource] — apply to the ability's source permanent

Example: Sengir Vampire puts counters on itself, while a spell puts counters on a target:

	// +1/+1 counter on self
	mage.AddCounters(mage.P1P1, mage.Fixed(1), mage.SelectSource)

	// +1/+1 counter on target creature
	mage.AddCounters(mage.P1P1, mage.Fixed(1), mage.SelectTarget)

# PlayerSelector

Effects that deal damage or interact with players use [PlayerSelector] to
determine which player(s) are affected:

  - [SelectController]() — the ability's controller
  - [SelectActivePlayer]() — the player whose turn it is
  - [SelectEachPlayer]() — all players
  - [SelectEachOpponent]() — all opponents
  - [SelectAttachedController]() — controller of the permanent this aura is attached to
  - [SelectEventController]() — the player from the triggering event

Examples:

	// Juzám Djinn: deals 1 damage to you each upkeep
	mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectController())

	// Pestilence: deals 1 damage to each creature and each player
	mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectEachPlayer())

	// Psychic Venom: deals 2 damage to enchanted land's controller when tapped
	mage.DealDamageToPlayers(mage.Fixed(2), mage.SelectAttachedController())

# Activated Abilities

Use [NewActivatedAbility] with a primary effect, a primary cost, and optional
[AbilityOption] modifiers ([WithCost], [WithTarget], [WithEffect]):

	// {B}: Regenerate (tap + mana cost)
	mage.NewActivatedAbility(mage.RegenerateSource(), mage.ManaCostOf("{B}"))

	// {T}: Destroy target Djinn or Efreet
	mage.NewActivatedAbility(
	    mage.DestroyTarget(),
	    mage.TapSourceCost(),
	    mage.WithTarget(mage.TargetCreature(
	        mage.Or(mage.HasSubType("Djinn"), mage.HasSubType("Efreet")),
	    )),
	)

Cost constructors: [ManaCostOf], [GenericCost], [TapSourceCost],
[SacrificeSourceCost], [SacrificeCreatureCost], [LifePayCost], [RemoveCountersCost].

# Triggered Abilities

Triggers listen for game events and fire effects. [NewTriggered] is the base
constructor; convenience wrappers set the condition for you:

  - [AttacksTrigger] — when this creature attacks
  - [DiesCreatureTrigger] — when another creature you control dies
  - [AnyCreatureDiesTrigger] — when any creature dies
  - [CreatureDealtDamageBySourceDiesTrigger] — when a creature damaged by this dies
  - [EntersBattlefieldTrigger] — when this enters the battlefield
  - [BeginningOfUpkeepTrigger] — at the beginning of your upkeep
  - [BeginningOfEachUpkeepTrigger] — at every player's upkeep
  - [DealsDamageToOpponentTrigger] — when this deals damage to an opponent
  - [WheneverSpellCastTrigger] — when a spell of a given color is cast
  - [WhenAttachedBecomesTappedTrigger] — when the enchanted permanent is tapped

For custom trigger conditions, use [NewTriggered] with [GenericTriggered.SetCondition]:

	mage.NewTriggered(mage.EvtCreatureDied, false, myEffect).
	    SetCondition(func(evt *mage.GameEvent, g *mage.Game, sourceID, _ uuid.UUID) bool {
	        return evt.SourceID == sourceID // only when this creature dies
	    })

# FuncEffect

For one-off card-specific logic that doesn't fit a pre-built effect, use [FuncEffect]:

	mage.FuncEffect("deal damage equal to Swamps", func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	    // custom logic here
	    return nil
	})
*/
package mage
