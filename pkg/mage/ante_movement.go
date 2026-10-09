package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AnteCardSet selects exact visits in the shared ante zone.
type AnteCardSet interface {
	anteCardSet()
	resolveAnteCards(*EffectContext) []capturedCard
}

type ownedAnteCards struct{ owner PlayerRef }

// AnteCardsOwnedBy selects cards with the specified current owner.
func AnteCardsOwnedBy(owner PlayerRef) AnteCardSet { return ownedAnteCards{owner} }
func (ownedAnteCards) anteCardSet()                {}
func (s ownedAnteCards) resolveAnteCards(ctx *EffectContext) []capturedCard {
	owner, ok := s.owner.resolvePlayer(ctx)
	if !ok || !ctx.Game.anteEnabled || ctx.Game.anteSettled {
		return nil
	}
	cards, err := ctx.Game.AnteCardsOwnedBy(owner)
	if err != nil {
		return nil
	}
	var result []capturedCard
	for _, card := range cards {
		if visit := ctx.Game.anteVisit(card.ID()); visit != uuid.Nil {
			result = append(result, capturedCard{id: card.ID(), visit: visit, zone: core.ZoneAnte})
		}
	}
	return result
}
func (s ownedAnteCards) inspectDefinition() definitionNode {
	return definitionNode{Role: "ante-card-set", children: []definitionEdge{{"owner", s.owner}}}
}

type otherAnteCards struct{ subjects AnteCardSet }

// OtherAnteCards excludes the physical source card after its move to ante.
func OtherAnteCards(subjects AnteCardSet) AnteCardSet { return otherAnteCards{subjects} }
func (otherAnteCards) anteCardSet()                   {}
func (s otherAnteCards) resolveAnteCards(ctx *EffectContext) []capturedCard {
	return slices.DeleteFunc(slices.Clone(s.subjects.resolveAnteCards(ctx)), func(card capturedCard) bool {
		return card.id == ctx.SourceID
	})
}
func (s otherAnteCards) inspectDefinition() definitionNode {
	return definitionNode{Role: "ante-card-set", children: []definitionEdge{{"subjects", s.subjects}}}
}

type anteGraveyardEffect struct{ subjects AnteCardSet }

// PutAnteCardsInGraveyards moves selected visits before it sends zone events.
func PutAnteCardsInGraveyards(subjects AnteCardSet) ResolvingEffect {
	return &anteGraveyardEffect{subjects}
}
func (*anteGraveyardEffect) Text() string {
	return "put the selected ante cards into their owners' graveyards"
}
func (*anteGraveyardEffect) Properties() EffectProperties { return EffectProperties{Mass: true} }
func (e *anteGraveyardEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"subjects", e.subjects}}}
}
func (e *anteGraveyardEffect) Apply(ctx *EffectContext) error {
	ctx.Game.putAnteCardsInGraveyards(e.subjects.resolveAnteCards(ctx))
	return nil
}

func (g *Game) putAnteCardsInGraveyards(subjects []capturedCard) {
	byOwner := make(map[uuid.UUID][]Card)
	visits := make(map[uuid.UUID]uuid.UUID)
	for _, subject := range subjects {
		if subject.zone != core.ZoneAnte {
			continue
		}
		card, ok := g.resolveAnteCard(subject.id, subject.visit)
		if !ok || g.GetPlayer(card.Owner()) == nil || visits[card.ID()] != uuid.Nil {
			continue
		}
		visits[card.ID()] = subject.visit
		byOwner[card.Owner()] = append(byOwner[card.Owner()], card)
	}
	for _, owner := range []Player{g.ActivePlayerObj(), g.NonActivePlayerObj()} {
		byOwner[owner.PlayerID()] = g.orderCardsForGraveyard(owner, byOwner[owner.PlayerID()])
	}
	var events []core.GameEvent
	for _, owner := range g.players {
		for _, card := range byOwner[owner.PlayerID()] {
			current, ok := g.resolveAnteCard(card.ID(), visits[card.ID()])
			if !ok || current.Owner() != owner.PlayerID() {
				continue
			}
			if removed, ok := g.removeAnteVisit(capturedCard{id: card.ID(), visit: visits[card.ID()], zone: core.ZoneAnte}); ok {
				owner.AddToGraveyard(removed)
				events = append(events, core.GameEvent{Type: core.EvtZoneChange, SourceID: card.ID(), PlayerID: owner.PlayerID(), FromZone: core.ZoneAnte, ToZone: core.ZoneGraveyard})
			}
		}
	}
	for _, event := range events {
		g.FireEvent(event)
	}
}

func (g *Game) removeAnteVisit(subject capturedCard) (Card, bool) {
	if subject.zone != core.ZoneAnte {
		return nil, false
	}
	if _, ok := g.resolveAnteCard(subject.id, subject.visit); !ok {
		return nil, false
	}
	for _, storage := range g.players {
		if card, ok := storage.RemoveFromAnte(subject.id); ok {
			return card, true
		}
	}
	return nil, false
}

func (e *anteGraveyardEffect) refersToSourcePermanent() bool {
	return anteCardSetReadsSourcePermanent(e.subjects)
}
func anteCardSetReadsSourcePermanent(subjects AnteCardSet) bool {
	switch s := subjects.(type) {
	case ownedAnteCards:
		return playerRefReadsSourcePermanent(s.owner)
	case otherAnteCards:
		return anteCardSetReadsSourcePermanent(s.subjects)
	default:
		return false
	}
}
