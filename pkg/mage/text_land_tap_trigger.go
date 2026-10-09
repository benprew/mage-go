package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithLandTapTextRevision permits checked replacement of a printed trigger.
// The declaration must use a typed opponent land-tap event on the battlefield.
func (t *GenericTriggered) WithLandTapTextRevision() *GenericTriggered {
	t.landTapTextRevision = true
	return t
}

func (t *GenericTriggered) supportsLandTapTextRevision() bool {
	_, typed := t.eventDefinition.(opponentLandTappedPattern)
	return typed && t.eventType == core.EvtTapped && t.sourceZone == core.ZoneBattlefield && len(t.zones) == 0 && !t.isStateTrigger
}

// RevisePrintedLandTapTrigger replaces one present printed land-tap trigger.
// The caller must supply the text layer's order and lifetime separately.
func (g *Game) RevisePrintedLandTapTrigger(source *Permanent, abilityID uuid.UUID, changes ...TextChange) error {
	return g.revisePrintedTrigger(source, abilityID, func(t *GenericTriggered) bool {
		return t.landTapTextRevision && t.supportsLandTapTextRevision()
	}, changes...)
}
