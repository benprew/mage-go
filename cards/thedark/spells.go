package thedark

import . "github.com/benprew/mage-go/pkg/mage"

func init() {
	registerSpells()
}

func registerSpells() {

	// Amnesia {3}{U}{U}{U}
	// Sorcery
	// Target player reveals their hand and discards all nonland cards.
	// TODO: implement
	Register("Amnesia", func() Card {
		return NewSorcery("Amnesia", "{3}{U}{U}{U}",
			NewSpellAbility(),
		)
	})

	// Blood of the Martyr {W}{W}{W}
	// Instant
	// Until end of turn, if damage would be dealt to any creature, you may have that damage dealt to you instead.
	// TODO: implement
	Register("Blood of the Martyr", func() Card {
		return NewInstant("Blood of the Martyr", "{W}{W}{W}",
			NewSpellAbility(),
		)
	})

	// Cleansing {W}{W}{W}
	// Sorcery
	// For each land, destroy that land unless any player pays 1 life.
	// TODO: implement
	Register("Cleansing", func() Card {
		return NewSorcery("Cleansing", "{W}{W}{W}",
			NewSpellAbility(),
		)
	})

	// Dust to Dust {1}{W}{W}
	// Sorcery
	// Exile two target artifacts.
	// TODO: implement
	Register("Dust to Dust", func() Card {
		return NewSorcery("Dust to Dust", "{1}{W}{W}",
			NewSpellAbility(),
		)
	})

	// Eternal Flame {2}{R}{R}
	// Sorcery
	// Eternal Flame deals X damage to target opponent or planeswalker and half X damage, rounded up, to you, where X is the number of Mountains you control.
	// TODO: implement
	Register("Eternal Flame", func() Card {
		return NewSorcery("Eternal Flame", "{2}{R}{R}",
			NewSpellAbility(),
		)
	})

	// Festival {W}
	// Instant
	// Cast this spell only during an opponent's upkeep.
	// Creatures can't attack this turn.
	// TODO: implement
	Register("Festival", func() Card {
		return NewInstant("Festival", "{W}",
			NewSpellAbility(),
		)
	})

	// Fire and Brimstone {3}{W}{W}
	// Instant
	// Fire and Brimstone deals 4 damage to target player who attacked this turn and 4 damage to you.
	// TODO: implement
	Register("Fire and Brimstone", func() Card {
		return NewInstant("Fire and Brimstone", "{3}{W}{W}",
			NewSpellAbility(),
		)
	})

	// Holy Light {2}{W}
	// Instant
	// Nonwhite creatures get -1/-1 until end of turn.
	// TODO: implement
	Register("Holy Light", func() Card {
		return NewInstant("Holy Light", "{2}{W}",
			NewSpellAbility(),
		)
	})

	// Inquisition {2}{B}
	// Sorcery
	// Target player reveals their hand. Inquisition deals damage to that player equal to the number of white cards in their hand.
	// TODO: implement
	Register("Inquisition", func() Card {
		return NewSorcery("Inquisition", "{2}{B}",
			NewSpellAbility(),
		)
	})

	// Martyr's Cry {W}{W}
	// Sorcery
	// Exile all white creatures. For each creature exiled this way, its controller draws a card.
	// TODO: implement
	Register("Martyr's Cry", func() Card {
		return NewSorcery("Martyr's Cry", "{W}{W}",
			NewSpellAbility(),
		)
	})

	// Riptide {U}
	// Instant
	// Tap all blue creatures.
	// TODO: implement
	Register("Riptide", func() Card {
		return NewInstant("Riptide", "{U}",
			NewSpellAbility(),
		)
	})

	// Tivadar's Crusade {1}{W}{W}
	// Sorcery
	// Destroy all Goblins.
	// TODO: implement
	Register("Tivadar's Crusade", func() Card {
		return NewSorcery("Tivadar's Crusade", "{1}{W}{W}",
			NewSpellAbility(),
		)
	})

}
