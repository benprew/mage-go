package mage

import "github.com/google/uuid"

type playerSnapshot struct {
	subject PlayerRef
	binding *PlayerBinding
}

// SnapshotPlayer saves a player when this instruction resolves.
// An absent subject clears the previous value of the binding.
func SnapshotPlayer(subject PlayerRef, binding *PlayerBinding) ResolvingEffect {
	return &playerSnapshot{subject: subject, binding: binding}
}

func (*playerSnapshot) Text() string                 { return "save player" }
func (*playerSnapshot) Properties() EffectProperties { return EffectProperties{} }
func (e *playerSnapshot) Apply(ctx *EffectContext) error {
	id, ok := e.subject.resolvePlayer(ctx)
	if ctx.Game.resolution.playerBindings == nil {
		ctx.Game.resolution.playerBindings = make(map[*PlayerBinding]uuid.UUID)
	}
	delete(ctx.Game.resolution.playerBindings, e.binding)
	if ok {
		ctx.Game.resolution.playerBindings[e.binding] = id
	}
	return nil
}
func (e *playerSnapshot) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-snapshot", bindingWrite: e.binding, children: []definitionEdge{{"subject", e.subject}, {"binding", e.binding}}}
}
