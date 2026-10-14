package mage

import (
	"slices"
	"strings"
)

// TargetHasSubtype requires the subtype and retains it in the target declaration.
func TargetHasSubtype(subtype string) PermanentTargetOption {
	return targetSubtypeOption{subtype: subtype}
}

type targetSubtypeOption struct{ subtype string }

func (targetSubtypeOption) permanentTargetOption() {}
func (o targetSubtypeOption) applyPermanentTarget(t *PermanentTarget) {
	if !slices.Contains(t.requiredSubtypes, o.subtype) {
		t.requiredSubtypes = append(slices.Clone(t.requiredSubtypes), o.subtype)
	}
}

func (o targetSubtypeOption) inspectDefinition() definitionNode {
	n := definitionNode{Role: "target-subtype", Text: o.subtype}
	if o.subtype == "" || strings.TrimSpace(o.subtype) != o.subtype {
		n.problem = "target subtype requires a nonempty name without outer spaces"
	}
	return n
}

// TargetRequiredSubtypes returns a separate list of enforced subtype constraints.
// An empty list supplies no proof about the target domain. Other filters can
// further restrict it. Choices and current game state cannot change this list.
func TargetRequiredSubtypes(target Target) []string {
	switch t := target.(type) {
	case *targetDeclaration:
		if t != nil {
			return TargetRequiredSubtypes(t.spec)
		}
	case *countBoundTarget:
		if t != nil {
			return TargetRequiredSubtypes(t.inner)
		}
	case *randomTarget:
		if t != nil {
			return TargetRequiredSubtypes(t.inner)
		}
	case *PermanentTarget:
		if t != nil {
			return slices.Clone(t.requiredSubtypes)
		}
	case TargetBinding:
		if !nilDefinition(t) {
			return TargetRequiredSubtypes(t.targetSpec())
		}
	}
	return nil
}
