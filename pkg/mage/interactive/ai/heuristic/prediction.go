package heuristic

import (
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

// PredictOption simulates an option with heuristic targets and X values.
// The stack resolves without responses. The live game does not change.
func (s *Strategy) PredictOption(g *mage.Game, playerID uuid.UUID, option interactive.ActionOption) (*mage.Game, bool) {
	branch := g.Clone()
	p := branch.GetPlayer(playerID)
	var err error
	switch option.Type {
	case interactive.ActionPass:
	case interactive.ActionPlayLand:
		err = branch.PlayLand(playerID, option.CardID)
	case interactive.ActionCastSpell:
		var card *mage.Card
		for _, c := range p.Hand() {
			if c.ID() == option.CardID {
				card = c
				break
			}
		}
		if card == nil {
			return nil, false
		}
		targets := s.autoSelectTargets(p, branch, card)
		x := s.bestXValue(branch, playerID, card, targets)
		err = branch.CastSpellByID(playerID, card.ID(), targets, x)
	case interactive.ActionActivateAbility:
		perm := branch.FindPermanent(option.PermanentID)
		if perm == nil || option.AbilityIndex < 0 || option.AbilityIndex >= len(perm.RuntimeAbilities) {
			return nil, false
		}
		ability, ok := mage.UnwrapAbility(perm.RuntimeAbilities[option.AbilityIndex]).(mage.ActivatedAbility)
		if !ok {
			return nil, false
		}
		targets, ok := s.autoSelectAbilityTargets(p, branch, perm, ability)
		if !ok {
			return nil, false
		}
		err = branch.ActivateAbilityByIndex(playerID, perm.ID(), option.AbilityIndex, targets)
	default:
		return nil, false
	}
	if err != nil {
		return nil, false
	}
	branch.ResolveStack()
	branch.CheckStateBasedActions()
	return branch, true
}
