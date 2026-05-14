package search

import (
	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

func applyMove(g *mage.Game, playerID uuid.UUID, m *Move) {
	switch m.Type {
	case interactive.ActionPlayLand:
		_ = g.PlayLand(playerID, m.CardID)
	case interactive.ActionCastSpell:
		_ = g.CastSpellByID(playerID, m.CardID, m.Targets, m.XValue)
		g.ResolveStack()
	case interactive.ActionActivateAbility:
		_ = g.ActivateAbilityByIndex(playerID, m.PermanentID, m.AbilityIndex, m.Targets)
		g.ResolveStack()
	}
	g.CheckStateBasedActions()
}
