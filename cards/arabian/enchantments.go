package arabian

import "github.com/mage/mage/pkg/mage"

func init() {
	registerEnchantments()
}

func registerEnchantments() {
	// ===== GLOBAL ENCHANTMENTS =====

	mage.Register("Cyclone", func() mage.Card {
		// At the beginning of your upkeep, put a wind counter on Cyclone, then
		// sacrifice Cyclone unless you pay {G} for each wind counter on it. If you
		// pay, Cyclone deals damage equal to the number of wind counters on it to
		// each creature and each player.
		c := mage.NewEnchantment("Cyclone", "{2}{G}{G}")
		return c
	})

	mage.Register("Drop of Honey", func() mage.Card {
		// At the beginning of your upkeep, destroy the creature with the least power.
		// It can't be regenerated. If two or more creatures are tied for least power,
		// you choose one of them.
		// When there are no creatures on the battlefield, sacrifice Drop of Honey.
		c := mage.NewEnchantment("Drop of Honey", "{G}")
		return c
	})

	mage.Register("Jihad", func() mage.Card {
		// As Jihad enters the battlefield, choose a color and an opponent.
		// White creatures get +2/+1 as long as the chosen player controls a nontoken
		// permanent of the chosen color.
		// When the chosen player controls no nontoken permanents of the chosen color,
		// sacrifice Jihad.
		c := mage.NewEnchantment("Jihad", "{W}{W}{W}")
		return c
	})

	// Magnetic Mountain already registered in Alpha

	mage.Register("Oubliette", func() mage.Card {
		// When Oubliette enters the battlefield, target creature phases out until
		// Oubliette leaves the battlefield. Tap that creature as it phases in this way.
		c := mage.NewEnchantment("Oubliette", "{1}{B}{B}")
		return c
	})

	// ===== AURAS =====

	mage.Register("Fishliver Oil", func() mage.Card {
		// Enchant creature
		// Enchanted creature has islandwalk.
		c := mage.NewAura("Fishliver Oil", "{1}{U}")
		return c
	})

	mage.Register("Unstable Mutation", func() mage.Card {
		// Enchant creature
		// Enchanted creature gets +3/+3.
		// At the beginning of the upkeep of enchanted creature's controller, put a
		// -1/-1 counter on that creature.
		c := mage.NewAura("Unstable Mutation", "{U}")
		return c
	})
}
