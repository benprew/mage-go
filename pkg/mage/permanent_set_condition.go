package mage

import "github.com/google/uuid"

type permanentSetCountCondition struct {
	subjects PermanentSet
	minimum  int
	empty    bool
}

// PermanentSetCountAtLeast checks current membership against a fixed minimum.
func PermanentSetCountAtLeast(subjects PermanentSet, minimum int) StateCondition {
	return permanentSetCountCondition{subjects: subjects, minimum: minimum}
}

// PermanentSetIsEmpty checks that the current set has no members.
func PermanentSetIsEmpty(subjects PermanentSet) StateCondition {
	return permanentSetCountCondition{subjects: subjects, empty: true}
}

func (permanentSetCountCondition) stateCondition() {}
func (c permanentSetCountCondition) checkState(reader GameReader, state StateContext) bool {
	game, ok := reader.(*Game)
	if !ok || nilDefinition(c.subjects) {
		return false
	}
	query := *game
	query.resolution.permanentBindings = state.permanents
	query.resolution.playerBindings = state.players
	ctx := &EffectContext{Game: &query, SourceID: state.SourceID, Controller: state.ControllerID, sourceCaptured: state.SourceIncarnation != uuid.Nil, sourceIncarnation: state.SourceIncarnation}
	count := len(c.subjects.resolvePermanents(ctx))
	if c.empty {
		return count == 0
	}
	return count >= c.minimum
}
func (c permanentSetCountCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: "current permanent set count", children: []definitionEdge{{"subjects", c.subjects}}}
	if c.minimum < 0 {
		n.problem = "set count requires a nonnegative minimum"
	}
	return n
}
