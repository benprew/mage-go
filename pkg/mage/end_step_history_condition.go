package mage

import "github.com/google/uuid"

type sourceAttackHistoryCondition struct{ attacked bool }

// SourceAttackHistoryThisTurn checks the original source's attack history.
// Departure and return cannot replace the saved incarnation.
func SourceAttackHistoryThisTurn(attacked bool) StateCondition {
	return sourceAttackHistoryCondition{attacked: attacked}
}

func (sourceAttackHistoryCondition) stateCondition() {}

func (c sourceAttackHistoryCondition) checkState(g GameReader, ctx StateContext) bool {
	incarnation := ctx.SourceIncarnation
	if incarnation == uuid.Nil {
		p := g.FindPermanent(ctx.SourceID)
		if p == nil || p.PhasedOut {
			return false
		}
		incarnation = p.incarnationID
	} else {
		p := g.FindPermanentIncludingPhased(ctx.SourceID)
		if p == nil || p.incarnationID != incarnation {
			game, ok := g.(*Game)
			if !ok {
				return false
			}
			lki := game.zones.LKIByIncarnation(incarnation)
			if lki == nil || lki.ID != ctx.SourceID || lki.Snapshot == nil {
				return false
			}
		}
	}
	return g.HasAttackedThisTurnIncarnation(ctx.SourceID, incarnation) == c.attacked
}

func (c sourceAttackHistoryCondition) inspectDefinition() definitionNode {
	text := "source did not attack this turn"
	if c.attacked {
		text = "source attacked this turn"
	}
	return definitionNode{Role: "state-condition", Text: text, battlefieldSource: true}
}

type creatureDiedThisTurnCondition struct{}

// CreatureDiedThisTurnCondition includes token deaths under either player's control.
func CreatureDiedThisTurnCondition() StateCondition { return creatureDiedThisTurnCondition{} }

func (creatureDiedThisTurnCondition) stateCondition() {}
func (creatureDiedThisTurnCondition) checkState(g GameReader, _ StateContext) bool {
	return g.CreatureDeaths() > 0
}
func (creatureDiedThisTurnCondition) inspectDefinition() definitionNode {
	return definitionNode{Role: "state-condition", Text: "a creature died this turn"}
}
