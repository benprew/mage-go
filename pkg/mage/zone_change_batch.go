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
	// Trigger checks need the completed death count and the former battlefield.
	batch.before.trackers.Turn.creatureDeaths = g.trackers.Turn.creatureDeaths
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
	g.destroyPermanents(permanents, false)
}

func (g *Game) destroyPermanents(permanents []*Permanent, cannotRegenerate bool) int {
	if len(permanents) == 0 {
		return 0
	}
	creatureDeaths := 0
	g.withZoneChangeBatch(func() {
		type candidate struct {
			permanent *Permanent
			creature  bool
		}
		var destroy []candidate
		initialCreature := make(map[uuid.UUID]bool, len(permanents))
		for _, p := range permanents {
			if p != nil {
				if current := g.FindPermanent(p.ID()); current != nil && current.incarnationID == p.incarnationID {
					initialCreature[p.ID()] = current.HasType(TypeCreature)
				}
			}
		}
		seen := make(map[uuid.UUID]bool)
		for _, p := range permanents {
			if p == nil {
				continue
			}
			current := g.FindPermanent(p.ID())
			if current == nil || current.incarnationID != p.incarnationID || current.PhasedOut || seen[p.ID()] {
				continue
			}
			seen[p.ID()] = true
			action := NewDestroyPermanentAction(uuid.Nil, p.ID())
			action.cannotRegenerate = cannotRegenerate
			if !current.HasKeyword(Indestructible) && g.effects.ApplyReplacements(action, g) != nil {
				destroy = append(destroy, candidate{current, initialCreature[p.ID()]})
			}
		}
		for _, item := range destroy {
			if current := g.FindPermanent(item.permanent.ID()); current != nil && current.incarnationID == item.permanent.incarnationID {
				if item.creature {
					creatureDeaths++
				}
				g.PutPermanentIntoGraveyard(current)
			}
		}
	})
	return creatureDeaths
}

// BouncePermanentsToHand returns one group of battlefield objects to their owners' hands.
func (g *Game) BouncePermanentsToHand(permanents []*Permanent) {
	if len(permanents) == 0 {
		return
	}
	g.withZoneChangeBatch(func() {
		seen := make(map[uuid.UUID]bool, len(permanents))
		for _, p := range permanents {
			if p == nil || seen[p.ID()] {
				continue
			}
			current := g.FindPermanent(p.ID())
			if current != nil && current.incarnationID == p.incarnationID && !current.PhasedOut {
				seen[p.ID()] = true
				g.BouncePermanentToHand(current)
			}
		}
	})
}
