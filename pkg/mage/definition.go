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
type savedBinding interface{ bindingName() string }

type definitionNode struct {
	isolatedBindings   bool
	localBindingScope  bool
	targetDeclaration  *targetDeclaration
	targetRead         *targetDeclaration
	Role, Text         string
	children           []definitionEdge
	problem            string
	battlefieldSource  bool
	bindingDeclaration savedBinding
	bindingRead        savedBinding
	bindingWrite       savedBinding
	bindingWrites      []savedBinding
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
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
	if e.invalidOption {
		n.problem = "supply a non-nil sacrifice result binding"
	} else if e.optionCount > 1 {
		n.problem = "duplicate sacrifice result binding"
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
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
		if e.groupedTargets && len(e.targets) == 0 {
			n.problem = "grouped targets require declarations"
		}
		for i, restriction := range e.dslRestrictions {
			if e.kind == ActionSpell {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("restrictions[%d]", i), spellStepDefinition{restriction.step, restriction.moreSteps}})
			} else {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("restrictions[%d]", i), restriction})
			}
		}
		n.children = append(n.children, declaredTargetNodes(e.targets, e.groupedTargets)...)
		for i, cost := range e.costs {
			if optional, ok := cost.(*optionalCost); ok && optional.capture {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if reveal, ok := cost.(*revealFromHandCost); ok && reveal.capture {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*tapNOtherMatchingCost); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if exile, ok := cost.(*exileMatchingCardFromGraveyardCost); ok && exile.capture {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if discard, ok := cost.(*discardCost); ok && discard.captureCard {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*xLifePayCost); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if life, ok := cost.(*lifePayCost); ok && life.capture {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if sacrifice, ok := cost.(*sacrificeMatchingCost); ok && (sacrifice.captureStats || sacrifice.captureManaValue || sacrifice.captureSubtype) {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*removeAnyCountersCost); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*capturedXCountersCost); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if counter, ok := cost.(*removeCountersCost); ok && counter.capture {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
		}
		for i, capture := range e.dslCaptures {
			n.children = append(n.children, definitionEdge{fmt.Sprintf("captures[%d]", i), capture})
		}
		if len(e.modes) > 0 {
			n.children = append(n.children, modeEdges(e.modes)...)
		} else {
			addEffects(e.effects)
		}
	case *GenericTriggered:
		n.children = append(n.children, declaredTargetNodes(e.targets, true)...)
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
		n.children = append(n.children, modeEdges(e.modes, true)...)
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
	var assigned map[savedBinding]bool
	var names map[string]any
	var targets map[*targetDeclaration]bool
	var castBody bool
	var visit func(any, string, bool, map[any]bool)
	visit = func(value any, path string, spell bool, active map[any]bool) {
		previousCastBody := castBody
		if action, ok := value.(*ActionDefinition); ok {
			castBody = action.kind == ActionSpell
		}
		defer func() { castBody = previousCastBody }()
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
			outer, outerTargets := assigned, targets
			assigned = maps.Clone(assigned)
			targets = maps.Clone(targets)
			defer func() { assigned, targets = outer, outerTargets }()
		}
		if n.localBindingScope {
			outerAssigned, outerTargets := assigned, targets
			assigned = maps.Clone(assigned)
			targets = make(map[*targetDeclaration]bool)
			defer func() { assigned, targets = outerAssigned, outerTargets }()
		}
		if n.bindingScope {
			outerAssigned, outerNames, outerTargets := assigned, names, targets
			assigned = make(map[savedBinding]bool)
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
		if !nilDefinition(n.bindingDeclaration) {
			binding := n.bindingDeclaration
			if prior, ok := names[binding.bindingName()]; ok && prior != binding {
				fail("duplicate binding name: " + binding.bindingName())
			}
			names[binding.bindingName()] = binding
		}
		if !nilDefinition(n.bindingRead) && !assigned[n.bindingRead] {
			fail("binding requires a producer before this read on every path")
		}
		if n.problem != "" {
			fail(n.problem)
		}
		if life, ok := value.(*lifePayCost); ok && life.capture && !nilDefinition(life.paid) && assigned[life.paid] {
			fail("duplicate paid life binding")
		}
		if exile, ok := value.(*exileMatchingCardFromGraveyardCost); ok && exile.capture && !nilDefinition(exile.paid) && assigned[exile.paid] {
			fail("duplicate paid graveyard exile binding")
		}
		if sacrifice, ok := value.(*sacrificeMatchingCost); ok && sacrifice.captureCard && !nilDefinition(sacrifice.paidCard) && assigned[sacrifice.paidCard] {
			fail("duplicate paid card binding")
		}
		if discard, ok := value.(*discardCost); ok && discard.captureCard && !nilDefinition(discard.paidCard) && assigned[discard.paidCard] {
			fail("duplicate paid card binding")
		}
		if reveal, ok := value.(*revealFromHandCost); ok && reveal.capture && !nilDefinition(reveal.paidCard) && assigned[reveal.paidCard] {
			fail("duplicate paid card binding")
		}
		if optional, ok := value.(*optionalCost); ok && optional.capture && !nilDefinition(optional.result) && assigned[optional.result] {
			fail("duplicate optional payment binding")
		}
		if life, ok := value.(*xLifePayCost); ok && !nilDefinition(life.paid) && assigned[life.paid] {
			fail("duplicate paid life binding")
		}
		if counter, ok := value.(*removeCountersCost); ok && counter.capture && !nilDefinition(counter.paid) && assigned[counter.paid] {
			fail("duplicate paid counter binding")
		}
		if counter, ok := value.(*capturedXCountersCost); ok && !nilDefinition(counter.paid) && assigned[counter.paid] {
			fail("duplicate paid counter binding")
		}
		if spell && n.battlefieldSource {
			fail("battlefield source action requires a permanent ability")
		}
		if _, cast := value.(*castBranch); cast && !castBody {
			fail("cast condition requires a spell ability")
		}
		if _, colors := value.(selfCastColorsValue); colors && !castBody {
			fail("colors spent value requires a spell ability")
		}
		for _, edge := range n.children {
			before := maps.Clone(assigned)
			visit(edge.value, path+"."+edge.name, spell, active)
			_, eventTrigger := value.(*GenericTriggered)
			_, capture := edge.value.(activationCapture)
			_, paidCost := edge.value.(*sacrificeMatchingCost)
			exileCost, exilePayment := edge.value.(*exileMatchingCardFromGraveyardCost)
			_, paidDiscard := edge.value.(*discardCost)
			reveal, paidReveal := edge.value.(*revealFromHandCost)
			_, paidCounters := edge.value.(*removeAnyCountersCost)
			_, paidXCounters := edge.value.(*capturedXCountersCost)
			optional, optionalPayment := edge.value.(*optionalCost)
			fixedCounter, fixedCounterCost := edge.value.(*removeCountersCost)
			life, paidLife := edge.value.(*lifePayCost)
			_, xPaidLife := edge.value.(*xLifePayCost)
			if !definitionSequential(value) && (!eventTrigger || edge.name != "event") && !capture && !paidCost && (!exilePayment || !exileCost.capture) && !paidDiscard && (!paidReveal || !reveal.capture) && !paidCounters && !paidXCounters && (!optionalPayment || !optional.capture) && (!fixedCounterCost || !fixedCounter.capture) && !xPaidLife && (!paidLife || !life.capture) {
				assigned = before
			}
		}
		if !nilDefinition(n.bindingWrite) {
			assigned[n.bindingWrite] = true
		}
		for _, binding := range n.bindingWrites {
			if !nilDefinition(binding) {
				assigned[binding] = true
			}
		}
	}
	for i, a := range card.Abilities() {
		targets = make(map[*targetDeclaration]bool)
		assigned = make(map[savedBinding]bool)
		names = make(map[string]any)
		spell := card.HasType(TypeInstant) || card.HasType(TypeSorcery) || (!nilDefinition(a) && a.Type() == AbilitySpell)
		if spell {
			if base, ok := card.(*BaseCard); ok {
				for j, cost := range base.AdditionalCosts() {
					visit(cost, fmt.Sprintf("additional-costs[%d]", j), true, make(map[any]bool))
				}
			}
		}
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
	case *nextStepEffect, *sequenceEffect, *compositeEffect, *PipelineData, *permanentSnapshot, *permanentLoop, *damageResultLoop, damageEventPattern, controllerDamageEventPattern, lifeLossEventPattern, attachedControllerUpkeepPattern, attachedBecomesTappedPattern:
		return true
	default:
		return false
	}
}
