package mage

// ControlledSinceCurrentTurnStart checks control since the current turn began.
// The controller's most recent turn can have a different start.
var ControlledSinceCurrentTurnStart = PermanentFilter{label: "controlled since current turn began", fn: func(p *Permanent, g *Game) bool {
	return p != nil && g != nil && !p.PhasedOut && p.turnControlGained < g.CurrentTurn()
}, query: &permanentCollectionFilterNode{operation: collectionFilterCurrentTurnControl, contextual: true}}
