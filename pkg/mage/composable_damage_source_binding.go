package mage

// DamageSourceBinding names a source saved by one damage instruction.
type DamageSourceBinding struct{ name string }

// BindDamageSource declares a source binding for one resolution.
func BindDamageSource(name string) *DamageSourceBinding {
	return &DamageSourceBinding{name: name}
}

// FromDamageSource reads a saved source with its saved characteristics.
func FromDamageSource(binding *DamageSourceBinding) DamageSourceRef {
	return savedDamageSourceRef{binding: binding}
}

type savedDamageSourceRef struct{ binding *DamageSourceBinding }

func (savedDamageSourceRef) damageSourceRef() {}
func (r savedDamageSourceRef) resolveDamageSource(ctx *EffectContext) DamageSource {
	return ctx.Game.resolution.damageSourceBindings[r.binding]
}
func (b *DamageSourceBinding) bindingName() string { return b.name }
func (b *DamageSourceBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "damage-source-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}
func (r savedDamageSourceRef) inspectDefinition() definitionNode {
	label := "saved damage source"
	if r.binding != nil {
		label = r.binding.name
	}
	return definitionNode{Role: "damage-source-ref", Text: label, bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}
