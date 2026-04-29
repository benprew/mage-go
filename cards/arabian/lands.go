package arabian

import (
	"fmt"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/dsl"
)

// exactHandSizeCost is a zero-cost activation condition: the ability can only
// be activated when the controller has exactly `size` cards in hand.
type exactHandSizeCost struct{ size int }

func (c *exactHandSizeCost) CanPay(_, controller uuid.UUID, g *Game) bool {
	p := g.GetPlayer(controller)
	return p != nil && len(p.Hand()) == c.size
}
func (c *exactHandSizeCost) Pay(_, _ uuid.UUID, _ *Game) error { return nil }
func (c *exactHandSizeCost) Text() string {
	return fmt.Sprintf("Activate only if you have exactly %d cards in hand", c.size)
}

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
				TapSourceCost(),
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
				TapSourceCost(),
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
				// TODO: convert to pipeline — needs ChoosePermanentStep with controller-relative filter
				FuncEffect("sacrifice creature, gain life equal to toughness",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
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
	})

	// Oracle: "{T}: Add {C}. {T}: Regenerate target Elephant."
	Register("Elephant Graveyard", func() Card {
		return NewLand("Elephant Graveyard",
			WithManaAbility(Colorless),
			WithActivatedAbility(
				RegenerateTarget(),
				TapSourceCost(),
				WithTarget(TargetCreature(HasSubType("Elephant"))),
			),
		)
	})

	// Oracle: "{T}: Target creature with flying has base power 0 until end of turn."
	Register("Island of Wak-Wak", func() Card {
		return NewLand("Island of Wak-Wak",
			WithActivatedAbility(
				SetPowerUntilEndOfTurn(0, SelectTarget),
				TapSourceCost(),
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
				TapSourceCost(),
				WithCost(&exactHandSizeCost{size: 7}),
			),
		)
	})

	// Oracle: "{T}: Prevent the next 1 damage that would be dealt to target creature
	// this turn."
	Register("Oasis", func() Card {
		return NewLand("Oasis",
			WithActivatedAbility(
				PreventDamageToTarget(Fixed(1)),
				TapSourceCost(),
				WithTarget(TargetCreature()),
			),
		)
	})
}
