package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// PutLibraryTopInHand moves the current top to its owner's hand without a draw.
// An absent player or empty library has no effect.
func PutLibraryTopInHand(player PlayerRef) ResolvingEffect {
	return &libraryTopMoveEffect{player: player, destination: core.ZoneHand}
}

// PutLibraryTopInGraveyard moves the current top without a mill instruction.
// An absent player or empty library has no effect.
func PutLibraryTopInGraveyard(player PlayerRef) ResolvingEffect {
	return &libraryTopMoveEffect{player: player, destination: core.ZoneGraveyard}
}

type libraryTopMoveEffect struct {
	player      PlayerRef
	destination core.Zone
}

func (e *libraryTopMoveEffect) Text() string {
	if e.destination == core.ZoneGraveyard {
		return "put the library top into its owner's graveyard"
	}
	return "put the library top into its owner's hand"
}
func (e *libraryTopMoveEffect) Properties() EffectProperties {
	if e.destination == core.ZoneGraveyard {
		return EffectProperties{Outcome: OutcomeDetriment}
	}
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *libraryTopMoveEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"player", e.player}}}
}
func (e *libraryTopMoveEffect) Apply(ctx *EffectContext) error {
	player := handOperationPlayer(e.player, ctx)
	if player == nil || len(player.Library()) == 0 {
		return nil
	}
	card := player.Library()[0]
	owner := ctx.Game.GetPlayer(card.Owner())
	if owner == nil {
		return ErrPlayerNotFound
	}
	player.SetLibrary(player.Library()[1:])
	if e.destination == core.ZoneHand {
		owner.AddToHand(card)
	} else {
		owner.AddToGraveyard(card)
	}
	ctx.Game.FireEvent(core.GameEvent{Type: core.EvtZoneChange, SourceID: card.ID(), PlayerID: owner.PlayerID(), FromZone: core.ZoneLibrary, ToZone: e.destination})
	return nil
}
func (e *libraryTopMoveEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
