package arabian

import . "github.com/mage/mage/pkg/mage"

func init() {
	registerEnchantments()
}

func registerEnchantments() {
	// ===== GLOBAL ENCHANTMENTS =====

	Register("Cyclone", func() Card {
		return NewEnchantment("Cyclone", "{2}{G}{G}")
	})

	Register("Drop of Honey", func() Card {
		return NewEnchantment("Drop of Honey", "{G}")
	})

	Register("Jihad", func() Card {
		return NewEnchantment("Jihad", "{W}{W}{W}")
	})

	Register("Oubliette", func() Card {
		return NewEnchantment("Oubliette", "{1}{B}{B}")
	})

	// ===== AURAS =====

	Register("Fishliver Oil", func() Card {
		return NewAura("Fishliver Oil", "{1}{U}")
	})

	Register("Unstable Mutation", func() Card {
		return NewAura("Unstable Mutation", "{U}")
	})
}
