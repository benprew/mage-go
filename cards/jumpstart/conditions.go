package jumpstart

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// controllerDidNotAttackThisTurn is satisfied when no creature controlled by
// the trigger's controller has attacked this turn. Used by Curious Obsession's
// "if you didn't attack with a creature this turn, sacrifice this Aura" check.
type controllerDidNotAttackThisTurn struct{}

func (controllerDidNotAttackThisTurn) CheckTriggerCond(_ *GameEvent, g GameReader, _ uuid.UUID, controllerID uuid.UUID) bool {
	for _, p := range g.FilterBattlefield(And(IsCreature, ControlledBy(controllerID))) {
		if g.HasAttackedThisTurn(p.ID()) {
			return false
		}
	}
	return true
}

// anotherAuraAttachedTo returns true if any aura other than self is attached
// to host. Used by Face of Divinity for "as long as another Aura is attached".
func anotherAuraAttachedTo(g *Game, self, host *Permanent) bool {
	if host == nil {
		return false
	}
	for _, p := range g.FilterBattlefield(HasSubType("Aura")) {
		if p.ID() == self.ID() {
			continue
		}
		if p.AttachedTo == host.ID() {
			return true
		}
	}
	return false
}
