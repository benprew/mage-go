package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

// Fight makes two referenced creatures deal damage equal to their power to each other.
func Fight(first, second PermanentRef) ResolvingEffect {
	return &fightReferencedEffect{first: first, second: second}
}

type fightReferencedEffect struct {
	first  PermanentRef
	second PermanentRef
}

func (*fightReferencedEffect) Text() string { return "creatures fight each other" }
func (*fightReferencedEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *fightReferencedEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"first", e.first}, {"second", e.second}}}
}
func (e *fightReferencedEffect) Apply(ctx *EffectContext) error {
	first, firstOK := e.first.resolvePermanent(ctx)
	second, secondOK := e.second.resolvePermanent(ctx)
	if !firstOK || !secondOK || !first.HasType(TypeCreature) || !second.HasType(TypeCreature) {
		return nil
	}
	fightPermanents(ctx, first, second)
	return nil
}
