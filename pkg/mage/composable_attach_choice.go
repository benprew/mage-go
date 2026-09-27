package mage

// MayAttach offers a legal attachment move to the resolving controller.
func MayAttach(attachment, host PermanentRef) ResolvingEffect {
	return &mayAttachEffect{attachment: attachment, host: host}
}

type mayAttachEffect struct {
	attachment PermanentRef
	host       PermanentRef
}

func (e *mayAttachEffect) Text() string               { return "you may attach permanent to permanent" }
func (*mayAttachEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *mayAttachEffect) Apply(ctx *EffectContext) error {
	attachment, ok := e.attachment.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	host, ok := e.host.resolvePermanent(ctx)
	if !ok || !ctx.Game.canAttach(attachment, host) {
		return nil
	}
	player := ctx.Game.GetPlayer(ctx.Controller)
	if player != nil && player.ChooseMayAbility("attach "+attachment.Name()+" to "+host.Name()) {
		ctx.Game.Attach(attachment.ID(), host.ID())
	}
	return nil
}
func (e *mayAttachEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"attachment", e.attachment}, {"host", e.host}}}
}
