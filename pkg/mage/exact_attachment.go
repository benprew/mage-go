package mage

// AttachToPermanent attaches to one exact host without a choice or target.
// An illegal move leaves the attachment on its current host.
func AttachToPermanent(attachment, host PermanentRef) ResolvingEffect {
	return &exactAttachmentEffect{attachment: attachment, host: host}
}

type exactAttachmentEffect struct {
	attachment PermanentRef
	host       PermanentRef
}

func (*exactAttachmentEffect) Text() string                 { return "attach permanent to permanent" }
func (*exactAttachmentEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *exactAttachmentEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"attachment", e.attachment}, {"host", e.host}}}
}
func (e *exactAttachmentEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.attachment) || permanentRefReadsSourcePermanent(e.host)
}
func (e *exactAttachmentEffect) Apply(ctx *EffectContext) error {
	attachment, ok := e.attachment.resolvePermanent(ctx)
	if !ok || attachment.PhasedOut {
		return nil
	}
	host, ok := e.host.resolvePermanent(ctx)
	if !ok || host.PhasedOut {
		return nil
	}
	ctx.Game.Attach(attachment.ID(), host.ID())
	return nil
}
