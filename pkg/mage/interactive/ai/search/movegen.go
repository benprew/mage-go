package search

import (
	"sort"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/eval"
)

// Move represents a single action the AI can take during a priority window.
type Move struct {
	Type         interactive.ActionType
	CardID       uuid.UUID
	CardName     string
	Targets      []uuid.UUID
	PermanentID  uuid.UUID
	AbilityIndex int
	Attackers    []uuid.UUID
	XValue       int // X value for X-cost spells (0 means not an X spell)
	ModeIndex    int // mode index for modal spells (0 = first mode or non-modal)

	IsCreature bool
	IsTactical bool // true for damage spells, removal, combat tricks
	heuristic  int
}

// GeneratePriorityMoves enumerates legal moves for playerID at the current
// priority window, sorted by heuristic value (descending).
func GeneratePriorityMoves(g *mage.Game, p mage.Player, landsPlayed int, mainPhase bool) []Move {
	playerID := p.PlayerID()
	moves := make([]Move, 0, 16)

	for _, c := range g.GetPlayableLands(playerID) {
		moves = append(moves, Move{
			Type:      interactive.ActionPlayLand,
			CardID:    c.ID(),
			CardName:  c.Name(),
			heuristic: 10,
		})
	}

	for _, card := range g.GetCastableSpells(playerID) {
		moves = append(moves, expandSpellMoves(p, g, card)...)
	}

	for _, info := range g.GetActivatableAbilities(playerID) {
		q := abilityQualityFromInfo(g, info)
		moves = append(moves, Move{
			Type:         interactive.ActionActivateAbility,
			PermanentID:  info.PermanentID,
			CardName:     info.PermanentName,
			AbilityIndex: info.AbilityIndex,
			IsTactical:   true, // activated abilities with q>=3 are tactical (damage, draw, etc.)
			heuristic:    q,
		})
	}

	moves = append(moves, Move{
		Type:      interactive.ActionPass,
		heuristic: 0,
	})

	sort.Slice(moves, func(i, j int) bool {
		return moves[i].heuristic > moves[j].heuristic
	})

	return moves
}

// GenerateAttackerSets produces a pruned set of attacker combinations.
// Generates: all, none, each solo, evasive-only, all-but-one (leave back for blocking),
// and top-N-by-value subsets for larger boards.
func GenerateAttackerSets(g *mage.Game, playerID uuid.UUID) [][]uuid.UUID {
	eligible := getEligibleAttackers(g, playerID)
	if len(eligible) == 0 {
		return [][]uuid.UUID{nil}
	}

	var sets [][]uuid.UUID

	all := make([]uuid.UUID, len(eligible))
	for i, p := range eligible {
		all[i] = p.ID()
	}
	sets = append(sets, all)

	sets = append(sets, nil)

	if len(eligible) > 1 {
		for _, p := range eligible {
			sets = append(sets, []uuid.UUID{p.ID()})
		}
	}

	var evasive []uuid.UUID
	for _, p := range eligible {
		if p.HasKeyword(core.Flying) ||
			p.HasKeyword(core.UnblockableKW) ||
			p.HasKeyword(core.Fear) ||
			p.HasKeyword(core.Islandwalk) ||
			p.HasKeyword(core.Swampwalk) ||
			p.HasKeyword(core.Forestwalk) ||
			p.HasKeyword(core.Mountainwalk) ||
			p.HasKeyword(core.Plainswalk) {
			evasive = append(evasive, p.ID())
		}
	}
	if len(evasive) > 0 && len(evasive) < len(eligible) {
		sets = append(sets, evasive)
	}

	// All-but-one: leave each creature back as a potential blocker.
	// This is key for defensive play while still pressuring.
	if len(eligible) > 2 && len(eligible) <= 8 {
		for skip := range eligible {
			subset := make([]uuid.UUID, 0, len(eligible)-1)
			for j, p := range eligible {
				if j != skip {
					subset = append(subset, p.ID())
				}
			}
			sets = append(sets, subset)
		}
	}

	// Value-sorted top-N subsets: attack with the N most valuable creatures.
	// Useful when we want to hold back cheap creatures.
	if len(eligible) > 3 {
		sorted := make([]*mage.Permanent, len(eligible))
		copy(sorted, eligible)
		sort.Slice(sorted, func(i, j int) bool {
			return eval.EvalCreatureInGame(sorted[i], g) > eval.EvalCreatureInGame(sorted[j], g)
		})
		// Top half by value.
		halfN := len(sorted) / 2
		if halfN >= 2 {
			topHalf := make([]uuid.UUID, halfN)
			for i := range halfN {
				topHalf[i] = sorted[i].ID()
			}
			sets = append(sets, topHalf)
		}
	}

	return sets
}

func getEligibleAttackers(g *mage.Game, playerID uuid.UUID) []*mage.Permanent {
	var eligible []*mage.Permanent
	for _, perm := range g.AllBattlefield() {
		if perm.Controller != playerID || !perm.CanDeclareAsAttacker(g) {
			continue
		}
		eligible = append(eligible, perm)
	}
	return eligible
}

func abilityQualityFromInfo(g *mage.Game, info mage.ActivatableInfo) int {
	perm := g.FindPermanent(info.PermanentID)
	if perm == nil {
		return 0
	}
	if info.AbilityIndex >= len(perm.RuntimeAbilities) {
		return 0
	}
	a := perm.RuntimeAbilities[info.AbilityIndex]
	aa, ok := a.(mage.ActivatedAbility)
	if !ok {
		return 0
	}
	return eval.AbilityQuality(aa)
}

func expandSpellMoves(p mage.Player, g *mage.Game, card mage.Card) []Move {
	mc := card.ManaCost()

	// Modal spells: generate one move per mode
	if modes := card.Modes(); len(modes) > 1 {
		return expandModalSpellMoves(p, g, card, modes)
	}

	if mc.HasX {
		return expandXSpellMoves(p, g, card)
	}
	return expandNonXSpellMoves(p, g, card, 0, 0)
}

// expandModalSpellMoves generates one set of moves per mode for modal spells.
func expandModalSpellMoves(p mage.Player, g *mage.Game, card mage.Card, modes []string) []Move {
	var allMoves []Move
	maxModes := min(len(modes), 3)
	for modeIdx := range maxModes {
		baseMoves := expandNonXSpellMoves(p, g, card, 0, modeIdx)
		allMoves = append(allMoves, baseMoves...)
	}
	return allMoves
}

// expandXSpellMoves generates moves for X-cost spells with target-aware values.
// In addition to 1, max/2, and max, it tries X values that are lethal to
// opponent creatures or the opponent's life total.
func expandXSpellMoves(p mage.Player, g *mage.Game, card mage.Card) []Move {
	playerID := p.PlayerID()
	mc := card.ManaCost()
	maxX := g.MaxXValue(playerID, mc, mage.SpellContextForCard(card))
	if maxX < 1 {
		return nil
	}

	// Standard X variants: 1, max/2, max
	xValues := []int{1}
	if half := maxX / 2; half > 1 {
		xValues = append(xValues, half)
	}
	if maxX > 1 {
		xValues = append(xValues, maxX)
	}

	// Target-aware X values: add toughness values of opponent creatures
	// and opponent life total if the spell deals damage.
	isDamageSpell := false
	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok || sa.Kind() != mage.ActionSpell {
			continue
		}
		for _, e := range sa.Effects() {
			if e.Properties().DamageValue != nil {
				isDamageSpell = true
				break
			}
		}
	}

	if isDamageSpell {
		opponent := g.GetOpponent(playerID)
		if opponent != nil {
			// Try X = opponent's life (lethal burn).
			if oppLife := opponent.Life(); oppLife > 0 && oppLife <= maxX {
				xValues = append(xValues, oppLife)
			}
			// Try X = each opponent creature's toughness (lethal removal).
			for _, perm := range g.AllBattlefield() {
				if perm.Controller == opponent.PlayerID() && perm.HasType(core.TypeCreature) {
					tough := perm.CurrentToughness(g)
					if tough > 0 && tough <= maxX {
						xValues = append(xValues, tough)
					}
				}
			}
		}
	}

	// Deduplicate
	seen := make(map[int]bool)
	var unique []int
	for _, x := range xValues {
		if !seen[x] && x >= 1 && x <= maxX {
			seen[x] = true
			unique = append(unique, x)
		}
	}

	var allMoves []Move
	for _, x := range unique {
		moves := expandNonXSpellMoves(p, g, card, x, 0)
		allMoves = append(allMoves, moves...)
	}
	return allMoves
}

// expandNonXSpellMoves generates targeting variants for a spell with an optional X value and mode index.
func expandNonXSpellMoves(p mage.Player, g *mage.Game, card mage.Card, xValue, modeIndex int) []Move {
	playerID := p.PlayerID()
	sv := eval.SpellValue(card, p, g)
	// Bonus for higher X values
	if xValue > 0 {
		sv += xValue
	}

	allTargetReqs := card.CastTargets()
	purpose, damage, outcome := targetPurposeForCard(g, playerID, card, xValue)

	// A spell is tactical if it has detriment or benefit effects (damage, removal,
	// combat tricks, draw). Used by quiescence search to extend at leaf nodes.
	tactical := outcome == mage.OutcomeDetriment || outcome == mage.OutcomeBenefit

	if len(allTargetReqs) == 0 {
		return []Move{{
			Type:       interactive.ActionCastSpell,
			CardID:     card.ID(),
			CardName:   card.Name(),
			IsCreature: card.HasType(core.TypeCreature),
			IsTactical: tactical,
			XValue:     xValue,
			ModeIndex:  modeIndex,
			heuristic:  sv,
		}}
	}

	combos := topTargetCombinations(g, playerID, card, allTargetReqs, purpose, damage)
	if len(combos) == 0 {
		return []Move{{
			Type:       interactive.ActionCastSpell,
			CardID:     card.ID(),
			CardName:   card.Name(),
			IsCreature: card.HasType(core.TypeCreature),
			IsTactical: tactical,
			XValue:     xValue,
			ModeIndex:  modeIndex,
			heuristic:  sv,
		}}
	}

	moves := make([]Move, 0, len(combos))
	for _, targets := range combos {
		h := sv + targetComboScore(g, playerID, targets, purpose, damage)
		if purpose == eval.TargetBurn {
			for _, tid := range targets {
				if tp := g.GetPlayer(tid); tp != nil && tp.PlayerID() != playerID && damage >= tp.Life() {
					h += 100
				}
			}
		}
		moves = append(moves, Move{
			Type:       interactive.ActionCastSpell,
			CardID:     card.ID(),
			CardName:   card.Name(),
			Targets:    targets,
			IsTactical: tactical,
			XValue:     xValue,
			ModeIndex:  modeIndex,
			heuristic:  h,
		})
	}
	return moves
}
