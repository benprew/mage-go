package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// SpellTargetPredicate retains one complete native singular spell target.
// The declaration stores no choices or spell departure information.
type SpellTargetPredicate struct {
	declaration *targetDeclaration
	spec        *typedSpellTarget
}

// InspectSpellTarget reads native type and color predicates and complete Boolean trees.
// Mana value can equal the inspected object's announced X.
// Battlefield and other contextual restrictions need separate contracts.
func InspectSpellTarget(reference StackObjectRef) (SpellTargetPredicate, bool) {
	binding, ok := reference.(*SpellTargetBinding)
	if !ok || binding == nil || binding.targetDeclaration == nil || binding.invalidSpellFilter {
		return SpellTargetPredicate{}, false
	}
	spec, ok := binding.spec.(*typedSpellTarget)
	if !ok || spec == nil || spec.Min() != 1 || spec.Max() != 1 || spec.targetsControlledPermanent || spec.filterCount < 0 || spec.filterCount > 1 {
		return SpellTargetPredicate{}, false
	}
	for _, filter := range spec.filters {
		if !pureSpellCardFilter(filter) {
			return SpellTargetPredicate{}, false
		}
	}
	if spec.filterCount == 1 && !pureSpellTargetFilter(spec.contextualFilter) {
		return SpellTargetPredicate{}, false
	}
	return SpellTargetPredicate{declaration: binding.targetDeclaration, spec: spec}, true
}

func pureSpellCardFilter(filter CardFilter) bool {
	if filter.fn == nil {
		return true
	}
	return pureSpellCardCharacteristics(filter.characteristics)
}

func pureSpellCardCharacteristics(node *cardCharacteristicNode) bool {
	if node == nil {
		return false
	}
	switch node.operation {
	case cardCharacteristicType:
		return node.kind >= core.TypeCreature && node.kind <= core.TypePlaneswalker
	case cardCharacteristicColor:
		return node.color >= core.White && node.color <= core.Green
	case cardCharacteristicAll, cardCharacteristicAny:
		for _, child := range node.children {
			if !pureSpellCardCharacteristics(child) {
				return false
			}
		}
		return true
	}
	return false
}

func pureSpellTargetFilter(filter SpellFilter) bool {
	switch filter.kind {
	case "card":
		return pureSpellCardFilter(filter.card)
	case "all", "any", "not":
		if len(filter.children) == 0 || filter.kind == "not" && len(filter.children) != 1 {
			return false
		}
		for _, child := range filter.children {
			if !pureSpellTargetFilter(child) {
				return false
			}
		}
		return true
	}
	return false
}

func pureSpellCardMatches(filter CardFilter, card Card) bool {
	return filter.fn == nil || filter.characteristics.matches(card)
}

func pureSpellTargetMatches(filter SpellFilter, card Card) bool {
	switch filter.kind {
	case "card":
		return pureSpellCardMatches(filter.card, card)
	case "not":
		return !pureSpellTargetMatches(filter.children[0], card)
	case "all":
		for _, child := range filter.children {
			if !pureSpellTargetMatches(child, card) {
				return false
			}
		}
		return true
	case "any":
		for _, child := range filter.children {
			if pureSpellTargetMatches(child, card) {
				return true
			}
		}
	}
	return false
}

// Select reads exact announced stack identity and current native legality.
// A nil identity with a true known result means an illegal target. Missing or
// repeated declarations and missing zone snapshots return unknown. No LKI is used.
func (predicate SpellTargetPredicate) Select(g *Game, obj *StackObject) (uuid.UUID, bool) {
	if predicate.declaration == nil || g == nil || obj == nil || g.GetPlayer(obj.Controller) == nil || len(obj.TargetSpecs) != len(obj.Targets) {
		return uuid.Nil, false
	}
	current, ok := InspectSpellTarget(&SpellTargetBinding{predicate.declaration})
	if !ok || current.spec != predicate.spec {
		return uuid.Nil, false
	}
	if predicate.spec.manaValueEqualX && obj.XValue < 0 {
		return uuid.Nil, false
	}
	index := -1
	for i, spec := range obj.TargetSpecs {
		if declaration, ok := spec.(*targetDeclaration); ok && declaration == predicate.declaration {
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
	zone, ok := obj.TargetZones[id]
	if !ok {
		return uuid.Nil, false
	}
	if zone != core.ZoneStack {
		return uuid.Nil, true
	}
	spell := g.stack.FindByID(id)
	if spell == nil || spell.leftStack || spell.IsAbility || nilDefinition(spell.Card) {
		return uuid.Nil, true
	}
	if predicate.spec.manaValueEqualX && stackSpellManaValue(spell) != obj.XValue {
		return uuid.Nil, true
	}
	for _, filter := range predicate.spec.filters {
		if !pureSpellCardMatches(filter, spellCharacteristicCard(spell)) {
			return uuid.Nil, true
		}
	}
	if predicate.spec.filterCount == 1 && !pureSpellTargetMatches(predicate.spec.contextualFilter, spellCharacteristicCard(spell)) {
		return uuid.Nil, true
	}
	return id, true
}
