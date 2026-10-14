package mage

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// SacrificeAnyMatchingCostInto saves power and new card visits for one payment.
// The payer can choose zero. The source can pay if it matches the filter.
func SacrificeAnyMatchingCostInto(filter PermanentFilter, text string, power *NumberBinding, cards *CardsBinding) Cost {
	return &pluralSacrificeCost{filter: filter, text: text, power: power, cards: cards}
}

// SacrificeSourceAndAnyMatchingCostInto saves separate source and subset results.
// All subjects leave in one batch. The source cannot enter the optional subset.
func SacrificeSourceAndAnyMatchingCostInto(filter PermanentFilter, text string, power *NumberBinding, cards *CardsBinding, source *CardBinding) Cost {
	return &pluralSacrificeCost{filter: filter, text: text, power: power, cards: cards, mandatorySource: true, sourceBinding: source}
}

type pluralSacrificeCost struct {
	filter          PermanentFilter
	text            string
	power           *NumberBinding
	cards           *CardsBinding
	locked          bool
	chooser         Player
	selected        []capturedPermanent
	total           int
	results         []capturedCard
	mandatorySource bool
	sourceBinding   *CardBinding
	sourceVisit     capturedPermanent
	sourceResult    capturedCard
}

func (c *pluralSacrificeCost) Text() string { return c.text }
func (c *pluralSacrificeCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "plural-sacrifice-cost", Text: c.text, bindingWrites: []savedBinding{c.power, c.cards}, children: []definitionEdge{{"power", c.power}, {"cards", c.cards}}}
	if c.filter.IsZero() || c.text == "" || c.power == nil || c.cards == nil {
		n.problem = "supply a filter, text and distinct non-nil result bindings"
	}
	if c.mandatorySource {
		n.battlefieldSource = true
		n.bindingWrites = append(n.bindingWrites, c.sourceBinding)
		n.children = append(n.children, definitionEdge{"source", c.sourceBinding})
		if c.sourceBinding == nil {
			n.problem = "supply a source card result binding"
		}
	}
	return n
}
func (c *pluralSacrificeCost) eligible(p *Permanent, controller uuid.UUID, g *Game) bool {
	return p != nil && !p.PhasedOut && p.ControllerID() == controller && !c.filter.IsZero() && c.filter.Match(p, g)
}
func (c *pluralSacrificeCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	if g.GetPlayer(controller) == nil || c.filter.IsZero() {
		return false
	}
	if c.mandatorySource {
		source := g.FindPermanent(sourceID)
		if source == nil || source.PhasedOut || source.ControllerID() != controller || (c.sourceVisit.incarnation != uuid.Nil && (source.ID() != c.sourceVisit.id || source.incarnationID != c.sourceVisit.incarnation)) {
			return false
		}
	}
	seen := make(map[uuid.UUID]bool, len(c.selected))
	if c.locked {
		for _, visit := range c.selected {
			p := g.FindPermanent(visit.id)
			if (c.mandatorySource && visit.id == sourceID) || seen[visit.id] || !c.eligible(p, controller, g) || p.incarnationID != visit.incarnation {
				return false
			}
			seen[visit.id] = true
		}
	}
	return true
}
func (c *pluralSacrificeCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	c.total, c.results, c.sourceResult = 0, nil, capturedCard{}
	if !c.CanPay(sourceID, controller, g) {
		return fmt.Errorf("cannot pay plural sacrifice cost")
	}
	if !c.locked {
		if c.mandatorySource && c.sourceVisit.incarnation == uuid.Nil {
			source := g.FindPermanent(sourceID)
			c.sourceVisit = capturedPermanent{source.ID(), source.incarnationID}
		}
		player := g.GetPlayer(controller)
		if c.chooser != nil {
			player = c.chooser
		}
		var candidates []*Permanent
		for _, p := range g.zones.battlefield {
			if c.eligible(p, controller, g) && (!c.mandatorySource || p.ID() != sourceID) {
				candidates = append(candidates, p)
			}
		}
		visits := make(map[uuid.UUID]capturedPermanent, len(candidates))
		for _, p := range candidates {
			visits[p.ID()] = capturedPermanent{p.ID(), p.incarnationID}
		}
		count := player.ChooseNumber(0, len(candidates), c.text)
		if count < 0 || count > len(candidates) {
			return fmt.Errorf("invalid sacrifice count")
		}
		c.selected = make([]capturedPermanent, 0, count)
		for range count {
			chosen := player.ChoosePermanent(candidates, c.text, g)
			index := -1
			for i, p := range candidates {
				if chosen != nil && chosen.ID() == p.ID() && chosen.incarnationID == visits[p.ID()].incarnation {
					index = i
					break
				}
			}
			if index < 0 {
				return fmt.Errorf("invalid sacrifice choice")
			}
			p := candidates[index]
			c.selected = append(c.selected, visits[p.ID()])
			candidates = append(candidates[:index:index], candidates[index+1:]...)
		}
	}
	locked := *c
	locked.locked = true
	if !locked.CanPay(sourceID, controller, g) {
		return fmt.Errorf("cannot sacrifice saved visits")
	}
	subjects := make([]*Permanent, len(c.selected))
	for i, visit := range c.selected {
		p := g.FindPermanent(visit.id)
		subjects[i] = p
	}
	if c.mandatorySource {
		subjects = append(subjects, g.FindPermanent(sourceID))
	}
	completed := 0
	g.controllersSacrificePermanentsResult(subjects, func(p *Permanent) bool {
		if c.mandatorySource && p.ID() == sourceID {
			return !p.PhasedOut && p.ControllerID() == controller && p.incarnationID == c.sourceVisit.incarnation
		}
		return c.eligible(p, controller, g)
	}, AnyPermanent, func(result *SacrificePermanentAction, token bool, power int) {
		completed++
		isSource := c.mandatorySource && result.permanentID == sourceID
		if !isSource {
			c.total += power
		}
		if token {
			return
		}
		var saved capturedCard
		if result.destination == core.ZoneGraveyard {
			if visit := g.graveyardVisit(result.permanentID); visit != uuid.Nil {
				saved = capturedCard{id: result.permanentID, visit: visit, zone: core.ZoneGraveyard}
			}
		} else if card := g.FindExiledCard(result.permanentID); card != nil {
			saved = capturedCard{id: result.permanentID, visit: card.Visit, zone: core.ZoneExile}
		}
		if isSource {
			c.sourceResult = saved
		} else if saved.visit != uuid.Nil {
			c.results = append(c.results, saved)
		}
	})
	if completed != len(subjects) {
		c.total, c.results, c.sourceResult = 0, nil, capturedCard{}
		return fmt.Errorf("plural sacrifice payment did not complete")
	}
	return nil
}
