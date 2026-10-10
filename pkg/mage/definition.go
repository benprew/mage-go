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
		n.problem = "supply a non-nil sacrifice result binding and a subtype"
	} else if e.optionCount > 1 {
		n.problem = "duplicate sacrifice result binding"
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	if e.subtypeResult != nil {
		n.bindingWrite = e.subtypeResult
		n.children = append(n.children, definitionEdge{"subtype result", e.subtypeResult})
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
	case *ManaAbility:
		if e.conditional != nil {
			n.children = append(n.children, definitionEdge{"conditional", e.conditional})
		}
	case *PrintedBasicLandwalkAbility:
		if !isBasicLandwalk(e.keyword) {
			n.problem = "printed landwalk requires a basic landwalk keyword"
		}
	case *ProtectionAbility:
		if e.colorTextRevision && !e.supportsColorTextRevision() {
			n.problem = "text protection revision requires nonempty color protection without an exception"
		}
	case *StaticAbilityHolder:
		if e.landTypeTextRevision && !e.supportsLandTypeTextRevision() {
			n.problem = "text land-type revision requires one global or attached battlefield basic-land-type effect"
		}
		if e.animationTextRevision && !e.supportsAnimationTextRevision() {
			n.problem = "text animation revision requires all layers from one battlefield animation rule"
		}
		if e.grantTextRevision && !e.supportsGrantTextRevision() {
			n.problem = "text grant revision requires battlefield ability grants with optional P/T modifications"
		}
		if e.abilityTextRevision && !e.supportsAbilityTextRevision() {
			n.problem = "text ability revision requires nonempty battlefield ability-layer effects"
		}
		if e.ptTextLayer && !e.supportsLayeredPTTextRevision() {
			n.problem = "text P/T layer requires validated battlefield P/T modifications"
		}
		if e.ptTextRevision && !e.supportsPTTextRevision() {
			n.problem = "text P/T revision requires nonempty battlefield P/T modifications"
		}
		for i, effect := range e.Effects {
			n.children = append(n.children, definitionEdge{fmt.Sprintf("rules[%d]", i), effect})
		}
	case *ModalSpellAbility:
		if e.textContinuation && !e.supportsTextContinuation() {
			n.problem = "modal spell text continuation requires separate modes without casting bindings"
		}
		for i, option := range e.dslOptions {
			n.children = append(n.children, definitionEdge{fmt.Sprintf("options[%d]", i), option})
		}
		n.children = append(n.children, modeEdges(e.modes)...)
	case *ActionDefinition:
		if e.spellTextContinuation && !e.supportsSpellTextContinuation() {
			n.problem = "spell text continuation requires an ordinary spell without casting bindings"
		}
		if e.textRevisionState && e.kind != ActionActivated {
			n.problem = "text revision state requires an activated action"
		}
		if e.activationManaXOptions > 0 {
			hasX := false
			for _, cost := range e.costs {
				if mana, ok := cost.(*ManaCostPayment); ok && mana.MC.HasX {
					hasX = true
				}
			}
			if e.kind != ActionActivated || e.activationManaXOptions != 1 || !hasX {
				n.problem = "supply one defined mana X on an activated action with an X mana cost"
			}
			n.children = append(n.children, definitionEdge{"activationManaX", e.activationManaX})
		}
		if e.activationHistoryOptions > 0 {
			if e.kind != ActionActivated || e.activationHistory == nil || e.activationHistoryOptions != 1 {
				n.problem = "supply one activation history binding on an activated ability"
			}
			n.children = append(n.children, definitionEdge{"activationHistory", activationHistoryDeclaration{e.activationHistory}})
		}
		if e.groupedTargets && len(e.targets) == 0 {
			n.problem = "grouped targets require declarations"
		}
		if len(e.castingConditions) > 0 && e.kind != ActionSpell {
			n.problem = "casting conditions require a spell action"
		}
		for i, condition := range e.castingConditions {
			n.children = append(n.children, definitionEdge{fmt.Sprintf("castingConditions[%d]", i), castingConditionDefinition{condition}})
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
			if _, ok := cost.(*sourceCardCost); ok {
				if e.kind != ActionActivated {
					n.problem = "paid source card requires an activated ability cost"
				}
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(activationTargetMana); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(targetSurcharge); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if mill, ok := cost.(*millCost); ok && mill.capture {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*discardRandomCost); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*eitherCost); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*exileFromGraveyardCost); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*optionalCost); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if reveal, ok := cost.(*revealFromHandCost); ok && reveal.capture {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*tapNOtherMatchingCost); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if exile, ok := cost.(*exileMatchingCardFromGraveyardCost); ok && (exile.capture || exile.captureCard) {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if discard, ok := cost.(*discardCost); ok && (discard.captureCard || discard.captureCards || discard.differentNames) {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*xLifePayCost); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if life, ok := cost.(*lifePayCost); ok && life.capture {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*sacrificeMatchingCost); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*removeAnyCountersCost); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*capturedXCountersCost); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*putCounterOnMatchingCost); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*removeCountersCost); ok {
				n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
			}
			if _, ok := cost.(*removeXCountersCost); ok {
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
	case *ManaBonusAbility:
		if e.controllerRecipient && (e.BonusMana < Colorless || e.BonusMana >= AnyColor || e.AttachedOnly || e.MatchProduced) {
			n.problem = "controller mana bonus requires a fixed global bonus"
		}
		if e.manaTapTextRevision && !e.supportsManaTapTextRevision() {
			n.problem = "text revision requires a valid fixed, attached or produced-type mana bonus"
		}
	case *GenericTriggered:
		if e.manaTapTextRevision && !e.supportsManaTapTextRevision() {
			n.problem = "text revision requires a typed battlefield land mana trigger"
		}
		if e.combatPairTextRevision && !e.supportsCombatPairTextRevision() {
			n.problem = "text revision requires a typed battlefield combat-pair trigger"
		}
		if e.landTapTextRevision && !e.supportsLandTapTextRevision() {
			n.problem = "text revision requires a typed battlefield opponent land-tap trigger"
		}
		if e.spellCastTextRevision && !e.supportsSpellCastTextRevision() {
			n.problem = "text revision requires a typed battlefield spell-cast trigger"
		}
		if e.eventDefinition != nil {
			n.Role = "event-trigger"
			n.children = append(n.children, definitionEdge{"event", e.eventDefinition})
		}
		if e.stepDefinition != nil {
			n.Role = "step-trigger"
			n.children = append(n.children, definitionEdge{"step", e.stepDefinition})
		}
		n.children = append(n.children, declaredTargetNodes(e.targets, true)...)
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
	if base, ok := card.(*BaseCard); ok && base != nil && base.manaTapSpellText != 0 {
		if _, err := manaTapSpellTextContext(card); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if base, ok := card.(*BaseCard); ok && base != nil && base.spellCastSpellText != 0 {
		if _, err := spellCastSpellTextContext(card); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if base, ok := card.(*BaseCard); ok && base != nil && base.globalStaticSpellText != 0 {
		if _, err := globalStaticSpellTextContext(card); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if base, ok := card.(*BaseCard); ok && base != nil && base.copyEntrySpellText != 0 {
		if _, err := copyEntrySpellTextContext(card); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if base, ok := card.(*BaseCard); ok && base != nil && base.intrinsicSpellText != 0 {
		if err := intrinsicSpellTextContext(card); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if base, ok := card.(*BaseCard); ok && base != nil && base.landAuraSpellText != 0 {
		if _, err := landAuraSpellTextContext(card); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if base, ok := card.(*BaseCard); ok && base != nil && base.abilityGroups != nil {
		if _, err := definitionAbilitiesByKey(base, false); err != nil {
			return err
		}
	}
	var problems []error
	if nilDefinition(card) {
		return fmt.Errorf("%s: card: supply a card definition", name)
	}
	historyOwners := make(map[*ActivationHistoryBinding]bool)
	var assigned map[savedBinding]bool
	var names map[string]any
	var targets map[*targetDeclaration]bool
	var castBody bool
	var visit func(any, string, bool, map[any]bool, bool)
	visit = func(value any, path string, spell bool, active map[any]bool, allocationAllowed bool) {
		previousCastBody := castBody
		switch value.(type) {
		case grantedTriggerDefinition, tokenAbilityDefinition:
			spell, castBody = false, false
		}
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
		if action, ok := value.(*ActionDefinition); ok && action.activationHistory != nil {
			if historyOwners[action.activationHistory] {
				fail("activation history must belong to one ability")
			}
			historyOwners[action.activationHistory] = true
		}
		if _, allocated := value.(*allocatedDamageEffect); allocated && !allocationAllowed {
			fail("allocated damage requires one instruction in a spell or activation body with one target group")
		}
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
		if choice, ok := value.(*wordPairChoice); ok && !nilDefinition(choice.binding) && assigned[choice.binding] {
			fail("duplicate text word pair producer")
		}
		if source, ok := value.(*sourceCardCost); ok && !nilDefinition(source.result) && assigned[source.result] {
			fail("duplicate paid source card binding")
		}
		if mill, ok := value.(*millCost); ok && mill.capture && !nilDefinition(mill.paid) && assigned[mill.paid] {
			fail("duplicate paid mill count binding")
		}
		if life, ok := value.(*lifePayCost); ok && life.capture && !nilDefinition(life.paid) && assigned[life.paid] {
			fail("duplicate paid life binding")
		}
		if exile, ok := value.(*exileMatchingCardFromGraveyardCost); ok && exile.capture && !nilDefinition(exile.paid) && assigned[exile.paid] {
			fail("duplicate paid graveyard exile binding")
		}
		if exile, ok := value.(*exileMatchingCardFromGraveyardCost); ok && exile.captureCard && !nilDefinition(exile.paidCard) && assigned[exile.paidCard] {
			fail("duplicate paid card binding")
		}
		if exile, ok := value.(*exileFromGraveyardCost); ok && exile.paidRequired && !nilDefinition(exile.paid) && assigned[exile.paid] {
			fail("duplicate paid cards binding")
		}
		if sacrifice, ok := value.(*sacrificeMatchingCost); ok && sacrifice.captureCard && !nilDefinition(sacrifice.paidCard) && assigned[sacrifice.paidCard] {
			fail("duplicate paid card binding")
		}
		if sacrifice, ok := value.(*sacrificeMatchingCost); ok && sacrifice.capturePermanent && !nilDefinition(sacrifice.paidPermanent) && assigned[sacrifice.paidPermanent] {
			fail("duplicate paid permanent binding")
		}
		if taps, ok := value.(*tapNOtherMatchingCost); ok && taps.capture && !nilDefinition(taps.paid) && assigned[taps.paid] {
			fail("duplicate paid permanent binding")
		}
		if sacrifice, ok := value.(*sacrificeMatchingCost); ok && sacrifice.captureTotalPower && !nilDefinition(sacrifice.paidTotalPower) && assigned[sacrifice.paidTotalPower] {
			fail("duplicate paid total power binding")
		}
		if discard, ok := value.(*discardCost); ok && discard.captureCard && !nilDefinition(discard.paidCard) && assigned[discard.paidCard] {
			fail("duplicate paid card binding")
		}
		if discard, ok := value.(*discardCost); ok && discard.captureCards && !nilDefinition(discard.paidCards) && assigned[discard.paidCards] {
			fail("duplicate paid cards binding")
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
			_, sequence := value.(*sequenceEffect)
			allowAllocation := allocationAllowed && sequence
			switch parent := value.(type) {
			case *ActionDefinition:
				if parent.kind == ActionSpell || parent.kind == ActionActivated {
					if len(parent.modes) == 0 {
						allowAllocation = allocatedBodyEdge(parent.effects, parent.targets, edge.value)
					} else if len(parent.targets) == 0 {
						_, allowAllocation = edge.value.(ModeDefinition)
					}
				}
			case *ModalSpellAbility:
				_, allowAllocation = edge.value.(ModeDefinition)
			case ModeDefinition:
				allowAllocation = allocationAllowed && allocatedBodyEdge(parent.mode.Effects, parent.mode.Targets, edge.value)
			}
			visit(edge.value, path+"."+edge.name, spell, active, allowAllocation)
			_, eventTrigger := value.(*GenericTriggered)
			_, capture := edge.value.(activationCapture)
			_, history := edge.value.(activationHistoryDeclaration)
			_, paidCost := edge.value.(*sacrificeMatchingCost)
			_, paidSource := edge.value.(*sourceCardCost)
			paidTap, tapCost := edge.value.(*tapNOtherMatchingCost)
			exileCost, exilePayment := edge.value.(*exileMatchingCardFromGraveyardCost)
			exileCards, exileCardsPayment := edge.value.(*exileFromGraveyardCost)
			_, paidDiscard := edge.value.(*discardCost)
			reveal, paidReveal := edge.value.(*revealFromHandCost)
			_, paidCounters := edge.value.(*removeAnyCountersCost)
			_, paidXCounters := edge.value.(*capturedXCountersCost)
			optional, optionalPayment := edge.value.(*optionalCost)
			fixedCounter, fixedCounterCost := edge.value.(*removeCountersCost)
			life, paidLife := edge.value.(*lifePayCost)
			_, xPaidLife := edge.value.(*xLifePayCost)
			mill, paidMill := edge.value.(*millCost)
			if !definitionSequential(value) && (!eventTrigger || edge.name != "event") && !capture && !history && !paidCost && !paidSource && (!tapCost || !paidTap.capture) && (!exilePayment || (!exileCost.capture && !exileCost.captureCard)) && (!exileCardsPayment || !exileCards.paidRequired) && !paidDiscard && (!paidReveal || !reveal.capture) && !paidCounters && !paidXCounters && (!optionalPayment || !optional.capture) && (!fixedCounterCost || !fixedCounter.capture) && !xPaidLife && (!paidLife || !life.capture) && (!paidMill || !mill.capture) {
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
					visit(cost, fmt.Sprintf("additional-costs[%d]", j), true, make(map[any]bool), false)
				}
			}
		}
		if action, ok := a.(*ActionDefinition); ok && action.spellSacrificeContinuationOptions > 0 {
			if _, err := spellTextAction(card); err != nil {
				problems = append(problems, fmt.Errorf("%s: abilities[%d]: spell sacrifice text continuation: %w", name, i, err))
			}
		}
		visit(a, fmt.Sprintf("abilities[%d]", i), spell, make(map[any]bool), false)
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
	case playerDamageAmount, *nextStepEffect, *sequenceEffect, *compositeEffect, *PipelineData, *permanentSnapshot, *permanentLoop, *savedCardLoop, *damageResultLoop, damageEventPattern, controllerDamageEventPattern, lifeLossEventPattern, attachedControllerUpkeepPattern, attachedBecomesTappedPattern:
		return true
	default:
		return false
	}
}
