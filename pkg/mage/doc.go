/*
Package mage implements a Magic: The Gathering game engine. Cards are defined
by registering factory functions, and the engine handles game state, the stack,
combat, targeting, and the rules layer system.

This doc is the comprehensive reference for implementing cards. It covers every
subsystem: card constructors, the registry, spells, effects, targets, filters,
costs, abilities (activated, triggered, static), the continuous effect / layer
system, the attr system, and the [*Game] API that effects use to mutate
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
Most accept variadic [CardOption] functions for customization. Instants and
sorceries accept spell effects/options directly, plus [CardOption] values:

	[NewCreature](name, manaCost, power, toughness, ...CardOption)
	[NewInstant](name, manaCost, effectsOrOptions...)
	[NewSorcery](name, manaCost, effectsOrOptions...)
	[NewEnchantment](name, manaCost, ...CardOption)
	[NewAura](name, manaCost, ...CardOption)           // TypeEnchantment + "Aura" subtype
	[NewArtifact](name, manaCost, ...CardOption)
	[NewEquipment](name, manaCost, ...CardOption)       // TypeArtifact + "Equipment" subtype
	[NewLand](name, ...CardOption)                      // no mana cost
	[NewPlaneswalker](name, manaCost, startingLoyalty, ...CardOption)
	[NewToken](name, power, toughness, types, subTypes, ...keywords)

# Planeswalkers (minimal)

The planeswalker primitive is intentionally minimal. [NewPlaneswalker] gives a
card TypePlaneswalker and an EntersWithNCounters(Loyalty, N) replacement so the
permanent enters with its starting loyalty (CR 614.1c / 306.5b). The 0-loyalty
state-based action (CR 704.5i) is implemented in CheckStateBasedActions.

XXX: NOT yet implemented:
  - Loyalty-activated abilities (+N / -N abilities, CR 606)
  - Attacking planeswalkers (CR 506.4 / 508.1) and the planeswalker as a valid
    attack/spell/ability target as an alternative to a player
  - Damage redirection rules (legacy CR 117.6, removed in 2018)

# CardOption Functions

Options configure a card during construction. They are passed as trailing
arguments to any card constructor:

	[WithSubTypes]("Human", "Soldier")     // creature/land subtypes
	[WithSuperTypes](core.Legendary)       // Legendary, Basic, Snow, World
	[WithKeyword](core.Flying)             // keyword attrs (seeds baseAttrs)
	[WithAbility](ability)                 // any Ability (triggered, protection, etc.)
	[WithAction](action)                   // spell or activated ability action
	[WithCardType](core.TypeArtifact)      // additional card type (artifact creature)
	[WithManaAbility](core.Green)          // tap for one mana of color
	[WithAnyColorMana]()                   // tap for any color (Birds of Paradise)
	[WithAdditionalCost](cost)             // extra cost when casting (sacrifice, discard)
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

Spells and non-mana activated abilities are both backed by [ActionDefinition],
which stores costs, effects, and real targeting requirements. Instants and
sorceries automatically wrap direct effects/options in a spell action.

Preferred spell constructor style:

	[NewInstant](name, cost, effectsOrOptions...)
	[NewSorcery](name, cost, effectsOrOptions...)

Explicit spell actions are also available:

	[NewSpell](effectsOrOptions...)            // effects plus options such as WithTarget

Examples:

	// Lightning Bolt: deal 3 damage to any target
	mage.NewInstant("Lightning Bolt", "{R}",
	    mage.DealDamage(mage.Fixed(3)),
	    mage.WithTarget(mage.TargetAnyTarget()),
	)

	// Wrath of God: destroy all creatures (no target)
	mage.NewSorcery("Wrath of God", "{2}{W}{W}",
	    mage.DestroyAllCreatures(),
	)

Older helpers remain available during migration:

	[NewSpellAbility](effects...)              // untargeted spell
	[NewTargetedSpell](target, effects...)     // targeted spell

Effects are the atomic actions. Each implements the [Effect] interface:

	Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error
	Text() string
	Properties() EffectProperties

Examples:

	// Ancestral Recall: target player draws 3
	mage.NewInstant("Ancestral Recall", "{U}",
	    mage.DrawCards(mage.Fixed(3)),
	    mage.WithTarget(mage.TargetPlayer()),
	)

	// Modal spell, legacy "branch on ModeValue" form (Healing Salve):
	c := mage.NewInstant("Healing Salve", "{W}", spell)
	c.SetModes([]string{"Gain 3 life", "Prevent 3 damage"})
	// Inside the FuncEffect, call g.ModeValue() to read the chosen mode (0 or 1).

	// Modal spell, full per-mode targets and effects (Crushing Canopy):
	canopy := mage.NewInstant("Crushing Canopy", "{3}{G}", nil)
	canopy.AddAbility(mage.NewModalSpell([]mage.Mode{
	    {
	        Label:   "Destroy target creature with flying",
	        Targets: []mage.Target{mage.TargetCreature(mage.HasKeywordFilter(core.Flying))},
	        Effects: []mage.Effect{mage.DestroyTarget()},
	    },
	    {
	        Label:   "Destroy target enchantment",
	        Targets: []mage.Target{mage.TargetPermanent(mage.IsEnchantment)},
	        Effects: []mage.Effect{mage.DestroyTarget()},
	    },
	}))
	// At cast time, the engine prompts Player.ChooseMode and gathers
	// targets only for the chosen mode (CR 700.2). At resolution, only
	// that mode's effects run. The chosen mode index is stored on
	// StackObject.ModeChoice; per-mode targets on StackObject.ModalTargets.

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
	[LoseLifeAmount](amount ValueSource)     // controller loses dynamic life
	[TargetPlayerLoseLife](amount ValueSource) // target player loses life
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
	[Scry](amount ValueSource)                   // CR 701.18 — controller scries N

Scry primitives:

	Game.PerformScry(player, n) — engine entry point: looks at the top N
	cards of player's library and rearranges them per
	Player.ChooseScryPlacement. Fires EvtScry with Amount = cards seen.

	Player.ChooseScryPlacement(top, reason, g) decides the split: returns
	(bottom, topOrder) ID slices that partition `top`. BasePlayer's default
	keeps every card on top in original order. TestPlayer queues decisions
	via TestGame.ChooseScry(p, bottom, topOrder).

Reveal-and-pick primitives (reveal.go):

	Game.RevealTopN(player, n) []Card
	    Returns a copy of the top N cards without modifying the library.
	    Powers "look at" / "reveal the top N" effects (Commune with
	    Dinosaurs, Silhana Wayfinder, Muxus, Lurking Predators, Sin Prodder).

	Game.RemoveTopN(player, n) []Card
	    Removes and returns the top N cards. Pair with the relocation
	    primitives below to put cards back in the desired places.

	Game.RevealAndPickFromTop(chooser, owner, n, filter, mayDecline, reason)
	    Reveals top N of `owner`'s library, asks `chooser` to pick one
	    card matching `filter`. Returns (chosen, revealed). chosen is nil
	    if no card matches; if mayDecline=false the chooser is forced to
	    pick from the candidates. The library is NOT mutated — caller
	    typically follows with RemoveTopN + relocation.

	Game.PutOnBottomInRandomOrder(player, cards)
	    Appends `cards` to the bottom of the library in uniformly random
	    order. The cards must already have been removed from the library.

	Game.PutOnBottomInChosenOrder(player, cards)
	    Asks the player (via Player.ChooseScryPlacement, reason
	    "put on bottom in any order") to order `cards` and appends them to
	    the bottom of the library in that order. The first ID in the
	    chosen ordering ends up just above the previous bottom card; the
	    last ID becomes the new deepest card. Powers "Put the rest on the
	    bottom of your library in any order" (Commune with Dinosaurs etc.).
	    Test players script the order via TestGame.ChooseScry(player,
	    ordering, nil).

	Game.PutOnTopInChosenOrder(player, cards)
	    Places `cards` on top in the supplied order (first = new top).

	Game.RevealHand(viewer, owner) []Card
	    Returns a copy of `owner`'s hand. The viewer parameter is
	    informational; the engine does not currently model private vs
	    public knowledge.

	Game.PickFromHand(chooser, owner, filter, mayDecline, reason) Card
	    Asks `chooser` to pick one card from `owner`'s hand matching the
	    filter. Returns nil if no card matches or the chooser declines
	    (only when mayDecline=true). The hand is NOT mutated — callers
	    move the returned card themselves (e.g. via Game.PlayerDiscard).
	    Powers Corpse Traders / Entomber Exarch ("you choose a card from
	    target opponent's hand; they discard it").

	Both reveal-and-pick choosers reuse Player.ChooseCardFromLibrary
	for the actual selection — TestPlayer scripts decisions via
	TestGame.ChooseFromLibrary(p, name).

Random-card primitives (random.go):

	Game.DiscardAtRandom(player, n) []Card
	    Discards up to n cards from the player's hand chosen uniformly
	    at random, firing EvtDiscard for each. Returns the discarded
	    cards. Powers Goblin Lore ("then discard three cards at random")
	    and similar self-discard-at-random effects. Distinct from the
	    DiscardRandom Effect, which targets an opponent.

	Game.RandomCardFromGraveyard(player, filter) Card
	    Returns one card chosen uniformly at random from the player's
	    graveyard that matches the filter, or nil if none match. Does
	    NOT remove the card. Powers Charmbreaker Devils ("instant or
	    sorcery card chosen at random") and Ghoulraiser ("Zombie card
	    at random").

	Game.RandomCardFromHand(player, filter) Card
	    Same pattern for hands — useful for "reveal a card at random
	    from your hand" effects.

	All three use math/rand directly, the same RNG used by
	ShuffleLibrary, DiscardRandomCost, and Aladdin's Lamp draw
	replacement.

Graveyard effects:

	[ReturnFromGraveyardToBattlefield]()          // reanimate (Animate Dead)
	[ReturnFromGraveyardToHandTarget]()           // Raise Dead, Regrowth
	[ReturnSourceToHand]()                        // return self from graveyard to hand (Rancor)

Permanent manipulation effects:

	[Tap]()                                    // tap (target as effect, source as cost)
	[UntapTarget]()                            // untap target permanent
	[UntapSource]()                            // untap self
	[TapOrUntapTarget]()                       // Twiddle: choose tap or untap
	[TapAllLands]()                            // tap all lands target player controls
	[TapAttachedCreature]()                    // tap enchanted creature

Combat effects:

	[Boost](p, t ValueSource)                       // +P/+T (default: target, EOT)
	  .Targeting(ToSource()/ToTarget()/ToMatching(f)/ToAllMatching(f)/ToGathered(v))
	  .Until(EndOfTurn/EndOfCombat)                 // composable target + duration
	[GrantKeyword](kw)                              // grant keyword (default: target, EOT)
	  .Targeting(ToSource()/ToTarget()/...)          // same composable pattern
	  .Until(EndOfTurn/EndOfCombat)
	[GrantType](ct)                                 // grant card type (default: target, Indefinite)
	  .Targeting(ToSource()/ToTarget()/...)          // same composable pattern
	  .Until(Indefinite/EndOfTurn/...)              // Indefinite = lasts while target on battlefield
	[DoubleTargetPower]()                           // double target's power (Berserk)
	[MakeUnblockableUntilEndOfTurn]()               // can't be blocked this turn
	[PreventAttackingTargetUntilEndOfTurn]()        // target creature can't attack this turn (CR 506.4a)
	[SetPTUntilEndOfTurn](p, t, sel)                // set base P/T until EOT
	[SetPowerUntilEndOfTurn](power, sel)            // set base power only until EOT
	[RemoveFromCombat]()                            // remove target from combat

Counter effects:

	[AddCounters](counterType, amount ValueSource).Targeting(sel).Max(n)  // defaults to target
	[RemoveCounters](counterType, amount int).Targeting(sel)             // defaults to source

Misc effects:

	[AddMana](color, amount)              // add mana to pool
	[AddAnyMana](amount, Color)           // add mana of any one color
	[CreateToken](name, p, t, types, subtypes, keywords...)
	[CreateTreasureToken]() / [CreateTreasureTokens](n)  // CR 111.10c predefined token
	[CreateFoodToken]() / [CreateFoodTokens](n)          // CR 111.10d predefined token
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
	    func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
	[TargetPermanent](IsLand)                              // land (= TargetLand)
	[TargetPermanent](IsArtifact)                          // artifact (= TargetArtifact)
	[TargetArtifactWithManaValueX]()                       // artifact with CMC == g.CurrentX (Detonate)
	[TargetPermanent](Or(IsArtifact, IsEnchantment))       // artifact or enchantment

Player targets:

	[TargetPlayer]()                    // any player
	[TargetOpponent]()                  // opponent only
	[TargetAnyTarget]()                 // creature or player ("any target")

Stack targets:

	[TargetSpellOnStack](filters ...CardFilter)  // spell on the stack

Zone targets:

	[TargetCardInYourGraveyard](filters ...CardFilter)        // any card in your graveyard (pass IsCreatureCard for creatures)
	[TargetAnyNumberOfCardsInYourGraveyard](filters)         // any number (0+) of cards in your graveyard
	[TargetCardInHand](filters ...CardFilter)                // any card in your hand (pass IsCreatureCard for creatures)

Multi-target shapes (used with [NewMultiTargetSpell]):

	[TargetUpToNCreatures](n, filters...)             // 0..n creatures (Dauntless Onslaught)
	[TargetUpToNCreaturesOrPlayers](n)                // 0..n creatures-or-players (divided damage)
	[TargetUpToNCardsInYourGraveyard](n, filters...)  // 0..n cards in your graveyard
	[TargetCreatureYouControl](filters...)            // a creature you control
	[TargetCreatureOpponentControls](filters...)      // a creature an opponent controls

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
	[PrintedInSet](string)            // card name was printed in set (by code, e.g. "ARN")

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

# Multi-Target Spells

Three target shapes don't fit the single-target pattern:

  - "Up to N target X" — controller picks 0..N legal targets.
  - "Target Y you control AND target Z an opponent controls" — two distinct
    Targets with disjoint predicates (Peel from Reality, Nature's Way).
  - "N damage divided as you choose among any number of targets" — controller
    chooses both the targets and the per-target damage at announcement
    (CR 601.2d). Flames of the Firebrand, Hail of Arrows.

Use [NewMultiTargetSpell] with a slice of [Target]s; the flat list of chosen
target IDs (in declaration order) is passed to each effect via ctx.Targets.

To act on every chosen target uniformly, select with [ToAllTargets]() — the
selector iterates ctx.Targets and skips uuid.Nil placeholders or targets that
have left the battlefield (CR 608.2b: a multi-target spell whose first target
becomes illegal still affects the surviving targets).

	// Dauntless Onslaught: Up to two target creatures get +2/+2 until EOT
	mage.NewSorcery("Dauntless Onslaught", "{1}{W}",
	    mage.NewMultiTargetSpell(
	        []mage.Target{mage.TargetUpToNCreatures(2)},
	        mage.Boost(mage.Fixed(2), mage.Fixed(2)).Targeting(mage.ToAllTargets()),
	    ),
	)

	// Peel from Reality: Return target creature you control AND target
	// creature an opponent controls to their owners' hands.
	mage.NewInstant("Peel from Reality", "{1}{U}",
	    mage.NewMultiTargetSpell(
	        []mage.Target{
	            mage.TargetCreatureYouControl(),
	            mage.TargetCreatureOpponentControls(),
	        },
	        // FuncEffect that iterates targets and bounces each.
	    ),
	)

	// Flames of the Firebrand: 3 damage divided as you choose among any number of targets.
	mage.NewSorcery("Flames of the Firebrand", "{2}{R}",
	    mage.NewMultiTargetSpell(
	        []mage.Target{mage.TargetUpToNCreaturesOrPlayers(3)},
	        mage.DealDividedDamage(mage.Fixed(3)),
	    ),
	)

For divided-damage spells the engine calls [Player.ChooseDamageDistribution]
at cast/activation time, validates the result (CR 601.2d: sum equals total,
keys are a subset of chosen targets, no negatives), and stores the map on the
[StackObject] as DamageDistribution. At resolution, [DealDividedDamage] reads
the map back, skipping targets that became illegal — only their share is
wasted, the remaining targets still take their assigned damage. Test code
scripts the distribution with TestGame.ChooseDamageDistribution(player, map).

# Cast-Time Snapshot (CR 608.2g)

Some spells reference values that are fixed when the spell is put on the
stack rather than re-queried at resolution. Oracle text cues are phrases
like "as you cast this spell" or "as ~ enters" — for example, Draconic
Roar's "If you revealed a Dragon card or controlled a Dragon as you cast
this spell". Per CR 608.2g, those values are locked in at cast time;
state changes between cast and resolution must not flip the condition.

The cast pipeline writes a [CastContext] onto the [StackObject] when the
spell goes on the stack, populated immediately after additional costs
have been paid. Effects read it back during resolution via
[Game.ResolvingCastContext]:

	ctx := g.ResolvingCastContext()
	if ctx.HasControlledSubtypeAtCast("Dragon") || ctx.RevealedSubtypeAtCast("Dragon") {
	    // bonus damage half — fixed at cast time, immune to in-response removal
	}

[CastContext] is intentionally minimal — extend with new fields as cards
require. Currently captured:

  - ControllerSubtypesAtCast: every subtype the controller had on a
    permanent at cast time (used by "controlled an X as you cast..." riders).
  - RevealedAtCast: cards revealed by [RevealFromHandCost] payments paid
    for this spell (used by "revealed an X as you cast..." riders).

[CastContext] is nil for activated and triggered abilities; helper methods
(HasControlledSubtypeAtCast, RevealedSubtypeAtCast) are nil-safe.

# PermanentSelector — Source vs Target

Effects that can apply to either the source permanent or a resolved target use
[PermanentSelector]:

	[SelectTarget]    // apply to first resolved target (targets[0])
	[SelectSource]    // apply to the ability's source permanent

	// Sengir Vampire: +1/+1 counter on self
	mage.AddCounters(core.P1P1, mage.Fixed(1)).Targeting(mage.ToSource())

	// Spell: +1/+1 counter on target creature
	mage.AddCounters(core.P1P1, mage.Fixed(1))

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
	[Tap]()                            // {T}: tap the source (also usable as effect)
	[SacrificeSourceCost]()            // sacrifice self
	[SacrificeMatchingCost](filter, text)  // sacrifice a permanent matching filter
	[SacrificeCreatureCost]()          // = SacrificeMatchingCost(IsCreature, ...)
	[SacrificeArtifactCost]()          // = SacrificeMatchingCost(IsArtifact, ...)
	[LifePayCost](amount)             // pay life
	[RemoveCountersCost](ct, n)        // remove n counters of type ct
	[DiscardCost](n)                   // discard n cards
	[DiscardRandomCost](n)             // discard n cards at random
	[DiscardCardsWithDifferentNamesCost](n)  // discard n cards w/ distinct names
	[ExileFromGraveyardCost](n)        // exile n cards from your graveyard
	[ExileSourceCost]()                // exile self
	[ReturnToHandCost](filter)         // bounce a permanent to hand (nil = any)
	[TapMatchingCost](filter, text)    // tap another untapped permanent matching filter
	[TapCreatureCost]()                // = TapMatchingCost(IsCreature, ...)

Additional costs on the card itself (paid when casting, not on an ability):

	mage.WithAdditionalCost(mage.SacrificeMatchingCost(mage.IsCreature, "Sacrifice a creature"))

# Activated Abilities

Non-mana activated abilities use the same [ActionDefinition] model as spells.
Prefer [NewActivated] with [WithAction]:

	mage.WithAction(mage.NewActivated(
	    mage.Tap(),
	    mage.DealDamage(mage.Fixed(1)),
	    mage.WithTarget(mage.TargetAnyTarget()),
	))

[NewActivatedAbility] and [WithActivatedAbility] remain as compatibility
wrappers. They take a primary effect, primary cost, and optional [AbilityOption]
modifiers:

	mage.NewActivatedAbility(effect, cost, ...AbilityOption)

ActionOption / AbilityOption modifiers:

	[WithCost](cost)                      // additional cost
	[WithTarget](target)                  // targeting requirement
	[WithEffect](effect)                  // additional effect
	[WithEffects](effects...)             // additional effects
	[WithTiming](TimingSorcery)           // timing rule
	[WithStepTiming](step)                // step-specific timing
	[WithUpkeepOnly]()                    // only during upkeep
	[WithOncePerTurn]()                   // once per turn limit
	[WithAnyPlayerMay]()                  // any player can activate
	[WithControlledSinceTurnStart]()      // must have controlled since turn start

Attach to a card with [WithActivatedAbility] or [WithAbility]:

	// Northern Paladin: {W}{W}, {T}: Destroy target black permanent
	mage.WithActivatedAbility(
	    mage.DestroyTargetPermanent(),
	    mage.ManaCostOf("{W}{W}"),
	    mage.WithCost(mage.Tap()),
	    mage.WithTarget(mage.TargetPermanent(mage.HasColorFilter(core.Black))),
	)

	// Prodigal Sorcerer: {T}: Deal 1 damage to any target
	mage.WithActivatedAbility(
	    mage.DealDamage(mage.Fixed(1)),
	    mage.Tap(),
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

When a triggered ability with one or more .AddTarget(...) declarations is put
on the stack (CR 603.3d), its controller is prompted via Player.ChooseTargets
for each declared Target — the same mechanism used for spell-cast targeting.
Chosen IDs become the StackObject's Targets and are passed to the ability's
effects at resolution time. Triggers with no declared targets keep the legacy
event-derived auto-binding (e.g. ETB triggers receive the entering permanent's
ID, EvtZoneChange (BF→GY, was creature) triggers receive the dead creature's ID, etc.) so cards
that read targets[0] from the firing event continue to work unchanged.

Convenience constructors (set condition automatically):

	[AttacksTrigger](effect, optional)                          // EvtDeclaredAttacker, source is self
	[BlocksTrigger](effect, optional)                           // EvtDeclaredBlocker + Flag=true (source is self; CR 509.3a; fires once per combat per blocker)
	[EntersBattlefieldTrigger](effect, optional)                // EvtZoneChange (To=Battlefield), source is self
	[DiesCreatureTrigger](effect, optional, filter)             // EvtZoneChange (BF→GY, was creature), another creature you control
	[AnyCreatureDiesTrigger](effect, optional)                  // EvtZoneChange (BF→GY, was creature), any creature
	[CreatureDealtDamageBySourceDiesTrigger](effect, optional)  // creature damaged by source dies
	[DealsDamageToOpponentTrigger](effect, optional)            // EvtDamageDealt to opponent
	[WhenDamageDealtToThisTrigger](effect, optional)            // EvtDamageDealt to self
	[BeginningOfUpkeepTrigger](effect, optional)                // EvtUpkeep, controller's upkeep
	[BeginningOfEachUpkeepTrigger](effect, optional)            // EvtUpkeep, every player
	[BeginningOfEachDrawStepTrigger](effect, optional)          // EvtDrawStep, while untapped
	[BeginningOfAttachedControllerUpkeepTrigger](effect, opt)   // EvtUpkeep, attached creature's controller
	[WheneverSpellCastTrigger](effect, optional, ...CardFilter)  // EvtSpellCast, any player, filtered
	[WheneverYouCastSpellTrigger](effect, optional, ...CardFilter) // EvtSpellCast, controller only
	[WhenAttachedBecomesTappedTrigger](effect, optional)        // EvtTapped, enchanted permanent
	[WheneverPermanentEntersBattlefieldTrigger](e, opt, filter) // EvtZoneChange (To=Battlefield), filtered
	[WhenOpponentPermanentBecomesTappedTrigger](e, opt, filter) // EvtTapped, opponent's matching permanent
	[PutIntoGraveyardFromBattlefieldTrigger](effect, optional)  // EvtZoneChange (BF→GY), self
	[SacrificeAtUpkeepUnlessPay](manaCost)                      // sacrifice unless pay at upkeep
	[WheneverYouGainLifeTrigger](effect, optional)              // EvtLifeGained, controller (CR 119.9)
	[WheneverPlayerGainsLifeTrigger](effect, optional)          // EvtLifeGained, any player
	[WheneverYouLoseLifeTrigger](effect, optional)              // EvtLifeLost, controller (CR 119.9)
	[WheneverOpponentLosesLifeTrigger](effect, optional)        // EvtLifeLost, opponent (Exquisite Blood)
	[WheneverPlayerLosesLifeTrigger](effect, optional)          // EvtLifeLost, any player
	[WheneverYouDiscardTrigger](effect, optional)               // EvtDiscard, controller
	[WheneverOpponentDiscardsTrigger](effect, optional)         // EvtDiscard, opponent (Fell Specter)
	[WheneverPlayerDiscardsTrigger](effect, optional)           // EvtDiscard, any player
	[WheneverYouSacrificeAnotherCreatureTrigger](effect, opt)   // EvtSacrifice, another creature you control (Kels)
	[WheneverYouSacrificeTrigger](effect, optional)             // EvtSacrifice, any non-self permanent you control
	[WheneverBecomesTargetTrigger](effect, optional)            // EvtBecomesTarget, source self (Departed Deckhand)
	[WheneverBecomesTargetFirstTimeEachTurnTrigger](e, opt)     // EvtBecomesTarget, first time each turn (Kira)
	[WheneverDealsCombatDamageToPlayerTrigger](effect, opt)     // EvtDamageDealt to a player, combat, source self
	[WheneverPermanentDealsCombatDamageToPlayerTrigger](e, opt, filter) // combat damage to player, controller's matching permanent (Coastal Piracy, Sharding Sphinx)
	[WheneverEnchantedPermanentDealsDamageToPlayerTrigger](e, opt)      // damage to player, source is enchanted permanent (Curiosity)
	[WhileInZoneTrigger](zone, evtType, effect, optional)               // CR 113.6 — trigger functions while source is in zone (Pia Nalaar from graveyard)
	[BeginningOfYourEndStepFromGraveyard](effect, optional)             // EvtEndStep, controller's, source in graveyard (Pia Nalaar)

By default, GenericTriggered functions only on the battlefield (CR 113.6). Use
[GenericTriggered.InZone] to declare an additional active zone (graveyard,
hand, exile). FireEvent scans each player's graveyard for card-level
GenericTriggered abilities whose ActiveZones include ZoneGraveyard, sets the
ability's source/controller transiently to the card and its owner, and queues
the trigger like any battlefield trigger.

EvtLifeGained / EvtLifeLost auto-binds preserve evt.Amount as the trigger's
EventAmount (readable via mage.EventAmountValue() in effects), but do NOT
bind a default target — "you gain that much life" triggers fall back to the
controller naturally. EvtDiscard and EvtSacrifice auto-bind the event's
PlayerID as targets[0] so effects like "that player loses 2 life" target
the discarder/sacrificer.

EvtBecameUntapped fires whenever a permanent becomes untapped (during the untap step or
by an effect like Twiddle). Use with NewTriggered for "when this becomes untapped" triggers.

EvtBecomesTarget fires whenever a permanent or player becomes the target of a
spell or activated ability (CR 603.6c, 119.5). It is fired immediately after
the spell is cast or the ability is activated (and copies pushed to the stack
via spell-copy effects), once per distinct target. Event fields:

  - SourceID — the source of the targeting (card ID for spells, permanent ID
    for activated abilities)
  - TargetID — the targeted object (permanent or player)
  - PlayerID — the controller of the spell/ability that is targeting
  - Flag    — true if the source is an activated ability, false if a spell

The auto-bind passes the targeted object as targets[0] and the offending
spell/ability source as targets[1] so a "counter that spell or ability"
effect (Kira) can find the spell on the stack via targets[1]. Game tracks
TimesTargetedThisTurn(id) for "first time each turn" predicates; the counter
resets at cleanup.

Custom triggers with SetConditionData (composable data predicates):

	// When this creature dies (not "another" — self)
	mage.NewTriggered(core.EvtZoneChange (BF→GY, was creature), false, effect).
	    SetConditionData(mage.EventSourceIsSelf{})

	// Whenever an opponent's Swamp becomes tapped
	mage.NewTriggered(core.EvtTapped, false, effect).
	    SetConditionData(mage.AndTriggerCond{Conditions: []mage.TriggerConditionData{
	        mage.EventSourceControlledByOpponent{},
	        mage.EventSourceHasSubType{SubType: "Swamp"},
	    }})

	// Source is attacking and blocked by a non-Wall
	mage.NewTriggered(core.EvtBlockersDecl, false, effect).
	    SetConditionData(mage.SourceBlockedByCreatureMatching{Filter: mage.Not(mage.HasSubType("Wall"))})

The optional flag (second arg) controls whether the controller may decline:
false = mandatory, true = "you may" (AI/player can decline).

## "You may pay {N}. If you do, ___" (CR 603.4)

Triggers of the form "Whenever X, you may pay {cost}. If you do, ___" prompt
the controller for an optional payment mid-resolution. Wrap the inner effect
with [MayPayMana]:

	mage.MayPayMana("{1}{W}", "put X +1/+1 counters on target creature",
	    inner)

On Apply, the wrapper calls Player.ChooseMayAbility with the description; if
the player accepts AND Game.TryPayCostFromLands succeeds for the cost, the
inner Effect runs with the same sourceID/controller/targets the outer
trigger received. If the player declines or the cost cannot be paid,
nothing happens (no partial payment is taken). Used by Cradle of Vitality,
Kels Fight Fixer, Emiel the Blessed, and similar abilities.

Predicates implement [TriggerConditionData]:

	type TriggerConditionData interface {
	    CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool
	}

Compose with [AndTriggerCond], [OrTriggerCond], [NotTriggerCond].

Common atomic predicates:

	[EventSourceIsSelf]                  // evt.SourceID == sourceID
	[EventTargetIsSelf]                  // evt.TargetID == sourceID
	[EventPlayerIsController]            // evt.PlayerID == controllerID
	[EventPlayerIsNotController]         // evt.PlayerID != controllerID
	[EventSourceControlledByOpponent]    // source permanent controlled by opponent
	[EventSourceHasType]{Type}           // source permanent has card type
	[EventSourceHasSubType]{SubType}     // source permanent has subtype
	[SourceIsBlockedAttacker]            // source is attacker with blockers
	[SourceIsUnblockedAttacker]          // source is attacker with no blockers
	[SourceInCombat]                     // source is attacking or blocking
	[EventSourceIsSelfDamageToPlayer]    // source dealt damage to any player
	[EventSourceIsSelfDamageToOpponent]  // source dealt damage to opponent
	[SpellCastIsType]{Type}              // spell cast has card type
	[ControllerHasNoPermanentMatching]{Filter} // controller has no matching permanent

For rare cases needing full closure access, SetCondition is still available.

When refining a constructor that already installed a filter (e.g.
WheneverPermanentEntersBattlefieldTrigger), use [GenericTriggered.AndConditionData]
to compose the new predicate with the constructor's filter (logical AND).
Calling SetConditionData would REPLACE the existing condition and silently
drop the constructor's filter.

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

# Layers (CR 613)

Continuous effects are applied in a fixed layer order (CR 613.1). The layer
system ensures interactions resolve deterministically — e.g., a type change in
layer 4 is visible to an ability-granting effect in layer 6, which in turn is
visible to a P/T modification in layer 7.

[EffectManager.Apply] resets all computed state (grantedAttrs, P/T bonuses,
SubTypeOverride, ColorOverride, RuntimeAbilities) then reapplies every active
effect in layer order:

	[LayerCopy]    (1) — copy effects (Doppelganger, Clone)
	[LayerControl] (2) — control changes (Control Magic, Aladdin)
	[LayerText]    (3) — text-changing effects (not currently used)
	[LayerType]    (4) — type changes, subtype overrides, animate effects
	[LayerColor]   (5) — color changes (lace effects)
	[LayerAbility] (6) — add/remove abilities, keywords, attrs
	[LayerPT]      (7) — power/toughness modifications

Layer 7 has four sublayers per CR 613.4 (not yet split in the engine):

	7a — Characteristic-defining abilities that set P/T (e.g. Tarmogoyf)
	7b — Effects that set P/T to a specific value (e.g. "becomes a 0/1")
	7c — Effects that modify P/T (e.g. +1/+1 counters, Giant Growth)
	7d — P/T switching effects (e.g. "switch power and toughness")

Within each layer/sublayer, effects are applied in timestamp order (CR 613.7).
An effect with an earlier timestamp is applied first. Dependency (CR 613.8) can
override timestamp order: if applying effect B would change what effect A does,
A depends on B and waits for B to apply first.

# Durations

	[WhileOnBattlefield]   // stays as long as source is on battlefield
	[EndOfTurn]            // removed at end of turn cleanup
	[EndOfCombat]          // removed at end of combat
	[Indefinite]           // never auto-removed; IsActive controls lifetime

# ActiveCondition

Controls when a FuncContinuousEffect is active:

	[SourceAttached]            // source is attached to another permanent
	[SourceUntapped]            // source is untapped (Winter Orb)
	[SourceTapped]              // source is tapped (Ashnod's Battle Gear)
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
	[BoostAttachedByCount](filter, powerFn, toughFn)   // P/T by controlled count

Target effects (apply to specific permanent by ID):

	[TemporaryBoost](targetID, power, toughness)           // +P/+T until EOT
	[TemporaryKeyword](targetID, keyword)                  // keyword until EOT
	[KeywordReplacement](targetID, from, to)               // replace keyword (Indefinite)
	[SetBasePT](targetID, power, toughness)                // set base P/T until EOT
	[SetBasePower](targetID, power)                        // set base power until EOT
	[ColorOverride](targetID, color)                       // change color (Indefinite)
	[TemporaryAnimate](targetID, power, toughness)         // animate until EOT
	[TemporaryAnimateUntilEndOfCombat](id, power, tough)   // animate until end combat
	[AnimateTargetLand](targetID, opts, duration)          // Elemental Uprising (full options)
	[AnimateLandWhileSourceOnBattlefield](targetID, opts)  // Awakener Druid pattern
	[AnimateAttachedLand](opts)                            // Vastwood Zendikon (aura animates host)
	[GrantManaAbilityToAttached](productions...)           // New Horizons (extra mana ability)
	[PreventBlockingUntilEndOfCombat](permID)              // can't block until end combat
	[PreventAttackingUntilEndOfTurn](permID)               // can't attack until EOT (CR 506.4a)

# Animate Land

CR 305.7 / 612 (layer system) require a land that "becomes a creature" to
remain a land while gaining creature type, P/T, and (optionally) extra
subtypes/colors/keywords. The engine exposes [AnimateLandOptions] and four
helpers built on the layer-aware effect primitives:

  - [AnimateTargetLand](id, opts, dur) — for spells like Elemental Uprising
    ("target land becomes a 4/4 Elemental creature with trample until end
    of turn. It's still a land."). Use core.EndOfTurn for instants.
  - [AnimateLandWhileSourceOnBattlefield](id, opts) — for Awakener Druid's
    ETB clause ("target Forest becomes a 4/5 green Treefolk creature for
    as long as Awakener Druid remains on the battlefield"). The effect's
    sourceID must be set to the source permanent before registration; the
    effect manager removes it automatically when the source leaves.
  - [AnimateAttachedLand](opts) — for auras like Vastwood Zendikon
    ("Enchanted land is a 6/4 green Elemental creature with trample. It's
    still a land."). Built on [AttachedEffect] and gated by SourceAttached.
  - [GrantManaAbilityToAttached](productions...) — companion helper for
    cards that grant additional mana abilities to the enchanted land
    (e.g. New Horizons' "{T}: Add {G}" clause).

All four helpers route through applyAnimateLand: they grant AttrIsCreature
/ AttrCanAttack / AttrCanBlock / AttrHasPowerToughness, set
BasePTOverride, append (not replace) subtypes via SubTypeOverride, append
colors via ColorOverride, and grant keyword attrs. AttrIsLand is left in
baseAttrs so the land remains a land per CR 305.7. Each Apply() cycle
recomputes from the card baseline so reverting (aura leaves, source
dies, EOT cleanup) restores the land to its non-creature state.

Global/source-based effects (while source on battlefield):

	[BoostAllCreatures](p, t, filter)                      // lord effect (excludes self)
	[BoostAllCreaturesIncludingSelf](p, t, filter)         // includes self
	[BoostControlledCreatures](p, t, filter)               // only your creatures
	[BoostSelf](p, t, SourceCondition)                     // conditional self-boost
	[PTEqualsCount](filter)                                // P/T = count of matching permanents
	[PTEqualsControlledCount](filter)                      // P/T = count you control
	[GrantKeywordToAll](keyword, filter)                   // grant keyword to matching
	[GrantActivatedAbilityToAll](effect, cost, filter)     // grant ability to matching
	[GrantTriggeredAbilityToAll](evt, opt, cond, filter, effects...) // grant trigger to matching
	[PreventUntapForMatching](filter)                      // Meekstone
	[PreventAllUntaps]()                                   // Stasis
	[AnimateArtifact](Attached)                            // Animate Artifact aura
	[AnimateArtifact](ForAll(duration))                    // Titania's Song (global)
	[AnimateArtifact](ForTarget(id, duration))             // Xenic Poltergeist (targeted)
	[AnimateLands](filter, power, toughness)               // Living Lands
	[LimitLandUntaps](limit)                               // Winter Orb (while untapped)
	[AllowUnlimitedLandPlays]()                            // Fastbond
	[IncreaseSpellCostForColor](color, amount)             // Gloom
	[ReduceSpellCostForColor](color, amount)               // cost reduction
	[ReduceSpellCostStatic](filter, amount, condition)     // conditional cost reduction (CR 601.2f)
	[ChangeSubTypesForAll](fromSubTypes, toSubTypes)       // Conversion
	[ManaConversion](from, to Color)                       // Sunglasses of Urza
	[BodyguardContinuous]()                                // Veteran Bodyguard
	[PersonalIncarnationRedirect]()                        // Personal Incarnation
	[PreventFromAttackingIfDefendingPlayerControls](f)     // Dandân

Combat restrictions (CR 509.1b/c, declared on attackers and blockers):

	[SourceCantBeBlockedExceptBy](filter)              // Gingerbrute, Invisibility-style
	[SourceCanBlockOnly](filter)                       // Rishadan Airship
	[SourceCantBeBlockedByFewerThan](n)                // Goblin Goon: needs 3+ blockers
	[TargetCantBeBlockedExceptBy](id, filter, dur)     // Ghirapur Guide
	[TargetMustBeBlockedIfAble](id, dur)               // Enlarge, Irresistible Prey
	[PreventBlockByPowerLessThanSource](filter)        // Champion of Lambholt
	[PreventAttackingIfDefenderControlsMore](filter)   // Goblin Goon attack clause

Filter helpers built for combat restrictions:

	[PowerLessOrEqual](n)        // creatures with power N or less
	[PowerGreaterThan](n)        // creatures with power > N

The [EffectManager] holds the per-cycle restriction maps:

	em.AddCantBeBlockedExceptBy(attackerID, filter)    // CR 509.1b
	em.AddCanBlockOnly(blockerID, filter)              // CR 509.1b
	em.AddMinBlockers(attackerID, n)                   // CR 509.1b ("N or more")

Filter-based restrictions are checked inside [CanBlock]. The minimum-blockers
constraint and "must be blocked if able" (AttrMustBeBlockedIfAble) are
enforced post-declaration in doDeclareBlockers via enforceMinimumBlockers
and enforceMustBeBlockedIfAble.

Damage prevention rules (continuous):

	[PreventDamageFromTo](from, toFactory, ...SourceCondition)
	    // Camel: prevent Desert damage to self and banded creatures

# Conditional Spell Cost Reduction (CR 601.2f)

Two flavors are supported. Both reduce only the *generic* portion of a mana
cost; colored requirements are unchanged. A spell's generic cost cannot drop
below zero (per cast).

 1. Static reductions sourced from a permanent on the battlefield, applying
    to spells the source's controller casts that match a spell-filter:

    [ReduceSpellCostStatic](filter, amount, condition) ContinuousEffect
    [ReduceSpellCostStaticLabeled](label, filter, amount, condition)

    Use as the argument to [WithStaticAbility]. The continuous effect
    registers a [SpellCostReducer] entry on [GameRules.SpellCostReducers]
    each Apply() cycle while the source is on the battlefield. Examples:
    Warden of Evos Isle, Dragonlord's Servant, Dragonspeaker Shaman,
    Herald's Horn.

 2. Self cost reductions intrinsic to the casting card itself, reducing
    only its own cost at cast time (works while the card is in hand —
    the cast pipeline walks the casting card's abilities directly):

    [WithSelfCostReduction](amount, condition) CardOption
    [SelfCostReduction](amount, condition) *SelfCostReductionAbility

    Examples: Bone Picker, Cryptic Serpent, Ghalta Primal Hunger.

Filters ([SpellPredicate]):

	[SpellAny]()                              // every spell
	[SpellHasType](CardType)                  // creature/instant/etc.
	[SpellHasSubType](string)                 // "Dragon", "Wizard"
	[SpellHasKeyword](Attr)                   // Flying, Trample
	[SpellIsSelf]()                           // the registered card itself
	[SpellSubTypeMatchesChosen]()             // Herald's Horn (ChosenSubtype)
	[SpellsAnd](preds...) / [SpellsOr](preds...)

Amounts ([SpellAmount]):

	[FixedAmount](n)                          // constant
	[AmountByGraveyardCount](CardFilter)      // Cryptic Serpent
	[AmountByTotalPower](PermanentFilter)     // Ghalta
	[AmountByPermanentCount](PermanentFilter)

Conditions ([SpellCondition]):

	[CondCreatureDiedThisTurn]()              // Bone Picker
	[CondControlsMatching](PermanentFilter)   // Wizard's Retort, Winged Words
	[CondAnd](conds...)
	(nil) — unconditional

Inspection (engine tests, AI):

	g.[ConditionalSpellCostReduction](controller, card) int
	    // total generic reduction that would apply to a hypothetical cast

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
	[AttrMustBeBlocked]       // Lure: every able blocker must block this
	[AttrMustBeBlockedIfAble] // CR 509.1c: must be blocked by at least one able blocker
	[AttrMayNotUntap]         // player may choose not to untap

Type-identity attrs:

	[AttrIsCreature], [AttrIsLand], [AttrIsArtifact], [AttrIsEnchantment]

Keyword attrs (all >= Flying):

	[Flying], [Reach], [FirstStrike], [DoubleStrike], [Trample], [Vigilance],
	[Haste], [Menace], [Fear], [Deathtouch], [Lifelink], [Defender], [Banding],
	[Indestructible], [Hexproof], [Shroud], [Forestwalk], [Islandwalk],
	[Swampwalk], [Mountainwalk], [Plainswalk], [Desertwalk], [UnblockableKW],
	[CantBeBlockedByWalls], [CantBeBlockedExceptByWalls], [CanBlockAny],
	[CanBlockAdditional], [BasiliskTouch], [CantRegenerate], [Flash]

Flash (CR 702.8 — "You may cast this spell any time you could cast an instant.")
is a casting-time keyword rather than a permanent ability: it is read off the
card while it is still in hand, by [Game.CastSpellByName] and
[Game.GetCastableSpells], to bypass the sorcery-speed gate. The keyword is
seeded onto the card via [WithKeyword] like any other keyword and is not
re-checked once the spell is on the battlefield.

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

[DamageSystem] has been largely gutted — most damage-related mechanisms are now
[ReplacementEffect] implementations in the replacement pipeline (see
"Replacement Effect System" above). What remains on DamageSystem:

	Reflection:  SetDamageReflection, GetDamageReflection  — Eye for an Eye (post-damage, not a replacement)
	Legacy API:  AddDamagePreventionRule (with [WithFrom], [WithTo], [WithOneShot]) — delegates to EffectManager.AddCycleReplacement
	             AddRegenerationShield — delegates to EffectManager.AddReplacement
	             SetArtifactDamageRedirect — delegates to EffectManager.AddCycleReplacement

Lifecycle: ResetPerCycle() is now empty. ClearEndOfTurn() clears reflection only.

# GameRules

[GameRules] owns all game-rule modifier state. Public fields set directly by
continuous effects:

	LandUntapMax       int             // max lands to untap per step (-1 = unlimited)
	ArtifactUntapMax   int             // max artifacts to untap per step (-1 = unlimited)
	UnlimitedLandPlays bool            // bypass one-land-per-turn
	SpellCostIncreases map[Color]int   // per-color cost increases (Gloom, etc.)
	SpellCostReductions map[Color]int  // per-color cost reductions
	SpellCostReducers  []SpellCostReducer // conditional generic-mana reducers (CR 601.2f)
	ManaConversion     map[Color]Color // forced mana conversion (Celestial Dawn)

Methods for special rules:

	Lich:      SetLichActive, IsLichActive, GetLichPermanent
	Channel:   SetChannelActive, IsChannelActive
	Sanctuary: SetSanctuaryActive, IsSanctuaryActive
	Skip draw: SetSkipNextDraw, ShouldSkipDraw
	Min life:  SetMinimumLife, HasMinimumLife
	Max hand:  SetMaxHandSize, GetMaxHandSize
	Cast block: AddExpansionCastBlock, IsCardExpansionBlocked

Lifecycle: ResetPerCycle() clears per-Apply state. ClearEndOfTurn() clears
end-of-turn flags. SyncManaConversions(players) writes mana conversions to
player mana pools.

# *Game — The Effect API

Effects receive a [*Game] pointer for mutations. Read-only callbacks (trigger
conditions, value sources, player selectors) receive a [GameReader] interface
to prevent accidental mutation.

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
	ResolvingCastZone() Zone
	ResolvingCastContext() *CastContext
	FindStackObject(uuid.UUID) *StackObject
	CombatGroups() []*CombatGroup

Mutation methods:

	PlayerGainLife(Player, int)
	FireEvent(GameEvent)
	PutOnBattlefield(Card, uuid.UUID) *Permanent
	RemoveFromBattlefield(*Permanent)
	DestroyPermanent(*Permanent)
	ExilePermanent(*Permanent)
	TapPermanent(*Permanent)
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

*Game also has proxy methods for the DamageSystem and GameRules
subsystems, so card effects call e.g. g.SetPreventCombatDamage() or
g.AddRegenerationShield(id) rather than reaching through g.Effects.Damage
directly. The full list is in game_mutator.go.

# Game Cloning & Search Execution (execute.go, clone.go)

[Game.Clone]() deep-copies the entire game state for AI search. Players are
wrapped in [SearchPlayer] for non-interactive choice defaults. All mutable state
(permanents, effects, stack, combat, exile) is deep-copied; immutable Card refs
are shared.

Headless execution methods drive cloned games programmatically (no player callbacks):

	g.ExecuteAttackers(playerID, attackerIDs)  // declare attackers + tap + events
	g.ExecuteBlockers(assignments)             // declare blockers + events
	g.ExecuteCombatDamage()                    // resolve first strike + normal damage
	g.RunRemainingSteps()                      // fast-forward from current step to end of turn

For spell/land/ability execution on clones, use the standard methods:

	g.PlayLand(playerID, cardID)                          // play + resolve stack
	g.CastSpellByID(playerID, cardID, targets, xValue)    // put on stack (call ResolveStack after)
	g.ActivateAbilityByIndex(playerID, permID, idx, tgts) // put on stack (or handle mana ability)
	g.ResolveStack()                                      // drain stack atomically

# Modal Spells (CR 700.2)

A modal spell offers the controller a choice of two or more options at
cast time. Use NewModalSpell to declare per-mode targets and effects:

	mage.NewModalSpell([]mage.Mode{
	    {Label: "...", Targets: []mage.Target{...}, Effects: []mage.Effect{...}},
	    {Label: "...", Targets: []mage.Target{...}, Effects: []mage.Effect{...}},
	})

The engine prompts Player.ChooseMode at cast time, then gathers targets
only for the chosen mode (CR 700.2d). The chosen mode index is recorded on
StackObject.ModeChoice; per-mode target lists on StackObject.ModalTargets.
At resolution, only the chosen mode's Effects run.

NewModalSpell panics with fewer than two modes (modal spells have at least
two options by definition).

For modal triggered abilities with per-mode targets (Entomber Exarch ETB,
"Choose one — …" trigger wording), use GenericTriggered.WithModes. Per CR
603.1f the mode is chosen as the trigger goes on the stack, and per CR
603.3d the chosen mode's targets are gathered at the same time. The engine
prompts Player.ChooseMode for the mode index, runs target selection for
that mode only, and pushes only that mode's Effects onto the stack object
— so resolution runs only the selected mode.

	mage.EntersBattlefieldTrigger(nil, false).WithModes(
	    mage.Mode{
	        Label:   "Return target creature card from your graveyard to your hand",
	        Targets: []mage.Target{mage.TargetCardInYourGraveyard(mage.IsCreatureCard)},
	        Effects: []mage.Effect{mage.ReturnFromGraveyardToHandTarget()},
	    },
	    mage.Mode{
	        Label:   "Target opponent reveals their hand …",
	        Targets: []mage.Target{mage.TargetOpponent()},
	        Effects: []mage.Effect{revealAndDiscard},
	    },
	)

WithModes panics with fewer than two modes (CR 700.2). When modes are
declared, AddTarget/AddEffect on the same trigger are ignored — the chosen
mode's lists fully replace them.

For modal triggers whose modes have no targets (Trusty Retriever's "put a
counter on this" / "draw a card"), ModalTriggerEffect remains an option.
It prompts ChooseMode at trigger resolution time rather than at
put-on-stack time, which is observably indistinguishable when no targets
are involved. Prefer WithModes for any modal trigger that has per-mode
targets — only WithModes implements the strict CR 603.1f / 603.3d ordering
(mode and targets chosen at stack placement, before priority passes).

	mage.ModalTriggerEffect("Trusty Retriever", []mage.ModalTriggerMode{
	    {Label: "Put a counter on this", Resolve: func(g, src, ctrl, targets) error {...}},
	    {Label: "Draw a card",            Resolve: func(g, src, ctrl, targets) error {...}},
	})

The legacy SetModes / g.ModeValue branch-inside-FuncEffect API still works
for cards whose modes don't differ in target shape; both APIs share the
same currentMode plumbing.

# Spell-Copy Primitive (CR 706, 707.10)

Game.CopySpellOnStack creates a duplicate StackObject of a spell already on
the stack. Used by Doublecast, Dualcaster Mage, Twincast, Reverberate, Fork,
Twinning Staff, Riku of Two Reflections, and similar effects.

	cp := g.CopySpellOnStack(originalSourceID, controller, mayChooseNewTargets)
	    // originalSourceID — SourceID (card ID) of the spell to copy;
	    //                    typically read from EvtSpellCast.SourceID.
	    // controller       — player who controls the copy (CR 706.10c).
	    // mayChooseNewTargets — true reprompts the controller for every
	    //                       declared Target on the spell's SpellAbility.
	    //                       For modal spells, only the chosen mode's
	    //                       targets are reprompted.

The returned StackObject:

  - Has a fresh ID and a freshly Copy()'d underlying Card (so spell-copy
    triggers don't double-fire on the same card ID).
  - Inherits the original's effects, X value, chosen mode, and divided-
    damage distribution.
  - Has IsCopy=true. ResolveStackObject honors the flag: a copy of a spell
    ceases to exist when it resolves OR fizzles (CR 707.10) — it does
    not enter the graveyard, the battlefield, or any other zone.

Returns nil if the original spell isn't on the stack (caller can no-op).
Use CopyStackObjectDirect when you already have the *StackObject in hand.

# Casting From Non-Hand Zones (Alternate Costs)

CR 117.9 / 601.2b: a player may sometimes be allowed to cast a card from a
non-hand zone (graveyard, exile) using either no mana cost ("without paying
its mana cost", an alternate cost of 0) or a different alternate cost. Cards
like Scholar of the Lost Trove, Scourge of Nel Toth, Etali Primal Storm,
Gonti Lord of Luxury and Maelstrom Archangel rely on this.

The engine exposes four helpers for this:

	g.CastCardFromZoneWithoutPaying(playerID, cardID, zone, targets, xValue)
	    — Move a card from any zone (Hand, Graveyard, Library, Exile) onto the
	      stack paying NO mana cost. Additional costs printed on the card
	      (sacrifice, discard, etc.) are still paid (CR 601.2b). Used by
	      "without paying its mana cost" effects.

	g.CastCardFromZoneWithAlternateCost(playerID, cardID, zone, mc, targets, xValue)
	    — Like the above but pays the supplied alternate ManaCost from the
	      controller's pool instead of the card's printed cost. The caller
	      provides both the zone and the mana cost; the card need not declare
	      the alt-cost itself. Used by effect-driven alt-casts where the
	      grant lives on a different card.

	WithAlternateCost(zone, mana, additional...) CardOption
	g.CastCardWithAlternateCost(playerID, cardID, altIdx, targets, xValue)
	    — Card-level alternate cost (CR 117.9). The card's constructor
	      registers one or more AlternateCost entries (zone + mana cost +
	      optional additional costs such as sacrifice/discard/pay-life), and
	      callers route through CastCardWithAlternateCost which:
	        1. Verifies the card is in the alt's source zone.
	        2. Auto-taps for the alt mana cost if the pool is short.
	        3. Validates that all additional costs can be paid together
	           (e.g. two SacrificeCreatureCost entries require two
	           sacrificable creatures, not just one).
	        4. Pays the additional costs, then enters the standard
	           cast-from-zone pipeline (paying the alt mana, removing the
	           card from its zone, pushing onto the stack, firing
	           EvtSpellCast). Used by Scourge of Nel Toth ("cast from your
	           graveyard by paying {B}{B} and sacrificing two creatures").
	      An optional Condition func gates the alt-cost (e.g. "only on your
	      turn"); when nil the alt is always available while the card sits
	      in the named zone.

	g.GrantCastFromExile(playerID, cardID, anyColorMana)
	g.CastFromExilePermissionFor(playerID, cardID) *CastableFromExilePermission
	g.ClearCastFromExilePermission(cardID)
	g.CastExiledCardWithPermission(playerID, cardID, targets, xValue)
	    — Per-card permission to cast a specific exiled card (Gonti, Lord of
	      Luxury). When AnyColorMana is true, the card's colored pips collapse
	      into generic for the cost calculation, modelling CR 609.4b "spend
	      mana as though it were mana of any color". The permission is
	      cleared automatically when the card leaves exile. The caster need
	      not own the exiled card (CR 706.10): Gonti's controller can cast a
	      card originally owned by an opponent.

	g.ExileCardFaceDown(card, exiledBy, revealedTo...)
	g.RevealExiledCardTo(cardID, playerID)
	ExiledCard.FaceDown / ExiledCard.RevealedTo / ExiledCard.VisibleTo(playerID)
	    — Face-down exile (CR 707, 408). A face-down exiled card hides its
	      identity from every player except those listed in RevealedTo. Used
	      by Gonti, Lord of Luxury: the chosen card is exiled face down and
	      only Gonti's controller is permitted to see it. RevealExiledCardTo
	      grants additional players permission to look at the identity later
	      (no-op once the entry has gone face-up or left exile). When the
	      card leaves exile (cast / removed) the face-down state is dropped
	      with the entry; UI/observers should query VisibleTo to decide
	      whether to display the card's name and characteristics.

	g.AddExileIfWouldGoToGraveyardThisTurn(cardID, sourceID)
	g.IsCardMarkedExileInsteadOfGraveyard(cardID) bool
	g.ClearExileInsteadOfGraveyardForTurn()
	    — Turn-scoped rider for the Scholar-of-the-Lost-Trove pattern: "If
	      that spell would be put into a graveyard this turn, exile it
	      instead." Tagged cards are exiled when ResolveStackObject would put
	      them in the graveyard, and a registered ReplacementEffect intercepts
	      the destroy/death path for permanents. The tag is cleared at the
	      cleanup step.

These helpers fire EvtSpellCast like the standard CastSpellByName, so cast
triggers (Storm, prowess, "whenever you cast a spell") see the cast.

# Replacement Effect System

The engine implements a generic replacement effect pipeline (MTG rule 614).
When a game mutation is about to happen — damage, destruction, life gain, or
card draw — it is wrapped in an [Action] struct and run through
[EffectManager.ApplyReplacements] before executing. Each registered
[ReplacementEffect] can match, modify, redirect, or fully prevent the action.

## Action Types

Six concrete action types represent pending mutations:

	[*DamageToPlayerAction]    — damage about to be dealt to a player
	[*DamageToCreatureAction]  — damage about to be dealt to a creature
	[*DestroyPermanentAction]  — a permanent about to be destroyed
	[*LifeGainAction]          — a player about to gain life
	[*DrawCardAction]          — a player about to draw a card
	[*AddCountersAction]       — counters about to be put on a permanent (CR 614.1c)

All implement the [Action] interface (single method: ActionSource() uuid.UUID).
Damage actions expose Amount(), IsCombatDamage(), PlayerID()/PermanentID(),
and a WithAmount(int) copy method for partial prevention.
[*AddCountersAction] exposes PermanentID(), CounterType(), Amount(), and
OnEntry() (true when the placement happens during enter-the-battlefield
resolution before EvtZoneChange (To=Battlefield)); WithAmount(int) returns a copy with
a different count.

## ReplacementEffect Interface

	type ReplacementEffect interface {
	    Matches(Action, GameReader) bool   // does this replacement apply?
	    Replace(Action, *Game) Action      // transform or prevent the action
	    SourceID() uuid.UUID               // permanent/spell that created this
	    IsActive(GameReader) bool          // still valid?
	    GetDuration() Duration             // when does this expire?
	    Clone() ReplacementEffect          // deep copy for game cloning
	}

Replace returns a modified action, a different action type (e.g., redirect
damage from player to creature), or nil to fully prevent the mutation. The
[replacementBase] struct provides default SourceID and GetDuration implementations.

## Replacement Durations

Each replacement declares its duration via GetDuration():

	EndOfTurn          — cleared at cleanup step (fog, forcefield, prevention
	                     shields, regeneration, one-shot redirects)
	WhileOnBattlefield — active while source permanent exists (Lich, cycle
	                     replacements re-registered by continuous effects)

## The ApplyReplacements Pipeline

[EffectManager.ApplyReplacements](action, g) loops over all registered
replacements. For each iteration it finds the first active, matching
replacement that has not yet fired for this event, applies it, and restarts.
Each replacement fires at most once per event (preventing infinite loops).
When no more replacements match, the (possibly transformed) action executes.
If any replacement returns nil, the action is fully prevented.

## Two Replacement Lists

The EffectManager holds two lists:

	em.replacements       — persistent replacements (one-shot shields, turn-scoped)
	em.cycleReplacements  — cleared each Apply() cycle, re-registered by continuous effects

Persistent replacements are checked first. Use [EffectManager.AddReplacement]
or [EffectManager.PrependReplacement] (inserts at front) for persistent ones.
Use [EffectManager.AddCycleReplacement] for cycle-scoped ones.

Continuous effects that need replacement behavior call AddCycleReplacement in
their Apply function. The EffectManager clears cycleReplacements at the start
of each Apply() cycle, so continuous effects re-register them every cycle.

At the cleanup step, [EffectManager.ClearReplacementsEndOfTurn] removes all
persistent replacements with EndOfTurn duration — this handles fog, forcefield,
prevention shields, regeneration shields, and one-shot redirects uniformly.

## Registration from Card Effects

Card effects use *Game proxy methods which internally create and register
the appropriate replacement:

	g.AddPreventionShield(playerID, amount)    → preventionShieldReplacement
	g.AddRegenerationShield(permID)            → regenerationReplacement
	g.SetPreventCombatDamage()                 → fogReplacement
	g.AddForcefieldShield(playerID)            → forcefieldReplacement
	g.AddColorPrevention(playerID, color)      → colorPreventionReplacement
	g.AddTypePrevention(playerID, cardType)    → typePreventionReplacement
	g.AddReverseDamageShield(playerID)         → reverseDamageReplacement
	g.SetCreatureDamageRedirect(cID, pID)      → creatureDamageRedirectReplacement
	g.SetAttackerDamageRedirect(aID, absID)    → attackerDamageRedirectReplacement
	g.SetSkipNextDraw(playerID)                → skipDrawReplacement
	g.SetDrawReplacement(playerID, count)      → drawReplacementEffect
	g.AddEmptyLibraryDrawReplacement(srcID,
	    playerID, callback)                    → emptyLibraryDrawReplacement
	                                              (Ormos, Archive Keeper; "if you would
	                                              draw while your library is empty")
	g.SetLichActive(playerID, sourceID)        → lichLifeGainReplacement
	g.PreventAllDamageFrom(sourceID)           → damagePreventionRuleReplacement
	g.SetMinimumLife(playerID)                 → minimumLifeReplacement (cycle)
	g.SetArtifactDamageRedirect(ctrlID, pID)   → artifactDamageRedirectReplacement (cycle)
	g.AddCounterDoubler(srcID, ct, filter)     → counterDoublerReplacement (Doubling Season,
	                                              Branching Evolution; CR 614.1c)
	g.AddETBAdditionalCounters(srcID, ct, n,
	                           filter, excludeSelf) → etbAdditionalCountersReplacement
	                                              (Oona's Blackguard, Winding Constrictor;
	                                              CR 614.1c)

For custom replacements, call [*Game.AddReplacementEffect](r) directly.

## Built-In Replacement Implementations (20)

All live in replacement.go:

	regenerationReplacement            — replaces destruction with tap + remove damage
	preventionShieldReplacement        — absorbs N damage then expires
	fogReplacement                     — prevents all combat damage
	forcefieldReplacement              — caps unblocked combat damage to 1
	colorPreventionReplacement         — prevents damage from matching color source
	typePreventionReplacement          — prevents damage from matching card type source
	reverseDamageReplacement           — prevents damage and gains life instead
	bodyguardReplacement               — redirects combat damage to bodyguard creature (cycle)
	playerDamageRedirectReplacement    — redirects all damage to creature (cycle)
	artifactDamageRedirectReplacement  — redirects artifact damage to creature (cycle)
	creatureDamageRedirectReplacement  — redirects creature damage to player
	attackerDamageRedirectReplacement  — redirects specific attacker damage to absorber
	lichLifeGainReplacement            — replaces life gain with card draw
	minimumLifeReplacement             — caps damage so life stays >= 1 (cycle)
	skipDrawReplacement                — skips next normal draw
	drawReplacementEffect              — Aladdin's Lamp draw replacement
	emptyLibraryDrawReplacement        — replaces draws from an empty library with a
	                                      card-supplied callback (Ormos, Archive Keeper)
	damagePreventionRuleReplacement    — from/to PermanentFilter-based prevention (cycle).
	                                      Supports combatOnly / noncombatOnly flags and a
	                                      toPlayerID gate for "damage dealt to <player>"
	                                      (e.g. Blessed Sanctuary). See
	                                      PreventNoncombatDamageToControllerAndCreatures.
	counterDoublerReplacement          — doubles +1/+1 (or other) counter placements on
	                                      matching permanents (CR 614.1c)
	etbAdditionalCountersReplacement   — adds N more counters when matching permanents
	                                      enter the battlefield (CR 614.1c)

## Example: Custom Replacement on a Card

	Register("Damage Halver", func() Card {
	    return NewEnchantment("Damage Halver", "{2}{W}",
	        WithStaticAbility(FuncContinuousEffect(LayerAbility, WhileOnBattlefield,
	            func(g *Game, sourceID uuid.UUID) error {
	                perm := g.FindPermanent(sourceID)
	                if perm == nil { return nil }
	                g.Effects.AddCycleReplacement(&halveDamageReplacement{
	                    replacementBase: replacementBase{sourceID: sourceID},
	                    playerID:        perm.Controller,
	                })
	                return nil
	            })),
	    )
	})

	type halveDamageReplacement struct {
	    replacementBase
	    playerID uuid.UUID
	}
	func (r *halveDamageReplacement) Matches(a Action, _ GameReader) bool {
	    act, ok := a.(*DamageToPlayerAction)
	    return ok && act.PlayerID() == r.playerID
	}
	func (r *halveDamageReplacement) Replace(a Action, _ *Game) Action {
	    act := a.(*DamageToPlayerAction)
	    return act.WithAmount(act.Amount() / 2)
	}
	func (r *halveDamageReplacement) IsActive(_ GameReader) bool { return true }

## Mutation Method Integration

The six Game methods that produce actions:

	DealDamageToPlayer  — creates DamageToPlayerAction, runs pipeline, executes via executeDamageToPlayer
	DealDamageToPermanent — creates DamageToCreatureAction, runs pipeline, executes via executeDamageToCreature
	DestroyPermanent    — creates DestroyPermanentAction, runs pipeline, removes from battlefield if not replaced
	PlayerGainLife      — creates LifeGainAction, runs pipeline, applies life gain if not replaced
	doDrawNormalDraw    — creates DrawCardAction, runs pipeline, draws card if not replaced
	AddCountersWithReplacement — creates AddCountersAction, runs pipeline, calls Permanent.AddCounter
	                              with the (possibly transformed) amount. Engine code that needs
	                              counter doubling / ETB-additional support uses this; raw
	                              Permanent.AddCounter bypasses the pipeline.

During PutOnBattlefield the entering permanent is transiently exposed via
g.enteringPermanent so FindPermanent (and therefore replacement Matches
predicates) can see it before it joins g.battlefield. EntersWithXCounters
and EntersWithNCounters route through AddCountersWithReplacement with
onEntry=true.

The executeAction dispatcher type-switches on the returned action and calls the
appropriate executor. If a replacement changes action type (e.g., redirect
DamageToPlayer to DamageToCreature), the dispatcher routes correctly.

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
	[EntersWithComputedCounters](counterType, compute) // ETB with counters computed from board state (Towering Titan)
	[CopyCreatureOnETB]()                 // clone ETB (Doppelganger)
	[ETBWithTargets](effect)              // run effect on ETB using spell targets
	[ETBEffect](effect)                   // run effect on ETB without targets
	[ETBChooseColor](reason)              // "as ~ enters, choose a color"
	[ETBChooseColorOtherThan](reason, c)  // "as ~ enters, choose a color other than X" (Thriving cycle)
	[ETBChooseOpponent]()                 // "as ~ enters, choose an opponent" (Nyxathid, Black Vise)
	[ETBChooseCreatureType](options)      // "as ~ enters, choose a creature type" (Herald's Horn)
	[ManaBonusAbility] / [NewManaBonusAbility](filter, color) // bonus mana on tap
	[NewAttachedManaBonusAbility](color)  // Wild Growth bonus mana

# "As ~ Enters the Battlefield, Choose ___" (CR 614.12)

An "as it enters" replacement effect makes the controller pick a value at
the moment a permanent enters the battlefield. The choice is recorded on
the permanent itself, so other abilities of the same permanent can read it
later. The four ETBChoose* constructors above each produce an
*ETBEffectAbility that PutOnBattlefield runs unconditionally during ETB
resolution BEFORE firing the EvtZoneChange (To=Battlefield) event — so the stored
choice is already in place when "when this enters" triggers fire.

Storage fields on Permanent:

	ChosenColor    Color   // ETBChooseColor / ETBChooseColorOtherThan
	ChosenPlayer   uuid.UUID // ETBChooseOpponent
	ChosenSubtype  string  // ETBChooseCreatureType

Example: a Thriving-cycle land that enters tapped and taps for either its
own color or the chosen color.

	mage.Register("Thriving Bluff", func() mage.Card {
	    return mage.NewLand("Thriving Bluff",
	        mage.WithKeyword(core.EntersTapped),
	        mage.WithAbility(mage.ETBChooseColorOtherThan(
	            "choose a color other than red", core.Red,
	        )),
	        mage.WithActivatedAbility(
	            // Custom mana ability reads perm.ChosenColor at activation time.
	            mage.FuncEffect("add {R} or chosen color", mage.EffectProperties{},
	                func(g *mage.Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
	                    perm := g.FindPermanent(sourceID)
	                    p := g.GetPlayer(controller)
	                    // ... add core.Red or perm.ChosenColor based on player choice
	                    return nil
	                }),
	            mage.Tap(),
	        ),
	    )
	})

Test harness scripting:

	tg.ChooseManaColor(PlayerA, core.Blue)   // for ETBChooseColor / ETBChooseColorOtherThan
	tg.ChooseString(PlayerA, "Goblin")       // for ETBChooseCreatureType

ETBChooseOpponent does not consult the player in 2-player; it pins
ChosenPlayer to the lone opponent at ETB-replacement time so downstream
references (Nyxathid's P/T-by-hand-size, ChosenPlayerUpkeepTrigger, etc.)
remain stable.

# Mana

Mana abilities are a special ability type:

	[NewManaAbility](color)          // tap for one mana of color
	[NewMultiManaAbility](prods...)  // custom productions (Sol Ring, dual mana, etc.)
	[NewManaAbility](AnyColor)       // tap for any color

Mana costs are parsed from strings like "{2}{W}{B}" via [ParseManaCost].
Colors: [White], [Blue], [Black], [Red], [Green], [Colorless].

Hybrid mana symbols (CR 107.4d, 117.7) are written {X/Y} where X and Y are
two of W/U/B/R/G — for example "{1}{W/U}{G/W}". A hybrid symbol can be paid
with mana of either listed color; the engine picks deterministically (the
more-abundant color in the pool, breaking ties toward the first listed
color). Each hybrid symbol contributes 1 to the cost's mana value (CR 202.3f)
and counts as both of its colors for color identity (CR 202.2c). Stored on
[ManaCost] as the [ManaCost.Hybrid] slice of [HybridSymbol]{A, B}.

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
	            mage.Tap(),
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
	                mage.AddCounters(core.P1P1, mage.Fixed(1)).Targeting(mage.ToSource()),
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
	        mage.DealDamage(mage.Fixed(3)),
	        mage.WithTarget(mage.TargetAnyTarget()),
	    )
	})

Untargeted sorcery:

	mage.Register("Wrath of God", func() mage.Card {
	    return mage.NewSorcery("Wrath of God", "{2}{W}{W}",
	        mage.DestroyAllCreatures(),
	    )
	})

X spell:

	mage.Register("Fireball", func() mage.Card {
	    return mage.NewSorcery("Fireball", "{X}{R}",
	        mage.DealDamage(mage.XValue()),
	        mage.WithTarget(mage.TargetAnyTarget()),
	    )
	})

Counterspell:

	mage.Register("Counterspell", func() mage.Card {
	    return mage.NewInstant("Counterspell", "{U}{U}",
	        mage.CounterSpell(),
	        mage.WithTarget(mage.TargetSpellOnStack()),
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
	            func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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

# Custom-keyword support: Secrets of Strixhaven (keyword_sos.go)

The Secrets of Strixhaven set introduces seven set-specific keywords that
ship as small composable helpers in keyword_sos.go (no engine-wide
constants beyond AttrPrepared):

	Prepared:
	    [WithPreparedSpell](spellFactory func() Card) CardOption
	    Game.IsPrepared(permID) bool
	    Game.SetPrepared(permID, prepared bool)
	    Game.CastPreparedSpellCopy(playerID, permID, spellFactory) error
	    AttrPrepared (core.Attr; in keyword range)
	    HasPreparedSpell(card) bool

	    The CardOption installs:
	      1) An ETB trigger that calls SetPrepared(self, true).
	      2) A free, sorcery-speed activated ability gated on
	         IsPrepared(self) that calls CastPreparedSpellCopy. The cast
	         pushes a freshly built copy of `spellFactory()` onto the
	         stack with IsCopy=true, prompts for any declared targets,
	         fires EvtSpellCast, then calls SetPrepared(self, false).

	Repartee:
	    [WheneverYouCastInstantOrSorceryTargetingCreatureTrigger](effect, optional)
	          *GenericTriggered

	    Fires only when the controller casts an instant/sorcery whose
	    declared targets include at least one creature on the battlefield.

	Opus / Increment ("amount of mana spent to cast"):
	    [ManaSpentToCast](*StackObject) int           — sums ColorsSpent
	    [ManaSpentForSpellEvent](evt, GameReader) int  — for trigger predicates
	    [OpusEffect](text, fn(g, src, ctrl, manaSpent)) Effect
	    [IncrementTrigger]() *GenericTriggered

	    OpusEffect wraps a closure that receives the total mana the
	    controller spent to cast the triggering spell. Inside an
	    EvtSpellCast trigger the helper reads the topmost non-ability
	    StackObject's CastContext.ColorsSpent.

	    IncrementTrigger() returns the standard Increment cast-trigger:
	    "Whenever you cast a spell, if the amount of mana you spent is
	    greater than this creature's power or toughness, put a +1/+1
	    counter on this creature."

	Infusion ("if you gained life this turn"):
	    [IfControllerGainedLifeThisTurn](GameReader, controller) bool
	    [LifeGainedThisTurnFor](GameReader, controller) int
	    [InfusionEffect](text, inner Effect) Effect

	    InfusionEffect wraps `inner` so it only resolves when the
	    controller has gained at least one life this turn. Backed by the
	    existing PlayerLifeGainedThisTurn tracker — life-gain effects
	    that fire EvtLifeGained populate the count automatically.

	Grandeur ("Discard another card with the same name as this"):
	    [DiscardAnotherCardNamedSelfCost]() Cost

	    A Cost suitable for an activated ability: payable iff the
	    controller's hand contains a card with the source's name that
	    isn't the source itself.

	Paradigm ("After you first resolve a spell with this name, …"):
	    Game.RecordParadigmResolution(playerID, name)
	    Game.HasResolvedParadigmSpell(playerID, name) bool
	    Game.RegisterParadigmExiledCopy(playerID, name, cardID)
	    Game.ParadigmExiledCopy(playerID, name) (uuid.UUID, bool)

	    These are per-game state hooks: a Paradigm spell, after first
	    resolving, calls RecordParadigmResolution + (after exile)
	    RegisterParadigmExiledCopy. A main-phase trigger (defined on the
	    card) consults HasResolvedParadigmSpell and ParadigmExiledCopy to
	    decide whether to offer the recurring free-cast.

# Custom Per-Game State

For set-specific state that doesn't fit any existing Game field, the engine
provides a per-game string-keyed bag (Game.customState) accessed through
typed helpers in the relevant keyword file (e.g. paradigmStateOf in
keyword_sos.go). The bag is allocated by NewGame and survives the lifetime
of the game.
*/
package mage
