package arabian

import "github.com/mage/mage/pkg/mage"

func init() {
	registerLands()
}

func registerLands() {
	// Mountain (basic land) already registered in lands.go

	mage.Register("Bazaar of Baghdad", func() mage.Card {
		// {T}: Draw two cards, then discard three cards.
		c := mage.NewLand("Bazaar of Baghdad")
		return c
	})

	mage.Register("City of Brass", func() mage.Card {
		// Whenever City of Brass becomes tapped, it deals 1 damage to you.
		// {T}: Add one mana of any color.
		c := mage.NewLand("City of Brass")
		return c
	})

	mage.Register("Desert", func() mage.Card {
		// {T}: Add {C}.
		// {T}: Desert deals 1 damage to target attacking creature. Activate only
		// during the end of combat step.
		c := mage.NewLand("Desert", "Desert")
		c.AddAbility(mage.NewManaAbility(mage.Colorless))
		c.AddAbility(mage.NewActivatedAbility(
			mage.DealDamage(mage.Fixed(1)),
			mage.TapSourceCost(),
			mage.WithTarget(mage.TargetCreature(mage.IsAttacking))))
		return c
	})

	mage.Register("Diamond Valley", func() mage.Card {
		// {T}, Sacrifice a creature: You gain life equal to the sacrificed creature's
		// toughness.
		c := mage.NewLand("Diamond Valley")
		return c
	})

	mage.Register("Elephant Graveyard", func() mage.Card {
		// {T}: Add {C}.
		// {T}: Regenerate target Elephant.
		c := mage.NewLand("Elephant Graveyard")
		return c
	})

	mage.Register("Island of Wak-Wak", func() mage.Card {
		// {T}: Target creature with flying has base power 0 until end of turn.
		c := mage.NewLand("Island of Wak-Wak")
		return c
	})

	mage.Register("Library of Alexandria", func() mage.Card {
		// {T}: Add {C}.
		// {T}: Draw a card. Activate only if you have exactly seven cards in hand.
		c := mage.NewLand("Library of Alexandria")
		return c
	})

	mage.Register("Oasis", func() mage.Card {
		// {T}: Prevent the next 1 damage that would be dealt to target creature this turn.
		c := mage.NewLand("Oasis")
		return c
	})
}
