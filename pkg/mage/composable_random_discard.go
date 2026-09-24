package mage

type randomDiscardSubjectEffect struct {
	subject PlayerRef
	amount  ResolutionValue
}

// DiscardAtRandom makes the selected player discard cards at random.
// It reads the amount once and uses the game's random queue for each card.
func DiscardAtRandom(subject PlayerRef, amount ResolutionValue) ResolvingEffect {
	return &randomDiscardSubjectEffect{subject, amount}
}

func (e *randomDiscardSubjectEffect) Text() string {
	return "discard at random for " + InspectDefinition(e.subject).Text
}
func (*randomDiscardSubjectEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *randomDiscardSubjectEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
}
func (e *randomDiscardSubjectEffect) Apply(ctx *EffectContext) error {
	id, ok := e.subject.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	for range max(0, amount) {
		hand := player.Hand()
		if len(hand) == 0 {
			break
		}
		card := hand[ctx.Game.RandIntn(len(hand))]
		ctx.Game.PlayerDiscardByEffect(player, card.ID(), ctx.SourceID)
	}
	return nil
}
