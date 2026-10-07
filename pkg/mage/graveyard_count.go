package mage

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
