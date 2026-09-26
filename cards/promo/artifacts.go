package promo

import (
	"github.com/benprew/mage-go/pkg/mage/core"
	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func init() {
	registerArtifacts()
}

func registerArtifacts() {
	// Mana Crypt {0}
	// Artifact
	// At the beginning of your upkeep, flip a coin. If you lose the flip, this artifact deals 3 damage to you.
	// {T}: Add {C}{C}.
	Register("Mana Crypt", func() Card {
		return NewArtifact("Mana Crypt", "{0}",
			WithAbility(At(YourUpkeep,
				FlipCoin(You,
					OnLoss(DealDamageToPlayers(Fixed(3), SelectController())),
				),
			)),
			WithAbility(Activate(
				Costs(TapCost(This)),
				AddManaAmount(core.Colorless, Fixed(2)),
			)),
		)
	})

}
