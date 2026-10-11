package mage

import (
	"slices"

	"github.com/google/uuid"
)

// SpellNotTargetedByPermanentAbility checks abilities that remain on the stack.
// Source information uses the exact battlefield visit, including departure data.
func SpellNotTargetedByPermanentAbility(filter PermanentFilter, excludeSource bool) SpellFilter {
	return SpellFilter{kind: "ability-target-exclusion", permanents: filter, excludeSource: excludeSource}
}

func (g *Game) spellTargetedByPermanentAbility(spellID uuid.UUID, source Card, filter PermanentFilter, excludeSource bool) bool {
	var excluded capturedPermanent
	if source != nil {
		if view, ok := source.(targetSourceLKI); ok {
			excluded = capturedPermanent{source.ID(), view.incarnation}
		} else if p := g.FindPermanentIncludingPhased(source.ID()); p != nil {
			excluded = capturedPermanent{p.ID(), p.incarnationID}
		}
	}
	for _, object := range g.stack.Objects() {
		if !object.IsAbility || object.sourceIncarnation == uuid.Nil || !slices.Contains(object.Targets, spellID) {
			continue
		}
		identity := capturedPermanent{object.SourceID, object.sourceIncarnation}
		if excludeSource && identity == excluded {
			continue
		}
		p, lki := g.readPermanentIncarnationInformation(identity)
		if p == nil && lki != nil {
			p = lki.Snapshot
		}
		if p != nil && filter.Match(p, g) {
			return true
		}
	}
	return false
}
