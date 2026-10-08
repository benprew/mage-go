package mage

// The wrapper keeps token status separate from shared stack definitions.
type permanentSpellCopyCard struct {
	Card
	abilities []Ability
}

func newPermanentSpellCopyCard(card Card) Card {
	abilities := make([]Ability, len(card.Abilities()))
	for i, ability := range card.Abilities() {
		abilities[i] = cloneTokenAbility(ability)
	}
	return permanentSpellCopyCard{Card: card, abilities: abilities}
}

func (permanentSpellCopyCard) IsToken() bool          { return true }
func (c permanentSpellCopyCard) Abilities() []Ability { return c.abilities }
func (c permanentSpellCopyCard) Copy() Card           { return newPermanentSpellCopyCard(c.Card.Copy()) }
