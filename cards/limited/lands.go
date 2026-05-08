package limited

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/dsl"
)

func init() {
	registerLands()
}

func registerLands() {
	// Basic lands
	for _, land := range []struct {
		name  string
		color Color
	}{
		{"Plains", White},
		{"Island", Blue},
		{"Swamp", Black},
		{"Mountain", Red},
		{"Forest", Green},
	} {
		name := land.name
		color := land.color
		Register(name, func() Card {
			return NewLand(name,
				WithSuperTypes(SuperBasic),
				WithSubTypes(name),
				WithManaAbility(color),
			)
		})
	}

	// Dual lands - each taps for one of two colors
	duals := []struct {
		name   string
		sub1   string
		sub2   string
		color1 Color
		color2 Color
	}{
		{"Badlands", "Swamp", "Mountain", Black, Red},
		{"Bayou", "Swamp", "Forest", Black, Green},
		{"Plateau", "Mountain", "Plains", Red, White},
		{"Savannah", "Forest", "Plains", Green, White},
		{"Scrubland", "Plains", "Swamp", White, Black},
		{"Taiga", "Mountain", "Forest", Red, Green},
		{"Tropical Island", "Forest", "Island", Green, Blue},
		{"Tundra", "Plains", "Island", White, Blue},
		{"Underground Sea", "Island", "Swamp", Blue, Black},
		{"Volcanic Island", "Island", "Mountain", Blue, Red},
	}

	for _, d := range duals {
		name := d.name
		sub1 := d.sub1
		sub2 := d.sub2
		color1 := d.color1
		color2 := d.color2
		Register(name, func() Card {
			return NewLand(name,
				WithSubTypes(sub1, sub2),
				WithManaAbility(color1),
				WithManaAbility(color2),
			)
		})
	}
}
