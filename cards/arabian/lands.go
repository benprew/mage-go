package arabian

import (
	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerLands()
}

func registerLands() {
	// Oracle: "{T}: Draw two cards, then discard three cards."
	Register("Bazaar of Baghdad", withExpansion(func() Card {
		return NewLand("Bazaar of Baghdad",
			WithActivatedAbility(
				FuncEffect("draw 2, discard 3",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						p.DrawCard()
						p.DrawCard()
						chosen := p.ChooseCardsFromHand(3, "discard", g)
						for _, card := range chosen {
							p.RemoveFromHand(card.ID())
							p.AddToGraveyard(card)
						}
						return nil
					}),
				TapSourceCost(),
			),
		)
	}))

	// Oracle: "Whenever City of Brass becomes tapped, it deals 1 damage to you.
	// {T}: Add one mana of any color."
	Register("City of Brass", withExpansion(func() Card {
		return NewLand("City of Brass",
			WithAnyColorMana(),
			WithAbility(
				NewTriggered(EvtTapped, false,
					DealDamageToPlayers(Fixed(1), SelectController()),
				).SetCondition(func(evt *GameEvent, _ *Game, sourceID, _ uuid.UUID) bool {
					return evt.SourceID == sourceID
				}),
			),
		)
	}))

	// Oracle: "{T}: Add {C}. {T}: Desert deals 1 damage to target attacking creature.
	// Activate only during the end of combat step."
	Register("Desert", withExpansion(func() Card {
		return NewLand("Desert",
			WithSubTypes("Desert"),
			WithManaAbility(Colorless),
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				TapSourceCost(),
				WithTarget(TargetCreature(IsAttacking)),
			),
		)
	}))

	// Oracle: "{T}, Sacrifice a creature: You gain life equal to the sacrificed
	// creature's toughness."
	Register("Diamond Valley", withExpansion(func() Card {
		return NewLand("Diamond Valley",
			WithActivatedAbility(
				FuncEffect("sacrifice creature, gain life equal to toughness",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						var creatures []*Permanent
						for _, perm := range g.FilterBattlefield(And(IsCreature, ControlledBy(controller))) {
							if perm.ID() != sourceID {
								creatures = append(creatures, perm)
							}
						}
						if len(creatures) == 0 {
							return nil
						}
						chosen := p.ChoosePermanent(creatures, "sacrifice", g)
						if chosen == nil {
							return nil
						}
						toughness := chosen.CurrentToughness(g)
						g.Sacrifice(chosen)
						g.PlayerGainLife(p, toughness)
						return nil
					}),
				TapSourceCost(),
			),
		)
	}))

	// Oracle: "{T}: Add {C}. {T}: Regenerate target Elephant."
	Register("Elephant Graveyard", withExpansion(func() Card {
		return NewLand("Elephant Graveyard",
			WithManaAbility(Colorless),
			WithActivatedAbility(
				RegenerateTarget(),
				TapSourceCost(),
				WithTarget(TargetCreature(HasSubType("Elephant"))),
			),
		)
	}))

	// Oracle: "{T}: Target creature with flying has base power 0 until end of turn."
	Register("Island of Wak-Wak", withExpansion(func() Card {
		return NewLand("Island of Wak-Wak",
			WithActivatedAbility(
				SetPowerUntilEndOfTurn(0, SelectTarget),
				TapSourceCost(),
				WithTarget(TargetCreature(HasKeywordFilter(Flying))),
			),
		)
	}))

	// Oracle: "{T}: Add {C}. {T}: Draw a card. Activate only if you have exactly
	// seven cards in hand."
	Register("Library of Alexandria", withExpansion(func() Card {
		return NewLand("Library of Alexandria",
			WithManaAbility(Colorless),
			WithActivatedAbility(
				FuncEffect("draw a card (if 7 in hand)",
					EffectProperties{Outcome: OutcomeBenefit, DrawCount: 1},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						if len(p.Hand()) != 7 {
							return nil
						}
						p.DrawCard()
						return nil
					}),
				TapSourceCost(),
			),
		)
	}))

	// Oracle: "{T}: Prevent the next 1 damage that would be dealt to target creature
	// this turn."
	Register("Oasis", withExpansion(func() Card {
		return NewLand("Oasis",
			WithActivatedAbility(
				PreventDamageToTarget(Fixed(1)),
				TapSourceCost(),
				WithTarget(TargetCreature()),
			),
		)
	}))
}
