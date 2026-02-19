package arabian

import . "github.com/mage/mage/pkg/mage"

func init() {
	registerSpells()
}

func registerSpells() {
	// ===== WHITE SPELLS =====

	Register("Army of Allah", func() Card {
		return NewInstant("Army of Allah", "{1}{W}{W}", nil)
	})

	Register("Eye for an Eye", func() Card {
		return NewInstant("Eye for an Eye", "{W}{W}", nil)
	})

	Register("Piety", func() Card {
		return NewInstant("Piety", "{2}{W}", nil)
	})

	Register("Shahrazad", func() Card {
		return NewSorcery("Shahrazad", "{W}{W}", nil)
	})

	// ===== GREEN SPELLS =====

	Register("Desert Twister", func() Card {
		return NewSorcery("Desert Twister", "{4}{G}{G}", nil)
	})

	Register("Metamorphosis", func() Card {
		return NewSorcery("Metamorphosis", "{G}", nil)
	})

	Register("Sandstorm", func() Card {
		return NewInstant("Sandstorm", "{G}", nil)
	})
}
