package limited

import (
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

func init() {
	registerLands()
}

func registerLands() {
	// Basic lands
	for _, land := range []struct {
		name  string
		color core.Color
	}{
		{"Plains", core.White},
		{"Island", core.Blue},
		{"Swamp", core.Black},
		{"Mountain", core.Red},
		{"Forest", core.Green},
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
		color1 core.Color
		color2 core.Color
	}{
		{"Badlands", "Swamp", "Mountain", core.Black, core.Red},
		{"Bayou", "Swamp", "Forest", core.Black, core.Green},
		{"Plateau", "Mountain", "Plains", core.Red, core.White},
		{"Savannah", "Forest", "Plains", core.Green, core.White},
		{"Scrubland", "Plains", "Swamp", core.White, core.Black},
		{"Taiga", "Mountain", "Forest", core.Red, core.Green},
		{"Tropical Island", "Forest", "Island", core.Green, core.Blue},
		{"Tundra", "Plains", "Island", core.White, core.Blue},
		{"Underground Sea", "Island", "Swamp", core.Blue, core.Black},
		{"Volcanic Island", "Island", "Mountain", core.Blue, core.Red},
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
