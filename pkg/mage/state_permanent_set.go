package mage

import (
	"slices"

	"github.com/google/uuid"
)

// WhenPermanentsPresent saves the exact visits that satisfy a state trigger.
// Later entries cannot change the pending trigger's subjects.
func WhenPermanentsPresent(subjects PermanentSet, result *PermanentSetBinding, body ResolvingEffect) TriggeredAbility {
	return WhenState(permanentSetState{subjects, result}, body)
}

type permanentSetState struct {
	subjects PermanentSet
	result   *PermanentSetBinding
}

func (permanentSetState) stateCondition() {}

func (s permanentSetState) inspectDefinition() definitionNode {
	return definitionNode{Role: "state-condition", Text: "permanents are present", bindingWrite: s.result, children: []definitionEdge{{"subjects", s.subjects}, {"result", s.result}}}
}

func (s permanentSetState) checkState(reader GameReader, ctx StateContext) bool {
	g, ok := reader.(*Game)
	if !ok {
		return false
	}
	_, present := s.detectState(g, ctx.SourceID, ctx.ControllerID)
	return present
}

func (s permanentSetState) detectState(g *Game, source, controller uuid.UUID) ([]Effect, bool) {
	ctx := &EffectContext{Game: g, SourceID: source, Controller: controller}
	visits := []capturedPermanent{}
	for _, p := range s.subjects.resolvePermanents(ctx) {
		if p == nil {
			continue
		}
		current := g.FindPermanent(p.ID())
		visit := capturedPermanent{p.ID(), p.incarnationID}
		if current != nil && !current.PhasedOut && current.incarnationID == p.incarnationID && !slices.Contains(visits, visit) {
			visits = append(visits, visit)
		}
	}
	if len(visits) == 0 {
		return nil, false
	}
	return []Effect{&statePermanentSetCapture{s.result, visits}}, true
}

type statePermanentSetCapture struct {
	binding *PermanentSetBinding
	visits  []capturedPermanent
}

func (*statePermanentSetCapture) Text() string                 { return "remember the detected permanents" }
func (*statePermanentSetCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *statePermanentSetCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "state-permanent-set-capture", bindingWrite: e.binding, children: []definitionEdge{{"result", e.binding}}}
}
func (e *statePermanentSetCapture) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.createdTokenBindings == nil {
		ctx.Game.resolution.createdTokenBindings = make(map[*PermanentSetBinding][]capturedPermanent)
	}
	ctx.Game.resolution.createdTokenBindings[e.binding] = slices.Clone(e.visits)
	return nil
}
