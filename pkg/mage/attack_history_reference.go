package mage

// AttackedThisTurn keeps attack history separate for each permanent incarnation.
var AttackedThisTurn = NewPermanentFilter("attacked this turn", func(p *Permanent, g *Game) bool {
	return p != nil && g != nil && g.HasAttackedThisTurnIncarnation(p.ID(), p.incarnationID)
})

// DidntAttackThisTurn excludes objects that declared an attack this turn.
var DidntAttackThisTurn = NewPermanentFilter("did not attack this turn", func(p *Permanent, g *Game) bool {
	return p != nil && g != nil && !AttackedThisTurn.Match(p, g)
})

type targetPlayerAttackedThisTurnOption struct{}

// TargetPlayerAttackedThisTurn checks history at announcement and resolution.
func TargetPlayerAttackedThisTurn() PlayerTargetOption         { return targetPlayerAttackedThisTurnOption{} }
func (targetPlayerAttackedThisTurnOption) playerTargetOption() {}
func (targetPlayerAttackedThisTurnOption) applyPlayerTarget(target Target) Target {
	switch target := target.(type) {
	case *PlayerTarget:
		target.attackedThisTurn = true
	case *OpponentTarget:
		target.attackedThisTurn = true
	}
	return target
}
func (targetPlayerAttackedThisTurnOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-player", Text: "attacked this turn"}
}
