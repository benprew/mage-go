package mage

import (
	"maps"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func (g *Game) captureHandTargetCards(object *StackObject) {
	for _, id := range object.Targets {
		if _, saved := object.targetHandCards[id]; saved {
			continue
		}
		visit := g.handVisit(id)
		if visit == uuid.Nil {
			continue
		}
		if captured, ok := object.targetIncarnations[id]; ok && captured != visit {
			continue
		}
		object.targetHandCards = maps.Clone(object.targetHandCards)
		if object.targetHandCards == nil {
			object.targetHandCards = make(map[uuid.UUID]capturedCard)
		}
		object.targetHandCards[id] = capturedCard{id: id, visit: visit, zone: core.ZoneHand}
	}
}

func (g *Game) cardTargetVisit(id uuid.UUID) uuid.UUID {
	if visit := g.handVisit(id); visit != uuid.Nil {
		return visit
	}
	return g.graveyardVisit(id)
}

func (g *Game) handTargetCards(targets []uuid.UUID) map[uuid.UUID]capturedCard {
	object := &StackObject{Targets: targets}
	g.captureHandTargetCards(object)
	return object.targetHandCards
}
