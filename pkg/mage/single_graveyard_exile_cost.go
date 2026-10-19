package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ExileMatchingCardsFromSingleGraveyardCost requires matching cards from one graveyard.
func ExileMatchingCardsFromSingleGraveyardCost(amount int, filter CardFilter, players PlayerSet) Cost {
	return &exileFromGraveyardCost{amount: amount, filter: filter, players: players, singleMatching: true}
}

func (c *exileFromGraveyardCost) eligibleGraveyards(sourceID, controller uuid.UUID, g *Game) []uuid.UUID {
	if c.amount < 0 || c.filter.IsZero() || nilDefinition(c.players) || g.GetPlayer(controller) == nil {
		return nil
	}
	var ids []uuid.UUID
	for _, id := range c.players.resolvePlayers(&EffectContext{Game: g, Controller: controller, SourceID: sourceID}) {
		player := g.GetPlayer(id)
		if player == nil || slices.Contains(ids, id) {
			continue
		}
		count := 0
		for _, card := range player.Graveyard() {
			if c.filter.Match(card) {
				count++
			}
		}
		if count >= c.amount {
			ids = append(ids, id)
		}
	}
	return ids
}

func (c *exileFromGraveyardCost) canPaySingleGraveyard(sourceID, controller uuid.UUID, g *Game) bool {
	ids := c.eligibleGraveyards(sourceID, controller, g)
	if !c.locked {
		return len(ids) > 0
	}
	if !slices.Contains(ids, c.graveyard) || len(c.selected) != c.amount {
		return false
	}
	player := g.GetPlayer(c.graveyard)
	seen := make(map[uuid.UUID]bool, len(c.selected))
	for _, selected := range c.selected {
		if seen[selected.id] || selected.visit == uuid.Nil || g.graveyardVisit(selected.id) != selected.visit || !slices.ContainsFunc(player.Graveyard(), func(card Card) bool { return card.ID() == selected.id && c.filter.Match(card) }) {
			return false
		}
		seen[selected.id] = true
	}
	return true
}

func (c *exileFromGraveyardCost) chooseSingleGraveyard(sourceID, controller uuid.UUID, g *Game) error {
	return c.chooseSingleGraveyardWithFallback(sourceID, controller, g, false)
}

func (c *exileFromGraveyardCost) chooseSingleGraveyardWithFallback(sourceID, controller uuid.UUID, g *Game, mandatory bool) error {
	ids := c.eligibleGraveyards(sourceID, controller, g)
	if len(ids) == 0 {
		return fmt.Errorf("cannot pay matching single graveyard exile cost")
	}
	c.selected = make([]capturedCard, 0, c.amount)
	c.graveyard = ids[0]
	if c.amount == 0 {
		c.locked = true
		return nil
	}
	var candidates []Card
	visits := make(map[uuid.UUID]uuid.UUID)
	holders := make(map[uuid.UUID]uuid.UUID)
	for _, id := range ids {
		for _, card := range g.GetPlayer(id).Graveyard() {
			if c.filter.Match(card) {
				candidates = append(candidates, card)
				visits[card.ID()] = g.graveyardVisit(card.ID())
				holders[card.ID()] = id
			}
		}
	}
	payer := g.GetPlayer(controller)
	for i := range c.amount {
		chosen := payer.ChooseCardFromGraveyard(candidates, c.Text(), g)
		index := slices.IndexFunc(candidates, func(card Card) bool { return chosen != nil && card.ID() == chosen.ID() })
		if index < 0 {
			if !mandatory || len(candidates) == 0 {
				return fmt.Errorf("invalid matching graveyard exile cost choice")
			}
			index = 0
		}
		chosen = candidates[index]
		if i == 0 {
			c.graveyard = holders[chosen.ID()]
			candidates = slices.DeleteFunc(candidates, func(card Card) bool { return holders[card.ID()] != c.graveyard })
		}
		c.selected = append(c.selected, capturedCard{id: chosen.ID(), visit: visits[chosen.ID()], zone: core.ZoneGraveyard})
		candidates = slices.DeleteFunc(candidates, func(card Card) bool { return card.ID() == chosen.ID() })
	}
	c.locked = true
	if !c.canPaySingleGraveyard(sourceID, controller, g) {
		return fmt.Errorf("chosen single graveyard exile cards changed")
	}
	return nil
}

func (g *Game) exileGraveyardCardsTogether(playerID uuid.UUID, ids []uuid.UUID, sourceID uuid.UUID) {
	player := g.GetPlayer(playerID)
	if player == nil {
		return
	}
	selected := make([]capturedCard, 0, len(ids))
	for _, id := range ids {
		selected = append(selected, capturedCard{id: id, visit: g.graveyardVisit(id), zone: core.ZoneGraveyard})
	}
	g.exileGraveyardVisitsTogether(playerID, selected, sourceID)
}
