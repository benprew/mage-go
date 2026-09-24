package mage

import "github.com/google/uuid"

type discardEntireHandEffect struct {
	subject PlayerRef
}

// DiscardEntireHand makes one player discard the hand they have at resolution.
func DiscardEntireHand(subject PlayerRef) ResolvingEffect {
	return &discardEntireHandEffect{subject: subject}
}

func (e *discardEntireHandEffect) Text() string {
	return "discard entire hand for " + InspectDefinition(e.subject).Text
}
func (*discardEntireHandEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *discardEntireHandEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}
func (e *discardEntireHandEffect) Apply(ctx *EffectContext) error {
	id, ok := e.subject.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	child := *ctx
	child.Targets = []uuid.UUID{id}
	return DiscardHand().Targeting(SelectTargetPlayer()).Apply(&child)
}
