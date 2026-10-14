package mage

import "github.com/google/uuid"

type playerControlsSetCondition struct {
	player   PlayerRef
	subjects PermanentSet
}

// PlayerControlsAny checks current control of a member of the live set.
// The exact source visit must remain present for the query.
func PlayerControlsAny(player PlayerRef, subjects PermanentSet) StateCondition {
	return playerControlsSetCondition{player, subjects}
}
func (playerControlsSetCondition) stateCondition() {}
func (c playerControlsSetCondition) inspectDefinition() definitionNode {
	n := playerControlsCondition{player: c.player}.inspectDefinition()
	n.battlefieldSource = true
	n.children = append(n.children, definitionEdge{"subjects", c.subjects})
	return n
}
func (c playerControlsSetCondition) checkState(reader GameReader, ctx StateContext) bool {
	g, ok := reader.(*Game)
	if !ok || nilDefinition(c.subjects) {
		return false
	}
	source := g.FindPermanent(ctx.SourceID)
	if source == nil || (ctx.SourceIncarnation != uuid.Nil && source.incarnationID != ctx.SourceIncarnation) {
		return false
	}
	player := resolveStatePlayer(g, ctx, c.player)
	if player == uuid.Nil || g.GetPlayer(player) == nil {
		return false
	}
	query := &EffectContext{Game: g, SourceID: source.ID(), Controller: ctx.ControllerID, sourceCaptured: true, sourceIncarnation: source.incarnationID}
	for _, p := range c.subjects.resolvePermanents(query) {
		if p.ControllerID() == player {
			return true
		}
	}
	return false
}
