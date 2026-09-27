package mage

import "github.com/google/uuid"

type highestLifePlayerRef struct{}

// PlayerWithUniqueHighestLife selects the sole player with the most life.
// A tie has no player to select.
func PlayerWithUniqueHighestLife() PlayerRef { return highestLifePlayerRef{} }

func (highestLifePlayerRef) playerRef()       {}
func (highestLifePlayerRef) damageRecipient() {}
func (r highestLifePlayerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (highestLifePlayerRef) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	players := ctx.Game.AllPlayers()
	if len(players) < 2 {
		return uuid.Nil, false
	}
	best := players[0]
	tied := false
	for _, player := range players[1:] {
		switch {
		case player.Life() > best.Life():
			best = player
			tied = false
		case player.Life() == best.Life():
			tied = true
		}
	}
	if tied {
		return uuid.Nil, false
	}
	return best.PlayerID(), true
}
func (highestLifePlayerRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "player with the most life"}
}
