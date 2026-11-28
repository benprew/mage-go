package mage

import (
	"fmt"
	"maps"
	"slices"
	"uuid"
)

func (g *Game) announceDividedDamage(effects []Effect, card *Card, controller uuid.UUID, targets []uuid.UUID, x int) (distribution map[uuid.UUID]int, amount int, err error) {
	if len(targets) == 0 {
		return nil, 0, nil
	}
	for _, effect := range effects {
		value := DividedDamageTotal(effect)
		if value == nil {
			continue
		}
		ctx := NewEffectContext(g, card.ID(), controller, targets)
		ctx.SourceCard = card
		ctx.XValue = x
		total := value.Resolve(ctx)
		if total < len(targets) {
			return nil, 0, fmt.Errorf("cannot divide %d damage among %d targets", total, len(targets))
		}
		shares := make(map[uuid.UUID]int, len(targets))
		for _, id := range targets {
			if id == uuid.Nil() || shares[id] != 0 {
				return nil, 0, fmt.Errorf("divided damage requires distinct targets")
			}
			shares[id] = 1
		}

		player := g.GetPlayer(controller)
		if player != nil {
			chosen := player.ChooseDamageDistribution(targets, total, card.Name(), g)
			if validTargetDivision(chosen, targets, total) {
				return maps.Clone(chosen), total, nil
			}
		}
		shares[targets[0]] += total - len(targets)
		return shares, total, nil
	}
	return nil, 0, nil
}

func validTargetDivision(shares map[uuid.UUID]int, targets []uuid.UUID, total int) bool {
	if len(shares) != len(targets) {
		return false
	}
	remaining := total
	for i, id := range targets {
		amount := shares[id]
		if id == uuid.Nil() || slices.Contains(targets[:i], id) || amount <= 0 || amount > remaining {
			return false
		}
		remaining -= amount
	}
	return remaining == 0
}
