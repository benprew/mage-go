package mage

import "fmt"

// ModeDefinition is an immutable mode body and its target declarations.
type ModeDefinition struct{ mode Mode }

// ModeOption is a closed option for a mode body.
type ModeOption interface {
	modeOption()
	applyMode(*Mode)
}

func (targetOptions) modeOption() {}
func (o targetOptions) applyMode(m *Mode) {
	for _, b := range o.bindings {
		if nilDefinition(b) {
			panic("mode targets: supply a non-nil binding")
		}
		m.Targets = append(m.Targets, b.targetSpec())
	}
}

// ModeOf declares one mode. Targets belong only to that mode.
func ModeOf(label string, effect ResolvingEffect, options ...ModeOption) ModeDefinition {
	m := Mode{Label: label, Effects: []Effect{effect}}
	for _, o := range options {
		if nilDefinition(o) {
			panic("mode: supply a non-nil option")
		}
		o.applyMode(&m)
	}
	return ModeDefinition{m}
}
func modeValues(modes []ModeDefinition) []Mode {
	if len(modes) < 2 {
		panic("modes: supply at least two mode bodies")
	}
	result := make([]Mode, len(modes))
	for i, m := range modes {
		result[i] = m.mode
	}
	return result
}

// SpellModes creates a spell that chooses one mode through the existing casting path.
func SpellModes(modes ...ModeDefinition) *ModalSpellAbility { return NewModalSpell(modeValues(modes)) }

// ActivateModes creates a modal activation with the existing payment transaction.
func ActivateModes(costs CostGroup, modes ...ModeDefinition) *ActionDefinition {
	a := Activate(costs, nil)
	a.effects = nil
	a.modes = modeValues(modes)
	return a
}
func (m ModeDefinition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "mode", Text: m.mode.Label, localBindingScope: true, children: declaredTargetNodes(m.mode.Targets)}
	if m.mode.Label == "" {
		n.problem = "supply a mode label"
	}
	for i, e := range m.mode.Effects {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("effects[%d]", i), e})
	}
	return n
}
func modeEdges(modes []Mode) []definitionEdge {
	result := make([]definitionEdge, 0, len(modes))
	for i, m := range modes {
		result = append(result, definitionEdge{fmt.Sprintf("modes[%d]", i), ModeDefinition{m}})
	}
	return result
}
