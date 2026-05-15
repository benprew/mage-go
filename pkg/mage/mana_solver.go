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
//   - Conversions: one-way mana conversions (Sunglasses of Urza style:
//     Red→White lets red mana / red sources pay white slots). Optional.
type ManaSolverInputs struct {
	Pool        *ManaPool
	Cost        ManaCost
	Sources     []manaSourceInfo
	Scores      []int
	BonusFor    func(uuid.UUID) int
	Conversions map[Color]Color
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
	conv := in.Conversions

	// Track pool availability per color across the run; consume*ForColor
	// debits both exact and conversion-eligible entries.
	avail := map[Color]int{}
	for _, color := range manaPoolColors {
		if c := pool.Count(color); c > 0 {
			avail[color] = c
		}
	}

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
	for _, cn := range colorNeeds {
		consumed := consumePoolForColor(avail, cn.color, cn.need, conv)
		needed[cn.color] = cn.need - consumed
	}

	// Hybrid resolution: try to satisfy from pool, otherwise commit a
	// concrete color requirement that the colored pass will satisfy.
	// XXX: source-side hybrid choice still uses sourceBucketColor (single
	// declared color), so a Tundra is counted toward {W/x} availability but
	// not {U/x}. Driving the source pre-check off Colors[] would be more
	// accurate; the colored pass below handles correlation correctly via
	// sourceProducesAny.
	availableForHybridFromSources := map[Color]int{}
	for _, src := range sources {
		availableForHybridFromSources[sourceBucketColor(src)] += src.Amount
	}
	for _, h := range mc.Hybrid {
		if consumePoolForColor(avail, h.A, 1, conv) > 0 {
			continue
		}
		if consumePoolForColor(avail, h.B, 1, conv) > 0 {
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

	poolRemaining := 0
	for _, n := range avail {
		poolRemaining += n
	}
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
				if src.PermanentID == uuid.Nil || !sourceProducesAny(src, cn.color, conv) {
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
				// Conversions translate the output color into the pool color
				// it will arrive as, then debit one unit of the matched color.
				total := 0
				consumed := false
				for c, amt := range src.PerTapOutput {
					total += amt
					effC := c
					if conv != nil {
						if to, ok := conv[c]; ok {
							effC = to
						}
					}
					if !consumed && effC == cn.color {
						amt--
						consumed = true
					}
					if effC == Colorless {
						surplusMana += amt
					} else {
						coloredSurplus[effC] += amt
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

// consumePoolForColor debits up to `need` mana of color `c` from `avail`,
// first using exact-color entries, then any color X whose conv[X] == c
// (one-way conversions like Sunglasses of Urza's Red→White). Returns the
// amount actually consumed (≤ need).
func consumePoolForColor(avail map[Color]int, c Color, need int, conv map[Color]Color) int {
	if need == 0 {
		return 0
	}
	consumed := 0
	if avail[c] > 0 {
		use := min(avail[c], need)
		avail[c] -= use
		consumed += use
		need -= use
	}
	if need == 0 || conv == nil {
		return consumed
	}
	for from, to := range conv {
		if to != c || from == c {
			continue
		}
		if avail[from] > 0 {
			use := min(avail[from], need)
			avail[from] -= use
			consumed += use
			need -= use
			if need == 0 {
				break
			}
		}
	}
	return consumed
}

// sourceProducesAny reports whether src can produce color c, accounting for
// one-way conversions: a source whose declared color X satisfies c if
// conv[X] == c.
func sourceProducesAny(src manaSourceInfo, c Color, conv map[Color]Color) bool {
	if sourceProduces(src, c) {
		return true
	}
	if conv == nil {
		return false
	}
	for _, sc := range src.Colors {
		if conv[sc] == c {
			return true
		}
	}
	return false
}
