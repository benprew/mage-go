package mage

import "reflect"

// The wrapper keeps token status separate from shared stack definitions.
type permanentSpellCopyCard struct {
	Card
	abilities []Ability
}

func newPermanentSpellCopyCard(card Card) Card {
	abilities := make([]Ability, len(card.Abilities()))
	for i, ability := range card.Abilities() {
		abilities[i] = cloneTokenAbility(ability)
		if holder, ok := abilities[i].(*StaticAbilityHolder); ok {
			holder.Effects = make([]ContinuousEffect, len(holder.Effects))
			original := ability.(*StaticAbilityHolder)
			for j, effect := range original.Effects {
				if cloned, ok := effect.(deepCloneableEffect); ok {
					holder.Effects[j] = cloned.cloneEffect()
				} else {
					// Stateless effects still have a separate source identity.
					value := reflect.ValueOf(effect)
					if value.Kind() == reflect.Pointer && value.Elem().Kind() == reflect.Struct {
						clone := reflect.New(value.Elem().Type())
						clone.Elem().Set(value.Elem())
						holder.Effects[j] = clone.Interface().(ContinuousEffect)
					} else {
						holder.Effects[j] = effect
					}
				}
			}
		}
	}
	return permanentSpellCopyCard{Card: card, abilities: abilities}
}

func (permanentSpellCopyCard) IsToken() bool          { return true }
func (c permanentSpellCopyCard) Abilities() []Ability { return c.abilities }
func (c permanentSpellCopyCard) Copy() Card           { return newPermanentSpellCopyCard(c.Card.Copy()) }
