package mage

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type cardOwnershipEffect struct {
	card  CardRef
	owner PlayerRef
}

// AcquireCardOwnership changes the owner of a present card without moving it.
func AcquireCardOwnership(card CardRef, owner PlayerRef) ResolvingEffect {
	return &cardOwnershipEffect{card, owner}
}
func (*cardOwnershipEffect) Text() string { return "acquire ownership of the card" }
func (*cardOwnershipEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *cardOwnershipEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"card", e.card}, {"owner", e.owner}}}
}
func (e *cardOwnershipEffect) Apply(ctx *EffectContext) error {
	if !ctx.Game.anteEnabled || ctx.Game.anteSettled {
		return nil
	}
	card, ok := e.card.resolveCard(ctx)
	if !ok || card.IsToken() {
		return nil
	}
	owner, ok := e.owner.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(owner) == nil {
		return nil
	}
	return ctx.Game.ChangeOwner(card.ID(), owner)
}
func (e *cardOwnershipEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.owner)
}

type anteExchangeEffect struct {
	card   CardRef
	player PlayerRef
}

// ExchangeAnteCardWithLibraryTop completes both moves before it sends events.
// An empty library receives the ante card under the cross-zone exchange rule.
func ExchangeAnteCardWithLibraryTop(card CardRef, player PlayerRef) ResolvingEffect {
	return &anteExchangeEffect{card, player}
}
func (*anteExchangeEffect) Text() string                 { return "exchange the ante card with the library top" }
func (*anteExchangeEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *anteExchangeEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"card", e.card}, {"player", e.player}}}
}
func (e *anteExchangeEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
func (e *anteExchangeEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	card, ok := e.card.resolveCard(ctx)
	if !ok || card.IsToken() {
		return nil
	}
	playerID, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := g.GetPlayer(playerID)
	if player == nil || card.Owner() != playerID {
		return nil
	}
	subject := capturedCard{id: card.ID(), visit: g.anteVisit(card.ID()), zone: core.ZoneAnte}
	if _, ok := g.resolveAnteCard(subject.id, subject.visit); !ok {
		return nil
	}
	library := slices.Clone(player.Library())
	var top Card
	if len(library) > 0 {
		top = library[0]
		if top.Owner() != playerID || top.IsToken() {
			return nil
		}
		library = library[1:]
	}
	removed, ok := g.removeAnteVisit(subject)
	if !ok {
		return nil
	}
	player.SetLibrary(append([]Card{removed}, library...))
	if top != nil {
		player.AddToAnte(top)
	}
	g.FireEvent(core.GameEvent{Type: core.EvtZoneChange, SourceID: card.ID(), PlayerID: playerID, FromZone: core.ZoneAnte, ToZone: core.ZoneLibrary})
	if top != nil {
		g.FireEvent(core.GameEvent{Type: core.EvtZoneChange, SourceID: top.ID(), PlayerID: playerID, FromZone: core.ZoneLibrary, ToZone: core.ZoneAnte})
	}
	return nil
}
