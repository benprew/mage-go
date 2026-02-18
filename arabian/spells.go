package arabian

import "github.com/mage/mage"

func init() {
	registerSpells()
}

func registerSpells() {
	// ===== WHITE SPELLS =====

	mage.Register("Army of Allah", func() mage.Card {
		// Attacking creatures get +2/+0 until end of turn.
		c := mage.NewInstant("Army of Allah", "{1}{W}{W}")
		return c
	})

	mage.Register("Eye for an Eye", func() mage.Card {
		// The next time a source of your choice would deal damage to you this turn,
		// instead that source deals that much damage to you and Eye for an Eye deals
		// that much damage to that source's controller.
		c := mage.NewInstant("Eye for an Eye", "{W}{W}")
		return c
	})

	mage.Register("Piety", func() mage.Card {
		// Blocking creatures get +0/+3 until end of turn.
		c := mage.NewInstant("Piety", "{2}{W}")
		return c
	})

	mage.Register("Shahrazad", func() mage.Card {
		// Players play a Magic subgame, using their libraries as their decks. Each
		// player who doesn't win the subgame loses half their life, rounded up.
		c := mage.NewSorcery("Shahrazad", "{W}{W}")
		return c
	})

	// ===== GREEN SPELLS =====

	mage.Register("Desert Twister", func() mage.Card {
		// Destroy target permanent.
		c := mage.NewSorcery("Desert Twister", "{4}{G}{G}")
		return c
	})

	mage.Register("Metamorphosis", func() mage.Card {
		// As an additional cost to cast this spell, sacrifice a creature.
		// Add X mana of any one color, where X is 1 plus the sacrificed creature's
		// mana value. Spend this mana only to cast creature spells.
		c := mage.NewSorcery("Metamorphosis", "{G}")
		return c
	})

	mage.Register("Sandstorm", func() mage.Card {
		// Sandstorm deals 1 damage to each attacking creature.
		c := mage.NewInstant("Sandstorm", "{G}")
		return c
	})
}
