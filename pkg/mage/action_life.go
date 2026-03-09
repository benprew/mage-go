package mage

import "github.com/google/uuid"

// LifeGainAction represents a player about to gain life.
type LifeGainAction struct {
	source   uuid.UUID
	playerID uuid.UUID
	amount   int
}

func NewLifeGainAction(source, playerID uuid.UUID, amount int) *LifeGainAction {
	return &LifeGainAction{
		source:   source,
		playerID: playerID,
		amount:   amount,
	}
}

func (a *LifeGainAction) ActionSource() uuid.UUID { return a.source }
func (a *LifeGainAction) PlayerID() uuid.UUID     { return a.playerID }
func (a *LifeGainAction) Amount() int             { return a.amount }
