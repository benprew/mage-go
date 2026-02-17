package cards

import "github.com/mage/mage"

func init() {
	registerLands()
}

func registerLands() {
	for _, land := range []struct {
		name  string
		color mage.Color
	}{
		{"Plains", mage.White},
		{"Island", mage.Blue},
		{"Swamp", mage.Black},
		{"Mountain", mage.Red},
		{"Forest", mage.Green},
	} {
		name := land.name
		color := land.color
		mage.Register(name, func() mage.Card {
			c := mage.NewLand(name, name) // subtype matches name (e.g., Plains has subtype "Plains")
			c.AddAbility(mage.NewManaAbility(color))
			return c
		})
	}
}
