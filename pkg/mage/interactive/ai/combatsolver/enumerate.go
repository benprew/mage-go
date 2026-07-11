package combatsolver

import (
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

// enumerateAttackerSets returns candidate attacker subsets for playerID to
// declare. It is intentionally redundant — duplicates are deduped by the
// solver via a key-based seen map. Subsets included:
//
//   - none (skip combat)
//   - all eligible
//   - each single attacker
//   - all evasive (flying / unblockable / landwalk)
//   - all-but-one (when 3 ≤ n ≤ 8)
//   - top half by EvalCreatureInGame value (when n > 3)
//   - full powerset when n ≤ 5 (covers everything else)
func enumerateAttackerSets(g *mage.Game, playerID uuid.UUID) [][]uuid.UUID {
	eligible := eligibleAttackers(g, playerID)
	if len(eligible) == 0 {
		return [][]uuid.UUID{nil}
	}

	sets := [][]uuid.UUID{nil}

	all := permIDs(eligible)
	sets = append(sets, all)

	if len(eligible) > 1 {
		for _, p := range eligible {
			sets = append(sets, []uuid.UUID{p.ID()})
		}
	}

	var evasive []uuid.UUID
	for _, p := range eligible {
		if isEvasive(p) {
			evasive = append(evasive, p.ID())
		}
	}
	if len(evasive) > 0 && len(evasive) < len(eligible) {
		sets = append(sets, evasive)
	}

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

	if len(eligible) > 3 {
		sorted := make([]*mage.Permanent, len(eligible))
		copy(sorted, eligible)
		sort.Slice(sorted, func(i, j int) bool {
			return eval.EvalCreatureInGame(sorted[i], g) > eval.EvalCreatureInGame(sorted[j], g)
		})
		halfN := len(sorted) / 2
		if halfN >= 2 {
			topHalf := make([]uuid.UUID, halfN)
			for i := range halfN {
				topHalf[i] = sorted[i].ID()
			}
			sets = append(sets, topHalf)
		}
	}

	if len(eligible) <= 5 {
		full := powerset(all)
		sets = append(sets, full...)
	}

	return dedupeIDSets(sets)
}

// enumerateBlockerSets returns candidate blocker assignments for the player
// whose creatures are being attacked. Includes:
//
//   - no blocks
//   - each valid single block (one blocker on one attacker)
//   - greedy 1:1 assignment (block as many attackers as possible)
//   - gang-block the biggest attacker (2 blockers) plus 1:1 on the rest
//   - full enumeration of valid 1-blocker-per-attacker permutations when small
//     (≤3 attackers, ≤4 blockers)
//
// Blockers with CanBlockAdditional / CanBlockAny are deferred to a follow-up
// phase (rare in early sets); current enumeration assigns each blocker to at
// most one attacker.
func enumerateBlockerSets(g *mage.Game, defenderID uuid.UUID) [][]mage.BlockAssignment {
	attackers := attackersAgainst(g, defenderID)
	if len(attackers) == 0 {
		return [][]mage.BlockAssignment{nil}
	}

	blockers := eligibleBlockers(g, defenderID)
	if len(blockers) == 0 {
		return [][]mage.BlockAssignment{nil}
	}

	sets := [][]mage.BlockAssignment{nil}

	// Single blocks per attacker.
	for _, atk := range attackers {
		if mage.HasLandwalkEvasion(atk, defenderID, g) {
			continue
		}
		for _, blk := range blockers {
			if !mage.CanBlock(blk, atk, g) {
				continue
			}
			sets = append(sets, []mage.BlockAssignment{{
				BlockerID:  blk.ID(),
				AttackerID: atk.ID(),
			}})
		}
	}

	// Greedy 1:1 assignment.
	if greedy := greedyAssignment(attackers, blockers, defenderID, g); len(greedy) > 0 {
		sets = append(sets, greedy)
	}

	// Gang-block biggest attacker + 1:1 on the rest.
	if gang := gangBlockBiggest(attackers, blockers, defenderID, g); len(gang) > 0 {
		sets = append(sets, gang)
	}

	// Full permutation enumeration when small.
	if len(attackers) <= 3 && len(blockers) <= 4 {
		enumerateBlockerPermutations(attackers, blockers, defenderID, g, &sets)
	}

	return dedupeBlockSets(sets)
}

func eligibleAttackers(g *mage.Game, playerID uuid.UUID) []*mage.Permanent {
	var out []*mage.Permanent
	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() != playerID || !perm.CanDeclareAsAttacker(g) {
			continue
		}
		out = append(out, perm)
	}
	return out
}

func eligibleBlockers(g *mage.Game, playerID uuid.UUID) []*mage.Permanent {
	var out []*mage.Permanent
	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() != playerID || !perm.CanDeclareAsBlocker(g) {
			continue
		}
		out = append(out, perm)
	}
	return out
}

func attackersAgainst(g *mage.Game, defenderID uuid.UUID) []*mage.Permanent {
	var out []*mage.Permanent
	for _, group := range g.CombatGroups() {
		if group.DefenderID != defenderID {
			continue
		}
		atk := g.FindPermanent(group.AttackerID)
		if atk != nil {
			out = append(out, atk)
		}
	}
	return out
}

func isEvasive(p *mage.Permanent) bool {
	switch {
	case p.HasKeyword(core.Flying),
		p.HasKeyword(core.UnblockableKW),
		p.HasKeyword(core.Fear),
		p.HasKeyword(core.Islandwalk),
		p.HasKeyword(core.Swampwalk),
		p.HasKeyword(core.Forestwalk),
		p.HasKeyword(core.Mountainwalk),
		p.HasKeyword(core.Plainswalk):
		return true
	}
	return false
}

func permIDs(perms []*mage.Permanent) []uuid.UUID {
	out := make([]uuid.UUID, len(perms))
	for i, p := range perms {
		out[i] = p.ID()
	}
	return out
}

func powerset(ids []uuid.UUID) [][]uuid.UUID {
	n := len(ids)
	out := make([][]uuid.UUID, 0, 1<<n)
	for mask := 0; mask < 1<<n; mask++ {
		var sub []uuid.UUID
		for i := range n {
			if mask&(1<<i) != 0 {
				sub = append(sub, ids[i])
			}
		}
		out = append(out, sub)
	}
	return out
}

func greedyAssignment(attackers, blockers []*mage.Permanent, defenderID uuid.UUID, g *mage.Game) []mage.BlockAssignment {
	used := make(map[uuid.UUID]bool)
	var out []mage.BlockAssignment
	for _, atk := range attackers {
		if mage.HasLandwalkEvasion(atk, defenderID, g) {
			continue
		}
		for _, blk := range blockers {
			if used[blk.ID()] {
				continue
			}
			if !mage.CanBlock(blk, atk, g) {
				continue
			}
			out = append(out, mage.BlockAssignment{BlockerID: blk.ID(), AttackerID: atk.ID()})
			used[blk.ID()] = true
			break
		}
	}
	return out
}

func gangBlockBiggest(attackers, blockers []*mage.Permanent, defenderID uuid.UUID, g *mage.Game) []mage.BlockAssignment {
	if len(attackers) == 0 {
		return nil
	}
	biggest := attackers[0]
	for _, atk := range attackers[1:] {
		if atk.CurrentPower(g) > biggest.CurrentPower(g) {
			biggest = atk
		}
	}
	if mage.HasLandwalkEvasion(biggest, defenderID, g) {
		return nil
	}

	var out []mage.BlockAssignment
	used := make(map[uuid.UUID]bool)
	gangCount := 0
	for _, blk := range blockers {
		if gangCount >= 2 {
			break
		}
		if !mage.CanBlock(blk, biggest, g) {
			continue
		}
		out = append(out, mage.BlockAssignment{BlockerID: blk.ID(), AttackerID: biggest.ID()})
		used[blk.ID()] = true
		gangCount++
	}
	if gangCount < 2 {
		return nil
	}

	for _, atk := range attackers {
		if atk.ID() == biggest.ID() || mage.HasLandwalkEvasion(atk, defenderID, g) {
			continue
		}
		for _, blk := range blockers {
			if used[blk.ID()] || !mage.CanBlock(blk, atk, g) {
				continue
			}
			out = append(out, mage.BlockAssignment{BlockerID: blk.ID(), AttackerID: atk.ID()})
			used[blk.ID()] = true
			break
		}
	}
	return out
}

func enumerateBlockerPermutations(attackers, blockers []*mage.Permanent,
	defenderID uuid.UUID, g *mage.Game, sets *[][]mage.BlockAssignment) {

	type pair struct{ blkIdx, atkIdx int }
	var validPairs []pair
	for bi, blk := range blockers {
		for ai, atk := range attackers {
			if mage.CanBlock(blk, atk, g) && !mage.HasLandwalkEvasion(atk, defenderID, g) {
				validPairs = append(validPairs, pair{bi, ai})
			}
		}
	}

	const maxSets = 32
	if len(*sets) >= maxSets {
		return
	}
	maxAdd := maxSets - len(*sets)

	seen := make(map[string]bool)
	var generate func(start int, current []pair, usedBlk, usedAtk map[int]bool)
	generate = func(start int, current []pair, usedBlk, usedAtk map[int]bool) {
		if len(seen) >= maxAdd {
			return
		}
		if len(current) > 0 {
			set := make([]mage.BlockAssignment, len(current))
			var key strings.Builder
			for i, p := range current {
				set[i] = mage.BlockAssignment{
					BlockerID:  blockers[p.blkIdx].ID(),
					AttackerID: attackers[p.atkIdx].ID(),
				}
				key.WriteString(blockers[p.blkIdx].ID().String() + ">" + attackers[p.atkIdx].ID().String() + ",")
			}
			if !seen[key.String()] {
				seen[key.String()] = true
				*sets = append(*sets, set)
			}
		}
		for i := start; i < len(validPairs); i++ {
			p := validPairs[i]
			if usedBlk[p.blkIdx] || usedAtk[p.atkIdx] {
				continue
			}
			usedBlk[p.blkIdx] = true
			usedAtk[p.atkIdx] = true
			generate(i+1, append(current, p), usedBlk, usedAtk)
			delete(usedBlk, p.blkIdx)
			delete(usedAtk, p.atkIdx)
		}
	}
	generate(0, nil, map[int]bool{}, map[int]bool{})
}

func dedupeIDSets(sets [][]uuid.UUID) [][]uuid.UUID {
	seen := make(map[string]bool, len(sets))
	out := make([][]uuid.UUID, 0, len(sets))
	for _, s := range sets {
		sorted := make([]uuid.UUID, len(s))
		copy(sorted, s)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].String() < sorted[j].String() })
		var key strings.Builder
		for _, id := range sorted {
			key.WriteString(id.String() + ",")
		}
		if !seen[key.String()] {
			seen[key.String()] = true
			out = append(out, s)
		}
	}
	return out
}

func dedupeBlockSets(sets [][]mage.BlockAssignment) [][]mage.BlockAssignment {
	seen := make(map[string]bool, len(sets))
	out := make([][]mage.BlockAssignment, 0, len(sets))
	for _, s := range sets {
		pairs := make([]string, len(s))
		for i, a := range s {
			pairs[i] = a.BlockerID.String() + ">" + a.AttackerID.String()
		}
		sort.Strings(pairs)
		var key strings.Builder
		for _, p := range pairs {
			key.WriteString(p + ",")
		}
		if !seen[key.String()] {
			seen[key.String()] = true
			out = append(out, s)
		}
	}
	return out
}
