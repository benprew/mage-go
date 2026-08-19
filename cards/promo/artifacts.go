package promo

import (
	. "github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
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
			WithAbility(BeginningOfUpkeepTrigger(
				IfElse("flip a coin",
					FlipCoinCond{},
					nil,
					DealDamageToPlayers(Fixed(3), SelectController()),
				), false,
			)),
			WithMultiManaAbility(ManaProduction{Color: core.Colorless, Amount: 2}),
		)
	})

}
