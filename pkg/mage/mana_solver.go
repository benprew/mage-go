package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
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
//   - CostedSources: candidate targetless mana abilities with both a mana cost
//     and a tap cost. When ordinary sources cannot pay Cost, the solver searches
//     for an ordered activation plan that pays these intermediate costs.
//   - Scores: preservation score per source (same length as Sources); higher
//     = prefer to keep untapped. The solver picks lowest-score eligible
//     source at each step. Score is computed by the caller via
//     preservationScore so the solver stays caller-agnostic.
//   - BonusesFor: returns bonus mana from Mana Flare-style abilities elsewhere
//     on the battlefield. Each value is a fixed color or MatchProduced.
//   - Conversions: one-way mana conversions (Sunglasses of Urza style:
//     Red→White lets red mana / red sources pay white slots). Optional.
//   - SpellContext: payment context for restricted mana used on the final cost.
//     Activation costs are ability payments and therefore use no spell context.
type ManaSolverInputs struct {
	Pool          *ManaPool
	Cost          ManaCost
	Sources       []manaSourceInfo
	CostedSources []costedManaSource
	Scores        []int
	BonusesFor    func(uuid.UUID) []ManaBonusColor
	Conversions   map[Color]Color
	SpellContext  *SpellPaymentContext
}

// ManaTap describes one source the caller should tap and the color slot the
// solver picked it for. Color is the colored requirement (White/Blue/Black/
// Red/Green) the source should produce; Colorless means the tap was picked in
// the generic pass and any color the source produces is acceptable.
//
// The color matters for sources that have multiple separate mana abilities
// (e.g. Underground Sea: "{T}: Add {U}" and "{T}: Add {B}"). The caller
// (TapForManaWithColor) uses Color to select the matching ability instead of
// just running the first one.
type ManaTap struct {
	PermanentID uuid.UUID
	// AbilityIndex is -1 for an ordinary tap-for-mana source. Nonnegative
	// values identify the exact costed mana ability that must be activated.
	AbilityIndex int
	Color        Color
}

// ManaSolution describes the result of solving a mana payment. SourcesToTap is
// an ordered list of ordinary taps and exact costed-ability activations. The
// solver doesn't mutate state, so the caller is responsible for applying the
// solution in order.
type ManaSolution struct {
	SourcesToTap []ManaTap
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
//  4. If that fast path fails and CostedSources are present, search ordered
//     activations while simulating each activation cost and mana production.
//
// Bonus mana from ManaBonusAbility on other battlefield permanents (Mana Flare,
// Gauntlet of Might) and the extra mana from multi-mana sources tapped in the
// colored pass flow into a shared surplus counter that reduces generic need.
func SolveMana(in ManaSolverInputs) (*ManaSolution, error) {
	sources := append([]manaSourceInfo(nil), in.Sources...)
	sol, ok, err := solveMana(in, sources, true)
	if ok {
		return sol, nil
	}
	if len(in.CostedSources) > 0 {
		if plan, planned := planCostedManaPayment(in); planned {
			return &ManaSolution{SourcesToTap: plan}, nil
		}
	}
	return nil, err
}

// CanSolveMana reports whether the given mana inputs can pay the cost. It uses
// the same costed-source planning as SolveMana, but discards the resulting plan.
// Without costed sources it may mark entries in in.Sources as used; callers
// should pass disposable scratch storage.
func CanSolveMana(in ManaSolverInputs) bool {
	sources := in.Sources
	if len(in.CostedSources) > 0 {
		sources = append([]manaSourceInfo(nil), sources...)
	}
	_, ok, _ := solveMana(in, sources, false)
	if ok {
		return true
	}
	_, ok = planCostedManaPayment(in)
	return ok
}

func solveMana(in ManaSolverInputs, sources []manaSourceInfo, collectSolution bool) (*ManaSolution, bool, error) {
	mc := in.Cost
	scores := in.Scores

	pool := in.Pool
	conv := in.Conversions

	// Track pool availability per color across the run; consume*ForColor
	// debits both exact and conversion-eligible entries.
	var avail [AnyColor + 1]int
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
	var needed [AnyColor + 1]int
	for _, cn := range colorNeeds {
		consumed := consumePoolForColor(&avail, cn.color, cn.need, conv)
		needed[cn.color] = cn.need - consumed
	}

	// Hybrid resolution: try to satisfy from pool, otherwise commit a
	// concrete color requirement that the colored pass will satisfy.
	// XXX: source-side hybrid choice still uses sourceBucketColor (single
	// declared color), so a Tundra is counted toward {W/x} availability but
	// not {U/x}. Driving the source pre-check off Colors[] would be more
	// accurate; the colored pass below handles correlation correctly via
	// sourceProducesAny.
	var availableForHybridFromSources [AnyColor + 1]int
	for _, src := range sources {
		availableForHybridFromSources[sourceBucketColor(src)] += src.Amount
	}
	for _, h := range mc.Hybrid {
		if consumePoolForColor(&avail, h.A, 1, conv) > 0 {
			continue
		}
		if consumePoolForColor(&avail, h.B, 1, conv) > 0 {
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
			return nil, false, solveManaError(collectSolution, "insufficient mana for hybrid %s", h)
		}
	}

	poolRemaining := 0
	for _, n := range avail {
		poolRemaining += n
	}
	genericNeeded := mc.Generic - min(mc.Generic, poolRemaining)

	bonusesFor := in.BonusesFor
	if bonusesFor == nil {
		bonusesFor = func(uuid.UUID) []ManaBonusColor { return nil }
	}

	var toTap []ManaTap
	surplusMana := 0
	// coloredSurplus tracks colored mana left over after a source satisfies its
	// first slot, including multi-mana sources and mana-bonus effects.
	var coloredSurplus [AnyColor + 1]int

	// Colored pass: per slot, pick lowest-score source that can produce the
	// required color. Extra and bonus mana retain their colors so a source that
	// produces {U}{U}, including via Mana Flare, can satisfy two blue pips.
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
				if idx == -1 || scoreAt(scores, j) < scoreAt(scores, idx) {
					idx = j
				}
			}
			if idx == -1 {
				return nil, false, solveManaError(collectSolution, "cannot pay %s for cost %s: no untapped %s source available", cn.color, mc, cn.color)
			}
			src := sources[idx]
			sources[idx].PermanentID = uuid.Nil
			productionColor := sourceColorForRequirement(src, cn.color, conv)
			if collectSolution {
				toTap = append(toTap, ManaTap{PermanentID: src.PermanentID, AbilityIndex: -1, Color: productionColor})
			}
			needed[cn.color]--
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
				// Any leftover Amount beyond the per-tap colored output (rare;
				// indicates Amount > sum of PerTapOutput) falls into generic.
				if src.Amount > total {
					surplusMana += src.Amount - total
				}
			} else {
				extra := max(src.Amount-1, 0)
				effectiveColor := convertedManaColor(productionColor, conv)
				if effectiveColor == Colorless {
					surplusMana += extra
				} else {
					coloredSurplus[effectiveColor] += extra
				}
			}
			for _, bonus := range bonusesFor(src.PermanentID) {
				effectiveColor := convertedManaColor(bonus.Resolve(productionColor), conv)
				if effectiveColor == Colorless {
					surplusMana++
				} else {
					coloredSurplus[effectiveColor]++
				}
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
			if src.Amount+len(bonusesFor(src.PermanentID)) <= 1 {
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
			s := scoreAt(scores, j)
			if needEfficiency {
				produced := src.Amount + len(bonusesFor(src.PermanentID))
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
			return nil, false, solveManaError(collectSolution, "cannot pay cost %s: %d generic mana still needed, no untapped sources remain", mc, genericNeeded)
		}
		if collectSolution {
			toTap = append(toTap, ManaTap{PermanentID: sources[idx].PermanentID, AbilityIndex: -1, Color: Colorless})
		}
		produced := sources[idx].Amount + len(bonusesFor(sources[idx].PermanentID))
		genericNeeded -= produced
		sources[idx].PermanentID = uuid.Nil
	}

	if !collectSolution {
		return nil, true, nil
	}
	return &ManaSolution{SourcesToTap: toTap}, true, nil
}

func sourceColorForRequirement(src manaSourceInfo, required Color, conv map[Color]Color) Color {
	for _, color := range src.Colors {
		if color == required {
			return color
		}
	}
	for _, color := range src.Colors {
		if convertedManaColor(color, conv) == required {
			return color
		}
	}
	return required
}

func convertedManaColor(color Color, conv map[Color]Color) Color {
	if converted, ok := conv[color]; ok {
		return converted
	}
	return color
}

func scoreAt(scores []int, i int) int {
	if i < len(scores) {
		return scores[i]
	}
	return 0
}

func solveManaError(needed bool, format string, args ...any) error {
	if !needed {
		return nil
	}
	return fmt.Errorf(format, args...)
}

// consumePoolForColor debits up to `need` mana of color `c` from `avail`,
// first using exact-color entries, then any color X whose conv[X] == c
// (one-way conversions like Sunglasses of Urza's Red→White). Returns the
// amount actually consumed (≤ need).
func consumePoolForColor(avail *[AnyColor + 1]int, c Color, need int, conv map[Color]Color) int {
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
