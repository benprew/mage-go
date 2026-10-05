package mage

import "github.com/google/uuid"

// SourceTappedStateWithLKI checks the source state and retains departure information.
// A captured source incarnation permits a last-known state read at resolution.
func SourceTappedStateWithLKI(tapped bool) StateCondition {
	return sourceTappedLKICondition{tapped: tapped}
}

type sourceTappedLKICondition struct{ tapped bool }

func (sourceTappedLKICondition) stateCondition() {}

func (c sourceTappedLKICondition) checkState(g GameReader, ctx StateContext) bool {
	if ctx.SourceIncarnation == uuid.Nil {
		p := g.FindPermanent(ctx.SourceID)
		return p != nil && !p.PhasedOut && p.Tapped == c.tapped
	}
	if p := g.FindPermanentIncludingPhased(ctx.SourceID); p != nil && p.incarnationID == ctx.SourceIncarnation {
		return p.Tapped == c.tapped
	}
	game, ok := g.(*Game)
	if !ok {
		return false
	}
	lki := game.zones.LKIByIncarnation(ctx.SourceIncarnation)
	return lki != nil && lki.ID == ctx.SourceID && lki.Snapshot != nil && lki.Snapshot.Tapped == c.tapped
}

func (c sourceTappedLKICondition) inspectDefinition() definitionNode {
	text := "source is untapped, with last-known state"
	if c.tapped {
		text = "source is tapped, with last-known state"
	}
	return definitionNode{Role: "state-condition", Text: text, battlefieldSource: true}
}
