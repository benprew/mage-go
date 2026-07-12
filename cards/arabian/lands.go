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
			WithActivatedAbility(
				CompositeEffects(
					"draw 2, discard 3",
					DrawCards(Fixed(2)),
					DiscardCards(Fixed(3)).Targeting(SelectController()),
				),
				Tap(),
			),
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

	// Oracle: "{T}: Add {C}. {T}: Desert deals 1 damage to target attacking creature.
	// Activate only during the end of combat step."
	Register("Desert", func() Card {
		return NewLand("Desert",
			WithSubTypes("Desert"),
			WithManaAbility(Colorless),
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				Tap(),
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
				Tap(),
				WithCost(SacrificeCreatureCost()),
			),
		)
	})

	// Oracle: "{T}: Add {C}. {T}: Regenerate target Elephant."
	Register("Elephant Graveyard", func() Card {
		return NewLand("Elephant Graveyard",
			WithManaAbility(Colorless),
			WithActivatedAbility(
				RegenerateTarget(),
				Tap(),
				WithTarget(TargetCreature(HasSubType("Elephant"))),
			),
		)
	})

	// Oracle: "{T}: Target creature with flying has base power 0 until end of turn."
	Register("Island of Wak-Wak", func() Card {
		return NewLand("Island of Wak-Wak",
			WithActivatedAbility(
				SetPowerUntilEndOfTurn(0, SelectTarget),
				Tap(),
				WithTarget(TargetCreature(HasKeywordFilter(Flying))),
			),
		)
	})

	// Oracle: "{T}: Add {C}. {T}: Draw a card. Activate only if you have exactly
	// seven cards in hand."
	Register("Library of Alexandria", func() Card {
		return NewLand("Library of Alexandria",
			WithManaAbility(Colorless),
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				Tap(),
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
		return NewLand("Oasis",
			WithActivatedAbility(
				PreventDamageToTarget(Fixed(1)),
				Tap(),
				WithTarget(TargetCreature()),
			),
		)
	})
}
