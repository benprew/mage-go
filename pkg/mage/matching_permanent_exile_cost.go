package mage

import (
	"fmt"

	"github.com/google/uuid"
)

type matchingPermanentExileCost struct {
	filter   PermanentFilter
	text     string
	locked   bool
	selected capturedPermanent
	chooser  Player
}

// ExileMatchingPermanentCost exiles one matching permanent the payer controls.
// The source can pay if it matches the filter.
func ExileMatchingPermanentCost(filter PermanentFilter, text string) Cost {
	return &matchingPermanentExileCost{filter: filter, text: text}
}

func (c *matchingPermanentExileCost) Text() string { return c.text }

func (c *matchingPermanentExileCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-exile-cost", Text: c.text}
	if c.filter.IsZero() {
		n.problem = "supply a permanent exile cost filter"
	} else if c.text == "" {
		n.problem = "supply permanent exile cost text"
	}
	return n
}

func (c *matchingPermanentExileCost) eligible(p *Permanent, controller uuid.UUID, g *Game) bool {
	return p != nil && !p.PhasedOut && p.ControllerID() == controller && !c.filter.IsZero() && c.filter.Match(p, g)
}

func (c *matchingPermanentExileCost) CanPay(_, controller uuid.UUID, g *Game) bool {
	if g.GetPlayer(controller) == nil {
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

func (c *matchingPermanentExileCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	if !c.CanPay(sourceID, controller, g) {
		return fmt.Errorf("cannot pay matching permanent exile cost")
	}
	if !c.locked {
		var candidates []*Permanent
		for _, p := range g.zones.battlefield {
			if c.eligible(p, controller, g) {
				candidates = append(candidates, p)
			}
		}
		visits := make([]capturedPermanent, len(candidates))
		for i, p := range candidates {
			visits[i] = capturedPermanent{p.ID(), p.incarnationID}
		}
		player := g.GetPlayer(controller)
		if c.chooser != nil {
			player = c.chooser
		}
		chosen := player.ChoosePermanent(candidates, c.text, g)
		valid := false
		for _, visit := range visits {
			if chosen != nil && chosen.ID() == visit.id && chosen.incarnationID == visit.incarnation {
				c.selected = visit
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid permanent chosen for exile cost")
		}
	}
	locked := *c
	locked.locked = true
	if !locked.CanPay(sourceID, controller, g) {
		return fmt.Errorf("cannot exile the chosen permanent visit")
	}
	g.exilePermanent(g.FindPermanent(c.selected.id), sourceID)
	return nil
}
