package mage

import "github.com/google/uuid"

type permanentTargetPlayerDomain struct {
	controllerOnly  bool
	opponentOnly    bool
	controllerOwned bool
	opponentOwned   bool
}

func (t *PermanentTarget) playerDomain() permanentTargetPlayerDomain {
	return permanentTargetPlayerDomain{
		controllerOnly: t.controllerOnly, opponentOnly: t.opponentOnly,
		controllerOwned: t.controllerOwned, opponentOwned: t.opponentOwned,
	}
}

func (d permanentTargetPlayerDomain) matches(p *Permanent, controller uuid.UUID) bool {
	return (!d.controllerOnly || p.ControllerID() == controller) &&
		(!d.opponentOnly || p.ControllerID() != controller) &&
		(!d.controllerOwned || p.Card.Owner() == controller) &&
		(!d.opponentOwned || p.Card.Owner() != controller)
}
