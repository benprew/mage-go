package mage

import "github.com/google/uuid"

// HostOfPermanent reads the attachment host of an exact permanent visit.
// Departure saves the last attachment. Return cannot replace either visit.
func HostOfPermanent(attachment PermanentRef) PermanentRef {
	return permanentHostRef{attachment}
}

type permanentHostRef struct{ attachment PermanentRef }

func (permanentHostRef) permanentRef()    {}
func (permanentHostRef) damageRecipient() {}
func (r permanentHostRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	p, ok := r.resolvePermanent(ctx)
	if !ok {
		return uuid.Nil, false
	}
	return p.ID(), true
}

func (r permanentHostRef) informationIdentity(ctx *EffectContext) (capturedPermanent, bool) {
	ref, ok := r.attachment.(permanentInformationReference)
	if !ok || nilDefinition(ref) {
		return capturedPermanent{}, false
	}
	attachment, ok := ref.informationIdentity(ctx)
	if !ok {
		return capturedPermanent{}, false
	}
	if p := ctx.Game.FindPermanentIncludingPhased(attachment.id); p != nil && p.incarnationID == attachment.incarnation {
		host := capturedPermanent{p.AttachedTo, p.attachedIncarnation}
		return host, host.id != uuid.Nil && host.incarnation != uuid.Nil
	}
	host, ok := ctx.Game.zones.departedAttachments[attachment.incarnation]
	return host, ok
}

func (r permanentHostRef) resolvePermanent(ctx *EffectContext) (*Permanent, bool) {
	host, ok := r.informationIdentity(ctx)
	if !ok {
		return nil, false
	}
	p := ctx.Game.FindPermanent(host.id)
	return p, p != nil && !p.PhasedOut && p.incarnationID == host.incarnation
}

func (r permanentHostRef) inspectDefinition() definitionNode {
	return permanentInformationNode(r.attachment, "permanent-ref", "attachment host")
}
