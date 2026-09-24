package mage

import "github.com/google/uuid"

type discardSubjectEffect struct {
	subject PlayerRef
	amount  ResolutionValue
}

func (e *discardSubjectEffect) Text() string {
	return "discard for " + InspectDefinition(e.subject).Text
}
func (*discardSubjectEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *discardSubjectEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
}
func (e *discardSubjectEffect) Apply(ctx *EffectContext) error {
	id, ok := e.subject.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	child := *ctx
	child.Targets = []uuid.UUID{id}
	return DiscardCards(Fixed(amount)).Apply(&child)
}
