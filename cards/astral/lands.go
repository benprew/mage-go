package astral

import . "github.com/benprew/mage-go/pkg/mage"

func init() {
	registerLands()
}

func registerLands() {

	// Gem Bazaar
	// Land
	// When Gem Bazaar comes into play, choose a random color.
	// {T}: Add to your mana pool one mana of the color last chosen. Then choose a random color.
	Register("Gem Bazaar", func() Card {
		return NewLand("Gem Bazaar",
			WithAbility(EntersBattlefieldTrigger(
				SetSourceChosenColorAtRandom(),
				false,
			)),
			WithDynamicManaAbility(
				ChosenColorManaProductions,
				SetSourceChosenColorAtRandom(),
			),
		)
	})

}
