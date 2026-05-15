package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// ManaSolverInputs is the input bundle for SolveMana. The solver is pure —
// it doesn't read or mutate Game state, so all data it needs must be supplied
// here. Fields:
//   - Pool: floating mana available; the solver subtracts what the pool can
//     cover before tapping sources.
//   - Cost: the mana cost being paid.
//   - Sources: candidate untapped mana sources, already filtered for
//     ineligible ones (e.g., the activation source when {T} is in the cost).
//     The solver may use any of them.
//   - Scores: preservation score per source (same length as Sources); higher
//     = prefer to keep untapped. The solver picks lowest-score eligible
//     source at each step. Score is computed by the caller via
//     preservationScore so the solver stays caller-agnostic.
//   - BonusFor: returns the additional mana (from Mana Flare-style abilities
//     elsewhere on the battlefield) produced when the given permanent taps.
type ManaSolverInputs struct {
	Pool     *ManaPool
	Cost     ManaCost
	Sources  []manaSourceInfo
	Scores   []int
	BonusFor func(uuid.UUID) int
}

// ManaSolution describes the result of solving a mana payment: an ordered
// list of permanent IDs the caller should tap (via TapForMana) to produce the
// mana for the cost. The solver doesn't mutate state, so the caller is
// responsible for applying the solution.
type ManaSolution struct {
	SourcesToTap []uuid.UUID
}

// efficiencyBonusPerSavedTap is the score reduction applied per "saved tap"
// when the solver is forced to pick a multi-mana source for efficiency in the
// generic pass. Argentum uses 25; the magnitude here just needs to dominate
// the per-color and utility penalties so a multi-mana source actually wins
// when we need it.
const efficiencyBonusPerSavedTap = 25

// SolveMana decides which mana sources to tap to pay the given cost given the
// player's floating mana pool. Returns the tap order in a ManaSolution, or an
// error if the cost is unpayable from the supplied inputs.
//
// Algorithm (matches ManaSolver in argentum-engine, adapted for our types):
//  1. Subtract floating mana from the cost (colored slots first, then hybrids,
//     then generic).
//  2. For each remaining colored slot, pick the lowest-score eligible source
//     that can produce that color.
//  3. For each remaining generic mana, pick the lowest-score remaining source.
//     If single-mana sources alone can't cover the remainder, bias toward
//     multi-mana sources (efficiencyBonusPerSavedTap * (Amount - 1)) so the
//     solver doesn't tap N basics when one Sol Ring would do.
//
// Bonus mana from ManaBonusAbility on other battlefield permanents (Mana Flare,
// Gauntlet of Might) and the extra mana from multi-mana sources tapped in the
// colored pass flow into a shared surplus counter that reduces generic need.
func SolveMana(in ManaSolverInputs) (*ManaSolution, error) {
	mc := in.Cost
	sources := append([]manaSourceInfo(nil), in.Sources...)
	scores := in.Scores

	pool := in.Pool

	// Subtract floating mana for colored requirements.
	colorNeeds := [...]struct {
		color Color
		need  int
	}{
		{White, mc.White},
		{Blue, mc.Blue},
		{Black, mc.Black},
		{Red, mc.Red},
		{Green, mc.Green},
	}
	needed := map[Color]int{}
	poolUsedForColor := 0
	poolByColor := map[Color]int{}
	for _, cn := range colorNeeds {
		poolByColor[cn.color] = pool.Count(cn.color)
	}
	for _, cn := range colorNeeds {
		have := poolByColor[cn.color]
		used := min(have, cn.need)
		poolByColor[cn.color] -= used
		poolUsedForColor += used
		needed[cn.color] = cn.need - used
	}

	// Hybrid resolution: try to satisfy from pool, otherwise commit a
	// concrete color requirement that the colored pass will satisfy.
	// XXX: undercounts dual lands — sourceBucketColor uses only the first
	// declared color, so a Tundra contributes to {W/x} availability but not
	// {U/x}. Driving hybrid choice off Colors[] would be correct but the
	// pre-multi-color heuristic is preserved here to avoid behavior drift in
	// CanAfford / GetCastableSpells paths.
	availableForHybridFromSources := map[Color]int{}
	for _, src := range sources {
		availableForHybridFromSources[sourceBucketColor(src)] += src.Amount
	}
	for _, h := range mc.Hybrid {
		pa, pb := poolByColor[h.A], poolByColor[h.B]
		if pa >= pb && pa > 0 {
			poolByColor[h.A] = pa - 1
			poolUsedForColor++
			continue
		}
		if pb > 0 {
			poolByColor[h.B] = pb - 1
			poolUsedForColor++
			continue
		}
		sa, sb := availableForHybridFromSources[h.A], availableForHybridFromSources[h.B]
		if sa >= sb && sa > 0 {
			availableForHybridFromSources[h.A] = sa - 1
			needed[h.A]++
		} else if sb > 0 {
			availableForHybridFromSources[h.B] = sb - 1
			needed[h.B]++
		} else {
			return nil, fmt.Errorf("insufficient mana for hybrid %s", h)
		}
	}

	poolRemaining := pool.TotalMana() - poolUsedForColor
	genericNeeded := mc.Generic - min(mc.Generic, poolRemaining)

	bonusFor := in.BonusFor
	if bonusFor == nil {
		bonusFor = func(uuid.UUID) int { return 0 }
	}

	var toTap []uuid.UUID
	surplusMana := 0
	// coloredSurplus tracks colored mana left over from dual-emit sources
	// after their "trigger" color slot is satisfied. A future slot of the
	// same color drains the surplus before reaching for another source.
	coloredSurplus := map[Color]int{}

	// Colored pass: per slot, pick lowest-score source that can produce the
	// required color. Multi-mana sources contribute their excess (Amount - 1)
	// plus any bonus mana to surplusMana so the generic pass benefits.
	// Iterate colorNeeds (slice) rather than `needed` (map) so the tap order
	// in toTap is deterministic across runs.
	for _, cn := range colorNeeds {
		for needed[cn.color] > 0 {
			if coloredSurplus[cn.color] > 0 {
				coloredSurplus[cn.color]--
				needed[cn.color]--
				continue
			}
			idx := -1
			for j, src := range sources {
				if src.PermanentID == uuid.Nil || !sourceProduces(src, cn.color) {
					continue
				}
				if idx == -1 || scores[j] < scores[idx] {
					idx = j
				}
			}
			if idx == -1 {
				return nil, fmt.Errorf("cannot pay %s for cost %s: no untapped %s source available", cn.color, mc, cn.color)
			}
			src := sources[idx]
			sources[idx].PermanentID = uuid.Nil
			toTap = append(toTap, src.PermanentID)
			needed[cn.color]--
			bonus := bonusFor(src.PermanentID)
			if len(src.PerTapOutput) > 0 {
				// Dual-emit: route every produced color to its surplus
				// bucket (minus the one unit we just consumed for cn.color).
				total := 0
				for c, amt := range src.PerTapOutput {
					total += amt
					if c == cn.color {
						amt--
					}
					if c == Colorless {
						surplusMana += amt
					} else {
						coloredSurplus[c] += amt
					}
				}
				surplusMana += bonus
				// Any leftover Amount beyond the per-tap colored output (rare;
				// indicates Amount > sum of PerTapOutput) falls into generic.
				if src.Amount > total {
					surplusMana += src.Amount - total
				}
			} else {
				surplusMana += max(src.Amount-1, 0) + bonus
			}
		}
	}

	totalColoredSurplus := 0
	for _, amt := range coloredSurplus {
		totalColoredSurplus += amt
	}
	genericNeeded -= min(genericNeeded, surplusMana+totalColoredSurplus)

	// Generic pass: lowest-score remaining source. When the single-mana
	// supply can't cover the remainder, give a per-saved-tap bonus to
	// multi-mana sources so they win the tie against basics that would
	// require many taps.
	for genericNeeded > 0 {
		singleManaCount := 0
		for _, src := range sources {
			if src.PermanentID == uuid.Nil {
				continue
			}
			if src.Amount+bonusFor(src.PermanentID) <= 1 {
				singleManaCount++
			}
		}
		needEfficiency := singleManaCount < genericNeeded

		idx := -1
		bestScore := 0
		for j, src := range sources {
			if src.PermanentID == uuid.Nil {
				continue
			}
			s := scores[j]
			if needEfficiency {
				produced := src.Amount + bonusFor(src.PermanentID)
				saved := min(produced, genericNeeded) - 1
				if saved > 0 {
					s -= saved * efficiencyBonusPerSavedTap
				}
			}
			if idx == -1 || s < bestScore {
				idx = j
				bestScore = s
			}
		}
		if idx == -1 {
			return nil, fmt.Errorf("cannot pay cost %s: %d generic mana still needed, no untapped sources remain", mc, genericNeeded)
		}
		toTap = append(toTap, sources[idx].PermanentID)
		produced := sources[idx].Amount + bonusFor(sources[idx].PermanentID)
		genericNeeded -= produced
		sources[idx].PermanentID = uuid.Nil
	}

	return &ManaSolution{SourcesToTap: toTap}, nil
}
