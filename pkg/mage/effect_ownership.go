package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type changeOwnerGatheredEffect struct {
	varName  string
	newOwner PlayerSelector
}

// ChangeOwnerGathered changes ownership of the card whose ID is stored in a
// pipeline variable without moving it or changing its controller.
func ChangeOwnerGathered(varName string, newOwner PlayerSelector) Effect {
	return &changeOwnerGatheredEffect{varName: varName, newOwner: newOwner}
}

func (e *changeOwnerGatheredEffect) Text() string { return "change ownership" }
func (e *changeOwnerGatheredEffect) Properties() EffectProperties {
	return EffectProperties{}
}
func (e *changeOwnerGatheredEffect) Apply(ctx *EffectContext) error {
	cardID := ctx.TryGetUUID(e.varName)
	if cardID == uuid.Nil {
		return nil
	}
	if variable, ok := e.newOwner.(*VarPlayerSelector); ok {
		previous := variable.context
		variable.context = ctx
		defer func() { variable.context = previous }()
	}
	owners := e.newOwner.Select(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	if len(owners) != 1 {
		return fmt.Errorf("expected exactly one new owner, got %d", len(owners))
	}
	return ctx.Game.ChangeOwner(cardID, owners[0])
}

type moveExiledGatheredToGraveyardEffect struct {
	varName string
}

// MoveExiledGatheredToGraveyard moves the card whose ID is stored in a pipeline
// variable from exile to its current owner's graveyard.
func MoveExiledGatheredToGraveyard(varName string) Effect {
	return &moveExiledGatheredToGraveyardEffect{varName: varName}
}

func (e *moveExiledGatheredToGraveyardEffect) Text() string {
	return "put exiled card into its owner's graveyard"
}
func (e *moveExiledGatheredToGraveyardEffect) Properties() EffectProperties {
	return EffectProperties{}
}
func (e *moveExiledGatheredToGraveyardEffect) Apply(ctx *EffectContext) error {
	cardID := ctx.TryGetUUID(e.varName)
	if cardID == uuid.Nil {
		return nil
	}
	exiled := ctx.Game.FindExiledCard(cardID)
	if exiled == nil || exiled.Card == nil {
		return nil
	}
	owner := ctx.Game.GetPlayer(exiled.Card.Owner())
	if owner == nil {
		return ErrPlayerNotFound
	}
	card, ok := ctx.Game.RemoveFromExile(cardID)
	if !ok {
		return nil
	}
	owner.AddToGraveyard(card)
	ctx.Game.FireEvent(GameEvent{
		Type:     EvtZoneChange,
		SourceID: cardID,
		PlayerID: owner.PlayerID(),
		FromZone: ZoneExile,
		ToZone:   ZoneGraveyard,
	})
	return nil
}
