package mage

import (
	"errors"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// RevealLibraryTopInto saves the current top's visit and public reveal facts.
// An empty library or absent player clears the result.
func RevealLibraryTopInto(player PlayerRef, result *CardBinding) ResolvingEffect {
	return &libraryTopRevealEffect{player: player, result: result}
}

type libraryTopRevealEffect struct {
	player PlayerRef
	result *CardBinding
}

func (*libraryTopRevealEffect) Text() string                 { return "reveal the selected library's top card" }
func (*libraryTopRevealEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *libraryTopRevealEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", bindingWrite: e.result, children: []definitionEdge{{"player", e.player}, {"result", e.result}}}
}
func (e *libraryTopRevealEffect) Apply(ctx *EffectContext) error {
	delete(ctx.Game.resolution.cardBindings, e.result)
	player := handOperationPlayer(e.player, ctx)
	if player == nil || len(player.Library()) == 0 {
		return nil
	}
	card := player.Library()[0]
	visitor, ok := player.(interface{ libraryVisit(uuid.UUID) uuid.UUID })
	if !ok || visitor.libraryVisit(card.ID()) == uuid.Nil {
		return errors.New("library reveal requires an engine library-visit identity")
	}
	if ctx.Game.resolution.cardBindings == nil {
		ctx.Game.resolution.cardBindings = make(map[*CardBinding]capturedCard)
	}
	ctx.Game.resolution.cardBindings[e.result] = capturedCard{id: card.ID(), visit: visitor.libraryVisit(card.ID()), zone: core.ZoneLibrary, revealed: captureRevealedCardFacts(card)}
	ctx.Game.recordPublicReveal(card)
	return nil
}
func (e *libraryTopRevealEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}

func (p *BasePlayer) libraryTopIsPublic() bool { return p.libraryTopPublic }

func (p *BasePlayer) setLibraryTopPublic(public bool) {
	if p.libraryTopPublic && !public && len(p.library) != 0 {
		p.recordLibraryVisit(p.library[0].ID())
	}
	p.libraryTopPublic = public
}

func (g *Game) syncLibraryTopVisibility() {
	for _, player := range g.players {
		if state, ok := player.(interface{ setLibraryTopPublic(bool) }); ok {
			state.setLibraryTopPublic(g.IsTopCardRevealed(player.PlayerID()))
		}
	}
}

// Callers can change the library slice before they set its new order.
func (p *BasePlayer) libraryOrderSnapshot() []uuid.UUID { return slices.Clone(p.libraryOrder) }
