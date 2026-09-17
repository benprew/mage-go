package mage

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type definitionEdge struct {
	name  string
	value any
}
type definitionNode struct {
	isolatedBindings   bool
	localBindingScope  bool
	targetDeclaration  *targetDeclaration
	targetRead         *targetDeclaration
	Role, Text         string
	children           []definitionEdge
	problem            string
	battlefieldSource  bool
	bindingDeclaration *PermanentBinding
	bindingRead        *PermanentBinding
	bindingWrite       *PermanentBinding
	bindingScope       bool
}
type definitionInspector interface{ inspectDefinition() definitionNode }

func (c *statePredicate) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: c.label}
	if c.label == "" {
		n.Text = c.kind + " condition"
	}

	for i, child := range c.children {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("conditions[%d]", i), child})
	}
	if (c.kind == "all" || c.kind == "any") && len(c.children) == 0 {
		n.problem = "supply at least one condition"
	}
	if c.kind == "callback" {
		if c.fn == nil {
			n.problem = "supply a condition callback"
		}
		if len(c.dependencies) == 0 {
			n.problem = "declare callback dependencies"
		}
		if c.label == "" {
			n.problem = "supply a callback description"
		}
		for _, d := range c.dependencies {
			if d != StateBattlefield && d != StateSourceChoices {
				n.problem = "unknown state dependency"
			}
		}
	}
	return n
}
func (e *sacrificeEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}
func (sourceRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-ref", Text: "this", battlefieldSource: true}
}
func (e *sequenceEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "sequence"}
	if len(e.effects) == 0 {
		n.problem = "supply at least one effect"
	}
	for i, child := range e.effects {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("effects[%d]", i), child})
	}
	return n
}
func (e *stateBranch) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-branch", children: []definitionEdge{{"condition", e.condition}, {"then", e.effect}}}
}
func nilDefinition(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Slice, reflect.Map:
		return rv.IsNil()
	}
	return false
}
func describeDefinition(v any) definitionNode {
	if n, ok := v.(definitionInspector); ok {
		return n.inspectDefinition()
	}
	n := definitionNode{Role: "legacy"}
	addEffects := func(es []Effect) {
		for i, e := range es {
			n.children = append(n.children, definitionEdge{fmt.Sprintf("effects[%d]", i), e})
		}
	}
	switch e := v.(type) {
	case *StaticAbilityHolder:
		for i, effect := range e.Effects {
			n.children = append(n.children, definitionEdge{fmt.Sprintf("rules[%d]", i), effect})
		}
	case *ModalSpellAbility:
		n.children = append(n.children, modeEdges(e.modes)...)
	case *ActionDefinition:
		for i, restriction := range e.dslRestrictions {
			n.children = append(n.children, definitionEdge{fmt.Sprintf("restrictions[%d]", i), restriction})
		}
		n.children = append(n.children, declaredTargetNodes(e.targets)...)
		if len(e.modes) > 0 {
			n.children = append(n.children, modeEdges(e.modes)...)
		} else {
			addEffects(e.effects)
		}
	case *GenericTriggered:
		n.children = append(n.children, declaredTargetNodes(e.targets)...)
		if e.eventDefinition != nil {
			n.Role = "event-trigger"
			n.children = append(n.children, definitionEdge{"event", e.eventDefinition})
		}
		if e.stepDefinition != nil {
			n.Role = "step-trigger"
			n.children = append(n.children, definitionEdge{"step", e.stepDefinition})
		}
		if e.dslState {
			n.Role = "state-trigger"
			n.battlefieldSource = true
			n.children = append(n.children, definitionEdge{"condition", e.stateDefinition})
		}
		if len(e.modes) == 0 {
			addEffects(e.effects)
		}
		n.children = append(n.children, modeEdges(e.modes)...)
	case interface{ Effects() []Effect }:
		addEffects(e.Effects())
	case *RegisterDelayedTriggerData:
		n.bindingScope = true
		addEffects(e.Effects)
	case *PipelineData:
		addEffects(e.Steps)
	case *compositeEffect:
		addEffects(e.effects)
	case *IfElseData:
		if e.Then != nil {
			n.children = append(n.children, definitionEdge{"then", e.Then})
		}
		if e.Else != nil {
			n.children = append(n.children, definitionEdge{"else", e.Else})
		}
	case *ForEachPermanentData:
		n.children = append(n.children, definitionEdge{"body", e.Inner})
	case *ModalEffectData:
		addEffects(e.Modes)
	}
	return n
}

// DefinitionMetadata keeps ordered structural information for diagnostics and AI.
type DefinitionMetadata struct {
	Role       string
	Text       string
	Children   []DefinitionMetadata
	Properties EffectProperties
}

// InspectDefinition reports metadata without exposing an executable engine object.
func InspectDefinition(value any) DefinitionMetadata {
	if nilDefinition(value) {
		return DefinitionMetadata{Role: "nil"}
	}
	n := describeDefinition(value)
	out := DefinitionMetadata{Role: n.Role, Text: n.Text}
	if e, ok := value.(EffectMetadata); ok {
		out.Properties = e.Properties()
		out.Properties.AIRoles = slices.Clone(out.Properties.AIRoles)
		if out.Text == "" {
			out.Text = e.Text()
		}
	}
	for _, edge := range n.children {
		out.Children = append(out.Children, InspectDefinition(edge.value))
	}
	return out
}

// ValidateCardDefinition checks the expression families that have DSL adapters.
func ValidateCardDefinition(name string, card Card) error {
	var problems []error
	if nilDefinition(card) {
		return fmt.Errorf("%s: card: supply a card definition", name)
	}
	var assigned map[*PermanentBinding]bool
	var names map[string]any
	var targets map[*targetDeclaration]bool
	var visit func(any, string, bool, map[any]bool)
	visit = func(value any, path string, spell bool, active map[any]bool) {
		fail := func(message string) { problems = append(problems, fmt.Errorf("%s: %s: %s", name, path, message)) }
		if nilDefinition(value) {
			fail("supply a non-nil expression")
			return
		}
		if reflect.ValueOf(value).Comparable() {
			if active[value] {
				fail("remove cyclic definition dependency")
				return
			}
			active[value] = true
			defer delete(active, value)
		}
		n := describeDefinition(value)
		if n.isolatedBindings {
			outer := assigned
			assigned = maps.Clone(assigned)
			defer func() { assigned = outer }()
		}
		if n.localBindingScope {
			outerAssigned, outerTargets := assigned, targets
			assigned = make(map[*PermanentBinding]bool)
			targets = make(map[*targetDeclaration]bool)
			defer func() { assigned, targets = outerAssigned, outerTargets }()
		}
		if n.bindingScope {
			outerAssigned, outerNames, outerTargets := assigned, names, targets
			assigned = make(map[*PermanentBinding]bool)
			names = make(map[string]any)
			targets = make(map[*targetDeclaration]bool)
			defer func() { assigned, names, targets = outerAssigned, outerNames, outerTargets }()
		}
		if n.targetDeclaration != nil {
			b := n.targetDeclaration
			if prior, ok := names[b.name]; ok && prior != b {
				fail("duplicate target binding name: " + b.name)
			}
			if targets[b] {
				fail("declare each target binding only once")
			}
			names[b.name] = b
			targets[b] = true
		}
		if n.targetRead != nil && !targets[n.targetRead] {
			fail("target binding requires a declaration in this ability")
		}
		if n.bindingDeclaration != nil {
			binding := n.bindingDeclaration
			if prior, ok := names[binding.name]; ok && prior != binding {
				fail("duplicate binding name: " + binding.name)
			}
			names[binding.name] = binding
		}
		if n.bindingRead != nil && !assigned[n.bindingRead] {
			fail("binding requires a producer before this read on every path")
		}
		if n.problem != "" {
			fail(n.problem)
		}
		if spell && n.battlefieldSource {
			fail("battlefield source action requires a permanent ability")
		}
		for _, edge := range n.children {
			before := maps.Clone(assigned)
			visit(edge.value, path+"."+edge.name, spell, active)
			if !definitionSequential(value) {
				assigned = before
			}
		}
		if n.bindingWrite != nil {
			assigned[n.bindingWrite] = true
		}
	}
	for i, a := range card.Abilities() {
		targets = make(map[*targetDeclaration]bool)
		assigned = make(map[*PermanentBinding]bool)
		names = make(map[string]any)
		spell := card.HasType(TypeInstant) || card.HasType(TypeSorcery) || (!nilDefinition(a) && a.Type() == AbilitySpell)
		visit(a, fmt.Sprintf("abilities[%d]", i), spell, make(map[any]bool))
	}
	return errors.Join(problems...)
}

func validateFactory(name string, f CardFactory) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%s: factory: %v", name, p)
		}
	}()
	if f == nil {
		return fmt.Errorf("%s: factory: supply a factory", name)
	}
	return ValidateCardDefinition(name, f())
}

// ValidateDefinitions checks registered factories outside the registry lock.
func (r *Registry) ValidateDefinitions() error {
	r.mu.RLock()
	factories := make(map[string]CardFactory, len(r.factories))
	maps.Copy(factories, r.factories)
	r.mu.RUnlock()
	names := make([]string, 0, len(factories))
	for name := range factories {
		names = append(names, name)
	}
	slices.Sort(names)
	var problems []error
	for _, name := range names {
		if err := validateFactory(name, factories[name]); err != nil {
			problems = append(problems, err)
		}
	}
	return errors.Join(problems...)
}

func definitionSequential(value any) bool {
	switch value.(type) {
	case *sequenceEffect, *compositeEffect, *PipelineData, *permanentSnapshot, *permanentLoop:
		return true
	default:
		return false
	}
}
