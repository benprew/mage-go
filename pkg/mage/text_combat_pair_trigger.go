package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithCombatPairTextRevision permits checked replacement of a printed trigger.
// The declaration must use a typed combat-pair event on the battlefield.
func (t *GenericTriggered) WithCombatPairTextRevision() *GenericTriggered {
	t.combatPairTextRevision = true
	return t
}

func (t *GenericTriggered) supportsCombatPairTextRevision() bool {
	pattern, typed := t.eventDefinition.(filteredCombatPairPattern)
	if !typed || t.eventType != pattern.kind || t.sourceZone != core.ZoneBattlefield || len(t.zones) != 0 || t.isStateTrigger {
		return false
	}
	switch pattern.kind {
	case core.EvtDeclaredBlocker, core.EvtCreatureBlockedBy, core.EvtCombatPair:
	default:
		return false
	}
	return pattern.inspectDefinition().problem == ""
}

// RevisePrintedCombatPairTrigger replaces one present printed combat-pair trigger.
// The caller must supply the text layer's order and lifetime separately.
func (g *Game) RevisePrintedCombatPairTrigger(source *Permanent, abilityID uuid.UUID, changes ...TextChange) error {
	var kind core.EventType
	checked := false
	return g.revisePrintedTrigger(source, abilityID, func(t *GenericTriggered) bool {
		if !t.combatPairTextRevision || !t.supportsCombatPairTextRevision() {
			return false
		}
		if !checked {
			kind, checked = t.eventType, true
		}
		return t.eventType == kind
	}, changes...)
}
