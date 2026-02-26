package custom

import (
	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
)

func init() {
	registerModal()
}

func registerModal() {
	// Test card: modal activated ability with two modes.
	// Mode 1: Deal 1 damage to target opponent.
	// Mode 2: You gain 1 life.
	Register("Modal Test Artifact", func() Card {
		c := NewArtifact("Modal Test Artifact", "{1}",
			WithActivatedAbility(
				FuncEffect("deal 1 damage or gain 1 life",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						if g.ModeValue() == 0 {
							opp := g.GetOpponent(controller)
							if opp != nil {
								g.DealDamageToPlayer(opp, 1, sourceID)
							}
						} else {
							p := g.GetPlayer(controller)
							if p != nil {
								g.PlayerGainLife(p, 1)
							}
						}
						return nil
					}),
				ManaCostOf("{1}"),
			),
		)
		c.SetModes([]string{
			"Deal 1 damage to target opponent",
			"You gain 1 life",
		})
		return c
	})
}
