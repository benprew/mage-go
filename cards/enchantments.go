package cards

import "github.com/mage/mage"

func init() {
	registerEnchantments()
}

func registerEnchantments() {
	mage.Register("Rancor", func() mage.Card {
		c := mage.NewAura("Rancor", "{G}")
		c.AddAbility(mage.StaticAbility(
			mage.BoostAttached(2, 0, mage.AttachAura),
			mage.GrantAbilityToAttached(mage.Trample, mage.AttachAura),
		))
		// When Rancor is put into a graveyard from the battlefield,
		// return it to its owner's hand.
		c.AddAbility(mage.PutIntoGraveyardFromBattlefieldTrigger(
			mage.ReturnSourceToHand(), false,
		))
		return c
	})

	mage.Register("Pacifism", func() mage.Card {
		c := mage.NewAura("Pacifism", "{1}{W}")
		c.AddAbility(mage.StaticAbility(
			mage.PreventAttachedFromAttacking(mage.AttachAura),
		))
		return c
	})

	mage.Register("Holy Strength", func() mage.Card {
		c := mage.NewAura("Holy Strength", "{W}")
		c.AddAbility(mage.StaticAbility(
			mage.BoostAttached(1, 2, mage.AttachAura),
		))
		return c
	})
}
