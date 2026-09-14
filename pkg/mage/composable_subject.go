package mage

// PermanentRef identifies one battlefield incarnation.
type PermanentRef interface {
	permanentRef()
	resolvePermanent(*EffectContext) (*Permanent, bool)
}
type sourceRef struct{}

func (sourceRef) permanentRef() {}

// This refers to the source in the enclosing resolution context.
var This = sourceRef{}

func (sourceRef) resolvePermanent(ctx *EffectContext) (*Permanent, bool) {
	p := ctx.Game.FindPermanent(ctx.SourceID)
	if p == nil || p.PhasedOut {
		return nil, false
	}
	if ctx.sourceCaptured && p.incarnationID != ctx.sourceIncarnation {
		return nil, false
	}
	return p, true
}

type sacrificeEffect struct{ subject PermanentRef }

// Sacrifice instructs the stored controller to sacrifice the subject.
// An absent subject or a subject controlled by another player has no effect.
func Sacrifice(subject PermanentRef) ResolvingEffect  { return &sacrificeEffect{subject: subject} }
func (*sacrificeEffect) Text() string                 { return "sacrifice this permanent" }
func (*sacrificeEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *sacrificeEffect) Apply(ctx *EffectContext) error {
	if p, ok := e.subject.resolvePermanent(ctx); ok && p.ControllerID() == ctx.Controller {
		ctx.Game.DoSacrifice(p)
	}
	return nil
}
