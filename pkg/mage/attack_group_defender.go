package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithDefendingPlayer saves the defending player of a declared attack group.
// In a two-player game, this player is the declaring player's opponent.
func (p controlledAttackBatchPattern) WithDefendingPlayer(defender *PlayerBinding) controlledAttackBatchPattern {
	if p.captureDefender {
		p.duplicateDefender = true
	}
	p.defender = defender
	p.captureDefender = true
	return p
}

func (p controlledAttackBatchPattern) captureDetectedEvent(event *core.GameEvent, g GameReader, _ uuid.UUID) []Effect {
	if !p.captureDefender {
		return nil
	}
	defender := uuid.Nil
	if g.GetPlayer(event.PlayerID) != nil {
		if opponent := g.GetOpponent(event.PlayerID); opponent != nil {
			defender = opponent.PlayerID()
		}
	}
	return []Effect{&eventPlayerCapture{binding: p.defender, player: defender}}
}
