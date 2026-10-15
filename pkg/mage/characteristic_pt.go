package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type countCharacteristicPT struct {
	effectSource
	incarnation                  uuid.UUID
	filter                       PermanentFilter
	controlled, power, toughness bool
	opponents                    bool
	offset                       int
	entrySacrifice               bool
	entryLife                    bool
}

func newCountCharacteristicPT(filter PermanentFilter, controlled, power, toughness bool) ContinuousEffect {
	return &countCharacteristicPT{filter: filter, controlled: controlled, power: power, toughness: toughness}
}

func (*countCharacteristicPT) GetLayer() core.Layer         { return core.LayerPT }
func (*countCharacteristicPT) GetSublayer() core.SublayerPT { return core.SublayerPT_CDA }
func (*countCharacteristicPT) GetDuration() core.Duration   { return core.WhileOnBattlefield }
func (*countCharacteristicPT) Text() string                 { return "characteristic P/T from a permanent count" }
func (*countCharacteristicPT) Properties() EffectProperties { return EffectProperties{} }
func (e *countCharacteristicPT) IsActive(g *Game) bool {
	p := g.FindPermanent(e.SourceID())
	return p != nil && p.incarnationID == e.incarnation
}
func (e *countCharacteristicPT) cloneEffect() ContinuousEffect { cloned := *e; return &cloned }
func (e *countCharacteristicPT) inspectDefinition() definitionNode {
	n := definitionNode{Role: "characteristic-pt-count", Text: e.Text()}
	if !e.power && !e.toughness {
		n.problem = "select power or toughness for the characteristic count"
	}
	return n
}
func (e *countCharacteristicPT) count(g *Game, controller uuid.UUID, source *Permanent) int {
	if e.entryLife {
		if source == nil {
			return 0
		}
		return source.EntryLifePaid
	}
	if e.entrySacrifice {
		if source == nil {
			return 0
		}
		return source.EntrySacrificeCount
	}
	filter := e.filter
	if e.controlled {
		filter = And(ControlledBy(controller), filter)
	}
	if e.opponents {
		if g.GetPlayer(controller) == nil {
			return e.offset
		}
		opponent := g.GetOpponent(controller)
		if opponent == nil {
			return e.offset
		}
		filter = And(ControlledBy(opponent.PlayerID()), filter)
	}
	return e.offset + g.CountBattlefield(filter)
}
func (e *countCharacteristicPT) ApplyContinuous(ctx *EffectContext) error {
	p := ctx.Game.MutablePermanent(e.SourceID())
	if p == nil || p.incarnationID != e.incarnation {
		return nil
	}
	count := e.count(ctx.Game, p.ControllerID(), p)
	if e.power {
		p.SetCDAPower(count)
	}
	if e.toughness {
		p.SetCDAToughness(count)
	}
	return nil
}

func (g *Game) bindCharacteristicEffect(effect ContinuousEffect, p *Permanent) ContinuousEffect {
	if declaration, ok := effect.(*staticAnimationLayer); ok {
		bound := *declaration
		bound.incarnation = p.incarnationID
		return &bound
	}
	if declaration, ok := effect.(*countCharacteristicPT); ok {
		bound := *declaration
		bound.incarnation = p.incarnationID
		return &bound
	}
	return effect
}

// CardPowerToughness reads current P/T without changing the game.
// Cards without a controller use their owner for characteristic counts.
// This query does not supply information for an object that left a zone.
func (g *Game) CardPowerToughness(card Card) (power, toughness int) {
	if card == nil {
		return 0, 0
	}
	if p := g.FindPermanentIncludingPhased(card.ID()); p != nil {
		return p.CurrentPower(g), p.CurrentToughness(g)
	}
	controller := card.Owner()
	for _, obj := range g.StackObjects() {
		if obj.Card != nil && obj.Card.ID() == card.ID() {
			controller = obj.Controller
			break
		}
	}
	power, toughness = card.Power(), card.Toughness()
	for _, ability := range card.Abilities() {
		holder, ok := ability.(*StaticAbilityHolder)
		if !ok {
			continue
		}
		for _, effect := range holder.Effects {
			declaration, ok := effect.(*countCharacteristicPT)
			if !ok {
				continue
			}
			count := declaration.count(g, controller, nil)
			if declaration.power {
				power = count
			}
			if declaration.toughness {
				toughness = count
			}
		}
	}
	return power, toughness
}
