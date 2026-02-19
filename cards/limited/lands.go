package limited

import "github.com/mage/mage/pkg/mage"

func init() {
	registerLands()
}

func registerLands() {
	// Basic lands
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

	// Dual lands - each taps for one of two colors
	duals := []struct {
		name   string
		sub1   string
		sub2   string
		color1 mage.Color
		color2 mage.Color
	}{
		{"Badlands", "Swamp", "Mountain", mage.Black, mage.Red},
		{"Bayou", "Swamp", "Forest", mage.Black, mage.Green},
		{"Plateau", "Mountain", "Plains", mage.Red, mage.White},
		{"Savannah", "Forest", "Plains", mage.Green, mage.White},
		{"Scrubland", "Plains", "Swamp", mage.White, mage.Black},
		{"Taiga", "Mountain", "Forest", mage.Red, mage.Green},
		{"Tropical Island", "Forest", "Island", mage.Green, mage.Blue},
		{"Tundra", "Plains", "Island", mage.White, mage.Blue},
		{"Underground Sea", "Island", "Swamp", mage.Blue, mage.Black},
		{"Volcanic Island", "Island", "Mountain", mage.Blue, mage.Red},
	}

	for _, d := range duals {
		name := d.name
		sub1 := d.sub1
		sub2 := d.sub2
		color1 := d.color1
		color2 := d.color2
		mage.Register(name, func() mage.Card {
			c := mage.NewLand(name, sub1, sub2)
			c.AddAbility(mage.NewManaAbility(color1))
			c.AddAbility(mage.NewManaAbility(color2))
			return c
		})
	}
}
