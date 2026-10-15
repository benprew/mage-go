package mage

import (
	"slices"

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
	cards := make(map[capturedPermanent]capturedCard)
	for _, evt := range batch.events {
		if evt.Type == EvtZoneChange && evt.ToZone == ZoneGraveyard {
			identity := capturedPermanent{evt.SourceID, evt.SourceIncarnation}
			cards[identity] = capturedCard{id: evt.SourceID, visit: g.graveyardVisit(evt.SourceID), zone: ZoneGraveyard}
		}
	}
	for _, evt := range batch.events {
		g.fireEvent(evt, batch.before)
	}
	g.captureBatchAuraCards(batch.events, cards)
	g.CheckStateTriggers()
}

func isBattlefieldLeaveEvent(evt *GameEvent) bool {
	return (evt.Type == EvtZoneChange || evt.Type == EvtSacrifice) && evt.FromZone == ZoneBattlefield
}

// DestroyPermanents destroys the permanents as one simultaneous action.
// Replacement effects can prevent the destruction of individual permanents.
func (g *Game) DestroyPermanents(permanents []*Permanent) {
	g.destroyPermanents(permanents, false)
}

type destructionResult struct {
	creatureDeaths int
	visits         []capturedPermanent
	destroyed      int
}

func (g *Game) destroyPermanents(permanents []*Permanent, cannotRegenerate bool) destructionResult {
	return g.moveGraveyardBatch(permanents, nil, cannotRegenerate)
}

func (g *Game) moveGraveyardBatch(destroyed, direct []*Permanent, cannotRegenerate bool) destructionResult {
	permanents := append(append([]*Permanent(nil), destroyed...), direct...)
	bypass := make(map[uuid.UUID]bool, len(direct))
	for _, p := range direct {
		if p != nil {
			bypass[p.ID()] = true
		}
	}
	if len(permanents) == 0 {
		return destructionResult{}
	}
	var result destructionResult
	g.withZoneChangeBatch(func() {
		type candidate struct {
			permanent *Permanent
			creature  bool
			move      *permanentGraveyardAction
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
			var action Action
			if bypass[p.ID()] {
				action = &permanentGraveyardAction{subject: p.ID(), incarnation: p.incarnationID, destination: ZoneGraveyard}
			} else {
				if current.HasKeyword(Indestructible) {
					continue
				}
				proposal := NewDestroyPermanentAction(uuid.Nil, p.ID())
				proposal.cannotRegenerate = cannotRegenerate
				action = proposal
			}
			switch replaced := g.effects.ApplyReplacements(action, g).(type) {
			case *DestroyPermanentAction:
				destroy = append(destroy, candidate{permanent: current, creature: initialCreature[p.ID()]})
			case *permanentGraveyardAction:
				destroy = append(destroy, candidate{permanent: current, creature: initialCreature[p.ID()], move: replaced})
			}
		}
		for _, item := range destroy {
			if current := g.FindPermanent(item.permanent.ID()); current != nil && current.incarnationID == item.permanent.incarnationID {
				if item.move != nil {
					g.performPermanentGraveyardAction(item.move)
					if item.move.destination == ZoneGraveyard && item.creature {
						result.creatureDeaths++
						result.visits = append(result.visits, capturedPermanent{current.ID(), current.incarnationID})
					}
					continue
				}
				if item.creature {
					result.creatureDeaths++
					result.visits = append(result.visits, capturedPermanent{current.ID(), current.incarnationID})
				}
				g.putPermanentIntoGraveyard(current)
				result.destroyed++
			}
		}
	})
	return result
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

// ExilePermanents moves one group of battlefield objects to exile.
func (g *Game) ExilePermanents(permanents []*Permanent) {
	g.exilePermanents(permanents)
}

func (g *Game) exilePermanents(permanents []*Permanent) []exiledPermanentFact {
	if len(permanents) == 0 {
		return nil
	}
	var result []exiledPermanentFact
	g.withZoneChangeBatch(func() {
		type candidate struct {
			action *permanentGraveyardAction
			fact   exiledPermanentFact
		}
		var selected []candidate
		seen := make(map[uuid.UUID]bool)
		for _, p := range permanents {
			if p == nil || seen[p.ID()] {
				continue
			}
			current := g.FindPermanent(p.ID())
			if current == nil || current.incarnationID != p.incarnationID || current.PhasedOut {
				continue
			}
			seen[p.ID()] = true
			selected = append(selected, candidate{action: &permanentGraveyardAction{subject: current.ID(), incarnation: current.incarnationID, destination: ZoneExile}, fact: exiledPermanentFact{visit: capturedPermanent{current.ID(), current.incarnationID}, controller: current.ControllerID()}})
		}
		order := make(map[uuid.UUID]int)
		for i, id := range EachPlayer.resolvePlayers(&EffectContext{Game: g}) {
			order[id] = i
		}
		slices.SortStableFunc(selected, func(a, b candidate) int { return order[a.fact.controller] - order[b.fact.controller] })
		for i := range selected {
			selected[i].action, _ = g.effects.ApplyReplacements(selected[i].action, g).(*permanentGraveyardAction)
		}
		for _, item := range selected {
			action := item.action
			if action == nil {
				continue
			}
			current := g.FindPermanent(action.subject)
			if current == nil || current.PhasedOut || current.incarnationID != action.incarnation {
				continue
			}
			g.performPermanentGraveyardAction(action)
			if action.destination == ZoneExile && action.subject == item.fact.visit.id && action.incarnation == item.fact.visit.incarnation {
				result = append(result, item.fact)
			}
		}
	})
	return result
}
