package search

import (
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

const (
	maxExhaustiveAttackerN  = 0
	maxExhaustiveBlockPlans = 0
)

type attackingPlayer struct {
	*mage.SearchPlayer
	attackers []uuid.UUID
}

func (ap *attackingPlayer) DeclareAttackers(*mage.Game) []uuid.UUID {
	return ap.attackers
}

func (ap *attackingPlayer) GetBlockerOrder(g *mage.Game, attacker *mage.Permanent, blockers []*mage.Permanent, _ int) []uuid.UUID {
	if len(blockers) < 2 {
		return nil
	}
	ordered := append([]*mage.Permanent(nil), blockers...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return blockerDamageOrderLess(g, attacker, ordered[i], ordered[j])
	})
	out := make([]uuid.UUID, len(ordered))
	for i, blocker := range ordered {
		out[i] = blocker.ID()
	}
	return out
}

func (ap *attackingPlayer) GetCombatDamageAssignment(g *mage.Game, attacker *mage.Permanent, blockers []*mage.Permanent, totalPower int) map[uuid.UUID]int {
	if totalPower <= 0 || len(blockers) == 0 {
		return nil
	}
	assignment := make(map[uuid.UUID]int, len(blockers))
	remaining := totalPower
	for _, blocker := range blockers {
		if blocker == nil || remaining <= 0 {
			continue
		}
		lethal := lethalDamageTo(g, attacker, blocker)
		if lethal <= 0 {
			continue
		}
		dmg := min(lethal, remaining)
		assignment[blocker.ID()] = dmg
		remaining -= dmg
	}
	return assignment
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
	eligible := getEligibleAttackers(g, attackerPlayerID)
	if len(eligible) == 0 {
		return [][]uuid.UUID{nil}
	}

	var candidates [][]uuid.UUID
	if len(eligible) <= maxExhaustiveAttackerN {
		candidates = append(candidates, exhaustiveAttackPlans(eligible)...)
	} else {
		candidates = append(candidates, GenerateAttackerSets(g, attackerPlayerID)...)
	}

	candidates = dedupeAttackPlans(candidates)
	sortAttackPlans(g, attackerPlayerID, candidates)
	return candidates
}

func blockerSubsets(g *mage.Game, defenderID uuid.UUID) [][]mage.BlockAssignment {
	attackers := combatAttackersForDefender(g, defenderID)
	if len(attackers) == 0 {
		return [][]mage.BlockAssignment{nil}
	}

	blockers := legalBlockers(g, defenderID)
	if len(blockers) == 0 {
		return [][]mage.BlockAssignment{nil}
	}

	sets := [][]mage.BlockAssignment{nil}

	if exhaustive, ok := exhaustiveBlockPlans(g, attackers, blockers); ok {
		sets = append(sets, exhaustive...)
	} else {
		sets = append(sets, legacyBlockerSubsets(g, defenderID, attackers, blockers)...)
	}
	sets = append(sets, additionalBlockerPlans(g, defenderID, attackers, blockers)...)

	sets = repairBlockPlans(g, sets)
	sets = dedupeBlockPlans(sets)
	sortBlockPlans(g, defenderID, sets)
	return sets
}

func legacyBlockerSubsets(g *mage.Game, defenderID uuid.UUID, attackers, blockers []*mage.Permanent) [][]mage.BlockAssignment {
	var sets [][]mage.BlockAssignment
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

func additionalBlockerPlans(g *mage.Game, defenderID uuid.UUID, attackers, blockers []*mage.Permanent) [][]mage.BlockAssignment {
	var out [][]mage.BlockAssignment
	for _, blocker := range blockers {
		maxBlocks := 1
		if blocker.HasKeyword(core.CanBlockAny) {
			maxBlocks = 3
		} else if blocker.HasKeyword(core.CanBlockAdditional) {
			maxBlocks = 2
		}
		if maxBlocks < 2 {
			continue
		}
		var legal []*mage.Permanent
		for _, attacker := range attackers {
			if mage.HasLandwalkEvasion(attacker, defenderID, g) || !mage.CanBlock(blocker, attacker, g) {
				continue
			}
			legal = append(legal, attacker)
		}
		if len(legal) < 2 {
			continue
		}
		sort.SliceStable(legal, func(i, j int) bool {
			return attackerUrgency(g, legal[i]) > attackerUrgency(g, legal[j])
		})
		n := min(maxBlocks, len(legal))
		plan := make([]mage.BlockAssignment, 0, n)
		for _, attacker := range legal[:n] {
			plan = append(plan, mage.BlockAssignment{BlockerID: blocker.ID(), AttackerID: attacker.ID()})
		}
		out = append(out, plan)
	}
	return out
}

func combatAttackersForDefender(g *mage.Game, defenderID uuid.UUID) []*mage.Permanent {
	var attackers []*mage.Permanent
	for _, group := range g.CombatGroups() {
		if group.DefenderID != defenderID {
			continue
		}
		if atk := g.FindPermanent(group.AttackerID); atk != nil {
			attackers = append(attackers, atk)
		}
	}
	return attackers
}

func legalBlockers(g *mage.Game, defenderID uuid.UUID) []*mage.Permanent {
	var blockers []*mage.Permanent
	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() == defenderID && perm.CanDeclareAsBlocker(g) {
			blockers = append(blockers, perm)
		}
	}
	return blockers
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
		return creatureValue(g, p)
	}
	return 0
}

func creatureValue(g *mage.Game, p *mage.Permanent) int {
	if p == nil || !p.HasType(core.TypeCreature) {
		return 0
	}
	v := p.CurrentPower(g)*10 + p.CurrentToughness(g)*12 + p.Card.ManaCost().CMC()*2
	if p.HasKeyword(core.Flying) || p.HasKeyword(core.Fear) || p.HasKeyword(core.Menace) {
		v += 15
	}
	if p.HasKeyword(core.UnblockableKW) {
		v += 30
	}
	if p.HasKeyword(core.Trample) {
		v += 10
	}
	if p.HasKeyword(core.FirstStrike) {
		v += 15
	}
	if p.HasKeyword(core.DoubleStrike) {
		v += 30
	}
	if p.HasKeyword(core.Deathtouch) {
		v += 20
	}
	if p.HasKeyword(core.Lifelink) {
		v += 15
	}
	if p.HasKeyword(core.Vigilance) {
		v += 10
	}
	if p.HasKeyword(core.Indestructible) {
		v += 30
	}
	if p.HasAttr(core.AttrMustAttack) {
		v -= 5
	}
	if ev := eval.EvalCreatureInGame(p, g); ev > 0 {
		v += ev
	}
	if v < 1 {
		return 1
	}
	return v
}

func effectiveDamage(g *mage.Game, p *mage.Permanent) int {
	if p == nil {
		return 0
	}
	dmg := p.CurrentPower(g)
	if p.HasAttr(core.AttrAssignsDamageEqualToToughness) {
		dmg = p.CurrentToughness(g)
	}
	if p.HasKeyword(core.DoubleStrike) {
		dmg *= 2
	}
	return max(dmg, 0)
}

func canKill(g *mage.Game, source, target *mage.Permanent) bool {
	if source == nil || target == nil {
		return false
	}
	dmg := effectiveDamage(g, source)
	if dmg <= 0 {
		return false
	}
	return source.HasKeyword(core.Deathtouch) || dmg >= target.CurrentToughness(g)-target.Damage
}

func survivesAgainst(g *mage.Game, creature, opposing *mage.Permanent) bool {
	if creature == nil || opposing == nil {
		return false
	}
	if creature.HasKeyword(core.Indestructible) {
		return true
	}
	if opposing.HasKeyword(core.Deathtouch) && effectiveDamage(g, opposing) > 0 {
		return false
	}
	if opposing.HasKeyword(core.FirstStrike) && !creature.HasKeyword(core.FirstStrike) && !creature.HasKeyword(core.DoubleStrike) && canKill(g, opposing, creature) {
		return false
	}
	if creature.HasKeyword(core.FirstStrike) && !opposing.HasKeyword(core.FirstStrike) && !opposing.HasKeyword(core.DoubleStrike) && canKill(g, creature, opposing) {
		return true
	}
	return effectiveDamage(g, opposing) < creature.CurrentToughness(g)-creature.Damage
}

func lethalDamageTo(g *mage.Game, source, target *mage.Permanent) int {
	if source == nil || target == nil {
		return 0
	}
	if source.HasKeyword(core.Deathtouch) {
		return 1
	}
	return max(target.CurrentToughness(g)-target.Damage, 1)
}

func blockerDamageOrderLess(g *mage.Game, attacker, a, b *mage.Permanent) bool {
	if attacker != nil && attacker.HasKeyword(core.Trample) {
		la := lethalDamageTo(g, attacker, a)
		lb := lethalDamageTo(g, attacker, b)
		if la != lb {
			return la < lb
		}
	}
	va := creatureValue(g, a)
	vb := creatureValue(g, b)
	if va != vb {
		return va > vb
	}
	return a.ID().String() < b.ID().String()
}

func exhaustiveAttackPlans(eligible []*mage.Permanent) [][]uuid.UUID {
	n := len(eligible)
	out := make([][]uuid.UUID, 0, 1<<min(n, 16))
	for mask := 0; mask < (1 << n); mask++ {
		var set []uuid.UUID
		for i, perm := range eligible {
			if mask&(1<<i) != 0 {
				set = append(set, perm.ID())
			}
		}
		out = append(out, set)
	}
	return out
}

func shouldHeuristicAttack(g *mage.Game, atk *mage.Permanent, opponentID uuid.UUID) bool {
	if atk.CurrentPower(g) <= 0 && !atk.HasKeyword(core.DoubleStrike) {
		return false
	}
	if atk.HasAttr(core.AttrMustAttack) || atk.HasKeyword(core.Vigilance) || atk.HasKeyword(core.Indestructible) || isEvasiveTo(g, atk, opponentID) {
		return true
	}

	var bestBlocker *mage.Permanent
	bestScore := -1 << 30
	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() != opponentID || !perm.HasType(core.TypeCreature) || perm.Tapped {
			continue
		}
		if !mage.CanBlock(perm, atk, g) || mage.HasLandwalkEvasion(atk, opponentID, g) {
			continue
		}
		score := creatureValue(g, perm)
		if score > bestScore {
			bestScore = score
			bestBlocker = perm
		}
	}
	if bestBlocker == nil {
		return true
	}

	// Compare the defender's no-block option against its best material block.
	noBlockValue := effectiveDamage(g, atk) * 10
	blockValue := 0
	if canKill(g, atk, bestBlocker) {
		blockValue += creatureValue(g, bestBlocker)
	}
	if !survivesAgainst(g, atk, bestBlocker) {
		blockValue -= creatureValue(g, atk)
	}
	if atk.HasKeyword(core.Trample) {
		blockValue += max(effectiveDamage(g, atk)-lethalDamageTo(g, atk, bestBlocker), 0) * 10
	}
	defenderBest := min(noBlockValue, blockValue)
	return defenderBest > 0
}

func isEvasiveTo(g *mage.Game, atk *mage.Permanent, defenderID uuid.UUID) bool {
	if atk.HasKeyword(core.UnblockableKW) || mage.HasLandwalkEvasion(atk, defenderID, g) {
		return true
	}
	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() != defenderID || !perm.HasType(core.TypeCreature) || perm.Tapped {
			continue
		}
		if mage.CanBlock(perm, atk, g) {
			return false
		}
	}
	return true
}

func attackPlanScore(g *mage.Game, playerID uuid.UUID, plan []uuid.UUID) int {
	opponent := g.GetOpponent(playerID)
	if opponent == nil {
		return 0
	}
	opponentID := opponent.PlayerID()
	score := 0
	totalDamage := 0
	for _, id := range plan {
		atk := g.FindPermanent(id)
		if atk == nil {
			continue
		}
		totalDamage += effectiveDamage(g, atk)
		if atk.HasAttr(core.AttrMustAttack) {
			score += 100
		}
		if atk.HasKeyword(core.Lifelink) {
			score += effectiveDamage(g, atk) * 4
		}
		if !atk.HasKeyword(core.Vigilance) {
			score -= crackbackBlockerValue(g, atk, opponentID)
		}
		if shouldHeuristicAttack(g, atk, opponentID) {
			score += 20
		}
	}
	score += totalDamage * 10
	if totalDamage >= opponent.Life() {
		score += 100000
	}
	return score
}

func crackbackBlockerValue(g *mage.Game, atk *mage.Permanent, opponentID uuid.UUID) int {
	me := g.GetPlayer(atk.ControllerID())
	if me == nil || me.Life() > 8 {
		return 0
	}
	best := 0
	for _, opp := range g.AllBattlefield() {
		if opp.ControllerID() != opponentID || !opp.HasType(core.TypeCreature) {
			continue
		}
		if mage.CanBlock(atk, opp, g) {
			best = max(best, effectiveDamage(g, opp))
		}
	}
	return best * 8
}

func sortAttackPlans(g *mage.Game, playerID uuid.UUID, plans [][]uuid.UUID) {
	type scoredPlan struct {
		plan  []uuid.UUID
		score int
	}
	scored := make([]scoredPlan, len(plans))
	for i, plan := range plans {
		scored[i] = scoredPlan{plan: plan, score: attackPlanScore(g, playerID, plan)}
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return attackerSubsetLess(scored[i].plan, scored[j].plan)
	})
	for i := range scored {
		plans[i] = scored[i].plan
	}
}

func dedupeAttackPlans(plans [][]uuid.UUID) [][]uuid.UUID {
	seen := make(map[string]bool, len(plans))
	out := make([][]uuid.UUID, 0, len(plans))
	for _, plan := range plans {
		cp := append([]uuid.UUID(nil), plan...)
		sort.Slice(cp, func(i, j int) bool { return cp[i].String() < cp[j].String() })
		key := idsKey(cp)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, cp)
	}
	return out
}

func exhaustiveBlockPlans(g *mage.Game, attackers, blockers []*mage.Permanent) ([][]mage.BlockAssignment, bool) {
	choices := make([][][]uuid.UUID, len(blockers))
	total := 1
	for i, blocker := range blockers {
		choices[i] = append(choices[i], nil)
		var legal []uuid.UUID
		for _, attacker := range attackers {
			if mage.HasLandwalkEvasion(attacker, blocker.ControllerID(), g) {
				continue
			}
			if mage.CanBlock(blocker, attacker, g) {
				legal = append(legal, attacker.ID())
				choices[i] = append(choices[i], []uuid.UUID{attacker.ID()})
			}
		}
		maxBlocks := 1
		if blocker.HasKeyword(core.CanBlockAny) {
			maxBlocks = min(len(legal), 3)
		} else if blocker.HasKeyword(core.CanBlockAdditional) {
			maxBlocks = min(len(legal), 2)
		}
		if maxBlocks >= 2 {
			for a := 0; a < len(legal); a++ {
				for b := a + 1; b < len(legal); b++ {
					choices[i] = append(choices[i], []uuid.UUID{legal[a], legal[b]})
				}
			}
		}
		if maxBlocks >= 3 {
			for a := 0; a < len(legal); a++ {
				for b := a + 1; b < len(legal); b++ {
					for c := b + 1; c < len(legal); c++ {
						choices[i] = append(choices[i], []uuid.UUID{legal[a], legal[b], legal[c]})
					}
				}
			}
		}
		total *= len(choices[i])
		if total > maxExhaustiveBlockPlans {
			return nil, false
		}
	}

	var out [][]mage.BlockAssignment
	var cur []mage.BlockAssignment
	var rec func(int)
	rec = func(i int) {
		if i == len(blockers) {
			out = append(out, append([]mage.BlockAssignment(nil), cur...))
			return
		}
		blocker := blockers[i]
		for _, atkIDs := range choices[i] {
			if len(atkIDs) == 0 {
				rec(i + 1)
				continue
			}
			for _, atkID := range atkIDs {
				cur = append(cur, mage.BlockAssignment{BlockerID: blocker.ID(), AttackerID: atkID})
			}
			rec(i + 1)
			cur = cur[:len(cur)-len(atkIDs)]
		}
	}
	rec(0)
	return out, true
}

func gangKills(g *mage.Game, atk *mage.Permanent, blockers []*mage.Permanent) bool {
	if atk.HasKeyword(core.FirstStrike) && !hasFirstStrikeDamage(blockers) {
		survivingPower := 0
		for _, blocker := range blockers {
			if survivesAgainst(g, blocker, atk) {
				survivingPower += effectiveDamage(g, blocker)
			}
		}
		return survivingPower >= atk.CurrentToughness(g)-atk.Damage
	}
	total := 0
	for _, blocker := range blockers {
		total += effectiveDamage(g, blocker)
		if blocker.HasKeyword(core.Deathtouch) && effectiveDamage(g, blocker) > 0 {
			return true
		}
	}
	return total >= atk.CurrentToughness(g)-atk.Damage
}

func hasFirstStrikeDamage(blockers []*mage.Permanent) bool {
	for _, blocker := range blockers {
		if blocker.HasKeyword(core.FirstStrike) || blocker.HasKeyword(core.DoubleStrike) {
			return true
		}
	}
	return false
}

func estimatedDyingBlockerValue(g *mage.Game, atk *mage.Permanent, blockers []*mage.Permanent) int {
	ordered := append([]*mage.Permanent(nil), blockers...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return blockerDamageOrderLess(g, atk, ordered[i], ordered[j])
	})
	remaining := effectiveDamage(g, atk)
	total := 0
	for _, blocker := range ordered {
		if blocker.HasKeyword(core.Indestructible) {
			continue
		}
		lethal := lethalDamageTo(g, atk, blocker)
		if remaining >= lethal {
			total += creatureValue(g, blocker)
			remaining -= lethal
		}
	}
	return total
}

func attackerUrgency(g *mage.Game, atk *mage.Permanent) int {
	urgency := effectiveDamage(g, atk) * 10
	if atk.HasKeyword(core.Trample) || atk.HasKeyword(core.Lifelink) || atk.HasKeyword(core.DoubleStrike) {
		urgency += 20
	}
	if atk.HasKeyword(core.Deathtouch) || atk.HasKeyword(core.Menace) {
		urgency += 10
	}
	urgency += creatureValue(g, atk) / 4
	return urgency
}

func blockPlanScore(g *mage.Game, defenderID uuid.UUID, plan []mage.BlockAssignment) int {
	blockedByAttacker := make(map[uuid.UUID][]uuid.UUID)
	for _, ba := range plan {
		blockedByAttacker[ba.AttackerID] = append(blockedByAttacker[ba.AttackerID], ba.BlockerID)
	}
	me := g.GetPlayer(defenderID)
	life := 20
	if me != nil {
		life = me.Life()
	}
	score := 0
	incoming := 0
	for _, atk := range combatAttackersForDefender(g, defenderID) {
		blockers := blockedByAttacker[atk.ID()]
		if len(blockers) == 0 {
			incoming += effectiveDamage(g, atk)
			continue
		}
		if atk.HasKeyword(core.Trample) {
			overflow := effectiveDamage(g, atk)
			for _, blockerID := range blockers {
				if blocker := g.FindPermanent(blockerID); blocker != nil {
					overflow -= lethalDamageTo(g, atk, blocker)
				}
			}
			incoming += max(overflow, 0)
		}
		if gangKills(g, atk, permanentsByID(g, blockers)) {
			score += creatureValue(g, atk)
		}
		score -= estimatedDyingBlockerValue(g, atk, permanentsByID(g, blockers))
	}
	prevented := max(totalIncomingDamage(g, defenderID)-incoming, 0)
	lifeWeight := 3
	if incoming >= life {
		lifeWeight = 100
	} else if life <= 8 {
		lifeWeight = 15
	}
	score += prevented * lifeWeight
	score -= incoming * 2
	return score
}

func totalIncomingDamage(g *mage.Game, defenderID uuid.UUID) int {
	total := 0
	for _, atk := range combatAttackersForDefender(g, defenderID) {
		total += effectiveDamage(g, atk)
	}
	return total
}

func permanentsByID(g *mage.Game, ids []uuid.UUID) []*mage.Permanent {
	out := make([]*mage.Permanent, 0, len(ids))
	for _, id := range ids {
		if perm := g.FindPermanent(id); perm != nil {
			out = append(out, perm)
		}
	}
	return out
}

func sortBlockPlans(g *mage.Game, defenderID uuid.UUID, plans [][]mage.BlockAssignment) {
	type scoredPlan struct {
		plan  []mage.BlockAssignment
		score int
	}
	scored := make([]scoredPlan, len(plans))
	for i, plan := range plans {
		scored[i] = scoredPlan{plan: plan, score: blockPlanScore(g, defenderID, plan)}
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return blockerSubsetLess(scored[i].plan, scored[j].plan)
	})
	for i := range scored {
		plans[i] = scored[i].plan
	}
}

func dedupeBlockPlans(plans [][]mage.BlockAssignment) [][]mage.BlockAssignment {
	seen := make(map[string]bool, len(plans))
	out := make([][]mage.BlockAssignment, 0, len(plans))
	for _, plan := range plans {
		cp := append([]mage.BlockAssignment(nil), plan...)
		sort.Slice(cp, func(i, j int) bool {
			if cp[i].AttackerID.String() != cp[j].AttackerID.String() {
				return cp[i].AttackerID.String() < cp[j].AttackerID.String()
			}
			return cp[i].BlockerID.String() < cp[j].BlockerID.String()
		})
		key := blocksKey(cp)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, cp)
	}
	return out
}

func repairBlockPlans(g *mage.Game, plans [][]mage.BlockAssignment) [][]mage.BlockAssignment {
	out := make([][]mage.BlockAssignment, 0, len(plans))
	for _, plan := range plans {
		counts := make(map[uuid.UUID]int)
		for _, block := range plan {
			counts[block.AttackerID]++
		}
		repaired := plan[:0]
		for _, block := range plan {
			atk := g.FindPermanent(block.AttackerID)
			if atk == nil {
				continue
			}
			minBlockers := 1
			if atk.HasKeyword(core.Menace) {
				minBlockers = 2
			}
			if counts[block.AttackerID] < minBlockers {
				continue
			}
			repaired = append(repaired, block)
		}
		out = append(out, append([]mage.BlockAssignment(nil), repaired...))
	}
	return out
}

func idsKey(ids []uuid.UUID) string {
	if len(ids) == 0 {
		return "-"
	}
	var key strings.Builder
	for _, id := range ids {
		key.WriteString(id.String() + ";")
	}
	return key.String()
}

func blocksKey(blocks []mage.BlockAssignment) string {
	if len(blocks) == 0 {
		return "-"
	}
	var key strings.Builder
	for _, block := range blocks {
		key.WriteString(block.BlockerID.String() + ">" + block.AttackerID.String() + ";")
	}
	return key.String()
}
