package mage

type grantLandPlaysEffect struct {
	subject PlayerRef
	amount  ResolutionValue
}

// GrantLandPlays adds to the selected player's land allowance for this turn.
// It does not play a land or change when the player can play lands.
// An absent player or a nonpositive amount has no effect.
func GrantLandPlays(subject PlayerRef, amount ResolutionValue) ResolvingEffect {
	return &grantLandPlaysEffect{subject, amount}
}

func (e *grantLandPlaysEffect) Text() string {
	return "grant additional land plays this turn to " + InspectDefinition(e.subject).Text
}
func (*grantLandPlaysEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *grantLandPlaysEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
}
func (e *grantLandPlaysEffect) Apply(ctx *EffectContext) error {
	id, ok := e.subject.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	ctx.Game.GrantExtraLandPlay(id, e.amount.resolveValue(ctx))
	return nil
}
