package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// DiscardAction represents a card being discarded because of an effect.
type DiscardAction struct {
	source      uuid.UUID
	playerID    uuid.UUID
	cardID      uuid.UUID
	destination Zone
}

// NewDiscardAction creates an effect-caused discard whose normal destination
// is its owner's graveyard.
func NewDiscardAction(source, playerID, cardID uuid.UUID) *DiscardAction {
	return &DiscardAction{source: source, playerID: playerID, cardID: cardID, destination: ZoneGraveyard}
}

func (a *DiscardAction) ActionSource() uuid.UUID { return a.source }
func (a *DiscardAction) PlayerID() uuid.UUID     { return a.playerID }
func (a *DiscardAction) CardID() uuid.UUID       { return a.cardID }
func (a *DiscardAction) Destination() Zone       { return a.destination }

func (a *DiscardAction) withDestination(destination Zone) *DiscardAction {
	cp := *a
	cp.destination = destination
	return &cp
}

type discardToLibraryReplacement struct {
	replacementBase
	playerID uuid.UUID
}

func (r *discardToLibraryReplacement) Matches(action Action, _ GameReader) bool {
	discard, ok := action.(*DiscardAction)
	return ok && discard.PlayerID() == r.playerID && discard.Destination() == ZoneGraveyard
}

func (r *discardToLibraryReplacement) Replace(action Action, g *Game) Action {
	discard := action.(*DiscardAction)
	player := g.GetPlayer(r.playerID)
	if player == nil || !player.ChooseMayAbility("put discarded card on top of your library instead") {
		return action
	}
	return discard.withDestination(ZoneLibrary)
}

func (r *discardToLibraryReplacement) IsActive(g GameReader) bool {
	return g.FindPermanent(r.sourceID) != nil
}

func (r *discardToLibraryReplacement) Clone() ReplacementEffect {
	return &discardToLibraryReplacement{
		replacementBase: replacementBase{sourceID: r.sourceID, duration: r.duration},
		playerID:        r.playerID,
	}
}

// DiscardToLibraryReplacement creates a continuous optional replacement for
// effect-caused discards by the source's controller.
func DiscardToLibraryReplacement() ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		source := g.FindPermanent(sourceID)
		if source == nil {
			return nil
		}
		g.effects.AddCycleReplacement(&discardToLibraryReplacement{
			replacementBase: replacementBase{sourceID: sourceID, duration: WhileOnBattlefield},
			playerID:        source.ControllerID(),
		})
		return nil
	})
}
