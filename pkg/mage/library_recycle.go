package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type libraryRecycleEffect struct {
	subjects  PlayerSet
	graveyard bool
	drawMoved bool
}

// ShuffleHandsAndGraveyardsIntoLibraries moves both zones before any shuffle.
func ShuffleHandsAndGraveyardsIntoLibraries(subjects PlayerSet) ResolvingEffect {
	return &libraryRecycleEffect{subjects: subjects, graveyard: true}
}

// ShuffleHandsIntoLibrariesAndDraw keeps each player's count for this resolution.
func ShuffleHandsIntoLibrariesAndDraw(subjects PlayerSet) ResolvingEffect {
	return &libraryRecycleEffect{subjects: subjects, drawMoved: true}
}

func (*libraryRecycleEffect) Text() string {
	return "shuffle selected players' cards into their libraries"
}
func (*libraryRecycleEffect) Properties() EffectProperties { return EffectProperties{Mass: true} }
func (e *libraryRecycleEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}}}
}

func (e *libraryRecycleEffect) Apply(ctx *EffectContext) error {
	ids := e.subjects.resolvePlayers(ctx)
	counts := make(map[uuid.UUID]int, len(ids))
	var events []core.GameEvent
	for _, id := range ids {
		player := ctx.Game.GetPlayer(id)
		if player == nil {
			continue
		}
		for _, card := range slices.Clone(player.Hand()) {
			if moved, ok := player.RemoveFromHand(card.ID()); ok {
				player.AddToLibrary(moved)
				counts[id]++
				events = append(events, core.GameEvent{Type: core.EvtZoneChange, SourceID: card.ID(), PlayerID: id, FromZone: core.ZoneHand, ToZone: core.ZoneLibrary})
			}
		}
		if e.graveyard {
			count := 0
			for _, card := range slices.Clone(player.Graveyard()) {
				if moved, ok := player.RemoveFromGraveyard(card.ID()); ok {
					player.AddToLibrary(moved)
					count++
					events = append(events, core.GameEvent{Type: core.EvtZoneChange, SourceID: card.ID(), PlayerID: id, FromZone: core.ZoneGraveyard, ToZone: core.ZoneLibrary})
				}
			}
			if count > 0 {
				events = append(events, core.GameEvent{Type: core.EvtCardsLeftGraveyard, PlayerID: id, Amount: count})
			}
		}
	}
	// Events must see the complete zone change for all selected players.
	for _, event := range events {
		ctx.Game.FireEvent(event)
	}
	for _, id := range ids {
		if player := ctx.Game.GetPlayer(id); player != nil {
			player.ShuffleLibrary()
		}
	}
	if e.drawMoved {
		for _, id := range ids {
			child := *ctx
			child.Targets = []uuid.UUID{id}
			if err := DrawCards(Fixed(counts[id])).Apply(&child); err != nil {
				return err
			}
		}
	}
	return nil
}
