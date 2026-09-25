package limited

import (
	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func init() {
	registerLands()
}

func registerLands() {
	// Basic land types supply their mana abilities (CR 305.6).
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
		// Plains: ({T}: Add {W}.)
		// Island: ({T}: Add {U}.)
		// Swamp: ({T}: Add {B}.)
		// Mountain: ({T}: Add {R}.)
		// Forest: ({T}: Add {G}.)
		Register(name, func() Card {
			return NewLand(name,
				WithSuperTypes(SuperBasic),
				WithSubTypes(name),
				WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(color, Fixed(1)))),
			)
		})
	}

	// Each basic land type supplies a separate mana ability (CR 305.6).
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
		// Badlands: ({T}: Add {B} or {R}.)
		// Bayou: ({T}: Add {B} or {G}.)
		// Plateau: ({T}: Add {R} or {W}.)
		// Savannah: ({T}: Add {G} or {W}.)
		// Scrubland: ({T}: Add {W} or {B}.)
		// Taiga: ({T}: Add {R} or {G}.)
		// Tropical Island: ({T}: Add {G} or {U}.)
		// Tundra: ({T}: Add {W} or {U}.)
		// Underground Sea: ({T}: Add {U} or {B}.)
		// Volcanic Island: ({T}: Add {U} or {R}.)
		Register(name, func() Card {
			return NewLand(name,
				WithSubTypes(sub1, sub2),
				WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(color1, Fixed(1)))),
				WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(color2, Fixed(1)))),
			)
		})
	}
}
