package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func (g *Game) anteVisit(id uuid.UUID) uuid.UUID {
	for _, player := range g.players {
		if visitor, ok := player.(interface{ anteVisit(uuid.UUID) uuid.UUID }); ok {
			if visit := visitor.anteVisit(id); visit != uuid.Nil {
				return visit
			}
		}
	}
	return uuid.Nil
}

type anteCardTarget struct{ BaseTarget }

func (t *anteCardTarget) Choose(controller uuid.UUID, source Card, g *Game, chosen []uuid.UUID) error {
	if len(chosen) != 1 || !slices.Contains(t.Possible(controller, source, g), chosen[0]) {
		return fmt.Errorf("must target one card in ante")
	}
	t.chosen = slices.Clone(chosen)
	return nil
}

func (*anteCardTarget) Possible(_ uuid.UUID, _ Card, g *Game) []uuid.UUID {
	if !g.anteEnabled || g.anteSettled {
		return nil
	}
	cards, _ := g.AnteCards()
	var result []uuid.UUID
	for _, card := range cards {
		if g.anteVisit(card.ID()) != uuid.Nil {
			result = append(result, card.ID())
		}
	}
	return result
}

// DeclareAnteCardTarget declares one card in the shared ante zone.
func DeclareAnteCardTarget(name string) *CardTargetBinding {
	return &CardTargetBinding{&targetDeclaration{name: name, spec: &anteCardTarget{BaseTarget{min: 1, max: 1}}}}
}

func (g *Game) resolveAnteCard(id, visit uuid.UUID) (Card, bool) {
	if visit == uuid.Nil || g.anteVisit(id) != visit || !g.anteEnabled || g.anteSettled {
		return nil, false
	}
	cards, _ := g.AnteCards()
	for _, card := range cards {
		if card.ID() == id {
			return card, true
		}
	}
	return nil, false
}

func (g *Game) captureAnteTargetCards(object *StackObject) {
	for _, id := range object.Targets {
		if _, saved := object.targetAnteCards[id]; saved {
			continue
		}
		if _, saved := object.targetHandCards[id]; saved {
			continue
		}
		if _, saved := object.targetExileCards[id]; saved {
			continue
		}
		visit := g.anteVisit(id)
		if visit == uuid.Nil {
			continue
		}
		if captured, ok := object.targetIncarnations[id]; ok && captured != visit {
			continue
		}
		object.targetAnteCards = maps.Clone(object.targetAnteCards)
		if object.targetAnteCards == nil {
			object.targetAnteCards = make(map[uuid.UUID]capturedCard)
		}
		object.targetAnteCards[id] = capturedCard{id: id, visit: visit, zone: core.ZoneAnte}
	}
}

func (g *Game) anteTargetCards(targets []uuid.UUID) map[uuid.UUID]capturedCard {
	object := &StackObject{Targets: targets}
	g.captureAnteTargetCards(object)
	return object.targetAnteCards
}
