package cards

import "github.com/mage/mage"

func init() {
	registerSpells()
}

func registerSpells() {
	mage.Register("Doom Blade", func() mage.Card {
		c := mage.NewInstant("Doom Blade", "{1}{B}")
		sa := mage.NewSpellAbility(mage.DestroyTarget())
		sa.AddTarget(mage.TargetCreature(mage.Not(mage.HasColorFilter(mage.Black))))
		c.AddAbility(sa)
		return c
	})

	mage.Register("Lightning Bolt", func() mage.Card {
		c := mage.NewInstant("Lightning Bolt", "{R}")
		sa := mage.NewSpellAbility(mage.DealDamage(3))
		sa.AddTarget(mage.TargetAnyTarget())
		c.AddAbility(sa)
		return c
	})

	// Wrath of God equivalent for shroud testing
	mage.Register("Wrath of God", func() mage.Card {
		c := mage.NewSorcery("Wrath of God", "{2}{W}{W}")
		sa := mage.NewSpellAbility(mage.DestroyAllCreatures())
		c.AddAbility(sa)
		return c
	})

	mage.Register("Giant Growth", func() mage.Card {
		c := mage.NewInstant("Giant Growth", "{G}")
		sa := mage.NewSpellAbility(mage.BoostTargetUntilEndOfTurn(3, 3))
		sa.AddTarget(mage.TargetCreature())
		c.AddAbility(sa)
		return c
	})
}
