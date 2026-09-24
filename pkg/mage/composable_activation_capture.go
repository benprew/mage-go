package mage

import (
	"github.com/google/uuid"
)

// ActivationCaptureOption saves a value before activation payment.
type ActivationCaptureOption interface {
	ActivationOption
	GrantedActivationOption
	ModalActivationOption
}

type activationCapture interface {
	definitionInspector
	capture(*EffectContext)
}

type activationNumberCapture struct {
	value   ResolutionValue
	binding *NumberBinding
}

// CaptureNumberBeforePayment saves a value after target selection and cost
// validation, before mana production and payment. Copies retain that value.
func CaptureNumberBeforePayment(value ResolutionValue, binding *NumberBinding) ActivationCaptureOption {
	return activationNumberCapture{value, binding}
}

func (activationNumberCapture) activationOption()        {}
func (activationNumberCapture) grantedActivationOption() {}
func (activationNumberCapture) modalActivationOption()   {}
func (o activationNumberCapture) applyActivation(a *ActionDefinition) {
	a.dslCaptures = append(a.dslCaptures, o)
}
func (o activationNumberCapture) applyGrantedActivation(a *ActionDefinition) { o.applyActivation(a) }
func (o activationNumberCapture) applyModalActivation(a *ActionDefinition)   { o.applyActivation(a) }
func (o activationNumberCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "activation-number-capture", bindingWrite: o.binding,
		children: []definitionEdge{{"value", o.value}, {"binding", o.binding}}}
}

func (a *ActionDefinition) captureValues(g *Game, source *Permanent, controller uuid.UUID, specs []Target, targets []uuid.UUID) (numbers map[*NumberBinding]int, permanents map[*PermanentBinding]capturedPermanent) {
	if len(a.dslCaptures) == 0 {
		return nil, nil
	}
	outer := g.resolution.numberBindings
	outerPermanents := g.resolution.permanentBindings
	permanents = make(map[*PermanentBinding]capturedPermanent)
	g.resolution.permanentBindings = permanents
	defer func() { g.resolution.permanentBindings = outerPermanents }()
	values := make(map[*NumberBinding]int)
	g.resolution.numberBindings = values
	defer func() { g.resolution.numberBindings = outer }()
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: controller, Targets: targets,
		sourceCaptured: true, sourceIncarnation: source.incarnationID,
		targetBindings: make(map[*targetDeclaration][]capturedPermanent)}
	for i, spec := range expandTargetSpecs(specs, targets, g.resolution.X()) {
		if binding, ok := spec.(*targetDeclaration); ok && targets[i] != uuid.Nil {
			value := capturedPermanent{id: targets[i]}
			if p := g.FindPermanent(targets[i]); p != nil {
				value.incarnation = p.incarnationID
			}
			ctx.targetBindings[binding] = append(ctx.targetBindings[binding], value)
		}
	}
	for _, capture := range a.dslCaptures {
		capture.capture(ctx)
	}
	return values, permanents
}

func (o activationNumberCapture) capture(ctx *EffectContext) {
	ctx.Game.resolution.numberBindings[o.binding] = o.value.resolveValue(ctx)
}

type activationPermanentCapture struct {
	subject PermanentRef
	binding *PermanentBinding
}

// CapturePermanentBeforePayment saves a permanent after target and cost
// validation, before mana production and payment. Copies retain its identity.
func CapturePermanentBeforePayment(subject PermanentRef, binding *PermanentBinding) ActivationCaptureOption {
	return activationPermanentCapture{subject, binding}
}

func (activationPermanentCapture) activationOption()        {}
func (activationPermanentCapture) grantedActivationOption() {}
func (activationPermanentCapture) modalActivationOption()   {}
func (o activationPermanentCapture) applyActivation(a *ActionDefinition) {
	a.dslCaptures = append(a.dslCaptures, o)
}
func (o activationPermanentCapture) applyGrantedActivation(a *ActionDefinition) { o.applyActivation(a) }
func (o activationPermanentCapture) applyModalActivation(a *ActionDefinition)   { o.applyActivation(a) }
func (o activationPermanentCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "activation-permanent-capture", bindingWrite: o.binding,
		children: []definitionEdge{{"subject", o.subject}, {"binding", o.binding}}}
}
func (o activationPermanentCapture) capture(ctx *EffectContext) {
	snapshot := permanentSnapshot(o)
	_ = snapshot.Apply(ctx)
}
