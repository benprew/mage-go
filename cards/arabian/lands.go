package arabian

import (
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerLands()
}

func registerLands() {
	Register("Bazaar of Baghdad", func() Card {
		return NewLand("Bazaar of Baghdad")
	})

	Register("City of Brass", func() Card {
		return NewLand("City of Brass")
	})

	Register("Desert", func() Card {
		return NewLand("Desert",
			WithSubTypes("Desert"),
			WithManaAbility(Colorless),
			WithActivatedAbility(
				DealDamage(Fixed(1)),
				TapSourceCost(),
				WithTarget(TargetCreature(IsAttacking)),
			),
		)
	})

	Register("Diamond Valley", func() Card {
		return NewLand("Diamond Valley")
	})

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

	Register("Island of Wak-Wak", func() Card {
		return NewLand("Island of Wak-Wak")
	})

	Register("Library of Alexandria", func() Card {
		return NewLand("Library of Alexandria")
	})

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
