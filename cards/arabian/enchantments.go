package arabian

import (
	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

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
		return NewAura("Fishliver Oil", "{1}{U}",
			WithAbility(StaticAbility(GrantAbilityToAttached(Islandwalk, AttachAura))),
		)
	})

	Register("Unstable Mutation", func() Card {
		return NewAura("Unstable Mutation", "{U}",
			WithAbility(StaticAbility(BoostAttached(3, 3, AttachAura))),
			WithAbility(BeginningOfAttachedControllerUpkeepTrigger(
				FuncEffect("put a -1/-1 counter on enchanted creature",
					EffectProperties{},
					func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						src := g.FindPermanent(sourceID)
						if src == nil || src.AttachedTo == uuid.Nil {
							return nil
						}
						target := g.FindPermanent(src.AttachedTo)
						if target != nil {
							target.AddCounter(M1M1, 1)
						}
						return nil
					}), false,
			)),
		)
	})
}
