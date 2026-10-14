package mage

import (
	"slices"
	"strings"
)

// SpellTargetSourceWithDeclarations retains all applicable target groups.
// The caller supplies the selected mode before target choices.
func SpellTargetSourceWithDeclarations(card Card, declared []Target) Card {
	source := SpellTargetSource(card)
	if source == nil {
		return nil
	}
	view := source.(targetSourceLKI)
	view.spellDeclarations = slices.Clone(declared)
	view.spellDeclarationsKnown = true
	return view
}

// TargetSourceCanTargetOnlySubtype proves that every applicable group requires
// the subtype. Empty and unknown declarations cannot supply proof.
func TargetSourceCanTargetOnlySubtype(source Card, subtype string) bool {
	view, ok := source.(targetSourceLKI)
	if !ok {
		return false
	}
	var declared []Target
	if view.spell {
		if !view.spellDeclarationsKnown {
			return false
		}
		declared = view.spellDeclarations
	} else {
		if view.abilityDeclaration == nil {
			return false
		}
		declared = view.abilityDeclaration.targets
	}
	return len(declared) > 0 && !slices.ContainsFunc(declared, func(target Target) bool { return !TargetRequiresSubtype(target, subtype) })
}

// CanTargetOnlySubtypeFilter supplies a declaration filter for live targeting
// restrictions. Invalid subtype names supply an invalid filter.
func CanTargetOnlySubtypeFilter(subtype string) CardFilter {
	if subtype == "" || strings.TrimSpace(subtype) != subtype {
		return CardFilter{}
	}
	return NewCardFilter("can target only "+subtype, func(source Card) bool { return TargetSourceCanTargetOnlySubtype(source, subtype) })
}

func spellDeclarationSource(obj *StackObject, source Card) Card {
	view, ok := source.(targetSourceLKI)
	if !ok || !view.spell {
		return source
	}
	declaration := obj.spellTextDefinition
	if declaration == nil {
		declaration = obj.Card
	}
	if declaration == nil {
		return source
	}
	declared := declaration.CastTargets()
	if modal, ok := getModalSpellAbility(declaration); ok {
		if obj.ModeChoice < 0 || obj.ModeChoice >= len(modal.Modes()) {
			return source
		}
		declared = modal.Modes()[obj.ModeChoice].Targets
	}
	return SpellTargetSourceWithDeclarations(source, declared)
}
