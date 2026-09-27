package mage

import "slices"

// CreatedTokensBinding names one token-entry result set.
type CreatedTokensBinding struct{ name string }

// BindCreatedTokens declares one instance-local token result.
func BindCreatedTokens(name string) *CreatedTokensBinding { return &CreatedTokensBinding{name: name} }

func (b *CreatedTokensBinding) bindingName() string { return b.name }
func (b *CreatedTokensBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "created-tokens-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}

type createdTokensSet struct{ binding *CreatedTokensBinding }

// FromCreatedTokens selects surviving, unphased incarnations from one entry batch.
func FromCreatedTokens(binding *CreatedTokensBinding) PermanentSet {
	return createdTokensSet{binding: binding}
}
func (createdTokensSet) permanentSet() {}
func (s createdTokensSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "created-tokens-set", bindingRead: s.binding, children: []definitionEdge{{"binding", s.binding}}}
}
func (s createdTokensSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	values := ctx.Game.resolution.createdTokenBindings[s.binding]
	result := make([]*Permanent, 0, len(values))
	for _, value := range values {
		p := ctx.Game.FindPermanent(value.id)
		if p != nil && !p.PhasedOut && p.incarnationID == value.incarnation {
			result = append(result, p)
		}
	}
	return result
}

func cloneCreatedTokenBindings(bindings map[*CreatedTokensBinding][]capturedPermanent) map[*CreatedTokensBinding][]capturedPermanent {
	if bindings == nil {
		return nil
	}
	cloned := make(map[*CreatedTokensBinding][]capturedPermanent, len(bindings))
	for binding, values := range bindings {
		cloned[binding] = slices.Clone(values)
	}
	return cloned
}
