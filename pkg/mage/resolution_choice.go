package mage

import (
	"fmt"
	"slices"
)

// ResolutionOption supplies an action and an optional legality condition.
// The condition must include any restrictions on the choice.
type ResolutionOption struct {
	Label string
	Body  ResolvingEffect
	When  StateCondition
}

// ChooseAtResolution asks the selected player to choose one available action.
// The choice belongs to this invocation and uses the current resolution frame.
// Sequence bodies keep their saved inputs. Local results cannot supply another
// option or a later instruction outside the choice.
func ChooseAtResolution(player PlayerRef, options ...ResolutionOption) ResolvingEffect {
	return &resolutionChoice{player: player, options: slices.Clone(options)}
}

type resolutionChoice struct {
	player  PlayerRef
	options []ResolutionOption
}

func (*resolutionChoice) Text() string { return "choose an action" }
func (e *resolutionChoice) Properties() EffectProperties {
	var effects []Effect
	for _, option := range e.options {
		if !nilDefinition(option.Body) {
			effects = append(effects, option.Body)
		}
	}
	return CompositeEffects("", effects...).Properties()
}

func (e *resolutionChoice) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return nil
	}
	var labels []string
	var bodies []ResolvingEffect
	state := StateContext{SourceID: ctx.SourceID, ControllerID: ctx.Controller, SourceIncarnation: ctx.sourceIncarnation}
	for _, option := range e.options {
		if option.When != nil && !option.When.checkState(ctx.Game, state) {
			continue
		}
		labels = append(labels, option.Label)
		bodies = append(bodies, option.Body)
	}
	if len(bodies) == 0 {
		return nil
	}
	choice := 0
	if len(bodies) > 1 {
		choice = player.ChooseMode(labels, e.Text())
	}
	if choice < 0 || choice >= len(bodies) {
		choice = 0
	}
	return bodies[choice].Apply(ctx)
}

func (e *resolutionChoice) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolution-choice", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
	if len(e.options) == 0 {
		n.problem = "supply at least one resolution option"
	}
	for i, option := range e.options {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("options[%d]", i), option})
	}
	return n
}

func (o ResolutionOption) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolution-option", Text: o.Label, isolatedBindings: true}
	if o.Label == "" {
		n.problem = "supply a resolution option label"
	}
	if o.When != nil {
		n.children = append(n.children, definitionEdge{"condition", o.When})
	}
	n.children = append(n.children, definitionEdge{"body", o.Body})
	return n
}
