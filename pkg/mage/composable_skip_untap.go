package mage

// SkipNextUntap prevents untapping during the next untap step of the subject's
// current controller. It does not tap the subject or prevent other untap effects.
func SkipNextUntap(subject PermanentRef) ResolvingEffect {
	return &skipNextUntapEffect{subject}
}

type skipNextUntapEffect struct{ subject PermanentRef }

func (e *skipNextUntapEffect) Text() string {
	return InspectDefinition(e.subject).Text + " doesn't untap during its controller's next untap step"
}
func (*skipNextUntapEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *skipNextUntapEffect) Apply(ctx *EffectContext) error {
	if p, ok := e.subject.resolvePermanent(ctx); ok {
		ctx.Game.SkipNextUntap(p.ID())
	}
	return nil
}
func (e *skipNextUntapEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}

// SkipNextUntapOf prevents untapping during the selected player's next untap
// step. It captures the player at resolution, even if control changes later.
func SkipNextUntapOf(subject PermanentRef, player PlayerRef) ResolvingEffect {
	return &capturedUntapEffect{subject: subject, player: player}
}

type capturedUntapEffect struct {
	subject PermanentRef
	player  PlayerRef
}

func (e *capturedUntapEffect) Text() string {
	return InspectDefinition(e.subject).Text + " doesn't untap during the next untap step of " + InspectDefinition(e.player).Text
}
func (*capturedUntapEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *capturedUntapEffect) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	player, ok := e.player.resolvePlayer(ctx)
	if ok {
		ctx.Game.turns.skipNextUntapOf(p.ID(), player)
	}
	return nil
}
func (e *capturedUntapEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"player", e.player}}}
}
