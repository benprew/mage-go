package jumpstart

import (
	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
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

// eventSourcePowerGreaterThanAllOthers is satisfied when the event's source
// permanent is a creature whose current power is strictly greater than every
// other creature on the battlefield. Used by Selvala, Heart of the Wilds:
// "Whenever another creature enters, if its power is greater than each other
// creature's power, you may draw a card." Excludes the entering creature
// itself from the comparison.
type eventSourcePowerGreaterThanAllOthers struct{}

func (eventSourcePowerGreaterThanAllOthers) CheckTriggerCond(evt *GameEvent, g GameReader, _, _ uuid.UUID) bool {
	enterer := g.FindPermanent(evt.SourceID)
	if enterer == nil || !enterer.HasType(TypeCreature) {
		return false
	}
	enterPow := enterer.CurrentPower(g)
	for _, p := range g.FilterBattlefield(IsCreature) {
		if p.ID() == enterer.ID() {
			continue
		}
		if p.CurrentPower(g) >= enterPow {
			return false
		}
	}
	return true
}

// opponentDealt3PlusDamageThisTurnCond is satisfied when an opponent of the
// trigger's controller has been dealt 3 or more damage this turn (any source,
// combat or noncombat). Shared by Pia Nalaar, Consul of Revival and Lightning
// Phoenix's graveyard end-step triggers.
type opponentDealt3PlusDamageThisTurnCond struct{}

func (opponentDealt3PlusDamageThisTurnCond) CheckTriggerCond(_ *GameEvent, g GameReader, _, controllerID uuid.UUID) bool {
	for _, p := range g.AllPlayers() {
		if p.PlayerID() == controllerID {
			continue
		}
		if g.DamageTakenByPlayer(p.PlayerID()) >= 3 {
			return true
		}
	}
	return false
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
