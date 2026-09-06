package thedark

import . "github.com/benprew/mage-go/pkg/mage"

func init() {
	registerLands()
}

func registerLands() {

	// City of Shadows
	// Land
	// {T}, Exile a creature you control: Put a storage counter on this land.
	// {T}: Add {C} for each storage counter on this land.
	// TODO: implement
	Register("City of Shadows", func() Card {
		return NewLand("City of Shadows")
	})

	// Maze of Ith
	// Land
	// {T}: Untap target attacking creature. Prevent all combat damage that would be dealt to and dealt by that creature this turn.
	// TODO: implement
	Register("Maze of Ith", func() Card {
		return NewLand("Maze of Ith")
	})

	// Safe Haven
	// Land
	// {2}, {T}: Exile target creature you control.
	// At the beginning of your upkeep, you may sacrifice this land. If you do, return each card exiled with this land to the battlefield under its owner's control.
	// TODO: implement
	Register("Safe Haven", func() Card {
		return NewLand("Safe Haven")
	})

	// Sorrow's Path
	// Land
	// {T}: Choose two target blocking creatures controlled by the same opponent. If each of those creatures could block all creatures that the other is blocking, remove both of them from combat. Each one then blocks all creatures the other was blocking.
	// Whenever this land becomes tapped, it deals 2 damage to you and each creature you control.
	// TODO: implement
	Register("Sorrow's Path", func() Card {
		return NewLand("Sorrow's Path")
	})

}
