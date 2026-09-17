package mage

import (
	"maps"
	"slices"

	"github.com/google/uuid"
)

type nextStepEffect struct {
	pattern TurnStepPattern
	body    ResolvingEffect
}

// AtNext creates a one-shot trigger for the next matching turn-step event.
// Saved references and the resolving controller are captured when it resolves.
func AtNext(pattern TurnStepPattern, body ResolvingEffect) ResolvingEffect {
	return &nextStepEffect{pattern, body}
}
func (*nextStepEffect) Text() string { return "at the next matching turn step" }
func (e *nextStepEffect) Properties() EffectProperties {
	if nilDefinition(e.body) {
		return EffectProperties{}
	}
	return e.body.Properties()
}
func (e *nextStepEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "delayed-step", Text: e.Text(), isolatedBindings: true, children: []definitionEdge{
		{"pattern", e.pattern}, {"body", e.body},
	}}
}
func (e *nextStepEffect) Apply(ctx *EffectContext) error {
	event, controllerOnly := e.pattern.delayedStep()
	body := &capturedDelayedBody{
		body:              e.body,
		permanents:        maps.Clone(ctx.Game.resolution.permanentBindings),
		targets:           cloneDelayedTargets(ctx.targetBindings),
		sourceIncarnation: ctx.sourceIncarnation,
		x:                 ctx.Game.resolution.X(),
	}
	if !ctx.sourceCaptured {
		if source := ctx.Game.FindPermanentIncludingPhased(ctx.SourceID); source != nil {
			body.sourceIncarnation = source.incarnationID
		}
	}
	dt := &DelayedTrigger{
		EventType:         event,
		SourceID:          ctx.SourceID,
		Controller:        ctx.Controller,
		sourceCaptured:    true,
		sourceIncarnation: body.sourceIncarnation,
		Effects:           []Effect{body},
	}
	if controllerOnly {
		dt.MatchPlayerID = ctx.Controller
	}
	ctx.Game.RegisterDelayedTrigger(dt)
	return nil
}

type capturedDelayedBody struct {
	body              ResolvingEffect
	permanents        map[*PermanentBinding]capturedPermanent
	targets           map[*targetDeclaration][]capturedPermanent
	sourceIncarnation uuid.UUID
	x                 int
}

func cloneDelayedTargets(targets map[*targetDeclaration][]capturedPermanent) map[*targetDeclaration][]capturedPermanent {
	result := maps.Clone(targets)
	for binding, values := range result {
		result[binding] = slices.Clone(values)
	}
	return result
}
func (e *capturedDelayedBody) Text() string                 { return e.body.Text() }
func (e *capturedDelayedBody) Properties() EffectProperties { return e.body.Properties() }
func (e *capturedDelayedBody) Apply(ctx *EffectContext) error {
	state := &ctx.Game.resolution
	outer, x := state.permanentBindings, state.X()
	state.permanentBindings = maps.Clone(e.permanents)
	state.SetX(e.x)
	defer func() { state.permanentBindings = outer; state.SetX(x) }()
	child := *ctx
	child.targetBindings = cloneDelayedTargets(e.targets)
	child.sourceCaptured = true
	child.sourceIncarnation = e.sourceIncarnation
	return e.body.Apply(&child)
}
