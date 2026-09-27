package mage

import (
	"maps"
	"slices"

	"github.com/google/uuid"
)

// DelayedStepPattern describes a future turn-step event.
type DelayedStepPattern interface {
	delayedStep(uuid.UUID) *DelayedTrigger
}

// DelayedStepOption is a closed option for a delayed step trigger.
type DelayedStepOption interface {
	delayedStepOption()
	applyDelayedStep(*nextStepEffect)
}

func (targetOptions) delayedStepOption() {}
func (o targetOptions) applyDelayedStep(e *nextStepEffect) {
	for _, b := range o.bindings {
		if nilDefinition(b) {
			panic("delayed targets: supply a non-nil binding")
		}
		e.targets = append(e.targets, b.targetSpec())
	}
}

func (interveningIfOption) delayedStepOption() {}
func (o interveningIfOption) applyDelayedStep(e *nextStepEffect) {
	if nilDefinition(o.condition) {
		panic("intervening-if: supply a condition")
	}
	if e.condition != nil {
		panic("intervening-if: supply only one condition")
	}
	e.condition = o.condition
}

type nextStepEffect struct {
	modes     []Mode
	condition StateCondition
	targets   []Target
	pattern   DelayedStepPattern
	body      ResolvingEffect
}

// AtNext creates a one-shot trigger for the next matching turn-step event.
// Saved references and the resolving controller are captured when it resolves.
func AtNext(pattern DelayedStepPattern, body ResolvingEffect, options ...DelayedStepOption) ResolvingEffect {
	e := &nextStepEffect{pattern: pattern, body: body}
	for _, option := range options {
		if nilDefinition(option) {
			panic("delayed step: supply a non-nil option")
		}
		option.applyDelayedStep(e)
	}
	return e
}

// ModalDelayedStepOption is a mode or condition for a delayed step trigger.
type ModalDelayedStepOption interface {
	modalDelayedStepOption()
	applyModalDelayedStep(*nextStepEffect)
}

func (ModeDefinition) modalDelayedStepOption() {}
func (m ModeDefinition) applyModalDelayedStep(e *nextStepEffect) {
	e.modes = append(e.modes, m.mode)
}
func (interveningIfOption) modalDelayedStepOption() {}
func (o interveningIfOption) applyModalDelayedStep(e *nextStepEffect) {
	o.applyDelayedStep(e)
}

// AtNextModes creates a one-shot step trigger with a mode chosen at stack placement.
func AtNextModes(pattern DelayedStepPattern, options ...ModalDelayedStepOption) ResolvingEffect {
	if nilDefinition(pattern) {
		panic("delayed step: supply a pattern")
	}
	e := &nextStepEffect{pattern: pattern}
	for _, option := range options {
		if nilDefinition(option) {
			panic("delayed modes: supply a non-nil option")
		}
		option.applyModalDelayedStep(e)
	}
	if len(e.modes) < 2 {
		panic("modes: supply at least two mode bodies")
	}
	return e
}

func (*nextStepEffect) Text() string { return "at the next matching turn step" }
func (e *nextStepEffect) Properties() EffectProperties {
	if len(e.modes) > 0 {
		var effects []Effect
		for _, mode := range e.modes {
			effects = append(effects, mode.Effects...)
		}
		return Sequence(effects...).Properties()
	}
	if nilDefinition(e.body) {
		return EffectProperties{}
	}
	return e.body.Properties()
}
func (e *nextStepEffect) inspectDefinition() definitionNode {
	children := declaredTargetNodes(e.targets)
	children = append(children, definitionEdge{"pattern", e.pattern})
	if e.condition != nil {
		children = append(children, definitionEdge{"condition", e.condition})
	}
	if len(e.modes) > 0 {
		children = append(children, modeEdges(e.modes)...)
	} else {
		children = append(children, definitionEdge{"body", e.body})
	}
	return definitionNode{Role: "delayed-step", Text: e.Text(), isolatedBindings: true, children: children}
}
func (e *nextStepEffect) Apply(ctx *EffectContext) error {
	dt := e.pattern.delayedStep(ctx.Controller)
	body := &capturedDelayedBody{
		body:              e.body,
		permanents:        maps.Clone(ctx.Game.resolution.permanentBindings),
		createdTokens:     cloneCreatedTokenBindings(ctx.Game.resolution.createdTokenBindings),
		players:           maps.Clone(ctx.Game.resolution.playerBindings),
		numbers:           maps.Clone(ctx.Game.resolution.numberBindings),
		targets:           cloneDelayedTargets(ctx.targetBindings),
		sourceIncarnation: ctx.sourceIncarnation,
		x:                 ctx.Game.resolution.X(),
	}
	if !ctx.sourceCaptured {
		if source := ctx.Game.FindPermanentIncludingPhased(ctx.SourceID); source != nil {
			body.sourceIncarnation = source.incarnationID
		}
	}
	dt.interveningCondition = e.condition
	if e.condition != nil && e.body != nil {
		body.body = IfState(e.condition, body.body)
	}
	dt.targets = slices.Clone(e.targets)
	dt.SourceID = ctx.SourceID
	dt.Controller = ctx.Controller
	dt.sourceCaptured = true
	dt.sourceIncarnation = body.sourceIncarnation
	if len(e.modes) > 0 {
		dt.modes = slices.Clone(e.modes)
		for i, mode := range dt.modes {
			captured := *body
			captured.body = Sequence(mode.Effects...)
			if e.condition != nil {
				captured.body = IfState(e.condition, captured.body)
			}
			dt.modes[i].Effects = []Effect{&captured}
			dt.modes[i].Targets = slices.Clone(mode.Targets)
		}
	} else {
		dt.Effects = []Effect{body}
	}
	ctx.Game.RegisterDelayedTrigger(dt)
	return nil
}

type capturedDelayedBody struct {
	body              ResolvingEffect
	permanents        map[*PermanentBinding]capturedPermanent
	createdTokens     map[*CreatedTokensBinding][]capturedPermanent
	players           map[*PlayerBinding]uuid.UUID
	numbers           map[*NumberBinding]int
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
	outerNumbers := state.numberBindings
	state.numberBindings = maps.Clone(e.numbers)
	defer func() { state.numberBindings = outerNumbers }()
	outerPlayers := state.playerBindings
	state.playerBindings = maps.Clone(e.players)
	defer func() { state.playerBindings = outerPlayers }()
	outer, x := state.permanentBindings, state.X()
	state.permanentBindings = maps.Clone(e.permanents)
	state.SetX(e.x)
	defer func() { state.permanentBindings = outer; state.SetX(x) }()
	outerCreated := state.createdTokenBindings
	state.createdTokenBindings = cloneCreatedTokenBindings(e.createdTokens)
	defer func() { state.createdTokenBindings = outerCreated }()
	child := *ctx
	child.targetBindings = cloneDelayedTargets(e.targets)
	if child.targetBindings == nil {
		child.targetBindings = make(map[*targetDeclaration][]capturedPermanent)
	}
	for binding, values := range ctx.targetBindings {
		child.targetBindings[binding] = slices.Clone(values)
	}
	child.sourceCaptured = true
	child.sourceIncarnation = e.sourceIncarnation
	return e.body.Apply(&child)
}
