/*
Package mage implements a Magic: The Gathering game engine. Cards are defined
by registering factory functions, and the engine handles game state, the stack,
combat, targeting, and the rules layer system.

This doc is the comprehensive reference for implementing cards. It covers every
subsystem: card constructors, the registry, spells, effects, targets, filters,
costs, abilities (activated, triggered, static), the continuous effect / layer
system, the attr system, and the GameMutator API that effects use to mutate
game state.

# Card Registration and the Registry

Every card is a factory function registered with [Register]. The factory returns
a fresh [Card] (a [*BaseCard]) each time it is called. Registration happens in
init() functions inside cards/ packages (e.g. cards/limited/creatures.go).

	mage.Register("Serra Angel", func() mage.Card {
	    return mage.NewCreature("Serra Angel", "{3}{W}{W}", 4, 4,
	        mage.WithSubTypes("Angel"),
	        mage.WithKeyword(core.Flying),
	        mage.WithKeyword(core.Vigilance),
	    )
	})

The global [DefaultRegistry] stores all factories. [CreateCard](name) instantiates
a card by name. [CardRegistered](name) checks existence. Each card file must
ensure its init() runs — typically a blank identifier reference in a test.go file:

	var _ = registerCreatures

# Card Constructors

Type-specific constructors create the [*BaseCard] with the right types pre-set.
All accept variadic [CardOption] functions for customization:

	[NewCreature](name, manaCost, power, toughness, ...CardOption)
	[NewInstant](name, manaCost, *SpellAbility, ...CardOption)
	[NewSorcery](name, manaCost, *SpellAbility, ...CardOption)
	[NewEnchantment](name, manaCost, ...CardOption)
	[NewAura](name, manaCost, ...CardOption)           // TypeEnchantment + "Aura" subtype
	[NewArtifact](name, manaCost, ...CardOption)
	[NewEquipment](name, manaCost, ...CardOption)       // TypeArtifact + "Equipment" subtype
	[NewLand](name, ...CardOption)                      // no mana cost
	[NewToken](name, power, toughness, types, subTypes, ...keywords)

# CardOption Functions

Options configure a card during construction. They are passed as trailing
arguments to any card constructor:

	[WithSubTypes]("Human", "Soldier")     // creature/land subtypes
	[WithSuperTypes](core.Legendary)       // Legendary, Basic, Snow, World
	[WithKeyword](core.Flying)             // keyword attrs (seeds baseAttrs)
	[WithAbility](ability)                 // any Ability (triggered, protection, etc.)
	[WithCardType](core.TypeArtifact)      // additional card type (artifact creature)
	[WithManaAbility](core.Green)          // tap for one mana of color
	[WithAnyColorMana]()                   // tap for any color (Birds of Paradise)
	[WithAdditionalCost](cost)             // extra cost when casting (sacrifice, discard)
	[WithExpansion]("Arabian Nights")      // set name
	[WithCumulativeUpkeep]("{1}")           // adds upkeep trigger with age counters

	// Shortcuts that build and attach abilities inline:
	[WithActivatedAbility](effect, cost, ...AbilityOption)
	[WithStaticAbility](effects ...ContinuousEffect)
	[WithETBEffect](effect)                // runs effect on ETB with spell targets

# Card Template Helpers

Common card patterns have dedicated constructors that reduce boilerplate:

	[NewLuckyCharm](name, cost, color)     // {1} artifact: optionally gain 1 life on color spell
	[NewLandDestruction](name, cost)       // sorcery: destroy target land
	[NewBoostAura](name, cost, +P, +T)    // aura: enchanted creature gets +P/+T

# Spells and Effects

Instants and sorceries take a [*SpellAbility] that defines resolution behavior.
Build one with:

	[NewSpellAbility](effects...)              // untargeted spell
	[NewTargetedSpell](target, effects...)     // targeted spell

Effects are the atomic actions. Each implements the [Effect] interface:

	Apply(g GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error
	Text() string
	Properties() EffectProperties

Examples:

	// Lightning Bolt: deal 3 damage to any target
	mage.NewInstant("Lightning Bolt", "{R}",
	    mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)

	// Wrath of God: destroy all creatures (no target)
	mage.NewSorcery("Wrath of God", "{2}{W}{W}",
	    mage.NewSpellAbility(mage.DestroyAllCreatures()),
	)

	// Ancestral Recall: target player draws 3
	mage.NewInstant("Ancestral Recall", "{U}",
	    mage.NewTargetedSpell(mage.TargetPlayer(), mage.DrawCards(mage.Fixed(3))),
	)

	// Modal spell (Healing Salve): set modes on the card
	c := mage.NewInstant("Healing Salve", "{W}", spell)
	c.SetModes([]string{"Gain 3 life", "Prevent 3 damage"})
	// Inside the FuncEffect, call g.ModeValue() to read the chosen mode (0 or 1).

# Pre-Built Effects Catalog

Damage effects:

	[DealDamage](amount ValueSource)                          // damage to target
	[DealDamageToAllCreatures](amount ValueSource, filter)    // mass creature damage
	[DealDamageToPlayers](amount ValueSource, selector)       // damage to selected players
	[PreventAllCombatDamage]()                                // Fog
	[PreventDamageToTarget](amount ValueSource)               // prevention shield

Removal effects:

	[DestroyTarget]()                     // destroy first target
	[DestroyTargetPermanent]()            // destroy target permanent
	[DestroyTargetLand]()                 // destroy target land
	[DestroyTargetArtifact]()             // destroy target artifact
	[DestroyAllCreatures]()               // board wipe
	[DestroyAllLands]()                   // Armageddon
	[DestroyAllEnchantments]()            // Tranquility
	[DestroyAllMatching](filter, text)    // destroy all matching filter
	[DestroyAllMatchingNoRegen](f, text)  // can't be regenerated (Shatterstorm)
	[ExileTarget]()                       // exile target permanent
	[SacrificeSource]()                   // sacrifice self

Life effects:

	[GainLife](amount int)                   // controller gains life
	[GainLifeTarget](amount ValueSource)     // target player gains life
	[LoseLife](amount int)                   // controller loses life
	[SacrificeCreatureOrDamage](dmg int)     // sacrifice or take damage (Lord of the Pit)

Card manipulation effects:

	[DrawCards](amount ValueSource)              // target player draws
	[DrawCardsActivePlayer](amount ValueSource)  // active player draws (Howling Mine)
	[DiscardCards](amount ValueSource)           // target player discards
	[DiscardRandom](amount int)                  // discard at random
	[DiscardHandAndDraw](n int)                  // each player discards hand, draws n
	[SearchLibraryToHand]()                      // Demonic Tutor
	[SearchLibraryToTop]()                       // Vampiric Tutor
	[ShuffleLibrary]()                           // shuffle controller's library

Graveyard effects:

	[ReturnFromGraveyardToBattlefield]()          // reanimate (Animate Dead)
	[ReturnFromGraveyardToHandTarget]()           // Raise Dead, Regrowth
	[ReturnSourceToHand]()                        // return self from graveyard to hand (Rancor)

Permanent manipulation effects:

	[TapTarget]()                              // tap target permanent
	[UntapTarget]()                            // untap target permanent
	[UntapSource]()                            // untap self
	[TapOrUntapTarget]()                       // Twiddle: choose tap or untap
	[TapAllLands]()                            // tap all lands target player controls
	[TapAttachedCreature]()                    // tap enchanted creature

Combat effects:

	[BoostUntilEndOfTurn](p, t ValueSource, sel)   // +P/+T until EOT
	[BoostMatchingUntilEndOfTurn](p, t, filter)     // boost matching creatures you control
	[BoostAllMatchingUntilEndOfTurn](p, t, filter)  // boost matching regardless of controller
	[DoubleTargetPower]()                           // double target's power (Berserk)
	[GrantKeywordUntilEndOfTurn](kw, sel)           // grant keyword until EOT
	[MakeUnblockableUntilEndOfTurn]()               // can't be blocked this turn
	[SetPTUntilEndOfTurn](p, t, sel)                // set base P/T until EOT
	[SetPowerUntilEndOfTurn](power, sel)            // set base power only until EOT
	[RemoveFromCombat]()                            // remove target from combat

Counter effects:

	[AddCounters](counterType, amount ValueSource, PermanentSelector)
	[RemoveCountersFromSource](counterType, amount int)

Misc effects:

	[AddMana](color, amount)              // add mana to pool
	[AddAnyMana](amount, Color)           // add mana of any one color
	[CreateToken](name, p, t, types, subtypes, keywords...)
	[CloneTarget](additionalTypes...)     // Copy Artifact, Clone
	[CopySpellOnStack]()                  // Fork
	[AttachToTarget]()                    // attach aura/equipment
	[ControlChangeTarget]()              // steal (Control Magic)
	[ExtraTurn]()                         // Time Walk
	[ChangeColorEffect](color)            // lace effect (permanent color change)
	[CounterSpell]()                       // counter target spell
	[CounterSpellIfColor](color)           // conditional counter (REB/BEB)
	[CounterSpellIfXMeetsCMC]()            // counter if X >= CMC (Spell Blast)
	[RegenerateSource]()                   // regen shield on self
	[RegenerateTarget]()                   // regen shield on target
	[ChooseColor](reason)                  // store chosen color on permanent
	[DestroyTargetAtEndOfTurn]()           // delayed destruction
	[ReplaceKeywordEffect](from, to)       // Sleight of Mind

Composite:

	[CompositeEffects](text, effects...)   // apply multiple effects in sequence

# FuncEffect — Custom One-Off Effects

For card-specific logic that doesn't fit a pre-built effect, wrap a closure:

	mage.FuncEffect(
	    "exile target creature; its controller gains life equal to its power",
	    mage.EffectProperties{Outcome: mage.OutcomeDetriment},
	    func(g mage.GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	        perm := g.FindPermanent(targets[0])
	        if perm == nil { return nil }
	        power := perm.CurrentPower(g)
	        g.ExilePermanent(perm)
	        if p := g.GetPlayer(perm.Controller); p != nil && power > 0 {
	            g.PlayerGainLife(p, power)
	        }
	        return nil
	    },
	)

The [EffectProperties] struct declares metadata for the AI:

	type EffectProperties struct {
	    Outcome     Outcome     // OutcomeBenefit, OutcomeDetriment, OutcomeUnknown
	    DamageValue ValueSource // non-nil if this deals damage
	    DrawCount   int         // fixed cards drawn (0 for X or non-draw)
	    Mass        bool        // true for board-wide effects
	}

Always set Outcome so the AI targets correctly: OutcomeDetriment effects target
opponents/their creatures; OutcomeBenefit effects target yourself/your creatures.

# Targets

Target constructors define what a spell or ability can legally target.
Each returns a [Target] interface. The engine calls Possible() to get legal
targets, then Choose() to lock in the selection. Targets auto-check hexproof,
shroud, and protection via CanBeTargetedBy.

Battlefield targets:

	[TargetCreature](filters ...PermanentFilter)          // creature
	[TargetPermanent](filters ...PermanentFilter)         // any permanent
	[TargetControlledCreature]()                           // creature you control
	[TargetControlledPermanent]()                          // permanent you control
	[TargetLand]()                                         // land
	[TargetArtifact]()                                     // artifact
	[TargetArtifactOrEnchantment]()                        // artifact or enchantment

Player targets:

	[TargetPlayer]()                    // any player
	[TargetOpponent]()                  // opponent only
	[TargetAnyTarget]()                 // creature or player ("any target")

Stack targets:

	[TargetSpellOnStack](filters ...CardFilter)  // spell on the stack

Zone targets:

	[TargetCreatureInYourGraveyard]()                   // creature in your graveyard
	[TargetCardInYourGraveyard](filters ...CardFilter)  // any card in your graveyard
	[TargetCreatureInHand]()                             // creature in your hand

Restrict with filters:

	// Destroy target nonblack creature
	mage.TargetCreature(mage.Not(mage.HasColorFilter(core.Black)))

	// Destroy target Djinn or Efreet
	mage.TargetCreature(mage.Or(mage.HasSubType("Djinn"), mage.HasSubType("Efreet")))

	// Target permanent with power 3 or less
	mage.TargetCreature(mage.HasPowerLTE(3))

# Filters

Two filter types: [PermanentFilter] (for battlefield permanents) and [CardFilter]
(for cards in hand/graveyard/stack). Both are labeled predicates with a Match method.

Create custom filters:

	mage.NewPermanentFilter("tapped artifact", func(p *mage.Permanent, g *mage.Game) bool {
	    return p.HasType(core.TypeArtifact) && p.Tapped
	})

	mage.NewCardFilter("instant card", func(c mage.Card) bool {
	    return c.HasType(core.TypeInstant)
	})

Pre-built PermanentFilter values:

	[AnyPermanent]            // matches everything
	[IsCreature]              // TypeCreature
	[IsArtifact]              // TypeArtifact
	[IsEnchantment]           // TypeEnchantment
	[IsLand]                  // TypeLand
	[IsTapped]                // tapped permanents
	[IsUntapped]              // untapped permanents
	[IsAttacking]             // currently attacking
	[IsBlocking]              // currently blocking

PermanentFilter constructors:

	[HasSubType](string)              // "Wall", "Swamp", etc.
	[HasColorFilter](Color)           // card color
	[HasKeywordFilter](Keyword)       // has keyword
	[NotHasKeywordFilter](Keyword)    // lacks keyword
	[HasPowerLTE](n int)              // power <= n
	[HasPowerGTE](n int)              // power >= n
	[ControlledBy](playerID)          // controlled by player
	[NotControlledBy](playerID)       // not controlled by player
	[Named](string)                   // exact name match
	[NotID](uuid.UUID)                // exclude specific permanent
	[IsID](uuid.UUID)                 // match specific permanent
	[IsBandedWith](uuid.UUID)         // banded with in combat
	[HasExpansion](string)            // from a specific set

Combinators:

	[And](filters...)    // all must match
	[Or](filters...)     // any must match
	[Not](filter)        // negation

Pre-built CardFilter values:

	[IsCreatureCard]          // creature cards
	[IsArtifactCard]          // artifact cards

# ValueSource — Dynamic Values

Many effects accept [ValueSource] instead of raw int for dynamic resolution:

	[Fixed](n int)                                          // constant
	[XValue]()                                              // reads g.CurrentX
	[CountBattlefield](who PlayerSelector, filter)          // count permanents
	[CountZone](zone, who PlayerSelector, filter CardFilter) // count cards in zone

Examples:

	// Lightning Bolt: 3 damage
	mage.DealDamage(mage.Fixed(3))

	// Fireball: X damage
	mage.DealDamage(mage.XValue())

	// Damage equal to Swamps you control
	mage.DealDamage(mage.CountBattlefield(
	    mage.SelectController(),
	    mage.And(mage.IsLand, mage.HasSubType("Swamp")),
	))

# PermanentSelector — Source vs Target

Effects that can apply to either the source permanent or a resolved target use
[PermanentSelector]:

	[SelectTarget]    // apply to first resolved target (targets[0])
	[SelectSource]    // apply to the ability's source permanent

	// Sengir Vampire: +1/+1 counter on self
	mage.AddCounters(core.P1P1, mage.Fixed(1), mage.SelectSource)

	// Spell: +1/+1 counter on target creature
	mage.AddCounters(core.P1P1, mage.Fixed(1), mage.SelectTarget)

# PlayerSelector — Who Is Affected

[PlayerSelector] picks one or more players for an effect:

	[SelectController]()            // the ability's controller
	[SelectActivePlayer]()          // the player whose turn it is
	[SelectEachPlayer]()            // all players in the game
	[SelectEachOpponent]()          // all opponents of the controller
	[SelectAttachedController]()    // controller of the permanent this is attached to
	[SelectEventController]()       // the player from targets[0] (event-driven triggers)

	// Juzám Djinn: deals 1 damage to you each upkeep
	mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectController())

	// Howling Mine: active player draws
	mage.DrawCardsActivePlayer(mage.Fixed(1))

	// Psychic Venom: 2 damage to enchanted land's controller
	mage.DealDamageToPlayers(mage.Fixed(2), mage.SelectAttachedController())

# Costs

The [Cost] interface has CanPay, Pay, and Text methods. Cost constructors:

	[ManaCostOf]("{1}{R}")             // mana payment from pool
	[GenericCost](n)                   // n generic mana
	[XManaCost]()                      // pay X generic mana (g.CurrentX)
	[TapSourceCost]()                  // {T}: tap the source
	[SacrificeSourceCost]()            // sacrifice self
	[SacrificeCreatureCost]()          // sacrifice a creature you control
	[SacrificeArtifactCost]()          // sacrifice an artifact you control
	[LifePayCost](amount)             // pay life
	[RemoveCountersCost](ct, n)        // remove n counters of type ct
	[DiscardCost](n)                   // discard n cards
	[DiscardRandomCost](n)             // discard n cards at random
	[ExileFromGraveyardCost](n)        // exile n cards from your graveyard
	[ExileSourceCost]()                // exile self
	[ReturnToHandCost](filter)         // bounce a permanent to hand (nil = any)
	[TapCreatureCost]()                // tap another untapped creature you control

Additional costs on the card itself (paid when casting, not on an ability):

	mage.WithAdditionalCost(mage.SacrificeCreatureCost())

# Activated Abilities

[NewActivatedAbility] takes a primary effect, primary cost, and optional
[AbilityOption] modifiers:

	mage.NewActivatedAbility(effect, cost, ...AbilityOption)

AbilityOption modifiers:

	[WithCost](cost)                      // additional cost
	[WithTarget](target)                  // targeting requirement
	[WithEffect](effect)                  // additional effect
	[WithUpkeepOnly]()                    // only during upkeep
	[WithOncePerTurn]()                   // once per turn limit
	[WithAnyPlayerMay]()                  // any player can activate
	[WithControlledSinceTurnStart]()      // must have controlled since turn start

Attach to a card with [WithActivatedAbility] or [WithAbility]:

	// Northern Paladin: {W}{W}, {T}: Destroy target black permanent
	mage.WithActivatedAbility(
	    mage.DestroyTargetPermanent(),
	    mage.ManaCostOf("{W}{W}"),
	    mage.WithCost(mage.TapSourceCost()),
	    mage.WithTarget(mage.TargetPermanent(mage.HasColorFilter(core.Black))),
	)

	// Prodigal Sorcerer: {T}: Deal 1 damage to any target
	mage.WithActivatedAbility(
	    mage.DealDamage(mage.Fixed(1)),
	    mage.TapSourceCost(),
	    mage.WithTarget(mage.TargetAnyTarget()),
	)

	// Regeneration: {B}: Regenerate self
	mage.WithActivatedAbility(
	    mage.RegenerateSource(),
	    mage.ManaCostOf("{B}"),
	)

Equipment uses [NewEquipAbility] (sorcery-speed, targets creature you control):

	mage.NewEquipAbility(mage.ManaCostOf("{2}"))

# Triggered Abilities

Triggered abilities listen for [GameEvent] values fired by the engine. The base
constructor is [NewTriggered](eventType, optional bool, effects...). Use
.SetCondition() to filter which events actually trigger it, and .AddTarget()
to add targeting requirements.

Convenience constructors (set condition automatically):

	[AttacksTrigger](effect, optional)                          // EvtDeclaredAttacker, source is self
	[BlocksTrigger](effect, optional)                           // EvtDeclaredBlocker, source is self
	[EntersBattlefieldTrigger](effect, optional)                // EvtEntersBattlefield, source is self
	[DiesCreatureTrigger](effect, optional, filter)             // EvtCreatureDied, another creature you control
	[AnyCreatureDiesTrigger](effect, optional)                  // EvtCreatureDied, any creature
	[CreatureDealtDamageBySourceDiesTrigger](effect, optional)  // creature damaged by source dies
	[DealsDamageToOpponentTrigger](effect, optional)            // EvtDamageDealt to opponent
	[WhenDamageDealtToThisTrigger](effect, optional)            // EvtDamageDealt to self
	[BeginningOfUpkeepTrigger](effect, optional)                // EvtUpkeep, controller's upkeep
	[BeginningOfEachUpkeepTrigger](effect, optional)            // EvtUpkeep, every player
	[BeginningOfEachDrawStepTrigger](effect, optional)          // EvtDrawStep, while untapped
	[BeginningOfAttachedControllerUpkeepTrigger](effect, opt)   // EvtUpkeep, attached creature's controller
	[WheneverSpellCastTrigger](effect, optional, *Color)        // EvtSpellCast, optional color filter
	[WheneverEnchantmentCastTrigger](effect, optional)          // EvtSpellCast, controller's enchantment
	[WhenAttachedBecomesTappedTrigger](effect, optional)        // EvtTapped, enchanted permanent
	[WheneverLandEntersBattlefieldTrigger](effect, optional)    // EvtEntersBattlefield, any land
	[WhenOpponentPermanentBecomesTappedTrigger](e, opt, filter) // EvtTapped, opponent's matching permanent
	[PutIntoGraveyardFromBattlefieldTrigger](effect, optional)  // EvtPutIntoGraveyardFromBattlefield, self
	[SacrificeAtUpkeepUnlessPay](manaCost)                      // sacrifice unless pay at upkeep

Custom triggers with SetCondition:

	// When this creature dies (not "another" — self)
	mage.NewTriggered(core.EvtCreatureDied, false, effect).
	    SetCondition(func(evt *core.GameEvent, g *mage.Game, sourceID, _ uuid.UUID) bool {
	        return evt.SourceID == sourceID
	    })

	// Whenever an opponent's Swamp becomes tapped
	mage.NewTriggered(core.EvtTapped, false, effect).
	    SetCondition(func(evt *core.GameEvent, g *mage.Game, sourceID, controllerID uuid.UUID) bool {
	        perm := g.FindPermanent(evt.SourceID)
	        return perm != nil && perm.Controller != controllerID && perm.HasSubType("Swamp")
	    })

The optional flag (second arg) controls whether the controller may decline:
false = mandatory, true = "you may" (AI/player can decline).

The [TriggerCondition] signature is:

	func(evt *GameEvent, g *Game, sourceID, controllerID uuid.UUID) bool

Pre-built conditions:

	[IsThisSource]   // evt.SourceID == sourceID

# Static Abilities and Continuous Effects

Static abilities apply continuous effects while the source is on the battlefield.
Use [WithStaticAbility] or [StaticAbility]:

	mage.WithStaticAbility(
	    mage.BoostAllCreatures(1, 1, mage.HasSubType("Goblin")),
	)

# The Continuous Effect System

Continuous effects implement the [ContinuousEffect] interface:

	Apply(g *Game) error        // apply the effect
	GetLayer() Layer            // which MTG layer (1-7)
	GetDuration() Duration      // when it expires
	IsActive(g *Game) bool      // is it currently in effect?
	SourceID() uuid.UUID
	SetSourceID(uuid.UUID)

Three primitives build all continuous effects:

1. [FuncContinuousEffect](layer, duration, applyFunc, ...ActiveCondition) — the
general-purpose builder. applyFunc receives (g *Game, sourceID uuid.UUID). By
default active while source is on the battlefield.

2. [AttachedEffect](layer, applyFunc) — for aura/equipment effects. applyFunc
receives (g *Game, source, target *Permanent). Active while source is attached.

3. [TargetEffect](layer, duration, targetID, applyFunc) — applies to a specific
permanent by ID. applyFunc receives (g *Game, target *Permanent). Active while
target exists. Used for temporary effects (Giant Growth, Sorceress Queen).

# Layers

The MTG layer system determines the order effects are applied. Applied in order
by [EffectManager.Apply]:

	[LayerCopy]    (1) — copy effects (Doppelganger, Clone)
	[LayerControl] (2) — control changes (Control Magic, Aladdin)
	[LayerType]    (4) — type changes, subtype overrides, animate effects
	[LayerColor]   (5) — color changes (lace effects)
	[LayerAbility] (6) — add/remove abilities, keywords, attrs
	[LayerPT]      (7) — power/toughness modifications

# Durations

	[WhileOnBattlefield]   // stays as long as source is on battlefield
	[EndOfTurn]            // removed at end of turn cleanup
	[EndOfCombat]          // removed at end of combat
	[Indefinite]           // never auto-removed; IsActive controls lifetime

# ActiveCondition

Controls when a FuncContinuousEffect is active:

	[SourceOnBattlefield]       // default: source permanent exists
	[SourceAttached]            // source is attached to another permanent
	[SourceUntapped]            // source is untapped (Winter Orb)
	[TargetOnBattlefield](id)   // specific permanent exists
	[WithSourceCondition](cond) // bridges SourceCondition to ActiveCondition

# SourceCondition

Used by damage prevention rules and BoostSelf:

	[WhileSourceAttacking]           // source is declared attacker
	[WhileSourceUntapped]            // source is untapped
	[WhileControlling](filter)       // controller controls matching permanent

# Pre-Built Continuous Effects

Attached effects (aura/equipment — active while attached):

	[BoostAttached](power, toughness, AttachType)       // +P/+T to attached
	[GrantAbilityToAttached](keyword, AttachType)       // grant keyword
	[GrantProtectionToAttached](color, AttachType)      // protection from color
	[RemoveKeywordFromAttached](keyword, AttachType)    // remove keyword
	[ChangeAttachedSubTypes](newSubTypes)               // override subtypes
	[GrantActivatedAbilityToAttached](e, cost, at)      // grant activated ability
	[PreventAttachedFromUntapping](AttachType)          // doesn't untap
	[PreventAttachedFromAttacking](AttachType)          // can't attack
	[ControlChangeContinuous]()                         // steal (Control Magic)
	[BoostAttachedByForestCount]()                      // Aspect of Wolf

Target effects (apply to specific permanent by ID):

	[TemporaryBoost](targetID, power, toughness)           // +P/+T until EOT
	[TemporaryKeyword](targetID, keyword)                  // keyword until EOT
	[KeywordReplacement](targetID, from, to)               // replace keyword (Indefinite)
	[SetBasePT](targetID, power, toughness)                // set base P/T until EOT
	[SetBasePower](targetID, power)                        // set base power until EOT
	[ColorOverride](targetID, color)                       // change color (Indefinite)
	[TemporaryAnimate](targetID, power, toughness)         // animate until EOT
	[TemporaryAnimateUntilEndOfCombat](id, power, tough)   // animate until end combat
	[PreventBlockingUntilEndOfCombat](permID)              // can't block until end combat

Global/source-based effects (while source on battlefield):

	[BoostAllCreatures](p, t, filter)                      // lord effect (excludes self)
	[BoostAllCreaturesIncludingSelf](p, t, filter)         // includes self
	[BoostControlledCreatures](p, t, filter)               // only your creatures
	[BoostSelf](p, t, SourceCondition)                     // conditional self-boost
	[PTEqualsCount](filter)                                // P/T = count of matching permanents
	[PTEqualsControlledCount](filter)                      // P/T = count you control
	[GrantKeywordToAll](keyword, filter)                   // grant keyword to matching
	[GrantActivatedAbilityToAll](effect, cost, filter)     // grant ability to matching
	[PreventUntapForMatching](filter)                      // Meekstone
	[PreventAllUntaps]()                                   // Stasis
	[AnimateLands](filter, power, toughness)               // Living Lands
	[LimitLandUntaps](limit)                               // Winter Orb (while untapped)
	[AllowUnlimitedLandPlays]()                            // Fastbond
	[IncreaseSpellCostForColor](color, amount)             // Gloom
	[ReduceSpellCostForColor](color, amount)               // cost reduction
	[ChangeSubTypesForAll](fromSubTypes, toSubTypes)       // Conversion
	[ManaConversion](from, to Color)                       // Sunglasses of Urza
	[BodyguardContinuous]()                                // Veteran Bodyguard
	[PersonalIncarnationRedirect]()                        // Personal Incarnation
	[PreventFromAttackingIfDefendingPlayerControls](f)     // Dandân

Damage prevention rules (continuous):

	[PreventDamageFromTo](from, toFactory, ...SourceCondition)
	    // Camel: prevent Desert damage to self and banded creatures

# The Attr System

All keyword abilities, capabilities, and type identity are stored as [Attr]
values with additive/subtractive counts on each [Permanent]:

  - baseAttrs: intrinsic attrs set at creation (from card's AttrSeeds and type)
  - grantedAttrs: effect-cycle deltas (reset and recomputed each EffectManager.Apply())

HasAttr(a) returns true if baseAttrs[a] + grantedAttrs[a] > 0.

Capability attrs:

	[AttrCanAttack]           // can declare as attacker
	[AttrCanBlock]            // can declare as blocker
	[AttrHasPowerToughness]   // has P/T, takes combat damage
	[AttrSummonSick]          // summoning sickness (cleared at untap)
	[AttrDoesNotUntap]        // doesn't untap during untap step
	[AttrEntersTapped]        // enters tapped
	[AttrMustAttack]          // must attack each turn
	[AttrMustBeBlocked]       // must be blocked if possible
	[AttrMayNotUntap]         // player may choose not to untap

Type-identity attrs:

	[AttrIsCreature], [AttrIsLand], [AttrIsArtifact], [AttrIsEnchantment]

Keyword attrs (all >= Flying):

	[Flying], [Reach], [FirstStrike], [DoubleStrike], [Trample], [Vigilance],
	[Haste], [Menace], [Fear], [Deathtouch], [Lifelink], [Defender], [Banding],
	[Indestructible], [Hexproof], [Shroud], [Forestwalk], [Islandwalk],
	[Swampwalk], [Mountainwalk], [Plainswalk], [Desertwalk], [UnblockableKW],
	[CantBeBlockedByWalls], [CantBeBlockedExceptByWalls], [CanBlockAny],
	[CanBlockAdditional], [BasiliskTouch], [CantRegenerate]

Continuous effects use EffectManager.GrantAttr/RevokeAttr to modify grantedAttrs:

	g.Effects.GrantAttr(perm.ID(), core.Flying)   // give flying
	g.Effects.RevokeAttr(perm.ID(), core.AttrCanAttack)  // prevent attacking

The permanent's GrantBaseAttr/RevokeBaseAttr modify intrinsic attrs directly.

# The EffectManager

[EffectManager] manages all continuous effects and per-cycle state. Its Apply(g)
method is called frequently and:

1. Resets all grantedAttrs, powerBonus, toughBonus, SubTypeOverride,
   BasePTOverride, ColorOverride, and controller to owner on every permanent.
2. Strips effect-granted runtime abilities.
3. Removes effects whose source is gone (except EOT/EOC/Indefinite effects).
4. Applies effects in layer order (1→2→4→5→6→7).
5. Calls Rules.ResetPerCycle() and Damage.ResetPerCycle() before each cycle.
6. Syncs mana conversions and writes attrDeltas to permanents.

The EffectManager owns two subsystems accessed as public fields:

  g.Effects.Damage  — [DamageSystem] (damage_system.go)
  g.Effects.Rules   — [GameRules]    (game_rules.go)

Block pair restrictions (PreventBlockPair, IsBlockPrevented) and attr deltas
(GrantAttr, RevokeAttr) remain directly on EffectManager.

# DamageSystem

[DamageSystem] owns all damage-related transient state. Key methods:

  Regeneration:    AddRegenerationShield, ConsumeRegenerationShield, HasRegenerationShield
  Prevention:      AddPreventionShield, PreventDamage, AddColorPrevention, AddTypePrevention
  Prevention rules: AddDamagePreventionRule (with [WithFrom], [WithTo], [WithOneShot])
  Redirection:     SetBodyguard, GetBodyguard, SetPlayerDamageRedirect,
                   SetArtifactDamageRedirect, SetCreatureDamageRedirect
  Combat damage:   SetPreventCombatDamage, PreventsCombatDamage
  Forcefield:      SetForcefieldShield, HasForcefieldShield
  Reverse damage:  SetReverseDamageShield, HasReverseDamageShield
  Reflection:      SetDamageReflection, GetDamageReflection
  Draw replacement: SetDrawReplacement, GetDrawReplacement

Lifecycle: ResetPerCycle() clears per-Apply state. ClearEndOfTurn() clears
shields and one-shot rules at end of turn.

# GameRules

[GameRules] owns all game-rule modifier state. Public fields set directly by
continuous effects:

  LandUntapMax       int             // max lands to untap per step (-1 = unlimited)
  ArtifactUntapMax   int             // max artifacts to untap per step (-1 = unlimited)
  UnlimitedLandPlays bool            // bypass one-land-per-turn
  SpellCostIncreases map[Color]int   // per-color cost increases (Gloom, etc.)
  SpellCostReductions map[Color]int  // per-color cost reductions
  ManaConversion     map[Color]Color // forced mana conversion (Celestial Dawn)

Methods for special rules:

  Lich:      SetLichActive, IsLichActive, GetLichPermanent
  Channel:   SetChannelActive, IsChannelActive
  Sanctuary: SetSanctuaryActive, IsSanctuaryActive
  Skip draw: SetSkipNextDraw, ShouldSkipDraw
  Min life:  SetMinimumLife, HasMinimumLife
  Max hand:  SetMaxHandSize, GetMaxHandSize
  Cast block: AddExpansionCastBlock, IsExpansionBlocked

Lifecycle: ResetPerCycle() clears per-Apply state. ClearEndOfTurn() clears
end-of-turn flags. SyncManaConversions(players) writes mana conversions to
player mana pools.

# GameMutator — The Effect API

Effects receive a [GameMutator] interface (satisfied by *Game). It embeds
[GameReader] for all reads and adds mutation verbs.

Read methods (GameReader):

	GetPlayer(uuid.UUID) Player
	GetOpponent(uuid.UUID) Player
	ActivePlayerObj() Player
	FindPermanent(uuid.UUID) *Permanent
	FindPermanentByName(string, uuid.UUID) *Permanent
	FindCardAnywhere(uuid.UUID) Card
	AnyBattlefield(PermanentFilter) bool
	FilterBattlefield(PermanentFilter) []*Permanent
	CountBattlefield(PermanentFilter) int
	AllPlayers() []Player
	XValue() int
	ModeValue() int
	GetResolvingCard() Card
	FindStackObject(uuid.UUID) *StackObject
	CombatGroups() []*CombatGroup

Mutation methods (GameMutator):

	PlayerGainLife(Player, int)
	FireEvent(GameEvent)
	PutOnBattlefield(Card, uuid.UUID) *Permanent
	RemoveFromBattlefield(*Permanent)
	DestroyPermanent(*Permanent)
	ExilePermanent(*Permanent)
	Sacrifice(*Permanent)
	DealDamageToPlayer(Player, int, uuid.UUID)
	DealDamageToPermanent(*Permanent, int, uuid.UUID)
	CounterSpellOnStack(uuid.UUID)
	PushStack(*StackObject)
	Attach(sourceID, targetID uuid.UUID)
	RegisterDelayedTrigger(*DelayedTrigger)
	GrantExtraTurn(uuid.UUID)
	RemoveFromCombat(uuid.UUID)
	AddContinuousEffect(ContinuousEffect)
	ApplyContinuousEffects()
	TryPayCostFromLands(playerID uuid.UUID, manaCost string) bool
	FlipCoin(playerID uuid.UUID) bool

GameMutator also includes proxy methods for the DamageSystem and GameRules
subsystems, so card effects call e.g. g.SetPreventCombatDamage() or
g.AddRegenerationShield(id) rather than reaching through g.Effects.Damage
directly. The full list is in game_mutator.go.

# Protection

Protection abilities are special — they're not keywords but runtime abilities:

	[ProtectionFromColor](color)         // protection from a single color
	[ProtectionFromColors](colors...)    // protection from multiple colors
	[ProtectionFromCardType](cardType)   // protection from artifacts, etc.
	[ProtectionFromSubType](subtype)     // protection from Goblins, etc.
	[ProtectionFromAll]()                // protection from everything

Attach with WithAbility:

	mage.WithAbility(mage.ProtectionFromColor(core.Black))

# Special Ability Types

	[SacrificeUnlessLand](subtype)        // sacrifice if you don't control land type
	[EntersWithXCounters](counterType)    // ETB with X counters
	[CopyCreatureOnETB]()                 // clone ETB (Doppelganger)
	[ETBWithTargets](effect)              // run effect on ETB using spell targets
	[ETBEffect](effect)                   // run effect on ETB without targets
	[GraveyardReturnIfCreaturesAbove](n)  // return from graveyard (Nether Shadow)
	[ManaBonusAbility] / [NewManaBonusAbility](filter, color) // bonus mana on tap
	[NewAttachedManaBonusAbility](color)  // Wild Growth bonus mana

# Mana

Mana abilities are a special ability type:

	[NewManaAbility](color)         // tap for one mana of color
	[NewAnyColorManaAbility]()      // tap for any color

Mana costs are parsed from strings like "{2}{W}{B}" via [ParseManaCost].
Colors: [White], [Blue], [Black], [Red], [Green], [Colorless].

# Complete Card Examples

Vanilla creature:

	mage.Register("Grizzly Bears", func() mage.Card {
	    return mage.NewCreature("Grizzly Bears", "{1}{G}", 2, 2,
	        mage.WithSubTypes("Bear"),
	    )
	})

Creature with keywords:

	mage.Register("Serra Angel", func() mage.Card {
	    return mage.NewCreature("Serra Angel", "{3}{W}{W}", 4, 4,
	        mage.WithSubTypes("Angel"),
	        mage.WithKeyword(core.Flying),
	        mage.WithKeyword(core.Vigilance),
	    )
	})

Creature with activated ability:

	mage.Register("Prodigal Sorcerer", func() mage.Card {
	    return mage.NewCreature("Prodigal Sorcerer", "{2}{U}", 1, 1,
	        mage.WithSubTypes("Human", "Wizard"),
	        mage.WithActivatedAbility(
	            mage.DealDamage(mage.Fixed(1)),
	            mage.TapSourceCost(),
	            mage.WithTarget(mage.TargetAnyTarget()),
	        ),
	    )
	})

Creature with triggered ability:

	mage.Register("Sengir Vampire", func() mage.Card {
	    return mage.NewCreature("Sengir Vampire", "{3}{B}{B}", 4, 4,
	        mage.WithSubTypes("Vampire"),
	        mage.WithKeyword(core.Flying),
	        mage.WithAbility(
	            mage.CreatureDealtDamageBySourceDiesTrigger(
	                mage.AddCounters(core.P1P1, mage.Fixed(1), mage.SelectSource),
	                false,
	            ),
	        ),
	    )
	})

Creature with static ability (lord):

	mage.Register("Goblin King", func() mage.Card {
	    return mage.NewCreature("Goblin King", "{1}{R}{R}", 2, 2,
	        mage.WithSubTypes("Goblin"),
	        mage.WithStaticAbility(
	            mage.BoostAllCreatures(1, 1, mage.HasSubType("Goblin")),
	            mage.GrantKeywordToAll(core.Mountainwalk, mage.HasSubType("Goblin")),
	        ),
	    )
	})

Targeted instant:

	mage.Register("Lightning Bolt", func() mage.Card {
	    return mage.NewInstant("Lightning Bolt", "{R}",
	        mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	    )
	})

Untargeted sorcery:

	mage.Register("Wrath of God", func() mage.Card {
	    return mage.NewSorcery("Wrath of God", "{2}{W}{W}",
	        mage.NewSpellAbility(mage.DestroyAllCreatures()),
	    )
	})

X spell:

	mage.Register("Fireball", func() mage.Card {
	    return mage.NewSorcery("Fireball", "{X}{R}",
	        mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.XValue())),
	    )
	})

Counterspell:

	mage.Register("Counterspell", func() mage.Card {
	    return mage.NewInstant("Counterspell", "{U}{U}",
	        mage.NewTargetedSpell(mage.TargetSpellOnStack(), mage.CounterSpell()),
	    )
	})

Aura with static effect:

	mage.Register("Holy Strength", func() mage.Card {
	    return mage.NewBoostAura("Holy Strength", "{W}", 1, 2)
	})

	// Equivalent manual construction:
	mage.Register("Holy Strength", func() mage.Card {
	    return mage.NewAura("Holy Strength", "{W}",
	        mage.WithAbility(mage.StaticAbility(
	            mage.BoostAttached(1, 2, core.AttachAura),
	        )),
	    )
	})

Control aura:

	mage.Register("Control Magic", func() mage.Card {
	    return mage.NewAura("Control Magic", "{2}{U}{U}",
	        mage.WithAbility(mage.StaticAbility(
	            mage.ControlChangeContinuous(),
	        )),
	    )
	})

Enchantment with upkeep trigger:

	mage.Register("Black Vise", func() mage.Card {
	    return mage.NewArtifact("Black Vise", "{1}",
	        mage.WithAbility(
	            mage.BeginningOfEachUpkeepTrigger(mage.BlackViseEffect(), false),
	        ),
	    )
	})

Equipment:

	mage.Register("Bonesplitter", func() mage.Card {
	    return mage.NewEquipment("Bonesplitter", "{1}",
	        mage.WithAbility(mage.StaticAbility(
	            mage.BoostAttached(2, 0, core.AttachEquipment),
	        )),
	        mage.WithAbility(mage.NewEquipAbility(mage.ManaCostOf("{1}"))),
	    )
	})

Land:

	mage.Register("Forest", func() mage.Card {
	    return mage.NewLand("Forest",
	        mage.WithSuperTypes(core.Basic),
	        mage.WithSubTypes("Forest"),
	        mage.WithManaAbility(core.Green),
	    )
	})

Complex card with FuncEffect:

	mage.Register("Swords to Plowshares", func() mage.Card {
	    return mage.NewInstant("Swords to Plowshares", "{W}",
	        mage.NewTargetedSpell(mage.TargetCreature(), mage.FuncEffect(
	            "exile target creature; controller gains life equal to its power",
	            mage.EffectProperties{Outcome: mage.OutcomeDetriment},
	            func(g mage.GameMutator, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	                if len(targets) == 0 { return nil }
	                perm := g.FindPermanent(targets[0])
	                if perm == nil { return nil }
	                power := perm.CurrentPower(g)
	                g.ExilePermanent(perm)
	                if p := g.GetPlayer(perm.Controller); p != nil && power > 0 {
	                    g.PlayerGainLife(p, power)
	                }
	                return nil
	            },
	        )),
	    )
	})
*/
package mage
