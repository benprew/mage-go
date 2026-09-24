package mage

import "github.com/google/uuid"

type millSubjectEffect struct {
	subject PlayerRef
	amount  ResolutionValue
}

// Mill puts cards from the selected player's library into their graveyard.
// An absent player has no effect. The existing mill modifiers apply.
func Mill(subject PlayerRef, amount ResolutionValue) ResolvingEffect {
	return &millSubjectEffect{subject, amount}
}

func (e *millSubjectEffect) Text() string { return "mill for " + InspectDefinition(e.subject).Text }
func (*millSubjectEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *millSubjectEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
}
func (e *millSubjectEffect) Apply(ctx *EffectContext) error {
	id, ok := e.subject.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	child := *ctx
	child.Targets = []uuid.UUID{id}
	return MillTargetPlayer(Fixed(amount)).Apply(&child)
}
