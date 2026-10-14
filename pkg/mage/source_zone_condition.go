package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// GraveyardCardHasAtLeastAbove checks the cards above one exact graveyard visit.
func GraveyardCardHasAtLeastAbove(card CardRef, player PlayerRef, minimum int, filter CardFilter) StateCondition {
	return graveyardAboveCondition{card, player, minimum, filter}
}

type graveyardAboveCondition struct {
	card    CardRef
	player  PlayerRef
	minimum int
	filter  CardFilter
}

func (graveyardAboveCondition) stateCondition() {}
func (c graveyardAboveCondition) checkState(reader GameReader, state StateContext) bool {
	ctx := cardConditionContext(reader, state)
	if ctx == nil {
		return false
	}
	card, ok := c.card.resolveCard(ctx)
	if !ok {
		return false
	}
	id, ok := c.player.resolvePlayer(ctx)
	if !ok {
		return false
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return false
	}
	found, count := false, 0
	for _, member := range player.Graveyard() {
		if found && c.filter.Match(member) {
			count++
		}
		if member.ID() == card.ID() {
			found = true
		}
	}
	return found && count >= c.minimum
}
func (c graveyardAboveCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: "matching cards above a graveyard card", children: []definitionEdge{{"card", c.card}, {"player", c.player}}}
	if c.minimum < 0 {
		n.problem = "graveyard count must be nonnegative"
	}
	if _, ok := c.card.(savedCardRef); !ok {
		n.problem = "graveyard condition requires a saved card reference"
	}
	return n
}

// CardMatchesFilter checks the current card at its saved zone visit.
func CardMatchesFilter(card CardRef, filter CardFilter) StateCondition {
	return cardFilterCondition{card, filter}
}

type cardFilterCondition struct {
	card   CardRef
	filter CardFilter
}

func (cardFilterCondition) stateCondition() {}
func (c cardFilterCondition) checkState(reader GameReader, state StateContext) bool {
	ctx := cardConditionContext(reader, state)
	if ctx == nil {
		return false
	}
	card, ok := c.card.resolveCard(ctx)
	return ok && c.filter.Match(card)
}
func (c cardFilterCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: "saved card matches filter", children: []definitionEdge{{"card", c.card}}}
	if _, ok := c.card.(savedCardRef); !ok {
		n.problem = "card condition requires a saved card reference"
	}
	return n
}

func cardConditionContext(reader GameReader, state StateContext) *EffectContext {
	g, ok := reader.(*Game)
	if !ok {
		return nil
	}
	query := *g
	query.resolution.cardBindings = state.cards
	query.resolution.playerBindings = state.players
	return &EffectContext{Game: &query, SourceID: state.SourceID, Controller: state.ControllerID}
}

func (g *Game) sourceZoneCardFacts(t *GenericTriggered, sourceID uuid.UUID) map[*CardBinding]capturedCard {
	if t.sourceCardBinding == nil {
		return nil
	}
	visit := g.graveyardVisit(sourceID)
	if t.sourceZone == core.ZoneExile {
		object := g.FindExiledCard(sourceID)
		if object == nil || object.FaceDown {
			return nil
		}
		visit = object.Visit
	}
	return map[*CardBinding]capturedCard{t.sourceCardBinding: {id: sourceID, visit: visit, zone: t.sourceZone}}
}
