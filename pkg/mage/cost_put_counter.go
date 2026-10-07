package mage

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type putCounterOnMatchingCost struct {
	counter  core.CounterType
	filter   PermanentFilter
	selected capturedPermanent
	locked   bool
}

// PutCounterOnMatchingCost puts one counter on a matching permanent the payer controls.
// Replacement effects can change the placement without changing the payment.
func PutCounterOnMatchingCost(counter core.CounterType, filter PermanentFilter) Cost {
	return &putCounterOnMatchingCost{counter: counter, filter: filter}
}
func (c *putCounterOnMatchingCost) Text() string {
	return "Put a counter on a matching permanent you control"
}
func (c *putCounterOnMatchingCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cost", Text: c.Text()}
	if c.counter < 0 || c.counter >= core.NumCounters {
		n.problem = "invalid counter type"
	}
	return n
}
func (c *putCounterOnMatchingCost) eligible(p *Permanent, controller uuid.UUID, g *Game) bool {
	return p != nil && !p.PhasedOut && p.ControllerID() == controller && (c.filter.IsZero() || c.filter.Match(p, g))
}
func (c *putCounterOnMatchingCost) CanPay(_, controller uuid.UUID, g *Game) bool {
	if c.counter < 0 || c.counter >= core.NumCounters || g.GetPlayer(controller) == nil {
		return false
	}
	if c.locked {
		p := g.FindPermanent(c.selected.id)
		return c.eligible(p, controller, g) && p.incarnationID == c.selected.incarnation
	}
	for _, p := range g.zones.battlefield {
		if c.eligible(p, controller, g) {
			return true
		}
	}
	return false
}
func (c *putCounterOnMatchingCost) choose(controller uuid.UUID, g *Game, chooser Player) error {
	var candidates []*Permanent
	for _, p := range g.zones.battlefield {
		if c.eligible(p, controller, g) {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 || chooser == nil {
		return fmt.Errorf("no permanent for counter payment")
	}
	chosen := chooser.ChoosePermanent(candidates, c.Text(), g)
	for _, p := range candidates {
		if chosen != nil && chosen.ID() == p.ID() && chosen.incarnationID == p.incarnationID {
			c.selected = capturedPermanent{id: p.ID(), incarnation: p.incarnationID}
			c.locked = true
			if c.CanPay(uuid.Nil, controller, g) {
				return nil
			}
		}
	}
	return fmt.Errorf("invalid permanent for counter payment")
}
func (c *putCounterOnMatchingCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	locked := *c
	if !locked.locked {
		if err := locked.choose(controller, g, g.GetPlayer(controller)); err != nil {
			return err
		}
	}
	if !locked.CanPay(sourceID, controller, g) {
		return fmt.Errorf("cannot pay chosen counter placement")
	}
	g.AddCountersWithReplacement(g.FindPermanent(locked.selected.id), locked.counter, 1, sourceID, false)
	return nil
}
