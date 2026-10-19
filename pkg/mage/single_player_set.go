package mage

import "github.com/google/uuid"

// OnePlayer selects the present player supplied by a typed reference.
func OnePlayer(player PlayerRef) PlayerSet { return singlePlayerSet{player} }

type singlePlayerSet struct{ player PlayerRef }

func (singlePlayerSet) playerSet() {}
func (s singlePlayerSet) resolvePlayers(ctx *EffectContext) []uuid.UUID {
	id, ok := s.player.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(id) == nil {
		return nil
	}
	return []uuid.UUID{id}
}
func (s singlePlayerSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-set", children: []definitionEdge{{"player", s.player}}}
}
