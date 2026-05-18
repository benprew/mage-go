package search

import (
	"fmt"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

func applyMove(g *mage.Game, playerID uuid.UUID, m *Move) error {
	if m == nil {
		return fmt.Errorf("nil move")
	}
	switch m.Type {
	case interactive.ActionPlayLand:
		if err := g.PlayLand(playerID, m.CardID); err != nil {
			return err
		}
	case interactive.ActionCastSpell:
		if err := g.CastSpellByID(playerID, m.CardID, m.Targets, m.XValue); err != nil {
			return err
		}
		g.ResolveStack()
	case interactive.ActionActivateAbility:
		if err := g.ActivateAbilityByIndex(playerID, m.PermanentID, m.AbilityIndex, m.Targets); err != nil {
			return err
		}
		g.ResolveStack()
	case interactive.ActionPass:
		return nil
	default:
		return fmt.Errorf("unsupported move type %s", m.Type)
	}
	g.CheckStateBasedActions()
	return nil
}
