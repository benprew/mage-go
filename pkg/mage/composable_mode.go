package mage

import "fmt"

// ModeDefinition is an immutable mode body and its target declarations.
type ModeDefinition struct {
	mode    Mode
	grouped bool
}

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
	return ModeDefinition{mode: m}
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

// ModalActivationOption is a mode or a restriction for a modal activation.
type ModalActivationOption interface {
	modalActivationOption()
	applyModalActivation(*ActionDefinition)
}

func (ModeDefinition) modalActivationOption() {}
func (m ModeDefinition) applyModalActivation(a *ActionDefinition) {
	a.modes = append(a.modes, m.mode)
}

// ActivateModes creates a modal activation with the existing payment transaction.
func ActivateModes(costs CostGroup, options ...ModalActivationOption) *ActionDefinition {
	a := Activate(costs, nil)
	a.effects = nil
	for _, option := range options {
		if nilDefinition(option) {
			panic("modal activation: supply a non-nil option")
		}
		option.applyModalActivation(a)
	}
	if len(a.modes) < 2 {
		panic("modes: supply at least two mode bodies")
	}
	return a
}
func (m ModeDefinition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "mode", Text: m.mode.Label, localBindingScope: true, children: declaredTargetNodes(m.mode.Targets, m.grouped)}
	if m.mode.Label == "" {
		n.problem = "supply a mode label"
	}
	for i, e := range m.mode.Effects {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("effects[%d]", i), e})
	}
	return n
}
func modeEdges(modes []Mode, grouped ...bool) []definitionEdge {
	result := make([]definitionEdge, 0, len(modes))
	for i, m := range modes {
		result = append(result, definitionEdge{fmt.Sprintf("modes[%d]", i), ModeDefinition{mode: m, grouped: len(grouped) > 0 && grouped[0]}})
	}
	return result
}

// WhenEventModes creates an event trigger with one mode chosen at stack placement.
func WhenEventModes(pattern EventPattern, modes ...ModeDefinition) TriggeredAbility {
	if nilDefinition(pattern) {
		panic("event: supply a pattern")
	}
	t := pattern.makeTrigger(nil)
	t.eventDefinition = pattern
	t.effects = nil
	return t.WithModes(modeValues(modes)...)
}

// AtModes creates a turn-step trigger with one mode chosen at stack placement.
func AtModes(pattern TurnStepPattern, modes ...ModeDefinition) TriggeredAbility {
	if nilDefinition(pattern) {
		panic("step: supply a pattern")
	}
	t := pattern.makeStepTrigger(nil)
	t.stepDefinition = pattern
	t.effects = nil
	return t.WithModes(modeValues(modes)...)
}
