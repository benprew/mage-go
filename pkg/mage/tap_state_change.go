package mage

// InspectTapStateChange checks whether a native tap instruction can change state.
// Subjects must be current, selected battlefield visits. Invalid visits or an
// invalid operation return unknown. Layers and target legality must be current.
// A possible transition is not a prediction of its replacement or final result.
// This query does not run callbacks, make choices or change counters.
func (g *Game) InspectTapStateChange(operation TapOperationKind, subjects []PermanentVisit) (mayChange, known bool) {
	if g == nil || (operation != TapOperation && operation != UntapOperation && operation != MayTapOrUntapOperation) {
		return false, false
	}
	for _, visit := range subjects {
		p := g.FindPermanent(visit.ID)
		if p == nil || p.PhasedOut || p.incarnationID != visit.Incarnation {
			return false, false
		}
		mayChange = mayChange || operation == MayTapOrUntapOperation || (operation == TapOperation && !p.Tapped) || (operation == UntapOperation && p.Tapped)
	}
	return mayChange, true
}
