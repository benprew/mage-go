package mage

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type sourceCardCost struct {
	incarnation uuid.UUID
	exile       bool
	result      *CardBinding
}

// SacrificeSourceCostWithCard saves the source's new public-zone visit.
// A later zone visit cannot replace this payment result.
func SacrificeSourceCostWithCard(result *CardBinding) Cost {
	return &sourceCardCost{result: result}
}

// ExileSourceCostWithCard saves the source's new exile visit.
// Tokens supply no card result.
func ExileSourceCostWithCard(result *CardBinding) Cost {
	return &sourceCardCost{exile: true, result: result}
}

func (c *sourceCardCost) Text() string {
	if c.exile {
		return "Exile ~"
	}
	return "Sacrifice ~"
}
func (c *sourceCardCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cost", battlefieldSource: true, bindingWrites: []savedBinding{c.result}, children: []definitionEdge{{"result", c.result}}}
	if c.result == nil {
		n.problem = "supply a source card result"
	}
	return n
}
func (c *sourceCardCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	p := g.FindPermanent(sourceID)
	return p != nil && !p.PhasedOut && (c.exile || p.ControllerID() == controller) && (c.incarnation == uuid.Nil || p.incarnationID == c.incarnation)
}
func (c *sourceCardCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	_, err := c.payCard(sourceID, controller, g)
	return err
}
func (c *sourceCardCost) payCard(sourceID, controller uuid.UUID, g *Game) (capturedCard, error) {
	if !c.CanPay(sourceID, controller, g) {
		return capturedCard{}, ErrSourceNotFound
	}
	source := g.FindPermanent(sourceID)
	var result capturedCard
	var err error
	g.withZoneChangeBatch(func() {
		if c.exile {
			g.ExilePermanent(source)
		} else {
			g.CaptureSacrificed(source)
			g.DoSacrifice(source)
		}
		if p := g.FindPermanent(sourceID); p != nil && p.incarnationID == source.incarnationID {
			err = fmt.Errorf("source did not leave while paying %s", c.Text())
			return
		}
		if source.IsToken {
			return
		}
		if exiled := g.FindExiledCard(sourceID); exiled != nil {
			result = capturedCard{id: sourceID, visit: exiled.Visit, zone: core.ZoneExile}
		} else if visit := g.graveyardVisit(sourceID); visit != uuid.Nil {
			result = capturedCard{id: sourceID, visit: visit, zone: core.ZoneGraveyard}
		}
	})
	return result, err
}
