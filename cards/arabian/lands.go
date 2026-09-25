package arabian

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func init() {
	registerLands()
}

func registerLands() {
	// Oracle: "{T}: Draw two cards, then discard three cards."
	Register("Bazaar of Baghdad", func() Card {
		return NewLand("Bazaar of Baghdad",
			WithAbility(Activate(
				Costs(TapCost(This)),
				Sequence(Draw(You, Fixed(2)), Discard(You, Fixed(3))),
			)),
		)
	})

	// Oracle: "Whenever City of Brass becomes tapped, it deals 1 damage to you.
	// {T}: Add one mana of any color."
	Register("City of Brass", func() Card {
		return NewLand("City of Brass",
			WithAnyColorMana(),
			WithAbility(
				NewTriggered(EvtTapped, false,
					DealDamageToPlayers(Fixed(1), SelectController()),
				).SetConditionData(EventSourceIsSelf{}),
			),
		)
	})

	// {T}: Add {C}.
	// {T}: This land deals 1 damage to target attacking creature. Activate only during the end of combat step.
	Register("Desert", func() Card {
		return NewLand("Desert",
			WithSubTypes("Desert"),
			WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Colorless, Fixed(1)))),
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				TapCost(This),
				WithTarget(TargetCreature(IsAttacking)),
				WithStepOnly(EndCombat),
			),
		)
	})

	// Oracle: "{T}, Sacrifice a creature: You gain life equal to the sacrificed
	// creature's toughness."
	Register("Diamond Valley", func() Card {
		return NewLand("Diamond Valley",
			WithActivatedAbility(
				FuncEffect("gain life equal to the sacrificed creature's toughness",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						sacrificed := g.LastSacrificed()
						if sacrificed == nil {
							return nil
						}
						g.PlayerGainLife(p, sacrificed.Toughness)
						return nil
					}),
				TapCost(This),
				WithCost(SacrificeCreatureCost()),
			),
		)
	})

	// {T}: Add {C}.
	// {T}: Regenerate target Elephant.
	Register("Elephant Graveyard", func() Card {
		elephant := TargetPermanent("elephant", HasSubType("Elephant"))
		return NewLand("Elephant Graveyard",
			WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Colorless, Fixed(1)))),
			WithAbility(Activate(
				Costs(TapCost(This)),
				Regenerate(elephant),
				WithTargets(elephant),
			)),
		)
	})

	// Oracle: "{T}: Target creature with flying has base power 0 until end of turn."
	Register("Island of Wak-Wak", func() Card {
		creature := TargetPermanent("creature", And(IsCreature, HasKeywordFilter(Flying)))
		return NewLand("Island of Wak-Wak",
			WithAbility(Activate(Costs(TapCost(This)),
				Until(EndOfTurn, SetBasePower(creature, Fixed(0))),
				WithTargets(creature))),
		)
	})

	// {T}: Add {C}.
	// {T}: Draw a card. Activate only if you have exactly seven cards in hand.
	Register("Library of Alexandria", func() Card {
		return NewLand("Library of Alexandria",
			WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Colorless, Fixed(1)))),
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				TapCost(This),
				mage.WithActivationCondition(func(g *Game, _ *Permanent, controller uuid.UUID) bool {
					p := g.GetPlayer(controller)
					return p != nil && len(p.Hand()) == 7
				}),
			),
		)
	})

	// Oracle: "{T}: Prevent the next 1 damage that would be dealt to target creature
	// this turn."
	Register("Oasis", func() Card {
		creature := TargetPermanent("creature", IsCreature)
		return NewLand("Oasis",
			WithAbility(Activate(Costs(TapCost(This)), PreventNextDamage(creature, Fixed(1)),
				WithTargets(creature))),
		)
	})
}
