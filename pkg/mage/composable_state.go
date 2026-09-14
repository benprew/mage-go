package mage

import (
	"slices"
	"strings"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// StateCondition is a pure condition that can be checked more than once.
type StateCondition interface {
	stateCondition()
	checkState(GameReader, StateContext) bool
	inspectDefinition() definitionNode
}

// StateContext identifies the source and controller for a condition check.
type StateContext struct {
	SourceID     uuid.UUID
	ControllerID uuid.UUID
}

// StateDependency declares the data that a callback reads.
type StateDependency uint8

const (
	StateBattlefield StateDependency = iota + 1
	StateSourceChoices
)

// StatePermanent is a snapshot. It contains no mutable engine objects.
type StatePermanent struct {
	id, controller, chosenPlayer uuid.UUID
	chosenColor                  Color
	colors                       []Color
	token                        bool
}

func (p StatePermanent) ID() uuid.UUID           { return p.id }
func (p StatePermanent) ControllerID() uuid.UUID { return p.controller }
func (p StatePermanent) ChosenPlayer() uuid.UUID { return p.chosenPlayer }
func (p StatePermanent) ChosenColor() Color      { return p.chosenColor }
func (p StatePermanent) HasColor(c Color) bool   { return slices.Contains(p.colors, c) }
func (p StatePermanent) IsToken() bool           { return p.token }

// StateReader supplies immutable snapshots to pure condition callbacks.
type StateReader interface {
	Permanent(uuid.UUID) (StatePermanent, bool)
	Battlefield() []StatePermanent
	stateReader()
}
type stateSnapshot struct{ permanents []StatePermanent }

func (stateSnapshot) stateReader() {}
func (s stateSnapshot) Permanent(id uuid.UUID) (StatePermanent, bool) {
	for _, p := range s.permanents {
		if p.id == id {
			return p, true
		}
	}
	return StatePermanent{}, false
}
func (s stateSnapshot) Battlefield() []StatePermanent { return slices.Clone(s.permanents) }
func snapshotState(g GameReader) StateReader {
	s := stateSnapshot{}
	for _, p := range g.FilterBattlefield(AnyPermanent) {
		s.permanents = append(s.permanents, StatePermanent{id: p.ID(), controller: p.ControllerID(), chosenPlayer: p.ChosenPlayer, chosenColor: p.ChosenColor, colors: slices.Clone(p.Colors()), token: p.IsToken})
	}
	return s
}

type statePredicate struct {
	label        string
	filter       PermanentFilter
	kind         string
	children     []StateCondition
	dependencies []StateDependency
	fn           func(StateReader, StateContext) bool
}

func (*statePredicate) stateCondition() {}
func (c *statePredicate) checkState(g GameReader, ctx StateContext) bool {
	switch c.kind {
	case "controls", "none":
		found := false
		for _, p := range g.FilterBattlefield(c.filter) {
			if p.ControllerID() == ctx.ControllerID {
				found = true
				break
			}
		}
		return found == (c.kind == "controls")
	case "battlefield-none":
		return len(g.FilterBattlefield(c.filter)) == 0
	case "all":
		for _, child := range c.children {
			if !child.checkState(g, ctx) {
				return false
			}
		}
		return true
	case "any":
		for _, child := range c.children {
			if child.checkState(g, ctx) {
				return true
			}
		}
		return false
	case "not":
		return !c.children[0].checkState(g, ctx)
	default:
		return c.fn(snapshotState(g), ctx)
	}
}

// Controls includes the source if it matches the filter.
func Controls(f PermanentFilter) StateCondition {
	return &statePredicate{kind: "controls", filter: f, label: "you control " + f.Text()}
}

// ControlsNone checks all permanents of the current controller.
func ControlsNone(f PermanentFilter) StateCondition {
	return &statePredicate{kind: "none", filter: f, label: "you control no " + f.Text()}
}

// BattlefieldHasNone checks all players' permanents.
func BattlefieldHasNone(f PermanentFilter) StateCondition {
	return &statePredicate{kind: "battlefield-none", filter: f, label: "no " + f.Text() + " on the battlefield"}
}

// AllConditions requires each condition to be true.
func AllConditions(cs ...StateCondition) StateCondition {
	return &statePredicate{kind: "all", children: slices.Clone(cs)}
}

// AnyCondition requires at least one condition to be true.
func AnyCondition(cs ...StateCondition) StateCondition {
	return &statePredicate{kind: "any", children: slices.Clone(cs)}
}

// NotCondition reverses a condition.
func NotCondition(c StateCondition) StateCondition {
	return &statePredicate{kind: "not", children: []StateCondition{c}}
}

// StateConditionFunc wraps a pure callback. Declare all data that it reads.
// The callback must not use randomness, choices, mutation, or mutable captures.
func StateConditionFunc(label string, dependencies []StateDependency, fn func(StateReader, StateContext) bool) StateCondition {
	return &statePredicate{kind: "callback", label: label, dependencies: slices.Clone(dependencies), fn: fn}
}

// WhenState uses the existing pending-trigger and stack lifecycle.
func WhenState(condition StateCondition, effect ResolvingEffect) TriggeredAbility {
	t := NewStateTriggered(false, effect)
	t.stateDefinition = condition
	t.dslState = true
	t.Condition = func(_ *GameEvent, g GameReader, source, controller uuid.UUID) bool {
		return condition.checkState(g, StateContext{SourceID: source, ControllerID: controller})
	}
	return t
}

// IfState checks a pure condition when the effect resolves.
func IfState(condition StateCondition, effect ResolvingEffect) ResolvingEffect {
	return &stateBranch{condition: condition, effect: effect}
}

type stateBranch struct {
	condition StateCondition
	effect    ResolvingEffect
}

func (e *stateBranch) Text() string {
	return "if " + e.condition.inspectDefinition().Text + ", " + e.effect.Text()
}
func (e *stateBranch) Properties() EffectProperties { return e.effect.Properties() }
func (e *stateBranch) Apply(ctx *EffectContext) error {
	if e.condition.checkState(ctx.Game, StateContext{SourceID: ctx.SourceID, ControllerID: ctx.Controller}) {
		return e.effect.Apply(ctx)
	}
	return nil
}

// Sequence executes instructions in order through the existing effect executor.
func Sequence(effects ...ResolvingEffect) ResolvingEffect {
	return &sequenceEffect{effects: slices.Clone(effects)}
}

type sequenceEffect struct{ effects []ResolvingEffect }

func (e *sequenceEffect) Text() string {
	texts := make([]string, len(e.effects))
	for i, c := range e.effects {
		texts[i] = c.Text()
	}
	return strings.Join(texts, "; ")
}
func (e *sequenceEffect) Properties() EffectProperties {
	return CompositeEffects("", e.effects...).Properties()
}
func (e *sequenceEffect) Apply(ctx *EffectContext) error {
	return CompositeEffects("", e.effects...).Apply(ctx)
}
