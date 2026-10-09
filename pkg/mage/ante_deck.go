package mage

// WithAnteDeckRestriction excludes this physical card from non-ante decks.
func WithAnteDeckRestriction() CardOption {
	return func(card *BaseCard) { card.requiresAnte = true }
}

// RequiresAnte reports the physical card's deck restriction.
// Changes to its game characteristics do not change this restriction.
func (c *BaseCard) RequiresAnte() bool { return c.requiresAnte }

// PrepareCardsForAnte separates cards before deck or sideboard setup.
// Both results keep input order and use separate slice storage.
// Excluded cards do not enter a game zone.
func PrepareCardsForAnte(cards []Card, enabled bool) (retained, excluded []Card) {
	for _, card := range cards {
		restricted, ok := card.(interface{ RequiresAnte() bool })
		if !enabled && ok && restricted.RequiresAnte() {
			excluded = append(excluded, card)
		} else {
			retained = append(retained, card)
		}
	}
	return retained, excluded
}
