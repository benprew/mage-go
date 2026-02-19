package arabian

import (
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerLands()
}

func registerLands() {
	mage.Register("Bazaar of Baghdad", func() mage.Card {
		return mage.NewLand("Bazaar of Baghdad")
	})

	mage.Register("City of Brass", func() mage.Card {
		return mage.NewLand("City of Brass")
	})

	mage.Register("Desert", func() mage.Card {
		return mage.NewLand("Desert",
			mage.WithSubTypes("Desert"),
			mage.WithManaAbility(core.Colorless),
			mage.WithAbility(mage.NewActivatedAbility(
				mage.DealDamage(mage.Fixed(1)),
				mage.TapSourceCost(),
				mage.WithTarget(mage.TargetCreature(mage.IsAttacking)),
			)),
		)
	})

	mage.Register("Diamond Valley", func() mage.Card {
		return mage.NewLand("Diamond Valley")
	})

	mage.Register("Elephant Graveyard", func() mage.Card {
		return mage.NewLand("Elephant Graveyard")
	})

	mage.Register("Island of Wak-Wak", func() mage.Card {
		return mage.NewLand("Island of Wak-Wak")
	})

	mage.Register("Library of Alexandria", func() mage.Card {
		return mage.NewLand("Library of Alexandria")
	})

	mage.Register("Oasis", func() mage.Card {
		return mage.NewLand("Oasis")
	})
}
