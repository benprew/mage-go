package astral

import . "github.com/benprew/mage-go/pkg/mage"

func init() {
	registerArtifacts()
}

func registerArtifacts() {

	// Pandora's Box {5}
	// Artifact
	// {3}, {T}: Choose a random summon card from all players' decks. For each player, flip a coin. If the flip ends up heads, put a token creature into play and treat it as though an exact copy of the chosen summon card were just played.
	Register("Pandora's Box", func() Card {
		return NewArtifact("Pandora's Box", "{5}",
			WithActivatedAbility(
				PandorasBoxEffect(),
				ManaCostOf("{3}"),
				WithCost(Tap()),
			),
		)
	})

}
