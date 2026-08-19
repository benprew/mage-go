package promo

import (
	. "github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

func init() {
	registerCreatures()
}

func registerCreatures() {

	// ===== RED CREATURES =====

	// Windseeker Centaur {1}{R}{R}
	// Creature — Centaur
	// 2/2
	// Vigilance
	Register("Windseeker Centaur", func() Card {
		return NewCreature("Windseeker Centaur", "{1}{R}{R}", 2, 2,
			WithSubTypes("Centaur"),
			WithKeyword(Vigilance),
		)
	})

	// ===== GREEN CREATURES =====

	// Giant Badger {1}{G}{G}
	// Creature — Badger
	// 2/2
	// Whenever this creature blocks, it gets +2/+2 until end of turn.
	Register("Giant Badger", func() Card {
		return NewCreature("Giant Badger", "{1}{G}{G}", 2, 2,
			WithSubTypes("Badger"),
			WithAbility(BlocksTrigger(
				Boost(Fixed(2), Fixed(2)).Targeting(ToSource()).Until(EndOfTurn), false,
			)),
		)
	})
}
