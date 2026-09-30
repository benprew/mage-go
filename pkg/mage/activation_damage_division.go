package mage

import (
	"fmt"

	"github.com/google/uuid"
)

func (g *Game) prepareActivationDamageDivision(controller uuid.UUID, card Card, effects []Effect, targets []uuid.UUID) (map[uuid.UUID]int, error) {
	for _, effect := range effects {
		if !IsDividedDamageEffect(effect) {
			continue
		}
		total := max(0, DividedDamageTotal(effect).Resolve(g, card.ID(), controller, targets))
		if len(targets) == 0 {
			return nil, nil
		}
		// Each target needs a positive share before the player can pay costs.
		if sanitizeDamageDistribution(nil, targets, total) == nil {
			return nil, fmt.Errorf("cannot divide %d damage among %d ability targets", total, len(targets))
		}
		player := g.GetPlayer(controller)
		if player == nil {
			return nil, ErrPlayerNotFound
		}
		response := player.ChooseDamageDistribution(targets, total, card.Name(), g)
		return sanitizeDamageDistribution(response, targets, total), nil
	}
	return nil, nil
}
