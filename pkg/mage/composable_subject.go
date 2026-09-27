package mage

// PermanentRef identifies one battlefield incarnation.
type PermanentRef interface {
	DamageRecipientRef
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

type sacrificeEffect struct {
	subject       PermanentRef
	result        *NumberBinding
	subtypeResult *NumberBinding
	subtype       string
	invalidOption bool
	optionCount   int
}

// SacrificeResultOption selects an output for one sacrifice instruction.
type SacrificeResultOption interface {
	sacrificeResultOption()
	applySacrificeResult(*sacrificeEffect)
}

type sacrificeAmountOption struct{ binding *NumberBinding }

// SacrificeAmountInto saves one if the permanent was sacrificed, or zero.
func SacrificeAmountInto(binding *NumberBinding) SacrificeResultOption {
	return sacrificeAmountOption{binding}
}

func (sacrificeAmountOption) sacrificeResultOption() {}
func (o sacrificeAmountOption) applySacrificeResult(e *sacrificeEffect) {
	e.result = o.binding
	e.optionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

type sacrificeSubtypeOption struct {
	subtype string
	binding *NumberBinding
}

// SacrificeSubtypeInto saves one if the sacrificed permanent had the subtype.
func SacrificeSubtypeInto(subtype string, binding *NumberBinding) SacrificeResultOption {
	return sacrificeSubtypeOption{subtype: subtype, binding: binding}
}

func (sacrificeSubtypeOption) sacrificeResultOption() {}
func (o sacrificeSubtypeOption) applySacrificeResult(e *sacrificeEffect) {
	e.subtype = o.subtype
	e.subtypeResult = o.binding
	e.optionCount++
	if o.binding == nil || o.subtype == "" {
		e.invalidOption = true
	}
}

// Sacrifice instructs the stored controller to sacrifice the subject.
// An absent subject or a subject controlled by another player has no effect.
func Sacrifice(subject PermanentRef, options ...SacrificeResultOption) ResolvingEffect {
	e := &sacrificeEffect{subject: subject}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalidOption = true
			continue
		}
		option.applySacrificeResult(e)
	}
	return e
}
func (e *sacrificeEffect) Text() string {
	if _, ok := e.subject.(sourceRef); ok {
		return "sacrifice this permanent"
	}
	if subject, ok := e.subject.(definitionInspector); ok {
		return "sacrifice " + subject.inspectDefinition().Text
	}
	return "sacrifice permanent"
}
func (e *sacrificeEffect) Properties() EffectProperties {
	_, source := e.subject.(sourceRef)
	return EffectProperties{SacrificesSource: source}
}
func (e *sacrificeEffect) Apply(ctx *EffectContext) error {
	if e.result != nil || e.subtypeResult != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		if e.result != nil {
			ctx.Game.resolution.numberBindings[e.result] = 0
		}
		if e.subtypeResult != nil {
			ctx.Game.resolution.numberBindings[e.subtypeResult] = 0
		}
	}
	if p, ok := e.subject.resolvePermanent(ctx); ok && p.ControllerID() == ctx.Controller {
		matched := e.subtypeResult != nil && p.HasSubType(e.subtype)
		ctx.Game.DoSacrifice(p)
		if e.result != nil {
			ctx.Game.resolution.numberBindings[e.result] = 1
		}
		if matched {
			ctx.Game.resolution.numberBindings[e.subtypeResult] = 1
		}
	}
	return nil
}
