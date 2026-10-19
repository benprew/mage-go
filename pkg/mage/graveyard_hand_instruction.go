package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// GraveyardHandInstruction retains one graveyard card return and its reference.
// Execution uses the shared owner destination and graveyard movement services.
type GraveyardHandInstruction struct{ Subject CardRef }

// Effect builds the native return. A missing subject returns nil.
func (instruction GraveyardHandInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subject) {
		return nil
	}
	return ReturnCardToHand(instruction.Subject)
}

// InspectGraveyardHandInstruction retains a native return without reading its card.
// Consumers must check the reference before they use it in a pure query.
func InspectGraveyardHandInstruction(effect ResolvingEffect) (GraveyardHandInstruction, bool) {
	e, ok := effect.(*returnCardToHandEffect)
	if !ok || e == nil || nilDefinition(e.card) {
		return GraveyardHandInstruction{}, false
	}
	return GraveyardHandInstruction{Subject: e.card}, true
}

// GraveyardTargetPredicate retains one complete singular graveyard target.
// A zero value cannot select a card. No chosen state is saved here.
type GraveyardTargetPredicate struct {
	declaration  *targetDeclaration
	spec         Target
	filters      []CardFilter
	anyGraveyard bool
}

// InspectGraveyardTarget accepts native card characteristics and graveyard scope.
// Optional, numeric, source-excluding and opaque restrictions need other contracts.
func InspectGraveyardTarget(reference CardRef) (GraveyardTargetPredicate, bool) {
	binding, ok := reference.(*CardTargetBinding)
	if !ok || binding == nil || binding.targetDeclaration == nil || binding.optionalGraveyardCount != 0 {
		return GraveyardTargetPredicate{}, false
	}
	p := GraveyardTargetPredicate{declaration: binding.targetDeclaration, spec: binding.spec}
	switch spec := binding.spec.(type) {
	case *GraveyardCardTarget:
		if spec == nil || spec.Min() != 1 || spec.Max() != 1 || spec.excludeSource {
			return GraveyardTargetPredicate{}, false
		}
		p.filters = spec.Filters
	case *AnyGraveyardCardTarget:
		if spec == nil || spec.Min() != 1 || spec.Max() != 1 || spec.graveyardEnchant {
			return GraveyardTargetPredicate{}, false
		}
		p.filters = spec.Filters
		p.anyGraveyard = true
	default:
		return GraveyardTargetPredicate{}, false
	}
	for _, filter := range p.filters {
		if !pureSpellCardFilter(filter) {
			return GraveyardTargetPredicate{}, false
		}
	}
	return p, true
}

// Select reads the announced graveyard visit and current native target legality.
// Missing snapshots return unknown. A nil identity with known true is illegal.
// No card predicate callback or last known information is used.
func (p GraveyardTargetPredicate) Select(g *Game, obj *StackObject) (uuid.UUID, bool) {
	if p.declaration == nil || g == nil || obj == nil || g.GetPlayer(obj.Controller) == nil || len(obj.TargetSpecs) != len(obj.Targets) {
		return uuid.Nil, false
	}
	current, ok := InspectGraveyardTarget(&CardTargetBinding{p.declaration})
	if !ok || current.spec != p.spec {
		return uuid.Nil, false
	}
	p = current
	index := -1
	for i, spec := range obj.TargetSpecs {
		if declaration, ok := spec.(*targetDeclaration); ok && declaration == p.declaration {
			if index != -1 {
				return uuid.Nil, false
			}
			index = i
		}
	}
	if index < 0 {
		return uuid.Nil, false
	}
	id := obj.Targets[index]
	if id == uuid.Nil {
		return uuid.Nil, true
	}
	zone, hasZone := obj.TargetZones[id]
	visit := obj.targetIncarnations[id]
	if !hasZone || visit == uuid.Nil {
		return uuid.Nil, false
	}
	if zone != core.ZoneGraveyard || g.graveyardVisit(id) != visit {
		return uuid.Nil, true
	}
	for _, player := range g.players {
		if !p.anyGraveyard && player.PlayerID() != obj.Controller {
			continue
		}
		for _, card := range player.Graveyard() {
			if card.ID() != id {
				continue
			}
			for _, filter := range p.filters {
				if !pureSpellCardMatches(filter, card) {
					return uuid.Nil, true
				}
			}
			return id, true
		}
	}
	return uuid.Nil, true
}
