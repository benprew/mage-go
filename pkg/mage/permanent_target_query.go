package mage

import (
	"strings"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PermanentTargetPredicate retains one complete characteristic, state and player target declaration.
// A zero value cannot select targets. The declaration has no chosen state.
type PermanentTargetPredicate struct {
	declaration                  *targetDeclaration
	filters                      []*permanentCollectionFilterNode
	players                      permanentTargetPlayerDomain
	excludeSource                bool
	toughnessLessThanSourcePower bool
	powerLESourcePower           bool
	attachedToControllerOnly     bool
	manaValueEqualX              bool
}

// InspectPermanentTarget reads a singular characteristic, state and player target reference.
// Player domains can restrict current ownership or control to you or an opponent.
// Mana value can equal the inspected object's announced X.
// Extra restrictions and opaque filters reject without running callbacks.
func InspectPermanentTarget(reference PermanentRef) (PermanentTargetPredicate, bool) {
	binding, ok := reference.(*PermanentTargetBinding)
	if !ok || binding == nil || binding.targetDeclaration == nil {
		return PermanentTargetPredicate{}, false
	}
	return InspectPermanentTargetDeclaration(binding.targetDeclaration)
}

// InspectPermanentTargetDeclaration reads one complete native singular target.
// It accepts announced declarations without a resolving reference or callbacks.
func InspectPermanentTargetDeclaration(target Target) (PermanentTargetPredicate, bool) {
	declaration, ok := target.(*targetDeclaration)
	if !ok || declaration == nil {
		return PermanentTargetPredicate{}, false
	}
	spec, ok := declaration.spec.(*PermanentTarget)
	if !ok || spec == nil || spec.Min() != 1 || spec.Max() != 1 {
		return PermanentTargetPredicate{}, false
	}
	return inspectPermanentTargetDeclaration(declaration, spec)
}

func inspectPermanentTargetDeclaration(declaration *targetDeclaration, spec *PermanentTarget) (PermanentTargetPredicate, bool) {
	if spec == nil ||
		len(spec.typeDependencies) != 0 || len(spec.blockHistoryDependencies) != 0 ||
		len(spec.controllerRefs) != 0 || len(spec.excludedBindings) != 0 || len(spec.combatRelations) != 0 ||
		spec.chooserCount != 0 || !nilDefinition(spec.chooserRef) ||
		spec.opponentChooses ||
		spec.fixedCreatureEnchant || spec.fixedLandEnchant {
		return PermanentTargetPredicate{}, false
	}
	result := PermanentTargetPredicate{declaration: declaration, players: spec.playerDomain(), excludeSource: spec.excludeSource, toughnessLessThanSourcePower: spec.toughnessLessThanSourcePower, powerLESourcePower: spec.powerLESourcePower, attachedToControllerOnly: spec.attachedToControllerOnly, manaValueEqualX: spec.manaValueEqualX}
	for _, subtype := range spec.requiredSubtypes {
		if subtype == "" || strings.TrimSpace(subtype) != subtype {
			return PermanentTargetPredicate{}, false
		}
		result.filters = append(result.filters, &permanentCollectionFilterNode{operation: collectionFilterSubtype, subtype: subtype})
	}
	for _, domain := range spec.subtypeDomains {
		predicate, ok := inspectPermanentTargetDomain(domain)
		if !ok {
			return PermanentTargetPredicate{}, false
		}
		result.filters = append(result.filters, predicate)
	}
	for _, filter := range spec.Filters {
		predicate, ok := inspectContextualPermanentCollectionFilter(filter)
		if !ok || !permanentTargetFilterSupported(predicate) {
			return PermanentTargetPredicate{}, false
		}
		result.filters = append(result.filters, predicate)
	}
	return result, true
}

func permanentTargetFilterSupported(node *permanentCollectionFilterNode) bool {
	switch node.operation {
	case collectionFilterTypes, collectionFilterColor, collectionFilterSubtype,
		collectionFilterTapped, collectionFilterAttacking, collectionFilterBlocking,
		collectionFilterPowerEqual, collectionFilterToughnessEqual, collectionFilterToughnessGTE:
		return true
	case collectionFilterAll, collectionFilterAny, collectionFilterNot, collectionFilterAttachment:
		for _, child := range node.children {
			if !permanentTargetFilterSupported(child) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func inspectPermanentTargetDomain(domain PermanentTargetDomain) (*permanentCollectionFilterNode, bool) {
	if domain.inspectDefinition().problem != "" {
		return nil, false
	}
	switch domain.kind {
	case unknownTargetDomain:
		return &permanentCollectionFilterNode{operation: collectionFilterAll}, true
	case requiredSubtypeDomain:
		return &permanentCollectionFilterNode{operation: collectionFilterSubtype, subtype: domain.subtype}, true
	case excludedSubtypeDomain:
		child := &permanentCollectionFilterNode{operation: collectionFilterSubtype, subtype: domain.subtype}
		return &permanentCollectionFilterNode{operation: collectionFilterNot, children: []*permanentCollectionFilterNode{child}}, true
	case conjunctionTargetDomain, disjunctionTargetDomain:
		node := &permanentCollectionFilterNode{operation: collectionFilterAll}
		if domain.kind == disjunctionTargetDomain {
			node.operation = collectionFilterAny
		}
		for _, child := range domain.children {
			predicate, ok := inspectPermanentTargetDomain(child)
			if !ok {
				return nil, false
			}
			node.children = append(node.children, predicate)
		}
		return node, true
	default:
		return nil, false
	}
}

// Select reads the exact announced visit and its current target legality.
// Layers must be current. False reports an unknown query, not an illegal target.
// Protection uses shared source information, including ability source LKI.
// Player domains use the stack object's controller and current target information.
// Typed global target rules are inspected without predicate callbacks.
// Known turn and history conditions use native checks after full inspection.
// Opaque protection and conditional target rules remain unknown.
func (p PermanentTargetPredicate) Select(g *Game, obj *StackObject) ([]PermanentVisit, bool) {
	if p.declaration == nil || g == nil || obj == nil || len(obj.TargetSpecs) != len(obj.Targets) || p.manaValueEqualX && obj.XValue < 0 {
		return nil, false
	}
	index := -1
	for i, spec := range obj.TargetSpecs {
		if declaration, ok := spec.(*targetDeclaration); ok && declaration == p.declaration {
			if index != -1 {
				return nil, false
			}
			index = i
		}
	}
	if index < 0 || index >= len(obj.Targets) {
		return nil, false
	}
	return p.selectIndex(g, obj, index)
}

func (p PermanentTargetPredicate) selectIndex(g *Game, obj *StackObject, index int) ([]PermanentVisit, bool) {
	if p.manaValueEqualX && obj.XValue < 0 {
		return nil, false
	}
	id := obj.Targets[index]
	if id == uuid.Nil {
		return nil, true
	}
	visit := obj.targetIncarnations[id]
	zone, zoneKnown := obj.TargetZones[id]
	if visit == uuid.Nil || !zoneKnown {
		return nil, false
	}
	if zone != core.ZoneBattlefield {
		return nil, true
	}
	permanent := g.FindPermanent(id)
	if permanent == nil || permanent.PhasedOut || permanent.incarnationID != visit {
		return nil, true
	}
	if p.manaValueEqualX && !permanentTargetManaValueMatches(permanent, obj.XValue) {
		return nil, true
	}
	if !p.players.matches(permanent, obj.Controller) {
		return nil, true
	}
	if p.attachedToControllerOnly {
		host := currentAttachmentHost(permanent, g)
		if host == nil || host.ControllerID() != obj.Controller {
			return nil, true
		}
	}
	for _, predicate := range p.filters {
		if !predicate.matches(permanent, g) {
			return nil, true
		}
	}
	source := g.resolutionTargetSource(obj)
	if source == nil || source.HasSubType("Aura") {
		return nil, false
	}
	if p.toughnessLessThanSourcePower || p.powerLESourcePower {
		if obj.IsAbility && obj.sourceIncarnation == uuid.Nil {
			return nil, false
		}
		matches, known := sourceRelativeTargetMatches(permanent, source, g, p.toughnessLessThanSourcePower, p.powerLESourcePower)
		if !known {
			return nil, false
		}
		if !matches {
			return nil, true
		}
	}
	if p.excludeSource {
		if obj.IsAbility && obj.sourceIncarnation == uuid.Nil {
			return nil, false
		}
		if targetSourceIsPermanent(source, permanent) {
			return nil, true
		}
	}
	if restricted, known := g.inspectGlobalTargetRestrictions(permanent, obj, source); !known {
		return nil, false
	} else if restricted {
		return nil, true
	}
	if !permanent.FaceDown || isFaceDownCard(permanent.Card) {
		for _, ability := range permanent.RuntimeAbilities {
			if protection, ok := UnwrapAbility(ability).(*ProtectionAbility); ok && (protection == nil || (len(protection.FromColors) == 0 && protection.Filter.characteristics == nil)) {
				return nil, false
			}
		}
	}
	if !permanent.canBeTargetedByCharacteristics(source, obj.Controller, g) {
		return nil, true
	}
	return []PermanentVisit{{ID: id, Incarnation: visit}}, true
}
