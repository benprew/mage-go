package mage

// ControllerControls checks the subject's current controller and battlefield.
func ControllerControls(filter PermanentFilter) PermanentFilter {
	condition := PlayerControls(You, filter)
	return NewPermanentFilter("controller controls "+filter.Text(), func(subject *Permanent, g *Game) bool {
		if subject == nil || g == nil {
			return false
		}
		current := g.FindPermanent(subject.ID())
		if current == nil || current.PhasedOut || current.incarnationID != subject.incarnationID || g.GetPlayer(current.ControllerID()) == nil {
			return false
		}
		return condition.checkState(g, StateContext{ControllerID: current.ControllerID()})
	})
}
