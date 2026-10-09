package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AnteLibraryTops moves eligible library tops before it sends zone events.
// An empty or ineligible library does not stop other selected players.
func (g *Game) AnteLibraryTops(players []uuid.UUID) {
	if !g.anteEnabled || g.anteSettled {
		return
	}
	seen := make(map[uuid.UUID]bool, len(players))
	var events []core.GameEvent
	for _, id := range players {
		if seen[id] {
			continue
		}
		seen[id] = true
		player := g.GetPlayer(id)
		if player == nil || len(player.Library()) == 0 {
			continue
		}
		card := player.Library()[0]
		if card.Owner() != id || card.IsToken() {
			continue
		}
		player.SetLibrary(append([]Card(nil), player.Library()[1:]...))
		player.AddToAnte(card)
		events = append(events, core.GameEvent{Type: core.EvtZoneChange, SourceID: card.ID(), PlayerID: id, FromZone: core.ZoneLibrary, ToZone: core.ZoneAnte})
	}
	for _, event := range events {
		g.FireEvent(event)
	}
}

type anteTopEffect struct{ subjects PlayerSet }

// AnteTopEach antes one current library top for each selected player.
func AnteTopEach(subjects PlayerSet) ResolvingEffect { return &anteTopEffect{subjects: subjects} }

// AnteTopFor antes the selected player's current library top.
func AnteTopFor(player PlayerRef) ResolvingEffect { return AnteTopEach(libraryRecyclePlayer{player}) }

func (*anteTopEffect) Text() string                 { return "ante the top card of the library" }
func (*anteTopEffect) Properties() EffectProperties { return EffectProperties{Mass: true} }
func (e *anteTopEffect) Apply(ctx *EffectContext) error {
	ctx.Game.AnteLibraryTops(e.subjects.resolvePlayers(ctx))
	return nil
}
func (e *anteTopEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"subjects", e.subjects}}}
}
func (e *anteTopEffect) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.subjects)
}
