package mage

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type singletonCardSet struct{ card CardRef }

// SingletonCard selects one card while its reference remains valid.
func SingletonCard(card CardRef) CardSet { return singletonCardSet{card} }
func (singletonCardSet) cardSet()        {}
func (s singletonCardSet) resolveCards(ctx *EffectContext) []Card {
	card, ok := s.card.resolveCard(ctx)
	if !ok {
		return nil
	}
	return []Card{card}
}
func (s singletonCardSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "card-set", Text: "single card", children: []definitionEdge{{"card", s.card}}}
}

// ExileGraveyardCards moves all selected graveyard cards before zone events.
// References keep their saved visits. Overlapping groups move each card once.
func ExileGraveyardCards(subjects ...CardSet) ResolvingEffect {
	return &savedGraveyardExile{subjects: append([]CardSet(nil), subjects...)}
}

type savedGraveyardExile struct{ subjects []CardSet }

func (*savedGraveyardExile) Text() string { return "exile selected graveyard cards" }
func (*savedGraveyardExile) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *savedGraveyardExile) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text()}
	if len(e.subjects) == 0 {
		n.problem = "supply at least one card collection"
	}
	for i, subjects := range e.subjects {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("subjects[%d]", i), subjects})
	}
	return n
}
func (e *savedGraveyardExile) Apply(ctx *EffectContext) error {
	var selected []capturedCard
	for _, subjects := range e.subjects {
		for _, card := range subjects.resolveCards(ctx) {
			if visit := ctx.Game.graveyardVisit(card.ID()); visit != uuid.Nil {
				selected = append(selected, capturedCard{id: card.ID(), visit: visit, zone: core.ZoneGraveyard})
			}
		}
	}
	ctx.Game.exileSelectedGraveyardVisitsTogether(selected, ctx.SourceID)
	return nil
}

func (g *Game) exileSelectedGraveyardVisitsTogether(selected []capturedCard, sourceID uuid.UUID) {
	type departure struct {
		card   Card
		player uuid.UUID
	}
	var moved []departure
	counts := make(map[uuid.UUID]int)
	for _, subject := range selected {
		if subject.zone != core.ZoneGraveyard || subject.visit == uuid.Nil || g.graveyardVisit(subject.id) != subject.visit {
			continue
		}
		for _, player := range g.players {
			if card, ok := player.RemoveFromGraveyard(subject.id); ok {
				moved = append(moved, departure{card, player.PlayerID()})
				counts[player.PlayerID()]++
				break
			}
		}
	}
	for _, move := range moved {
		g.ExileCard(move.card, sourceID)
	}
	for _, move := range moved {
		g.FireEvent(core.GameEvent{Type: core.EvtZoneChange, SourceID: move.card.ID(), PlayerID: move.player, FromZone: core.ZoneGraveyard, ToZone: core.ZoneExile})
	}
	for _, player := range g.players {
		if count := counts[player.PlayerID()]; count > 0 {
			g.FireEvent(core.GameEvent{Type: core.EvtCardsLeftGraveyard, PlayerID: player.PlayerID(), Amount: count})
		}
	}
}
