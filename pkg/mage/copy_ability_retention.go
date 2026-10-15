package mage

// TriggeredAbilityBinding identifies one lexical triggered declaration.
// Its builder retains rules without retaining targets or game choices.
type TriggeredAbilityBinding struct {
	build func() *GenericTriggered
}

func (*TriggeredAbilityBinding) bindingName() string { return "copy ability" }
func (b *TriggeredAbilityBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "triggered-ability-reference", bindingRead: b}
	if b.build == nil {
		n.problem = "retained ability requires a bound declaration"
	}
	return n
}

// BindTriggeredAbility supplies a reference to the declaration being built.
// Each later build supplies a new reference and new declarations.
func BindTriggeredAbility(build func(*TriggeredAbilityBinding) *GenericTriggered) *GenericTriggered {
	if build == nil {
		return nil
	}
	binding := &TriggeredAbilityBinding{}
	binding.build = func() *GenericTriggered { return BindTriggeredAbility(build) }
	trigger := build(binding)
	if trigger != nil {
		trigger.copyAbility = binding
	}
	return trigger
}

type triggeredAbilityDeclaration struct{ binding *TriggeredAbilityBinding }

func (d triggeredAbilityDeclaration) inspectDefinition() definitionNode {
	n := definitionNode{Role: "triggered-ability-declaration", bindingDeclaration: d.binding, bindingWrite: d.binding}
	if d.binding == nil || d.binding.build == nil {
		n.problem = "retained ability requires a builder"
	}
	return n
}

// ResolvingCopyOption supplies an exception to a resolving copy operation.
type ResolvingCopyOption interface{ applyResolvingCopyOption(*resolvingCopyEffect) }

type retainedCopyAbility struct{ binding *TriggeredAbilityBinding }

// RetainCopyAbility adds the bound declaration to the resulting copiable rules.
func RetainCopyAbility(binding *TriggeredAbilityBinding) ResolvingCopyOption {
	return retainedCopyAbility{binding: binding}
}
func (o retainedCopyAbility) applyResolvingCopyOption(e *resolvingCopyEffect) {
	e.retained = o.binding
	e.retentionOptions++
}

func resolvingCopyOptions(e *resolvingCopyEffect, options []ResolvingCopyOption) *resolvingCopyEffect {
	for _, option := range options {
		if option == nil {
			e.invalidOption = true
		} else {
			option.applyResolvingCopyOption(e)
		}
	}
	return e
}
