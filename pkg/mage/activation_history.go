package mage

import (
	"fmt"

	"github.com/google/uuid"
)

// ActivationHistoryBinding keeps each ability's history separate.
type ActivationHistoryBinding struct{ name string }

// BindActivationHistory names the history of one activated ability.
func BindActivationHistory(name string) *ActivationHistoryBinding {
	return &ActivationHistoryBinding{name: name}
}
func (b *ActivationHistoryBinding) bindingName() string { return b.name }
func (b *ActivationHistoryBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "activation-history", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply an activation history name"
	}
	return n
}

// WithActivationHistory permits resolution to read completed activations.
func WithActivationHistory(binding *ActivationHistoryBinding) ActionOption {
	return func(a *ActionDefinition) { a.activationHistory = binding; a.activationHistoryOptions++ }
}

type activationHistoryKey struct {
	binding     *ActivationHistoryBinding
	incarnation uuid.UUID
}

type activationHistoryDeclaration struct{ binding *ActivationHistoryBinding }

func (d activationHistoryDeclaration) inspectDefinition() definitionNode {
	return definitionNode{Role: "activation-history-declaration", bindingWrite: d.binding, children: []definitionEdge{{"binding", d.binding}}}
}

// IfActivationsAtLeast checks history at resolution, including countered activations.
func IfActivationsAtLeast(binding *ActivationHistoryBinding, minimum int, body ResolvingEffect) ResolvingEffect {
	return &activationHistoryBranch{binding: binding, minimum: minimum, body: body}
}

type activationHistoryBranch struct {
	binding *ActivationHistoryBinding
	minimum int
	body    ResolvingEffect
}

func (e *activationHistoryBranch) Text() string {
	return fmt.Sprintf("if this ability has been activated at least %d times this turn", e.minimum)
}
func (e *activationHistoryBranch) Properties() EffectProperties {
	if e.body == nil {
		return EffectProperties{}
	}
	return e.body.Properties()
}
func (e *activationHistoryBranch) Apply(ctx *EffectContext) error {
	key := activationHistoryKey{binding: e.binding, incarnation: ctx.sourceIncarnation}
	if ctx.Game.trackers.Turn.abilityActivations[key] >= e.minimum {
		return e.body.Apply(ctx)
	}
	return nil
}
func (e *activationHistoryBranch) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolution-branch", bindingRead: e.binding, children: []definitionEdge{{"binding", e.binding}, {"then", e.body}}}
	if e.binding == nil {
		n.problem = "supply an activation history binding"
	} else if e.minimum < 0 {
		n.problem = "supply a nonnegative activation threshold"
	}
	return n
}
