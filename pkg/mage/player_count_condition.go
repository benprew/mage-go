package mage

// OpponentControlsMoreThanYou compares current control of matching permanents.
func OpponentControlsMoreThanYou(filter PermanentFilter) StateCondition {
	return opponentControlsMoreCondition{filter: filter}
}

type opponentControlsMoreCondition struct{ filter PermanentFilter }

func (opponentControlsMoreCondition) stateCondition() {}
func (c opponentControlsMoreCondition) checkState(g GameReader, ctx StateContext) bool {
	if g.GetPlayer(ctx.ControllerID) == nil {
		return false
	}
	opponent := g.GetOpponent(ctx.ControllerID)
	if opponent == nil {
		return false
	}
	own, other := 0, 0
	for _, p := range g.FilterBattlefield(c.filter) {
		if p.ControllerID() == ctx.ControllerID {
			own++
		}
		if p.ControllerID() == opponent.PlayerID() {
			other++
		}
	}
	return other > own
}
func (opponentControlsMoreCondition) inspectDefinition() definitionNode {
	return definitionNode{Role: "state-condition", Text: "opponent controls more matching permanents than controller"}
}
