package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// CardBinding declares a card saved during one resolution.
type CardBinding struct{ name string }

// BindCard declares a card binding with a diagnostic name.
func BindCard(name string) *CardBinding { return &CardBinding{name: name} }

type capturedCard struct {
	id, visit uuid.UUID
	zone      core.Zone
}
type savedCardRef struct{ binding *CardBinding }

// FromCard reads the saved card while its zone visit remains current.
func FromCard(binding *CardBinding) CardRef { return savedCardRef{binding: binding} }
func (savedCardRef) cardRef()               {}
func (r savedCardRef) resolveCard(ctx *EffectContext) (Card, bool) {
	value, ok := ctx.Game.resolution.cardBindings[r.binding]
	if !ok || value.visit == uuid.Nil {
		return nil, false
	}
	for _, player := range ctx.Game.players {
		visitor, hasVisit := player.(interface{ handVisit(uuid.UUID) uuid.UUID })
		if value.zone == core.ZoneHand && hasVisit && visitor.handVisit(value.id) == value.visit {
			for _, card := range player.Hand() {
				if card.ID() == value.id {
					return card, true
				}
			}
		}
		if value.zone == core.ZoneGraveyard && ctx.Game.graveyardVisit(value.id) == value.visit {
			for _, card := range player.Graveyard() {
				if card.ID() == value.id {
					return card, true
				}
			}
		}
	}
	return nil, false
}

func (b *CardBinding) bindingName() string { return b.name }
func (b *CardBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "card-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}
func (r savedCardRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "card-ref", Text: "saved graveyard card", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

type eventCardCapture struct {
	binding *CardBinding
	value   capturedCard
}

func (*eventCardCapture) Text() string                 { return "remember the event card" }
func (*eventCardCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *eventCardCapture) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.cardBindings == nil {
		ctx.Game.resolution.cardBindings = make(map[*CardBinding]capturedCard)
	}
	ctx.Game.resolution.cardBindings[e.binding] = e.value
	return nil
}
func (e *eventCardCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-card-capture", bindingWrite: e.binding, children: []definitionEdge{{"card", e.binding}}}
}
