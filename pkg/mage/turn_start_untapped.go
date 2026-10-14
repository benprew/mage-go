package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func (g *Game) captureTurnStartUntapped() {
	t := &g.trackers.Turn
	active := g.ActivePlayerObj().PlayerID()
	if t.untappedStartTurn == g.CurrentTurn() && t.untappedStartPlayer == active {
		return
	}
	g.ApplyContinuousEffects()
	t.untappedStartTurn = g.CurrentTurn()
	t.untappedStartPlayer = active
	t.untappedStartVisits = make(map[capturedPermanent]bool)
	t.untappedLandsAtTurnStart = make(map[uuid.UUID]int)
	for _, p := range g.zones.battlefield {
		if p.PhasedOut {
			continue
		}
		t.untappedStartVisits[capturedPermanent{p.ID(), p.incarnationID}] = !p.Tapped
		if !p.Tapped && p.HasType(core.TypeLand) {
			t.untappedLandsAtTurnStart[p.ControllerID()]++
		}
	}
}

// StartedTurnUntapped checks the saved battlefield visit in the current turn.
func (g *Game) StartedTurnUntapped(id, incarnation uuid.UUID) bool {
	t := &g.trackers.Turn
	return t.untappedStartTurn == g.CurrentTurn() && t.untappedStartPlayer == g.ActivePlayerObj().PlayerID() && t.untappedStartVisits[capturedPermanent{id, incarnation}]
}

// UntappedLandCountAtTurnStart reads the selected player's saved land count.
func UntappedLandCountAtTurnStart(player PlayerRef) ResolutionValue {
	return turnStartLandCount{player: player}
}

type turnStartLandCount struct{ player PlayerRef }

func (turnStartLandCount) Text() string { return "untapped lands at turn start" }
func (v turnStartLandCount) resolveValue(ctx *EffectContext) int {
	id, ok := v.player.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(id) == nil {
		return 0
	}
	return ctx.Game.UntappedLandsAtTurnStart(id)
}
func (v turnStartLandCount) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"player", v.player}}}
}
func (v turnStartLandCount) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(v.player)
}

// PermanentStartedTurnUntapped checks the source's saved battlefield visit.
// Departure does not change this historical fact.
func PermanentStartedTurnUntapped(subject PermanentRef) StateCondition {
	return turnStartUntappedCondition{subject: subject}
}

type turnStartUntappedCondition struct{ subject PermanentRef }

func (turnStartUntappedCondition) stateCondition() {}
func (c turnStartUntappedCondition) checkState(g GameReader, ctx StateContext) bool {
	if _, ok := c.subject.(sourceRef); !ok {
		return false
	}
	incarnation := ctx.SourceIncarnation
	if incarnation == uuid.Nil {
		source := g.FindPermanent(ctx.SourceID)
		if source == nil || source.PhasedOut {
			return false
		}
		incarnation = source.incarnationID
	}
	return g.StartedTurnUntapped(ctx.SourceID, incarnation)
}
func (c turnStartUntappedCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: "source started turn untapped", battlefieldSource: true}
	if _, ok := c.subject.(sourceRef); !ok {
		n.problem = "state condition requires This"
	}
	return n
}
