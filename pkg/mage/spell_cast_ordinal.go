package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AfterFirstInstant matches the caster's second and later instants each turn.
// The saved event count includes casts before this source entered.
func (p spellCastPattern) AfterFirstInstant() spellCastPattern {
	if p.afterFirstInstant {
		p.duplicateInstantOrdinal = true
	}
	p.afterFirstInstant = true
	return p
}

type spellCastAfterFirstInstant struct{}

func (spellCastAfterFirstInstant) CheckTriggerCond(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	return event.Type == core.EvtSpellCast && event.CastInstantOrdinal > 1
}
