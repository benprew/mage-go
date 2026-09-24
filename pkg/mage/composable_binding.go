package mage

import (
	"github.com/google/uuid"
)

// PermanentBinding declares a saved permanent. Each resolution stores its own value.
// The name is for diagnostics. The declaration identifies the binding.
type PermanentBinding struct{ name string }

// BindPermanent declares a snapshot binding with a diagnostic name.
func BindPermanent(name string) *PermanentBinding { return &PermanentBinding{name: name} }

type capturedPermanent struct{ id, incarnation uuid.UUID }
type savedPermanentRef struct{ binding *PermanentBinding }

// From reads a saved battlefield incarnation. An absent snapshot has no subject.
func From(binding *PermanentBinding) PermanentRef { return savedPermanentRef{binding: binding} }
func (savedPermanentRef) permanentRef()           {}
func (r savedPermanentRef) resolvePermanent(ctx *EffectContext) (*Permanent, bool) {
	value, ok := ctx.Game.resolution.permanentBindings[r.binding]
	if !ok {
		return nil, false
	}
	p := ctx.Game.FindPermanent(value.id)
	if p == nil || p.PhasedOut || p.incarnationID != value.incarnation {
		return nil, false
	}
	return p, true
}

type attachedPermanentRef struct{}

// AttachedPermanent reads the attachment of the source incarnation.
// A departed source supplies its last attachment; a returned host is absent.
var AttachedPermanent = attachedPermanentRef{}

func (attachedPermanentRef) permanentRef() {}
func (attachedPermanentRef) resolvePermanent(ctx *EffectContext) (*Permanent, bool) {
	source, ok := This.resolvePermanent(ctx)
	var host capturedPermanent
	if ok {
		host = capturedPermanent{id: source.AttachedTo, incarnation: source.attachedIncarnation}
	} else if ctx.sourceCaptured {
		source = ctx.Game.FindPermanentIncludingPhased(ctx.SourceID)
		if source != nil && source.incarnationID == ctx.sourceIncarnation {
			host = capturedPermanent{id: source.AttachedTo, incarnation: source.attachedIncarnation}
		} else {
			host = ctx.Game.zones.departedAttachments[ctx.sourceIncarnation]
		}
	} else {
		return nil, false
	}
	p := ctx.Game.FindPermanent(host.id)
	if p == nil || (host.incarnation != uuid.Nil && p.incarnationID != host.incarnation) {
		return nil, false
	}
	return p, true
}

type permanentSnapshot struct {
	subject PermanentRef
	binding *PermanentBinding
}

// Snapshot records a subject before later instructions can move it.
// An absent subject records absence and replaces any earlier snapshot.
func Snapshot(subject PermanentRef, binding *PermanentBinding) ResolvingEffect {
	return &permanentSnapshot{subject: subject, binding: binding}
}
func (*permanentSnapshot) Text() string                 { return "save permanent" }
func (*permanentSnapshot) Properties() EffectProperties { return EffectProperties{} }
func (e *permanentSnapshot) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.permanentBindings == nil {
		ctx.Game.resolution.permanentBindings = make(map[*PermanentBinding]capturedPermanent)
	}
	p, ok := e.subject.resolvePermanent(ctx)
	delete(ctx.Game.resolution.permanentBindings, e.binding)
	if ok {
		ctx.Game.resolution.permanentBindings[e.binding] = capturedPermanent{p.ID(), p.incarnationID}
	}
	return nil
}

func (b *PermanentBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}
func (r savedPermanentRef) inspectDefinition() definitionNode {
	label := "saved permanent"
	if r.binding != nil {
		label = r.binding.name
	}
	return definitionNode{Role: "permanent-ref", Text: label, bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}
func (attachedPermanentRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-ref", Text: "attached permanent", battlefieldSource: true}
}
func (e *permanentSnapshot) inspectDefinition() definitionNode {
	return definitionNode{Role: "snapshot", bindingWrite: e.binding, children: []definitionEdge{{"subject", e.subject}, {"binding", e.binding}}}
}

func (b *PermanentBinding) bindingName() string { return b.name }
