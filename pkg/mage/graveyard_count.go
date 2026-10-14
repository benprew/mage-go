package mage

import "github.com/google/uuid"

type graveyardCardCountValue struct {
	player PlayerRef
	filter CardFilter
}

// CountGraveyardCards reads the selected player's matching graveyard cards.
func CountGraveyardCards(player PlayerRef, filter CardFilter) ResolutionValue {
	return graveyardCardCountValue{player: player, filter: filter}
}

func (v graveyardCardCountValue) Text() string { return "number of matching graveyard cards" }
func (v graveyardCardCountValue) resolveValue(ctx *EffectContext) int {
	id, ok := v.player.resolvePlayer(ctx)
	if !ok {
		return 0
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return 0
	}
	count := 0
	for _, card := range player.Graveyard() {
		if v.filter.Match(card) {
			count++
		}
	}
	return count
}
func (v graveyardCardCountValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"player", v.player}}}
}

// CountGraveyardCardsAmong reads current graveyards of distinct selected players.
func CountGraveyardCardsAmong(players PlayerSet, filter CardFilter) ResolutionValue {
	return graveyardSetCountValue{players: players, filter: filter}
}

type graveyardSetCountValue struct {
	players PlayerSet
	filter  CardFilter
}

func (graveyardSetCountValue) Text() string {
	return "number of matching graveyard cards among players"
}
func (v graveyardSetCountValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"players", v.players}}}
}
func (v graveyardSetCountValue) resolveValue(ctx *EffectContext) int {
	count := 0
	seen := make(map[uuid.UUID]bool)
	for _, id := range v.players.resolvePlayers(ctx) {
		player := ctx.Game.GetPlayer(id)
		if player == nil || seen[id] {
			continue
		}
		seen[id] = true
		for _, card := range player.Graveyard() {
			if v.filter.Match(card) {
				count++
			}
		}
	}
	return count
}
