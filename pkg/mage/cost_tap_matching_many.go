package mage

import (
	"fmt"

	"github.com/google/uuid"
)

type tapNOtherMatchingCost struct {
	count    int
	filter   PermanentFilter
	text     string
	locked   bool
	selected []capturedPermanent
	chooser  Player
}

// TapNOtherMatchingCost taps exactly count other untapped matching permanents.
func TapNOtherMatchingCost(count int, filter PermanentFilter, text string) Cost {
	return &tapNOtherMatchingCost{count: count, filter: filter, text: text}
}

func (c *tapNOtherMatchingCost) Text() string { return c.text }

func (c *tapNOtherMatchingCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "tap-cost", Text: c.text}
	if c.count <= 0 {
		n.problem = "tap count must be positive"
	} else if c.text == "" {
		n.problem = "supply tap cost text"
	}
	return n
}

func (c *tapNOtherMatchingCost) eligible(p *Permanent, sourceID, controller uuid.UUID, g *Game) bool {
	return p != nil && !p.PhasedOut && !p.Tapped && p.ID() != sourceID && p.ControllerID() == controller && c.filter.Match(p, g)
}

func (c *tapNOtherMatchingCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	if c.count <= 0 {
		return false
	}
	if c.locked {
		if len(c.selected) != c.count {
			return false
		}
		seen := make(map[uuid.UUID]bool, c.count)
		for _, choice := range c.selected {
			if seen[choice.id] {
				return false
			}
			seen[choice.id] = true
			p := g.FindPermanent(choice.id)
			if !c.eligible(p, sourceID, controller, g) || p.incarnationID != choice.incarnation {
				return false
			}
		}
		return true
	}
	count := 0
	for _, p := range g.zones.battlefield {
		if c.eligible(p, sourceID, controller, g) {
			count++
		}
	}
	return count >= c.count
}

func (c *tapNOtherMatchingCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	if !c.CanPay(sourceID, controller, g) {
		return fmt.Errorf("not enough matching permanents to tap")
	}
	if c.locked {
		permanents := make([]*Permanent, 0, len(c.selected))
		for _, choice := range c.selected {
			permanents = append(permanents, g.FindPermanent(choice.id))
		}
		g.tapPermanentsTogether(permanents)
		return nil
	}
	player := g.GetPlayer(controller)
	if c.chooser != nil {
		player = c.chooser
	}
	chosenPermanents := make([]*Permanent, 0, c.count)
	selected := make(map[uuid.UUID]bool, c.count)
	for range c.count {
		var candidates []*Permanent
		for _, p := range g.zones.battlefield {
			if c.eligible(p, sourceID, controller, g) && !selected[p.ID()] {
				candidates = append(candidates, p)
			}
		}
		chosen := player.ChoosePermanent(candidates, c.text, g)
		var valid *Permanent
		for _, candidate := range candidates {
			if chosen != nil && chosen.ID() == candidate.ID() && chosen.incarnationID == candidate.incarnationID {
				valid = candidate
				break
			}
		}
		if valid == nil {
			return fmt.Errorf("invalid permanent chosen for tap cost")
		}
		selected[valid.ID()] = true
		chosenPermanents = append(chosenPermanents, valid)
		if c.selected != nil {
			c.selected = append(c.selected, capturedPermanent{id: valid.ID(), incarnation: valid.incarnationID})
		}
	}
	g.tapPermanentsTogether(chosenPermanents)
	return nil
}
