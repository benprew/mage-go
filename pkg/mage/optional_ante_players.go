package mage

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PlayerSetBinding declares a saved set in the shared resolution frames.
type PlayerSetBinding struct{ seats *PlayerNumbersBinding }

// BindPlayerSet declares a separate membership result for each player seat.
func BindPlayerSet(name string) *PlayerSetBinding { return &PlayerSetBinding{BindPlayerNumbers(name)} }
func (b *PlayerSetBinding) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-set-binding", children: []definitionEdge{{"seats", b.seats}}}
}

// FromPlayerSet reads the saved members in APNAP order.
func FromPlayerSet(binding *PlayerSetBinding) PlayerSet { return savedPlayerSet{binding} }

type savedPlayerSet struct{ binding *PlayerSetBinding }

func (savedPlayerSet) playerSet() {}
func (s savedPlayerSet) resolvePlayers(ctx *EffectContext) []uuid.UUID {
	var ids []uuid.UUID
	for _, id := range EachPlayer.resolvePlayers(ctx) {
		if ctx.Game.resolution.numberBindings[s.binding.seats.member(ctx.Game, id)] != 0 {
			ids = append(ids, id)
		}
	}
	return ids
}
func (s savedPlayerSet) inspectDefinition() definitionNode {
	n := definitionNode{Role: "player-set", children: []definitionEdge{{"binding", s.binding}}}
	if s.binding != nil && s.binding.seats != nil {
		for _, b := range s.binding.seats.members {
			n.children = append(n.children, definitionEdge{"member", FromNumber(b)})
		}
	}
	return n
}

// MayAnteTopEach saves the players who pay after all optional choices.
// The shared ante batch moves all selected cards before zone events.
func MayAnteTopEach(players PlayerSet, paid *PlayerSetBinding) ResolvingEffect {
	return &mayAnteTopEach{players, paid}
}

type mayAnteTopEach struct {
	players PlayerSet
	paid    *PlayerSetBinding
}

func (*mayAnteTopEach) Text() string                 { return "each player may ante the top card" }
func (*mayAnteTopEach) Properties() EffectProperties { return EffectProperties{Mass: true} }
func (e *mayAnteTopEach) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", children: []definitionEdge{{"players", e.players}, {"paid", e.paid}}}
	if e.paid != nil && e.paid.seats != nil {
		for _, b := range e.paid.seats.members {
			n.bindingWrites = append(n.bindingWrites, b)
		}
	}
	return n
}
func (e *mayAnteTopEach) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.players)
}
func (e *mayAnteTopEach) Apply(ctx *EffectContext) error {
	g := ctx.Game
	if g.resolution.numberBindings == nil {
		g.resolution.numberBindings = make(map[*NumberBinding]int)
	}
	for _, b := range e.paid.seats.members {
		g.resolution.numberBindings[b] = 0
	}
	selected := e.players.resolvePlayers(ctx)
	type offer struct {
		id   uuid.UUID
		cost *anteCost
	}
	var accepted []offer
	var previous []string
	for _, id := range EachPlayer.resolvePlayers(ctx) {
		if !slices.Contains(selected, id) {
			continue
		}
		cost := (&anteCost{top: true}).lock(id, g)
		if !cost.CanPay(ctx.SourceID, id, g) {
			continue
		}
		player := g.GetPlayer(id)
		reason := "Ante the top card of your library?"
		if len(previous) > 0 {
			reason += " Previous choices: " + strings.Join(previous, "; ")
		}
		paid := player.ChooseMayAbility(reason)
		previous = append(previous, fmt.Sprintf("%s: %t", player.Name(), paid))
		if paid {
			accepted = append(accepted, offer{id, cost})
		}
	}
	var payers []uuid.UUID
	for _, offer := range accepted {
		if offer.cost.CanPay(ctx.SourceID, offer.id, g) {
			payers = append(payers, offer.id)
			g.resolution.numberBindings[e.paid.seats.member(g, offer.id)] = 1
		}
	}
	g.AnteLibraryTops(payers)
	return nil
}

// SetLifeTotalEach applies the required gain or loss through shared replacements.
// All replacements run before ordinary life changes and their events.
func SetLifeTotalEach(players PlayerSet, total ResolutionValue) ResolvingEffect {
	return &setLifeTotalEach{players, total}
}

type setLifeTotalEach struct {
	players PlayerSet
	total   ResolutionValue
}

func (*setLifeTotalEach) Text() string                 { return "set the selected players' life totals" }
func (*setLifeTotalEach) Properties() EffectProperties { return EffectProperties{Mass: true} }
func (e *setLifeTotalEach) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"players", e.players}, {"total", e.total}}}
}
func (e *setLifeTotalEach) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.players) || resolutionValueReadsSourcePermanent(e.total)
}
func (e *setLifeTotalEach) Apply(ctx *EffectContext) error {
	g := ctx.Game
	total := e.total.resolveValue(ctx)
	selected := e.players.resolvePlayers(ctx)
	type change struct {
		player Player
		gain   bool
		amount int
	}
	var changes []change
	for _, id := range EachPlayer.resolvePlayers(ctx) {
		if !slices.Contains(selected, id) {
			continue
		}
		p := g.GetPlayer(id)
		old := p.Life()
		if total == old {
			continue
		}
		high, low := max(total, old), min(total, old)
		if low < 0 && high > math.MaxInt+low {
			return fmt.Errorf("life total change exceeds integer range")
		}
		changes = append(changes, change{p, total > old, high - low})
	}
	for i, c := range changes {
		if c.gain {
			changes[i].amount = g.prepareLifeGain(c.player, c.amount)
		} else {
			changes[i].amount = g.prepareLifeLoss(c.player, c.amount)
		}
	}
	for _, c := range changes {
		if c.gain {
			c.player.GainLife(c.amount)
		} else {
			c.player.LoseLife(c.amount)
		}
	}
	for _, c := range changes {
		if c.amount > 0 {
			event := core.EvtLifeLost
			if c.gain {
				event = core.EvtLifeGained
			}
			g.FireEvent(core.GameEvent{Type: event, PlayerID: c.player.PlayerID(), Amount: c.amount})
		}
	}
	return nil
}
