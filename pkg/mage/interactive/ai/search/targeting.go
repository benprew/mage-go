package search

import (
	"sort"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/eval"
)

const (
	maxTargetCandidatesPerSlot = 3
	maxTargetCombinations      = 18
)

type scoredTarget struct {
	id    uuid.UUID
	score int
}

func targetPurposeForCard(g *mage.Game, playerID uuid.UUID, card mage.Card, x int) (eval.TargetPurpose, int, mage.Outcome) {
	purpose := eval.TargetGeneric
	outcome := mage.OutcomeUnknown
	damage := 0
	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok || sa.Kind() != mage.ActionSpell {
			continue
		}
		outcome = mage.SpellOutcome(sa.Effects())
		purpose = eval.TargetPurposeForEffects(sa.Effects())
		for _, e := range sa.Effects() {
			if dv := e.Properties().DamageValue; dv != nil {
				damage = dv.Resolve(g, card.ID(), playerID, nil)
				if x > 0 {
					damage = x
				}
			}
		}
		break
	}
	return purpose, damage, outcome
}

func topTargetCombinations(g *mage.Game, playerID uuid.UUID, source mage.Card, reqs []mage.Target, purpose eval.TargetPurpose, damage int) [][]uuid.UUID {
	if len(reqs) == 0 {
		return [][]uuid.UUID{{}}
	}

	slots := make([][]scoredTarget, 0, len(reqs))
	for _, req := range reqs {
		possible := req.Possible(playerID, source, g)
		if len(possible) == 0 {
			return nil
		}
		scored := make([]scoredTarget, 0, len(possible))
		for _, id := range possible {
			score := eval.TargetValueForPurpose(g, playerID, id, purpose, damage)
			scored = append(scored, scoredTarget{id: id, score: score})
		}
		sort.SliceStable(scored, func(i, j int) bool {
			return scored[i].score > scored[j].score
		})
		if len(scored) > maxTargetCandidatesPerSlot {
			scored = scored[:maxTargetCandidatesPerSlot]
		}
		slots = append(slots, scored)
	}

	var combos [][]uuid.UUID
	var build func(int, []uuid.UUID)
	build = func(slot int, current []uuid.UUID) {
		if len(combos) >= maxTargetCombinations {
			return
		}
		if slot == len(slots) {
			cp := append([]uuid.UUID(nil), current...)
			combos = append(combos, cp)
			return
		}
		for _, cand := range slots[slot] {
			build(slot+1, append(current, cand.id))
		}
	}
	build(0, nil)

	sort.SliceStable(combos, func(i, j int) bool {
		return targetComboScore(g, playerID, combos[i], purpose, damage) > targetComboScore(g, playerID, combos[j], purpose, damage)
	})
	if len(combos) > maxTargetCombinations {
		combos = combos[:maxTargetCombinations]
	}
	return combos
}

func targetComboScore(g *mage.Game, playerID uuid.UUID, targets []uuid.UUID, purpose eval.TargetPurpose, damage int) int {
	score := 0
	for _, id := range targets {
		score += eval.TargetValueForPurpose(g, playerID, id, purpose, damage)
	}
	return score
}

func abilityDamageForTargets(g *mage.Game, playerID, sourceID uuid.UUID, effects []mage.Effect, targets []uuid.UUID) int {
	damage := 0
	for _, e := range effects {
		if dv := e.Properties().DamageValue; dv != nil {
			damage = dv.Resolve(g, sourceID, playerID, targets)
		}
	}
	return damage
}
