package mage

// MayTapOrUntap lets the resolving controller change the subject's tapped state.
// The controller chooses at resolution. An absent subject has no effect.
func MayTapOrUntap(subject PermanentRef) ResolvingEffect {
	return &tapChoiceEffect{subject}
}

type tapChoiceEffect struct{ subject PermanentRef }

func (e *tapChoiceEffect) Text() string {
	return "you may tap or untap " + InspectDefinition(e.subject).Text
}
func (*tapChoiceEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *tapChoiceEffect) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(ctx.Controller)
	if player == nil {
		return nil
	}
	tapped := p.Tapped
	verb := "tap "
	if tapped {
		verb = "untap "
	}
	if !player.ChooseMayAbility(verb + p.Name()) {
		return nil
	}
	if tapped {
		ctx.Game.UntapPermanent(p)
	} else {
		ctx.Game.TapPermanent(p)
	}
	return nil
}
func (e *tapChoiceEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "may-tap-or-untap", Text: e.Text(), children: []definitionEdge{
		{"player", You}, {"tap", TapSubject(e.subject)}, {"untap", UntapPermanent(e.subject)},
	}}
}
