package custom

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
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
				ModalEffect("deal 1 damage or gain 1 life",
					DealDamageToPlayersStep(Fixed(1), SelectEachOpponent()),
					GainLifeStep(1),
				),
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
