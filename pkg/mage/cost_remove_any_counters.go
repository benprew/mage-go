package mage

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type removeAnyCountersCost struct {
	ct          core.CounterType
	paid        *NumberBinding
	count       int
	incarnation uuid.UUID
	locked      bool
}

// RemoveAnyCountersCost lets the payer choose the number of source counters to remove.
func RemoveAnyCountersCost(counter core.CounterType, paid *NumberBinding) Cost {
	return &removeAnyCountersCost{ct: counter, paid: paid}
}

func (c *removeAnyCountersCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cost", bindingWrite: c.paid, battlefieldSource: true, children: []definitionEdge{{"paid", c.paid}}}
	if c.paid == nil {
		n.problem = "supply a non-nil paid counter binding"
	}
	if c.ct < 0 || c.ct >= core.NumCounters {
		n.problem = "invalid counter type"
	}
	return n
}

func (c *removeAnyCountersCost) Text() string {
	return fmt.Sprintf("Remove any number of %s counters", c.ct)
}

func (c *removeAnyCountersCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	if c.ct < 0 || c.ct >= core.NumCounters {
		return false
	}
	p := g.FindPermanent(sourceID)
	return p != nil && (!c.locked || p.incarnationID == c.incarnation) && c.count >= 0 && p.Counters[c.ct] >= c.count
}

func (c *removeAnyCountersCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	if !c.CanPay(sourceID, controller, g) {
		return fmt.Errorf("cannot pay chosen counter cost")
	}
	if c.count == 0 {
		return nil
	}
	if !g.removeCountersFrom(g.FindPermanent(sourceID), c.ct, c.count, g.counterPlacementSource(sourceID)) {
		return fmt.Errorf("cannot remove chosen counters")
	}
	return nil
}
