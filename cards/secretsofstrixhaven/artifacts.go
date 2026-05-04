package secretsofstrixhaven

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

func init() {
	registerArtifacts()
}

func registerArtifacts() {

// Ark of Hunger {2}{R}{W}
// Artifact
// Whenever one or more cards leave your graveyard, this artifact deals 1 damage to each opponent and you gain 1 life.
// {T}: Mill a card. You may play that card this turn.
	Register("Ark of Hunger", func() Card {
		// XXX: "Whenever one or more cards leave your graveyard" — the engine
		// has no EvtLeaveGraveyard event, so the triggered ability cannot be
		// implemented.
		// XXX: "You may play that card this turn" — graveyard play permissions
		// are not supported in the engine (only exile via GrantCastFromExile).
		// The tap ability mills one card from the controller's library only.
		millOne := FuncEffect(
			"mill a card",
			EffectProperties{Outcome: OutcomeUnknown},
			func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
				p := g.GetPlayer(controller)
				if p == nil {
					return nil
				}
				lib := p.Library()
				if len(lib) == 0 {
					return nil
				}
				card := lib[len(lib)-1]
				p.SetLibrary(lib[:len(lib)-1])
				p.AddToGraveyard(card)
				return nil
			},
		)
		return NewArtifact("Ark of Hunger", "{2}{R}{W}",
			WithActivatedAbility(millOne, TapSourceCost()),
		)
	})


// Cauldron of Essence {1}{B}{G}
// Artifact
// Whenever a creature you control dies, each opponent loses 1 life and you gain 1 life.
// {1}{B}{G}, {T}, Sacrifice a creature: Return target creature card from your graveyard to the battlefield. Activate only as a sorcery.
	Register("Cauldron of Essence", func() Card {
		drainEffect := FuncEffect(
			"each opponent loses 1 life and you gain 1 life",
			EffectProperties{Outcome: OutcomeBenefit},
			func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
				opp := g.GetOpponent(controller)
				if opp != nil {
					g.PlayerLoseLife(opp, 1)
				}
				ctrl := g.GetPlayer(controller)
				if ctrl != nil {
					g.PlayerGainLife(ctrl, 1)
				}
				return nil
			},
		)
		return NewArtifact("Cauldron of Essence", "{1}{B}{G}",
			WithAbility(
				AnyCreatureDiesTrigger(drainEffect, false).
					AndConditionData(EventPlayerIsController{}),
			),
			WithActivatedAbility(
				ReturnFromGraveyardToBattlefield(),
				ManaCostOf("{1}{B}{G}"),
				WithCost(TapSourceCost()),
				WithCost(SacrificeCreatureCost()),
				WithTarget(TargetCreatureInYourGraveyard()),
				WithSorcerySpeed(),
			),
		)
	})


// Diary of Dreams {2}
// Artifact — Book
// Whenever you cast an instant or sorcery spell, put a page counter on this artifact.
// {5}, {T}: Draw a card. This ability costs {1} less to activate for each page counter on this artifact.
	Register("Diary of Dreams", func() Card {
		// XXX: "page counter" — the engine's core/counter.go has no Page counter type.
		// Charge counters are used as a substitute. All functional behavior is preserved,
		// but AssertCounterCount will show Charge counters, not page counters.
		counterTrigger := WheneverYouCastSpellTrigger(
			AddCounters(Charge, Fixed(1)).Targeting(ToSource()),
			false, IsInstantOrSorceryCard,
		)
		// Continuous effect: reduce the activation cost by 1 for each charge
		// counter on this artifact. ActivationCostReductions is reset each
		// Apply() cycle, so the FuncContinuousEffect re-registers it every cycle.
		costReduction := FuncContinuousEffect(LayerAbility, WhileOnBattlefield,
			func(g *Game, sourceID uuid.UUID) error {
				src := g.FindPermanent(sourceID)
				if src == nil {
					return nil
				}
				n := int(src.Counters[Charge])
				if n > 0 {
					g.AddActivationCostReduction(sourceID, n)
				}
				return nil
			},
		)
		return NewArtifact("Diary of Dreams", "{2}",
			WithSubTypes("Book"),
			WithAbility(counterTrigger),
			WithStaticAbility(costReduction),
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				GenericCost(5),
				WithCost(TapSourceCost()),
			),
		)
	})


// Potioner's Trove {3}
// Artifact
// {T}: Add one mana of any color.
// {T}: You gain 2 life. Activate only if you've cast an instant or sorcery spell this turn.
	Register("Potioner's Trove", func() Card {
		lifeGainCondition := func(g *Game, src *Permanent, controller uuid.UUID) bool {
			return g.GetInstantOrSorceryCastThisTurn(controller) > 0
		}
		return NewArtifact("Potioner's Trove", "{3}",
			WithAnyColorMana(),
			WithActivatedAbility(
				GainLife(2),
				TapSourceCost(),
				WithActivationCondition(lifeGainCondition),
			),
		)
	})


// Resonating Lute {2}{U}{R}
// Artifact
// Lands you control have "{T}: Add two mana of any one color. Spend this mana only to cast instant and sorcery spells."
// {T}: Draw a card. Activate only if you have seven or more cards in your hand.
	Register("Resonating Lute", func() Card {
		// XXX: "Lands you control have '{T}: Add two mana of any one color.
		// Spend this mana only to cast instant and sorcery spells.'" —
		// GrantActivatedAbilityToAll only works for creatures, not lands.
		// Additionally, mana spend restrictions are not supported in the engine.
		// The land mana ability grant is not implemented.
		drawCondition := func(g *Game, src *Permanent, controller uuid.UUID) bool {
			p := g.GetPlayer(controller)
			if p == nil {
				return false
			}
			return len(p.Hand()) >= 7
		}
		return NewArtifact("Resonating Lute", "{2}{U}{R}",
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				TapSourceCost(),
				WithActivationCondition(drawCondition),
			),
		)
	})


// Strixhaven Skycoach {3}
// Artifact — Vehicle
// 3/2
// Flying
// When this Vehicle enters, you may search your library for a basic land card, reveal it, put it into your hand, then shuffle.
// Crew 2 (Tap any number of creatures you control with total power 2 or more: This Vehicle becomes an artifact creature until end of turn.)
	Register("Strixhaven Skycoach", func() Card {
		// XXX: Vehicle/Crew mechanics are not implemented in the engine.
		// The 3/2 P/T and Crew 2 ability are not modeled.
		// XXX: "reveal it" — reveal to opponent is not modeled; the card just
		// goes to hand.
		searchForBasicLand := FuncEffect(
			"search your library for a basic land card, put it into your hand, then shuffle",
			EffectProperties{Outcome: OutcomeBenefit},
			func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
				searchBasicLandToHandSOS(g, controller)
				return nil
			},
		)
		return NewArtifact("Strixhaven Skycoach", "{3}",
			WithSubTypes("Vehicle"),
			WithKeyword(Flying),
			WithAbility(EntersBattlefieldTrigger(searchForBasicLand, true)),
		)
	})


// Tablet of Discovery {2}{R}
// Artifact
// When this artifact enters, mill a card. You may play that card this turn. (To mill a card, put the top card of your library into your graveyard.)
// {T}: Add {R}.
// {T}: Add {R}{R}. Spend this mana only to cast instant and sorcery spells.
	Register("Tablet of Discovery", func() Card {
		// XXX: "You may play that card this turn" — graveyard play permissions
		// are not supported in the engine (only exile via GrantCastFromExile).
		// XXX: "Spend this mana only to cast instant and sorcery spells" —
		// mana spend restrictions are not supported in the engine.
		millSelf := FuncEffect(
			"mill a card",
			EffectProperties{Outcome: OutcomeUnknown},
			func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
				p := g.GetPlayer(controller)
				if p == nil {
					return nil
				}
				lib := p.Library()
				if len(lib) == 0 {
					return nil
				}
				card := lib[len(lib)-1]
				p.SetLibrary(lib[:len(lib)-1])
				p.AddToGraveyard(card)
				return nil
			},
		)
		return NewArtifact("Tablet of Discovery", "{2}{R}",
			WithAbility(EntersBattlefieldTrigger(millSelf, false)),
			WithManaAbility(Red),
			WithAbility(NewMultiManaAbility(ManaProduction{Color: Red, Amount: 2})),
		)
	})

}
