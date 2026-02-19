package arabian

import "github.com/mage/mage/pkg/mage"

func init() {
	registerSpells()
}

func registerSpells() {
	// ===== WHITE SPELLS =====

	mage.Register("Army of Allah", func() mage.Card {
		return mage.NewInstant("Army of Allah", "{1}{W}{W}")
	})

	mage.Register("Eye for an Eye", func() mage.Card {
		return mage.NewInstant("Eye for an Eye", "{W}{W}")
	})

	mage.Register("Piety", func() mage.Card {
		return mage.NewInstant("Piety", "{2}{W}")
	})

	mage.Register("Shahrazad", func() mage.Card {
		return mage.NewSorcery("Shahrazad", "{W}{W}")
	})

	// ===== GREEN SPELLS =====

	mage.Register("Desert Twister", func() mage.Card {
		return mage.NewSorcery("Desert Twister", "{4}{G}{G}")
	})

	mage.Register("Metamorphosis", func() mage.Card {
		return mage.NewSorcery("Metamorphosis", "{G}")
	})

	mage.Register("Sandstorm", func() mage.Card {
		return mage.NewInstant("Sandstorm", "{G}")
	})
}
