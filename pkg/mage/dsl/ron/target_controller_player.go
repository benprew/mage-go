package ron

import "github.com/benprew/mage-go/pkg/mage"

type targetControllerPlayerPlan struct {
	target *permanentTargetPlan
}

func (p targetControllerPlayerPlan) build(bindings instanceBindings) mage.PlayerRef {
	return mage.ControllerOfWithLKI(bindings.permanents[p.target])
}
