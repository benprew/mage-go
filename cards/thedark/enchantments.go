package thedark

import . "github.com/benprew/mage-go/pkg/mage/dsl"

func init() {
	registerEnchantments()
}

func registerEnchantments() {
	// Blood Moon {2}{R}
	// Enchantment
	// Nonbasic lands are Mountains.
	Register("Blood Moon", func() Card {
		return NewEnchantment("Blood Moon", "{2}{R}",
			WithStaticAbility(
				BecomesBasicLandsEffect(
					NewPermanentFilter("nonbasic lands", func(permanent *Permanent, _ *Game) bool {
						return permanent.HasType(TypeLand) && !permanent.Card.HasSuperType(SuperBasic)
					}),
					"Mountain",
				),
			),
		)
	})
}
