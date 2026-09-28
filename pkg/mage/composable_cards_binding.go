package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// CardsBinding names a collection saved by one payment or instruction.
type CardsBinding struct{ name string }

// BindCards declares a card collection binding.
func BindCards(name string) *CardsBinding { return &CardsBinding{name: name} }

// CardSet is a sealed collection of cards in nonbattlefield zones.
type CardSet interface {
	cardSet()
	resolveCards(*EffectContext) []Card
}

type savedCardsSet struct{ binding *CardsBinding }

// FromCards reads the saved cards that remain in their captured zone visits.
func FromCards(binding *CardsBinding) CardSet { return savedCardsSet{binding: binding} }
func (savedCardsSet) cardSet()                {}
func (s savedCardsSet) resolveCards(ctx *EffectContext) []Card {
	var cards []Card
	for _, captured := range ctx.Game.resolution.cardsBindings[s.binding] {
		if captured.zone == core.ZoneExile {
			if exiled := ctx.Game.FindExiledCard(captured.id); exiled != nil && exiled.Visit == captured.visit && exiled.VisibleTo(ctx.Controller) {
				cards = append(cards, exiled.Card)
			}
			continue
		}
		if captured.zone != core.ZoneGraveyard {
			continue
		}
		if ctx.Game.graveyardVisit(captured.id) != captured.visit {
			continue
		}
		for _, player := range ctx.Game.players {
			for _, card := range player.Graveyard() {
				if card.ID() == captured.id {
					cards = append(cards, card)
					break
				}
			}
		}
	}
	return cards
}
func (b *CardsBinding) bindingName() string { return b.name }
func (b *CardsBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cards-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}
func (s savedCardsSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "card-set", Text: "saved cards", bindingRead: s.binding, children: []definitionEdge{{"binding", s.binding}}}
}

type cardCountValue struct{ subjects CardSet }

// CountCards counts cards still present in their captured zone visits.
func CountCards(subjects CardSet) ResolutionValue { return cardCountValue{subjects: subjects} }
func (cardCountValue) Text() string               { return "number of cards in collection" }
func (v cardCountValue) resolveValue(ctx *EffectContext) int {
	return len(v.subjects.resolveCards(ctx))
}
func (v cardCountValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "card-count", Text: v.Text(), children: []definitionEdge{{"subjects", v.subjects}}}
}
