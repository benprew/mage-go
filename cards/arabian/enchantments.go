package arabian

import "github.com/mage/mage/pkg/mage"

func init() {
	registerEnchantments()
}

func registerEnchantments() {
	// ===== GLOBAL ENCHANTMENTS =====

	mage.Register("Cyclone", func() mage.Card {
		return mage.NewEnchantment("Cyclone", "{2}{G}{G}")
	})

	mage.Register("Drop of Honey", func() mage.Card {
		return mage.NewEnchantment("Drop of Honey", "{G}")
	})

	mage.Register("Jihad", func() mage.Card {
		return mage.NewEnchantment("Jihad", "{W}{W}{W}")
	})

	mage.Register("Oubliette", func() mage.Card {
		return mage.NewEnchantment("Oubliette", "{1}{B}{B}")
	})

	// ===== AURAS =====

	mage.Register("Fishliver Oil", func() mage.Card {
		return mage.NewAura("Fishliver Oil", "{1}{U}")
	})

	mage.Register("Unstable Mutation", func() mage.Card {
		return mage.NewAura("Unstable Mutation", "{U}")
	})
}
