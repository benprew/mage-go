package arabian

import "github.com/mage/mage/pkg/mage"

func init() {
	registerArtifacts()
}

func registerArtifacts() {
	mage.Register("Aladdin's Lamp", func() mage.Card {
		// {X}, {T}: The next time you would draw a card this turn, instead look at
		// the top X cards of your library, put all but one of them on the bottom of
		// your library in a random order, then draw a card. X can't be 0.
		c := mage.NewArtifact("Aladdin's Lamp", "{10}")
		return c
	})

	mage.Register("Aladdin's Ring", func() mage.Card {
		// {8}, {T}: Aladdin's Ring deals 4 damage to any target.
		c := mage.NewArtifact("Aladdin's Ring", "{8}")
		return c
	})

	mage.Register("Bottle of Suleiman", func() mage.Card {
		// {1}, Sacrifice Bottle of Suleiman: Flip a coin. If you win the flip, create
		// a 5/5 colorless Djinn artifact creature token with flying. If you lose the
		// flip, Bottle of Suleiman deals 5 damage to you.
		c := mage.NewArtifact("Bottle of Suleiman", "{4}")
		return c
	})

	mage.Register("City in a Bottle", func() mage.Card {
		// Whenever one or more other nontoken permanents with a name originally
		// printed in the Arabian Nights expansion are on the battlefield, their
		// controllers sacrifice them.
		// Players can't cast spells or play lands with a name originally printed in
		// the Arabian Nights expansion.
		c := mage.NewArtifact("City in a Bottle", "{2}")
		return c
	})

	mage.Register("Ebony Horse", func() mage.Card {
		// {2}, {T}: Untap target attacking creature you control. Prevent all combat
		// damage that would be dealt to and dealt by that creature this turn.
		c := mage.NewArtifact("Ebony Horse", "{3}")
		return c
	})

	mage.Register("Flying Carpet", func() mage.Card {
		// {2}, {T}: Target creature gains flying until end of turn.
		c := mage.NewArtifact("Flying Carpet", "{4}")
		return c
	})

	mage.Register("Jandor's Ring", func() mage.Card {
		// {2}, {T}, Discard the last card you drew this turn: Draw a card.
		c := mage.NewArtifact("Jandor's Ring", "{6}")
		return c
	})

	mage.Register("Jandor's Saddlebags", func() mage.Card {
		// {3}, {T}: Untap target creature.
		c := mage.NewArtifact("Jandor's Saddlebags", "{2}")
		return c
	})

	mage.Register("Jeweled Bird", func() mage.Card {
		// Remove Jeweled Bird from your deck before playing if you're not playing
		// for ante.
		// {T}: Ante Jeweled Bird. If you do, put all other cards you own from the
		// ante into your graveyard, then draw a card.
		c := mage.NewArtifact("Jeweled Bird", "{1}")
		return c
	})

	mage.Register("Pyramids", func() mage.Card {
		// {2}: Choose one —
		// • Destroy target Aura attached to a land.
		// • The next time target land would be destroyed this turn, remove all damage
		//   marked on it instead.
		c := mage.NewArtifact("Pyramids", "{6}")
		return c
	})

	mage.Register("Ring of Ma'rûf", func() mage.Card {
		// {5}, {T}, Exile Ring of Ma'rûf: The next time you would draw a card this
		// turn, instead put a card you own from outside the game into your hand.
		c := mage.NewArtifact("Ring of Ma'rûf", "{5}")
		return c
	})

	mage.Register("Sandals of Abdallah", func() mage.Card {
		// {2}, {T}: Target creature gains islandwalk until end of turn. When that
		// creature dies this turn, destroy Sandals of Abdallah.
		c := mage.NewArtifact("Sandals of Abdallah", "{4}")
		return c
	})
}
