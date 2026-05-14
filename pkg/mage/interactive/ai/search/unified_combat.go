package search

import (
	"sort"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

type attackingPlayer struct {
	*mage.SearchPlayer
	attackers []uuid.UUID
}

func (ap *attackingPlayer) DeclareAttackers(*mage.Game) []uuid.UUID {
	return ap.attackers
}

type blockingPlayer struct {
	*mage.SearchPlayer
	blocks []mage.BlockAssignment
}

func (bp *blockingPlayer) DeclareBlockers(*mage.Game) []mage.BlockAssignment {
	return bp.blocks
}

func installAttackerOverride(g *mage.Game, idx int, attackers []uuid.UUID) {
	sp, ok := g.PlayerAt(idx).(*mage.SearchPlayer)
	if !ok {
		return
	}
	g.SetPlayerAt(idx, &attackingPlayer{SearchPlayer: sp, attackers: attackers})
}

func installBlockerOverride(g *mage.Game, idx int, blocks []mage.BlockAssignment) {
	sp, ok := g.PlayerAt(idx).(*mage.SearchPlayer)
	if !ok {
		return
	}
	g.SetPlayerAt(idx, &blockingPlayer{SearchPlayer: sp, blocks: blocks})
}

func attackerSubsets(g *mage.Game, attackerPlayerID uuid.UUID) [][]uuid.UUID {
	return GenerateAttackerSets(g, attackerPlayerID)
}

func blockerSubsets(g *mage.Game, defenderID uuid.UUID) [][]mage.BlockAssignment {
	groups := g.CombatGroups()
	if len(groups) == 0 {
		return [][]mage.BlockAssignment{nil}
	}

	var attackers []*mage.Permanent
	for _, group := range groups {
		if group.DefenderID != defenderID {
			continue
		}
		if atk := g.FindPermanent(group.AttackerID); atk != nil {
			attackers = append(attackers, atk)
		}
	}
	if len(attackers) == 0 {
		return [][]mage.BlockAssignment{nil}
	}

	var blockers []*mage.Permanent
	for _, perm := range g.AllBattlefield() {
		if perm.Controller == defenderID && perm.CanDeclareAsBlocker(g) {
			blockers = append(blockers, perm)
		}
	}
	if len(blockers) == 0 {
		return [][]mage.BlockAssignment{nil}
	}

	sets := [][]mage.BlockAssignment{nil}
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

	if greedy := greedy1to1(attackers, blockers, defenderID, g); len(greedy) > 0 {
		sets = append(sets, greedy)
	}
	sets = append(sets, gangBlockSets(attackers, blockers, defenderID, g)...)

	return sets
}

func gangBlockSets(attackers, blockers []*mage.Permanent, defenderID uuid.UUID, g *mage.Game) [][]mage.BlockAssignment {
	if len(blockers) < 2 {
		return nil
	}
	var out [][]mage.BlockAssignment
	for _, atk := range attackers {
		if mage.HasLandwalkEvasion(atk, defenderID, g) {
			continue
		}
		var legal []*mage.Permanent
		for _, blk := range blockers {
			if mage.CanBlock(blk, atk, g) {
				legal = append(legal, blk)
			}
		}
		if len(legal) < 2 {
			continue
		}
		set := make([]mage.BlockAssignment, 0, len(legal))
		for _, blk := range legal {
			set = append(set, mage.BlockAssignment{BlockerID: blk.ID(), AttackerID: atk.ID()})
		}
		out = append(out, set)
	}
	return out
}

func greedy1to1(attackers, blockers []*mage.Permanent, defenderID uuid.UUID, g *mage.Game) []mage.BlockAssignment {
	atk := append([]*mage.Permanent(nil), attackers...)
	blk := append([]*mage.Permanent(nil), blockers...)
	sort.Slice(atk, func(i, j int) bool { return permValue(atk[i], g) > permValue(atk[j], g) })
	sort.Slice(blk, func(i, j int) bool { return permValue(blk[i], g) > permValue(blk[j], g) })

	used := make(map[uuid.UUID]bool, len(blk))
	var out []mage.BlockAssignment
	for _, a := range atk {
		if mage.HasLandwalkEvasion(a, defenderID, g) {
			continue
		}
		for _, b := range blk {
			if used[b.ID()] {
				continue
			}
			if !mage.CanBlock(b, a, g) {
				continue
			}
			out = append(out, mage.BlockAssignment{BlockerID: b.ID(), AttackerID: a.ID()})
			used[b.ID()] = true
			break
		}
	}
	return out
}

func permValue(p *mage.Permanent, g *mage.Game) int {
	if p.HasType(core.TypeCreature) {
		return p.CurrentPower(g) + p.CurrentToughness(g)
	}
	return 0
}
