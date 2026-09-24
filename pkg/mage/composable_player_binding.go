package mage

import "github.com/google/uuid"

// PlayerBinding declares a saved player. Each resolution has its own value.
type PlayerBinding struct{ name string }

// BindPlayer declares a player binding with a diagnostic name.
func BindPlayer(name string) *PlayerBinding { return &PlayerBinding{name: name} }

// FromPlayer reads a saved player. An absent binding has no subject.
func FromPlayer(binding *PlayerBinding) PlayerRef { return savedPlayerRef{binding} }

type savedPlayerRef struct{ binding *PlayerBinding }

func (savedPlayerRef) playerRef()       {}
func (savedPlayerRef) damageRecipient() {}
func (r savedPlayerRef) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	id, ok := ctx.Game.resolution.playerBindings[r.binding]
	return id, ok && ctx.Game.GetPlayer(id) != nil
}
func (r savedPlayerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (b *PlayerBinding) bindingName() string { return b.name }
func (b *PlayerBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "player-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}
func (r savedPlayerRef) inspectDefinition() definitionNode {
	label := "saved player"
	if r.binding != nil {
		label = r.binding.name
	}
	return definitionNode{Role: "player-ref", Text: label, bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

// ChooseRandomPlayer saves one random member at resolution without targeting.
// An empty set clears the binding and does not use a random value.
func ChooseRandomPlayer(subjects PlayerSet, binding *PlayerBinding) ResolvingEffect {
	return &randomPlayerChoice{subjects, binding}
}

type randomPlayerChoice struct {
	subjects PlayerSet
	binding  *PlayerBinding
}

func (*randomPlayerChoice) Text() string                 { return "choose a random player" }
func (*randomPlayerChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *randomPlayerChoice) Apply(ctx *EffectContext) error {
	candidates := e.subjects.resolvePlayers(ctx)
	if ctx.Game.resolution.playerBindings == nil {
		ctx.Game.resolution.playerBindings = make(map[*PlayerBinding]uuid.UUID)
	}
	delete(ctx.Game.resolution.playerBindings, e.binding)
	if len(candidates) == 0 {
		return nil
	}
	ctx.Game.resolution.playerBindings[e.binding] = candidates[ctx.Game.RandIntn(len(candidates))]
	return nil
}
func (e *randomPlayerChoice) inspectDefinition() definitionNode {
	return definitionNode{Role: "random-player-choice", bindingWrite: e.binding, children: []definitionEdge{{"subjects", e.subjects}, {"binding", e.binding}}}
}
