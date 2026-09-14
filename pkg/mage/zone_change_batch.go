package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type zoneChangeBatch struct {
	before *Game
	events []GameEvent
}

// withZoneChangeBatch preserves the state before simultaneous zone changes.
func (g *Game) withZoneChangeBatch(apply func()) {
	if g.triggers.zoneBatch != nil {
		apply()
		return
	}
	batch := &zoneChangeBatch{before: g.Clone()}
	g.triggers.zoneBatch = batch
	defer func() { g.triggers.zoneBatch = nil }()
	apply()
	g.triggers.zoneBatch = nil
	g.effects.Apply(g)
	for _, evt := range batch.events {
		g.fireEvent(evt, batch.before)
	}
	g.CheckStateTriggers()
}

func isBattlefieldLeaveEvent(evt *GameEvent) bool {
	return evt.Type == EvtZoneChange && evt.FromZone == ZoneBattlefield
}

// DestroyPermanents destroys the permanents as one simultaneous action.
// Replacement effects can prevent the destruction of individual permanents.
func (g *Game) DestroyPermanents(permanents []*Permanent) {
	if len(permanents) == 0 {
		return
	}
	g.withZoneChangeBatch(func() {
		var destroy []*Permanent
		seen := make(map[uuid.UUID]bool)
		for _, p := range permanents {
			current := g.FindPermanent(p.ID())
			if current == nil || current.incarnationID != p.incarnationID || seen[p.ID()] {
				continue
			}
			seen[p.ID()] = true
			if !current.HasKeyword(Indestructible) && g.effects.ApplyReplacements(NewDestroyPermanentAction(uuid.Nil, p.ID()), g) != nil {
				destroy = append(destroy, current)
			}
		}
		for _, p := range destroy {
			if current := g.FindPermanent(p.ID()); current != nil && current.incarnationID == p.incarnationID {
				g.PutPermanentIntoGraveyard(current)
			}
		}
	})
}
