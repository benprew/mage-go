package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// InspectDestructionStateChange checks whether direct destruction can propose an action.
// Subjects must be current, selected battlefield visits. Layers and target
// legality must be current. Invalid visits return unknown. Empty groups and
// indestructible visits cannot propose destruction. All other visits retain a
// possible change, including regeneration and canceled destruction. This query
// does not run callbacks, choose replacements or consume shields.
func (g *Game) InspectDestructionStateChange(subjects []PermanentVisit) (mayChange, known bool) {
	if g == nil {
		return false, false
	}
	for _, visit := range subjects {
		p := g.FindPermanent(visit.ID)
		if p == nil || p.PhasedOut || p.incarnationID != visit.Incarnation {
			return false, false
		}
		mayChange = mayChange || !p.HasKeyword(core.Indestructible)
	}
	return mayChange, true
}
