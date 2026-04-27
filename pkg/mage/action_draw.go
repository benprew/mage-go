package mage

import "github.com/google/uuid"

// DrawCardAction represents a player about to draw a card.
type DrawCardAction struct {
	source       uuid.UUID
	playerID     uuid.UUID
	isNormalDraw bool // true for the draw-step draw, false for effect draws
}

func NewDrawCardAction(source, playerID uuid.UUID, isNormalDraw bool) *DrawCardAction {
	return &DrawCardAction{
		source:       source,
		playerID:     playerID,
		isNormalDraw: isNormalDraw,
	}
}

func (a *DrawCardAction) ActionSource() uuid.UUID { return a.source }
func (a *DrawCardAction) PlayerID() uuid.UUID     { return a.playerID }
func (a *DrawCardAction) IsNormalDraw() bool      { return a.isNormalDraw }
