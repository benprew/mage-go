package search

import (
	"sort"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
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

func topTargetCombinations(g *mage.Game, playerID uuid.UUID, source mage.Card, reqs []mage.Target, purpose eval.TargetPurpose, damage int, xValues ...int) [][]uuid.UUID {
	x := 0
	if len(xValues) > 0 {
		x = xValues[0]
	}
	return topTargetCombinationsFiltered(g, playerID, source, reqs, purpose, damage, x, nil)
}

func topAbilityTargetCombinations(g *mage.Game, playerID uuid.UUID, source *mage.Permanent, ability mage.ActivatedAbility, purpose eval.TargetPurpose, damage int) [][]uuid.UUID {
	var allow func(uuid.UUID) bool
	if len(ability.Targets()) == 1 {
		allow = func(targetID uuid.UUID) bool {
			return !ai.AbilityActivationIsRedundant(g, playerID, source.ID(), ability.Effects(), []uuid.UUID{targetID})
		}
	}
	combos := topTargetCombinationsFiltered(g, playerID, source.Card, ability.Targets(), purpose, damage, 0, allow)
	result := combos[:0]
	for _, combo := range combos {
		if !ai.AbilityActivationIsRedundant(g, playerID, source.ID(), ability.Effects(), combo) {
			result = append(result, combo)
		}
	}
	return result
}

func topTargetCombinationsFiltered(g *mage.Game, playerID uuid.UUID, source mage.Card, reqs []mage.Target, purpose eval.TargetPurpose, damage, x int, allow func(uuid.UUID) bool) [][]uuid.UUID {
	if len(reqs) == 0 {
		return [][]uuid.UUID{{}}
	}

	groups := make([][][]scoredTarget, 0, len(reqs))
	for _, req := range reqs {
		possible := req.Possible(playerID, source, g)
		minTargets, maxTargets := mage.TargetBounds(req, x)
		if len(possible) < minTargets {
			return nil
		}
		scored := make([]scoredTarget, 0, len(possible))
		for _, id := range possible {
			if allow != nil && !allow(id) {
				continue
			}
			score := eval.TargetValueForPurpose(g, playerID, id, purpose, damage)
			scored = append(scored, scoredTarget{id: id, score: score})
		}
		if len(scored) < minTargets {
			return nil
		}
		sort.SliceStable(scored, func(i, j int) bool {
			return scored[i].score > scored[j].score
		})
		candidateLimit := max(maxTargetCandidatesPerSlot, maxTargets)
		if len(scored) > candidateLimit {
			scored = scored[:candidateLimit]
		}
		maxTargets = min(maxTargets, len(scored))
		var selections [][]scoredTarget
		var choose func(start, remaining int, current []scoredTarget)
		choose = func(start, remaining int, current []scoredTarget) {
			if len(selections) >= maxTargetCombinations {
				return
			}
			if remaining == 0 {
				selections = append(selections, append([]scoredTarget(nil), current...))
				return
			}
			for i := start; i <= len(scored)-remaining; i++ {
				choose(i+1, remaining-1, append(current, scored[i]))
			}
		}
		for count := maxTargets; count >= minTargets; count-- {
			choose(0, count, nil)
		}
		groups = append(groups, selections)
	}

	var combos [][]uuid.UUID
	var build func(int, []uuid.UUID)
	build = func(group int, current []uuid.UUID) {
		if len(combos) >= maxTargetCombinations {
			return
		}
		if group == len(groups) {
			cp := append([]uuid.UUID(nil), current...)
			combos = append(combos, cp)
			return
		}
		for _, selection := range groups[group] {
			next := append([]uuid.UUID(nil), current...)
			for _, candidate := range selection {
				next = append(next, candidate.id)
			}
			build(group+1, next)
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
