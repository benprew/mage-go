package jumpstart

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

func init() {
	registerLands()
}

func registerLands() {

// Buried Ruin 
// Land
// {T}: Add {C}.
// {2}, {T}, Sacrifice this land: Return target artifact card from your graveyard to your hand.
// TODO: implement
	Register("Buried Ruin", func() Card {
		return NewLand("Buried Ruin")
	})


// Mirrodin's Core 
// Land
// {T}: Add {C}.
// {T}: Put a charge counter on this land.
// {T}, Remove a charge counter from this land: Add one mana of any color.
// TODO: implement
	Register("Mirrodin's Core", func() Card {
		return NewLand("Mirrodin's Core")
	})


// Phyrexian Tower 
// Legendary Land
// {T}: Add {C}.
// {T}, Sacrifice a creature: Add {B}{B}.
// TODO: implement
	Register("Phyrexian Tower", func() Card {
		return NewLand("Phyrexian Tower",
			WithSuperTypes(SuperLegendary),
		)
	})


// Riptide Laboratory 
// Land
// {T}: Add {C}.
// {1}{U}, {T}: Return target Wizard you control to its owner's hand.
// TODO: implement
	Register("Riptide Laboratory", func() Card {
		return NewLand("Riptide Laboratory")
	})


// Rupture Spire 
// Land
// This land enters tapped.
// When this land enters, sacrifice it unless you pay {1}.
// {T}: Add one mana of any color.
// TODO: implement
	Register("Rupture Spire", func() Card {
		return NewLand("Rupture Spire")
	})


// Terramorphic Expanse 
// Land
// {T}, Sacrifice this land: Search your library for a basic land card, put it onto the battlefield tapped, then shuffle.
// TODO: implement
	Register("Terramorphic Expanse", func() Card {
		return NewLand("Terramorphic Expanse")
	})


// Thriving Bluff 
// Land
// This land enters tapped. As it enters, choose a color other than red.
// {T}: Add {R} or one mana of the chosen color.
// TODO: implement
	Register("Thriving Bluff", func() Card {
		return NewLand("Thriving Bluff")
	})


// Thriving Grove 
// Land
// This land enters tapped. As it enters, choose a color other than green.
// {T}: Add {G} or one mana of the chosen color.
// TODO: implement
	Register("Thriving Grove", func() Card {
		return NewLand("Thriving Grove")
	})


// Thriving Heath 
// Land
// This land enters tapped. As it enters, choose a color other than white.
// {T}: Add {W} or one mana of the chosen color.
// TODO: implement
	Register("Thriving Heath", func() Card {
		return NewLand("Thriving Heath")
	})


// Thriving Isle 
// Land
// This land enters tapped. As it enters, choose a color other than blue.
// {T}: Add {U} or one mana of the chosen color.
// TODO: implement
	Register("Thriving Isle", func() Card {
		return NewLand("Thriving Isle")
	})


// Thriving Moor 
// Land
// This land enters tapped. As it enters, choose a color other than black.
// {T}: Add {B} or one mana of the chosen color.
// TODO: implement
	Register("Thriving Moor", func() Card {
		return NewLand("Thriving Moor")
	})

}
