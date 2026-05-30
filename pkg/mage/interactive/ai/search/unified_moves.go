package search

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

func legalPriorityMoves(g *mage.Game, p mage.Player, isRoot bool) []Move {
	playerID := p.PlayerID()
	var moves []Move

	if isRoot {
		for _, land := range g.GetPlayableLands(playerID) {
			moves = append(moves, Move{
				Type:     interactive.ActionPlayLand,
				CardID:   land.ID(),
				CardName: land.Name(),
			})
		}
		for _, card := range g.GetCastableSpells(playerID) {
			moves = append(moves, expandSpell(g, p, card)...)
		}
	}

	for _, info := range g.GetActivatableAbilities(playerID) {
		moves = append(moves, expandAbility(g, p, info)...)
	}

	moves = append(moves, Move{Type: interactive.ActionPass})
	return moves
}

func expandSpell(g *mage.Game, p mage.Player, card mage.Card) []Move {
	playerID := p.PlayerID()
	mc := card.ManaCost()

	xValues := []int{0}
	if mc.HasX {
		availMana := eval.CountAvailableMana(g, playerID)
		fixedCost := mc.CMC()
		maxX := availMana - fixedCost
		if maxX < 1 {
			return nil
		}
		const maxXVariants = 8
		if maxX > maxXVariants {
			maxX = maxXVariants
		}
		xValues = xValues[:0]
		for x := 1; x <= maxX; x++ {
			xValues = append(xValues, x)
		}
	}

	modeIndices := []int{0}
	if modes := card.Modes(); len(modes) > 1 {
		modeIndices = nil
		for i := range modes {
			modeIndices = append(modeIndices, i)
		}
	}

	var out []Move
	for _, mode := range modeIndices {
		for _, x := range xValues {
			out = append(out, expandTargets(g, playerID, card, mode, x)...)
		}
	}
	return out
}

func expandTargets(g *mage.Game, playerID uuid.UUID, card mage.Card, mode, x int) []Move {
	targets := card.CastTargets()
	if len(targets) == 0 {
		return []Move{{
			Type:      interactive.ActionCastSpell,
			CardID:    card.ID(),
			CardName:  card.Name(),
			XValue:    x,
			ModeIndex: mode,
		}}
	}
	purpose, damage, _ := targetPurposeForCard(g, playerID, card, x)
	combos := topTargetCombinations(g, playerID, card, targets, purpose, damage)
	if len(combos) == 0 {
		return nil
	}
	out := make([]Move, 0, len(combos))
	for _, combo := range combos {
		out = append(out, Move{
			Type:      interactive.ActionCastSpell,
			CardID:    card.ID(),
			CardName:  card.Name(),
			Targets:   combo,
			XValue:    x,
			ModeIndex: mode,
			heuristic: targetComboScore(g, playerID, combo, purpose, damage),
		})
	}
	return out
}

func expandAbility(g *mage.Game, p mage.Player, info mage.ActivatableInfo) []Move {
	playerID := p.PlayerID()
	perm := g.FindPermanent(info.PermanentID)
	if perm == nil || info.AbilityIndex >= len(perm.RuntimeAbilities) {
		return nil
	}
	aa, ok := mage.UnwrapAbility(perm.RuntimeAbilities[info.AbilityIndex]).(mage.ActivatedAbility)
	if !ok {
		return nil
	}
	targets := aa.Targets()
	if len(targets) == 0 {
		return []Move{{
			Type:         interactive.ActionActivateAbility,
			PermanentID:  info.PermanentID,
			AbilityIndex: info.AbilityIndex,
			CardName:     info.PermanentName,
		}}
	}
	purpose := eval.TargetPurposeForEffects(aa.Effects())
	damage := abilityDamageForTargets(g, playerID, perm.ID(), aa.Effects(), nil)
	combos := topTargetCombinations(g, playerID, perm.Card, targets, purpose, damage)
	if len(combos) == 0 {
		return nil
	}
	out := make([]Move, 0, len(combos))
	for _, combo := range combos {
		out = append(out, Move{
			Type:         interactive.ActionActivateAbility,
			PermanentID:  info.PermanentID,
			AbilityIndex: info.AbilityIndex,
			CardName:     info.PermanentName,
			Targets:      combo,
			heuristic:    targetComboScore(g, playerID, combo, purpose, damage),
		})
	}
	return out
}
