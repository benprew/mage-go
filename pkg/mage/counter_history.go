package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type counterPlacementKey struct {
	source    capturedPermanent
	recipient capturedPermanent
	counter   core.CounterType
}

func (g *Game) recordCounterPlacement(source capturedPermanent, recipient *Permanent, counter core.CounterType, amount int) {
	if source.incarnation == uuid.Nil || amount <= 0 {
		return
	}
	if g.zones.counterPlacements == nil {
		g.zones.counterPlacements = make(map[counterPlacementKey]bool)
	}
	g.zones.counterPlacements[counterPlacementKey{source, capturedPermanent{recipient.ID(), recipient.incarnationID}, counter}] = true
}

func (g *Game) counterPlacementSource(sourceID uuid.UUID) capturedPermanent {
	if p := g.FindPermanentIncludingPhased(sourceID); p != nil {
		return capturedPermanent{sourceID, p.incarnationID}
	}
	if p := g.zones.EnteringPermanent(); p != nil && p.ID() == sourceID {
		return capturedPermanent{sourceID, p.incarnationID}
	}
	return capturedPermanent{}
}

type counterHistoryPermanents struct {
	source     PermanentRef
	counter    core.CounterType
	candidates PermanentSet
}

// PermanentsGivenCountersBy selects visits that received this kind of counter from the source.
// Counter removal does not erase a placement.
func PermanentsGivenCountersBy(source PermanentRef, counter core.CounterType, candidates PermanentSet) PermanentSet {
	return counterHistoryPermanents{source, counter, candidates}
}
func (counterHistoryPermanents) permanentSet() {}
func (s counterHistoryPermanents) resolvePermanents(ctx *EffectContext) []*Permanent {
	ref, ok := s.source.(permanentInformationReference)
	if !ok || nilDefinition(ref) || nilDefinition(s.candidates) {
		return nil
	}
	source, ok := ref.informationIdentity(ctx)
	if !ok || source.incarnation == uuid.Nil {
		return nil
	}
	var selected []*Permanent
	for _, p := range s.candidates.resolvePermanents(ctx) {
		if p != nil && !p.PhasedOut && ctx.Game.zones.counterPlacements[counterPlacementKey{source, capturedPermanent{p.ID(), p.incarnationID}, s.counter}] {
			selected = append(selected, p)
		}
	}
	return selected
}
func (s counterHistoryPermanents) inspectDefinition() definitionNode {
	n := permanentInformationNode(s.source, "permanent-set", "permanents given counters by source")
	n.children = append(n.children, definitionEdge{"candidates", s.candidates})
	if s.counter < 0 || s.counter >= core.NumCounters {
		n.problem = "supply a valid counter type"
	}
	return n
}
