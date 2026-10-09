package mage

import (
	"slices"

	"github.com/google/uuid"
)

// HandLookObserver receives private card copies during resolution.
type HandLookObserver interface{ ObserveHandLook([]Card, GameReader) }

// HandLooks returns independent snapshots of the viewer's past observations.
// This history does not grant access to a current hand.
func (g *Game) HandLooks(viewer uuid.UUID) []PublicCardReveal {
	return clonePublicReveals(g.handLooks[viewer])
}

// LookAtHand shows the selected player's current hand only to the viewer.
func LookAtHand(viewer, player PlayerRef) ResolvingEffect {
	return &handInspectionEffect{viewer: viewer, player: player}
}

type handInspectionEffect struct{ viewer, player PlayerRef }

func (*handInspectionEffect) Text() string                 { return "look at the selected player's hand" }
func (*handInspectionEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *handInspectionEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"viewer", e.viewer}, {"player", e.player}}}
}
func (e *handInspectionEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.viewer) || playerRefReadsSourcePermanent(e.player)
}
func (e *handInspectionEffect) Apply(ctx *EffectContext) error {
	viewer, player := handOperationPlayer(e.viewer, ctx), handOperationPlayer(e.player, ctx)
	if viewer == nil || player == nil || len(player.Hand()) == 0 {
		return nil
	}
	observations := make([]PublicCardReveal, len(player.Hand()))
	for i, card := range player.Hand() {
		copied := card.Copy()
		copied.SetID(card.ID())
		observations[i] = PublicCardReveal{CardID: card.ID(), Owner: card.Owner(), Name: card.Name(), Types: slices.Clone(card.Types()), Card: copied}
	}
	if ctx.Game.handLooks == nil {
		ctx.Game.handLooks = make(map[uuid.UUID][]PublicCardReveal)
	}
	ctx.Game.handLooks[viewer.PlayerID()] = append(ctx.Game.handLooks[viewer.PlayerID()], observations...)
	if observer, ok := viewer.(HandLookObserver); ok {
		copies := clonePublicReveals(observations)
		cards := make([]Card, len(copies))
		for i, observation := range copies {
			cards[i] = observation.Card
		}
		observer.ObserveHandLook(cards, ctx.Game)
	}
	return nil
}
