package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PutChosenHandCardOnLibrary requires one current hand card at resolution.
// Saved visits prevent a returned card from replacing an offered object.
func PutChosenHandCardOnLibrary(player PlayerRef) ResolvingEffect {
	return &handLibraryChoice{player: player}
}

type handLibraryChoice struct{ player PlayerRef }

func (*handLibraryChoice) Text() string { return "put a card from hand on top of its owner's library" }
func (*handLibraryChoice) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *handLibraryChoice) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
}
func (e *handLibraryChoice) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
func (e *handLibraryChoice) Apply(ctx *EffectContext) error {
	player := handOperationPlayer(e.player, ctx)
	if player == nil || len(player.Hand()) == 0 {
		return nil
	}
	candidates := slices.Clone(player.Hand())
	visits := make(map[uuid.UUID]uuid.UUID, len(candidates))
	for _, card := range candidates {
		visits[card.ID()] = ctx.Game.handVisit(card.ID())
	}
	chosen := player.ChooseCardFromHand(candidates, e.Text(), ctx.Game)
	current := func(id uuid.UUID) Card {
		visit := visits[id]
		if visit == uuid.Nil || ctx.Game.handVisit(id) != visit {
			return nil
		}
		for _, card := range player.Hand() {
			if card.ID() == id {
				return card
			}
		}
		return nil
	}
	var card Card
	if chosen != nil {
		card = current(chosen.ID())
	}
	if card == nil {
		for _, candidate := range candidates {
			if card = current(candidate.ID()); card != nil {
				break
			}
		}
	}
	if card == nil {
		return nil
	}
	owner := ctx.Game.GetPlayer(card.Owner())
	if owner == nil {
		return ErrPlayerNotFound
	}
	if _, ok := player.RemoveFromHand(card.ID()); !ok {
		return nil
	}
	owner.SetLibrary(append([]Card{card}, owner.Library()...))
	ctx.Game.FireEvent(core.GameEvent{Type: core.EvtZoneChange, SourceID: card.ID(), PlayerID: owner.PlayerID(), FromZone: core.ZoneHand, ToZone: core.ZoneLibrary})
	return nil
}
