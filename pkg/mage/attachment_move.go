package mage

import "slices"

// AttachPermanentToChoice moves an attachment to a required legal host choice.
// The choice does not target. Return cannot replace either saved visit.
func AttachPermanentToChoice(player PlayerRef, attachment PermanentRef, hosts PermanentSet) ResolvingEffect {
	result := BindPermanent("attachment host")
	return &attachmentMoveEffect{
		attachment: attachment,
		hosts:      hosts,
		choice:     &permanentChoice{player, legalAttachmentHosts{attachment, hosts}, result},
	}
}

type attachmentMoveEffect struct {
	attachment PermanentRef
	hosts      PermanentSet
	choice     *permanentChoice
}

func (*attachmentMoveEffect) Text() string                 { return "attach permanent to a chosen permanent" }
func (*attachmentMoveEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *attachmentMoveEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"choice", e.choice}}}
}
func (e *attachmentMoveEffect) Apply(ctx *EffectContext) error {
	delete(ctx.Game.resolution.permanentBindings, e.choice.binding)
	attachment, ok := e.attachment.resolvePermanent(ctx)
	if !ok || attachment.PhasedOut {
		return nil
	}
	visit := capturedPermanent{attachment.ID(), attachment.incarnationID}
	if err := e.choice.Apply(ctx); err != nil {
		return err
	}
	attachment = ctx.Game.FindPermanent(visit.id)
	if attachment == nil || attachment.PhasedOut || attachment.incarnationID != visit.incarnation {
		return nil
	}
	host, ok := From(e.choice.binding).resolvePermanent(ctx)
	if !ok || host.PhasedOut || !ctx.Game.canAttach(attachment, host) {
		return nil
	}
	for _, candidate := range e.hosts.resolvePermanents(ctx) {
		if candidate.ID() == host.ID() && candidate.incarnationID == host.incarnationID {
			ctx.Game.Attach(attachment.ID(), host.ID())
			break
		}
	}
	return nil
}
func (e *attachmentMoveEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.attachment) || playerRefReadsSourcePermanent(e.choice.player) || permanentSetReadsSourcePermanent(e.hosts)
}

type legalAttachmentHosts struct {
	attachment PermanentRef
	hosts      PermanentSet
}

func (legalAttachmentHosts) permanentSet() {}
func (s legalAttachmentHosts) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-set", Text: "legal attachment hosts", children: []definitionEdge{{"attachment", s.attachment}, {"hosts", s.hosts}}}
}
func (s legalAttachmentHosts) resolvePermanents(ctx *EffectContext) []*Permanent {
	attachment, ok := s.attachment.resolvePermanent(ctx)
	if !ok || attachment.PhasedOut {
		return nil
	}
	var hosts []*Permanent
	for _, host := range s.hosts.resolvePermanents(ctx) {
		if host == nil {
			continue
		}
		current := ctx.Game.FindPermanent(host.ID())
		if current != nil && !current.PhasedOut && current.incarnationID == host.incarnationID && ctx.Game.canAttach(attachment, current) && !slices.Contains(hosts, current) {
			hosts = append(hosts, current)
		}
	}
	return hosts
}
