package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ReturnExiledCardsToOwnersHands moves the selected visits to their owners' hands.
func ReturnExiledCardsToOwnersHands(subjects ExiledCardSet) ResolvingEffect {
	return &exileTransportEffect{subjects: subjects, destination: core.ZoneHand}
}

// PutExiledCardsInOwnersGraveyards moves the selected visits to their owners' graveyards.
func PutExiledCardsInOwnersGraveyards(subjects ExiledCardSet) ResolvingEffect {
	return &exileTransportEffect{subjects: subjects, destination: core.ZoneGraveyard}
}

type exileTransportEffect struct {
	subjects    ExiledCardSet
	destination core.Zone
}

func (*exileTransportEffect) Text() string {
	return "move selected exiled cards to their owners' zones"
}
func (*exileTransportEffect) Properties() EffectProperties { return EffectProperties{Mass: true} }
func (e *exileTransportEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"subjects", e.subjects}}}
}
func (e *exileTransportEffect) refersToSourcePermanent() bool {
	if set, ok := e.subjects.(linkedExileSet); ok {
		return permanentRefReadsSourcePermanent(set.source)
	}
	return false
}
func (e *exileTransportEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	visits := make(map[uuid.UUID]uuid.UUID)
	byOwner := make(map[uuid.UUID][]Card)
	for _, subject := range e.subjects.resolveExiledCards(ctx) {
		if subject.zone != core.ZoneExile || subject.visit == uuid.Nil || visits[subject.id] != uuid.Nil {
			continue
		}
		exiled := g.FindExiledCard(subject.id)
		if exiled == nil || exiled.Visit != subject.visit || g.GetPlayer(exiled.Card.Owner()) == nil {
			continue
		}
		visits[subject.id] = subject.visit
		byOwner[exiled.Card.Owner()] = append(byOwner[exiled.Card.Owner()], exiled.Card)
	}
	if e.destination == core.ZoneGraveyard {
		for _, owner := range []Player{g.ActivePlayerObj(), g.NonActivePlayerObj()} {
			byOwner[owner.PlayerID()] = g.orderCardsForGraveyard(owner, byOwner[owner.PlayerID()])
		}
	}
	var events []core.GameEvent
	for _, owner := range g.players {
		for _, card := range byOwner[owner.PlayerID()] {
			exiled := g.FindExiledCard(card.ID())
			if exiled == nil || exiled.Visit != visits[card.ID()] || exiled.Card.Owner() != owner.PlayerID() {
				continue
			}
			moved, ok := g.RemoveFromExile(card.ID())
			if !ok {
				continue
			}
			if e.destination == core.ZoneHand {
				owner.AddToHand(moved)
			} else {
				owner.AddToGraveyard(moved)
			}
			events = append(events, core.GameEvent{Type: core.EvtZoneChange, SourceID: card.ID(), PlayerID: owner.PlayerID(), FromZone: core.ZoneExile, ToZone: e.destination})
		}
	}
	for _, event := range events {
		g.FireEvent(event)
	}
	return nil
}
