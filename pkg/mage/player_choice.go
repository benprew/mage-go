package mage

import (
	"slices"

	"github.com/google/uuid"
)

type playerChoice struct {
	player   PlayerRef
	subjects PlayerSet
	binding  *PlayerBinding
}

// ChoosePlayer saves a required player choice at resolution without targeting.
// An empty set or absent chooser clears the saved result.
func ChoosePlayer(player PlayerRef, subjects PlayerSet, binding *PlayerBinding) ResolvingEffect {
	return &playerChoice{player, subjects, binding}
}

func (*playerChoice) Text() string                 { return "choose a player" }
func (*playerChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *playerChoice) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.playerBindings == nil {
		ctx.Game.resolution.playerBindings = make(map[*PlayerBinding]uuid.UUID)
	}
	delete(ctx.Game.resolution.playerBindings, e.binding)
	chooser, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	candidates := e.subjects.resolvePlayers(ctx)
	if len(candidates) == 0 {
		return nil
	}
	selected := candidates[0]
	if len(candidates) > 1 {
		answer := ctx.Game.GetPlayer(chooser).ChooseTargets(slices.Clone(candidates), 1, 1, ctx.Game)
		if len(answer) == 1 && slices.Contains(candidates, answer[0]) {
			selected = answer[0]
		}
	}
	ctx.Game.resolution.playerBindings[e.binding] = selected
	return nil
}

func (e *playerChoice) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-choice", bindingWrite: e.binding, children: []definitionEdge{{"player", e.player}, {"subjects", e.subjects}, {"binding", e.binding}}}
}

func (e *playerChoice) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || playerSetReadsSourcePermanent(e.subjects)
}

type sorceryCastPlayers struct{}

// PlayersWhoCastSorceryThisTurn selects players from the current turn's cast history.
// A countered or unresolved cast remains in that history until cleanup ends.
func PlayersWhoCastSorceryThisTurn() PlayerSet { return sorceryCastPlayers{} }

func (sorceryCastPlayers) playerSet() {}
func (sorceryCastPlayers) resolvePlayers(ctx *EffectContext) []uuid.UUID {
	var selected []uuid.UUID
	for _, id := range EachPlayer.resolvePlayers(ctx) {
		if ctx.Game.GetSorceriesCastThisTurn(id) > 0 {
			selected = append(selected, id)
		}
	}
	return selected
}
func (sorceryCastPlayers) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-set", Text: "players who cast a sorcery this turn"}
}
