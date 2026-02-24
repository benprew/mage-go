package arabian

import . "github.com/mage/mage/pkg/mage"

func init() {
	registerSpells()
}

func registerSpells() {
	// ===== WHITE SPELLS =====

	// Oracle: "Attacking creatures get +2/+0 until end of turn."
	Register("Army of Allah", func() Card {
		return NewInstant("Army of Allah", "{1}{W}{W}",
			NewSpellAbility(BoostAllMatchingUntilEndOfTurn(Fixed(2), Fixed(0), IsAttacking)),
		)
	})

	// Oracle: "The next time a source of your choice would deal damage to you this turn,
	// instead that source deals that much damage to you and Eye for an Eye deals that much
	// damage to that source's controller."
	// XXX: Eye for an Eye deferred — needs damage source tracking + replacement effect
	Register("Eye for an Eye", func() Card {
		return NewInstant("Eye for an Eye", "{W}{W}", nil)
	})

	// Oracle: "Blocking creatures get +0/+3 until end of turn."
	Register("Piety", func() Card {
		return NewInstant("Piety", "{2}{W}",
			NewSpellAbility(BoostAllMatchingUntilEndOfTurn(Fixed(0), Fixed(3), IsBlocking)),
		)
	})

	// Oracle: "Players play a Magic subgame, using their libraries as their decks.
	// Each player who doesn't win the subgame loses half their life, rounded up."
	// XXX: Shahrazad skipped — subgame mechanic, banned in all formats
	Register("Shahrazad", func() Card {
		return NewSorcery("Shahrazad", "{W}{W}", nil)
	})

	// ===== GREEN SPELLS =====

	// Oracle: "Destroy target permanent."
	Register("Desert Twister", func() Card {
		return NewSorcery("Desert Twister", "{4}{G}{G}",
			NewTargetedSpell(TargetPermanent(), DestroyTargetPermanent()),
		)
	})

	// Oracle: "As an additional cost to cast this spell, sacrifice a creature.
	// Add X mana of any one color, where X is 1 plus the sacrificed creature's mana value.
	// Spend this mana only to cast creature spells."
	// XXX: Metamorphosis deferred — needs sacrifice as additional cost + restricted mana generation
	Register("Metamorphosis", func() Card {
		return NewSorcery("Metamorphosis", "{G}", nil)
	})

	// Oracle: "Sandstorm deals 1 damage to each attacking creature."
	Register("Sandstorm", func() Card {
		return NewInstant("Sandstorm", "{G}",
			NewSpellAbility(DealDamageToAllCreatures(Fixed(1), IsAttacking)),
		)
	})
}
