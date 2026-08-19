package fallen_empires

import (
	"math/rand"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func init() {
	registerArtifacts()
}

func registerArtifacts() {
	// Aeolipile {2}
	// Artifact
	// {1}, {T}, Sacrifice this artifact: It deals 2 damage to any target.
	Register("Aeolipile", withExpansion(func() Card {
		return NewArtifact("Aeolipile", "{2}",
			WithActivatedAbility(
				DealDamage(Fixed(2)),
				GenericCost(1),
				WithCost(Tap()),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetDamageAnyTarget()),
			),
		)
	}))

	// Balm of Restoration {2}
	// Artifact
	// {1}, {T}, Sacrifice this artifact: Choose one —
	// • You gain 2 life.
	// • Prevent the next 2 damage that would be dealt to any target this turn.
	Register("Balm of Restoration", withExpansion(func() Card {
		c := NewArtifact("Balm of Restoration", "{2}")
		c.AddAbility(NewModalActivated(GenericCost(1), []Mode{
			{
				Label:   "You gain 2 life",
				Effects: []Effect{GainLife(2)},
			},
			{
				Label:   "Prevent the next 2 damage that would be dealt to any target this turn",
				Targets: []Target{TargetDamageAnyTarget()},
				Effects: []Effect{PreventDamageToTarget(Fixed(2))},
			},
		}, WithCost(Tap()), WithCost(SacrificeSourceCost())))
		return c
	}))

	// Conch Horn {2}
	// Artifact
	// {1}, {T}, Sacrifice this artifact: Draw two cards, then put a card from your hand on top of your library.
	// TODO: convert to pipeline — needs DrawCards step + ChooseCardsFromHand + PutOnTopOfLibrary steps
	Register("Conch Horn", withExpansion(func() Card {
		return NewArtifact("Conch Horn", "{2}",
			WithActivatedAbility(
				FuncEffect("draw two cards, then put a card from your hand on top of your library",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						// Draw two cards
						g.PlayerDrawCard(p)
						g.PlayerDrawCard(p)
						// Put a card from hand on top of library
						hand := p.Hand()
						if len(hand) == 0 {
							return nil
						}
						chosen := p.ChooseCardsFromHand(1, "put a card on top of your library", g)
						if len(chosen) > 0 {
							p.RemoveFromHand(chosen[0].ID())
							lib := p.Library()
							newLib := make([]Card, 0, len(lib)+1)
							newLib = append(newLib, chosen[0])
							newLib = append(newLib, lib...)
							p.SetLibrary(newLib)
						}
						return nil
					}),
				GenericCost(1),
				WithCost(Tap()),
				WithCost(SacrificeSourceCost()),
			),
		)
	}))

	// Delif's Cone {0}
	// Artifact
	// {T}, Sacrifice this artifact: This turn, when target creature you control attacks and isn't blocked, you may gain life equal to its power. If you do, it assigns no combat damage this turn.
	// TODO: implement
	Register("Delif's Cone", withExpansion(func() Card {
		return NewArtifact("Delif's Cone", "{0}")
	}))

	// Delif's Cube {1}
	// Artifact
	// {2}, {T}: This turn, when target creature you control attacks and isn't blocked, it assigns no combat damage this turn and you put a cube counter on this artifact.
	// {2}, Remove a cube counter from this artifact: Regenerate target creature.
	// TODO: implement
	Register("Delif's Cube", withExpansion(func() Card {
		return NewArtifact("Delif's Cube", "{1}")
	}))

	// Draconian Cylix {3}
	// Artifact
	// {2}, {T}, Discard a card at random: Regenerate target creature.
	Register("Draconian Cylix", withExpansion(func() Card {
		return NewArtifact("Draconian Cylix", "{3}",
			WithActivatedAbility(
				RegenerateTarget(),
				GenericCost(2),
				WithCost(Tap()),
				WithCost(DiscardRandomCost(1)),
				WithTarget(TargetCreature()),
			),
		)
	}))

	// Elven Lyre {2}
	// Artifact
	// {1}, {T}, Sacrifice this artifact: Target creature gets +2/+2 until end of turn.
	Register("Elven Lyre", withExpansion(func() Card {
		return NewArtifact("Elven Lyre", "{2}",
			WithActivatedAbility(
				Boost(Fixed(2), Fixed(2)),
				GenericCost(1),
				WithCost(Tap()),
				WithCost(SacrificeSourceCost()),
				WithTarget(TargetCreature()),
			),
		)
	}))

	// Implements of Sacrifice {2}
	// Artifact
	// {1}, {T}, Sacrifice this artifact: Add two mana of any one color.
	Register("Implements of Sacrifice", withExpansion(func() Card {
		return NewArtifact("Implements of Sacrifice", "{2}",
			WithActivatedAbility(
				AddAnyMana(2, core.Colorless),
				GenericCost(1),
				WithCost(Tap()),
				WithCost(SacrificeSourceCost()),
			),
		)
	}))

	// Ring of Renewal {5}
	// Artifact
	// {5}, {T}: Discard a card at random, then draw two cards.
	// TODO: convert to pipeline — needs DiscardRandom targeting controller (not opponent) + DrawCards step
	Register("Ring of Renewal", withExpansion(func() Card {
		return NewArtifact("Ring of Renewal", "{5}",
			WithActivatedAbility(
				FuncEffect(
					"discard a card at random, then draw two cards",
					EffectProperties{Outcome: OutcomeBenefit, DrawCount: 2},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						p := g.GetPlayer(controller)
						if p == nil {
							return nil
						}
						// Discard a card at random
						hand := p.Hand()
						if len(hand) > 0 {
							idx := rand.Intn(len(hand))
							p.DiscardCard(hand[idx].ID())
						}
						// Then draw two cards
						for range 2 {
							g.PlayerDrawCard(p)
						}
						return nil
					},
				),
				GenericCost(5),
				WithCost(Tap()),
			),
		)
	}))

	// Spirit Shield {3}
	// Artifact
	// You may choose not to untap this artifact during your untap step.
	// {2}, {T}: Target creature gets +0/+2 for as long as this artifact remains tapped.
	Register("Spirit Shield", withExpansion(func() Card {
		return NewArtifact("Spirit Shield", "{3}",
			WithKeyword(AttrMayNotUntap),
			WithActivatedAbility(
				Boost(Fixed(0), Fixed(2)).
					Targeting(ToTarget()).
					Until(WhileOnBattlefield).
					WhileSourceTapped(),
				GenericCost(2),
				WithCost(Tap()),
				WithTarget(TargetCreature()),
			),
		)
	}))

	// Zelyon Sword {3}
	// Artifact
	// You may choose not to untap this artifact during your untap step.
	// {3}, {T}: Target creature gets +2/+0 for as long as this artifact remains tapped.
	Register("Zelyon Sword", withExpansion(func() Card {
		return NewArtifact("Zelyon Sword", "{3}",
			WithKeyword(AttrMayNotUntap),
			WithActivatedAbility(
				Boost(Fixed(2), Fixed(0)).
					Targeting(ToTarget()).
					Until(WhileOnBattlefield).
					WhileSourceTapped(),
				GenericCost(3),
				WithCost(Tap()),
				WithTarget(TargetCreature()),
			),
		)
	}))

}
