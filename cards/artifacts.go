package cards

import "github.com/mage/mage"

func init() {
	registerArtifacts()
}

func registerArtifacts() {
	mage.Register("Bonesplitter", func() mage.Card {
		c := mage.NewEquipment("Bonesplitter", "{1}")
		c.AddAbility(mage.StaticAbility(
			mage.BoostAttached(2, 0, mage.AttachEquipment),
		))
		c.AddAbility(mage.NewEquipAbility(mage.GenericCost(1)))
		return c
	})

	mage.Register("Lightning Greaves", func() mage.Card {
		c := mage.NewEquipment("Lightning Greaves", "{2}")
		c.AddAbility(mage.StaticAbility(
			mage.GrantAbilityToAttached(mage.Haste, mage.AttachEquipment),
			mage.GrantAbilityToAttached(mage.Shroud, mage.AttachEquipment),
		))
		c.AddAbility(mage.NewEquipAbility(mage.GenericCost(0)))
		return c
	})
}
