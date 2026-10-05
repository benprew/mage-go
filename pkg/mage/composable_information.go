package mage

import "github.com/google/uuid"

type permanentInformationReference interface {
	PermanentRef
	informationIdentity(*EffectContext) (capturedPermanent, bool)
}

func (b *PermanentTargetBinding) informationIdentity(ctx *EffectContext) (capturedPermanent, bool) {
	return firstTargetValue(ctx.targetBindings[b.targetDeclaration])
}

func (r savedPermanentRef) informationIdentity(ctx *EffectContext) (capturedPermanent, bool) {
	value, ok := ctx.Game.resolution.permanentBindings[r.binding]
	return value, ok
}

func (sourceRef) informationIdentity(ctx *EffectContext) (capturedPermanent, bool) {
	if ctx.sourceCaptured {
		return capturedPermanent{ctx.SourceID, ctx.sourceIncarnation}, ctx.sourceIncarnation != uuid.Nil
	}
	p := ctx.Game.FindPermanent(ctx.SourceID)
	if p == nil {
		return capturedPermanent{}, false
	}
	return capturedPermanent{p.ID(), p.incarnationID}, true
}

func (attachedPermanentRef) informationIdentity(ctx *EffectContext) (capturedPermanent, bool) {
	source, ok := This.informationIdentity(ctx)
	if !ok {
		return capturedPermanent{}, false
	}
	if p := ctx.Game.FindPermanentIncludingPhased(source.id); p != nil && p.incarnationID == source.incarnation {
		host := capturedPermanent{p.AttachedTo, p.attachedIncarnation}
		return host, host.id != uuid.Nil && host.incarnation != uuid.Nil
	}
	host, ok := ctx.Game.zones.departedAttachments[source.incarnation]
	return host, ok
}

func readPermanentInformation(subject PermanentRef, ctx *EffectContext) (*Permanent, *PermanentLKI) {
	ref, ok := subject.(permanentInformationReference)
	if !ok || nilDefinition(ref) {
		return nil, nil
	}
	identity, ok := ref.informationIdentity(ctx)
	if !ok {
		return nil, nil
	}
	if p := ctx.Game.FindPermanentIncludingPhased(identity.id); p != nil && p.incarnationID == identity.incarnation {
		if !p.PhasedOut {
			return p, nil
		}
		return nil, ctx.Game.zones.phaseInformation[identity.incarnation]
	}
	lki := ctx.Game.zones.LKIByIncarnation(identity.incarnation)
	if lki == nil || lki.ID != identity.id || lki.Snapshot == nil {
		return nil, nil
	}
	return nil, lki
}

func permanentInformationNode(subject PermanentRef, role, text string) definitionNode {
	n := definitionNode{Role: role, Text: text, children: []definitionEdge{{"subject", subject}}}
	if _, ok := subject.(permanentInformationReference); !ok {
		n.problem = "information requires a source, attachment, saved permanent or singular target"
	}
	return n
}

// ControllerOfWithLKI reads the exact incarnation, including its last controller.
// Information reads must not make a departed object available for an action.
func ControllerOfWithLKI(subject PermanentRef) PlayerRef {
	return permanentInformationController{subject}
}

type permanentInformationController struct{ subject PermanentRef }

func (permanentInformationController) playerRef()       {}
func (permanentInformationController) damageRecipient() {}
func (r permanentInformationController) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	p, lki := readPermanentInformation(r.subject, ctx)
	if lki != nil {
		p = lki.Snapshot
	}
	if p == nil {
		return uuid.Nil, false
	}
	return p.ControllerID(), ctx.Game.GetPlayer(p.ControllerID()) != nil
}
func (r permanentInformationController) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (r permanentInformationController) inspectDefinition() definitionNode {
	return permanentInformationNode(r.subject, "player-ref", "controller with last-known information")
}

// PowerOfWithLKI reads current power or the power saved before departure.
func PowerOfWithLKI(subject PermanentRef) ResolutionValue {
	return permanentInformationValue{subject: subject}
}

// ManaValueOfWithLKI reads current mana value or its last-known value.
func ManaValueOfWithLKI(subject PermanentRef) ResolutionValue {
	return permanentInformationValue{subject: subject, mana: true}
}

type permanentInformationValue struct {
	subject PermanentRef
	mana    bool
}

func (v permanentInformationValue) Text() string {
	if v.mana {
		return "mana value with last-known information"
	}
	return "power with last-known information"
}
func (v permanentInformationValue) resolveValue(ctx *EffectContext) int {
	p, lki := readPermanentInformation(v.subject, ctx)
	if lki != nil {
		if !v.mana {
			return lki.Power
		}
		p = lki.Snapshot
	}
	if p == nil {
		return 0
	}
	if v.mana {
		return p.Card.ManaCost().CMC()
	}
	return p.CurrentPower(ctx.Game)
}
func (v permanentInformationValue) inspectDefinition() definitionNode {
	return permanentInformationNode(v.subject, "permanent-value", v.Text())
}
