package mage

import (
	"maps"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func (g *Game) captureExileTargetCards(object *StackObject) {
	for _, id := range object.Targets {
		if _, saved := object.targetHandCards[id]; saved {
			continue
		}
		if _, saved := object.targetExileCards[id]; saved {
			continue
		}
		visit := g.exileTargetVisit(id)
		if visit == uuid.Nil {
			continue
		}
		if captured, ok := object.targetIncarnations[id]; ok && captured != visit {
			continue
		}
		object.targetExileCards = maps.Clone(object.targetExileCards)
		if object.targetExileCards == nil {
			object.targetExileCards = make(map[uuid.UUID]capturedCard)
		}
		object.targetExileCards[id] = capturedCard{id: id, visit: visit, zone: core.ZoneExile}
	}
}

func (g *Game) exileTargetVisit(id uuid.UUID) uuid.UUID {
	if object := g.FindExiledCard(id); object != nil {
		return object.Visit
	}
	return uuid.Nil
}

func (g *Game) exileTargetCards(targets []uuid.UUID) map[uuid.UUID]capturedCard {
	object := &StackObject{Targets: targets}
	g.captureExileTargetCards(object)
	return object.targetExileCards
}
