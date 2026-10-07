package mage

import (
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// AddAllCreaturesMustBlock adds one independent requirement for each matching blocker.
func (em *EffectManager) AddAllCreaturesMustBlock(attackerID uuid.UUID, filter PermanentFilter) {
	if em.allCreaturesMustBlockRules == nil {
		em.allCreaturesMustBlockRules = make(map[uuid.UUID][]PermanentFilter)
	}
	em.allCreaturesMustBlockRules[attackerID] = append(em.allCreaturesMustBlockRules[attackerID], filter)
}

type blockingChoice struct {
	assignment BlockAssignment
	groups     []int
	direct     int
	weight     int
}

type blockingRow struct {
	choices   []blockingChoice
	preferred []int
	capacity  int
}

type blockingPlan struct {
	groups       []*CombatGroup
	minimum      []int
	single       []int
	counts       []int
	directCounts []int
	rows         []blockingRow
	suffixWeight []int
}

func blockingAttributeCount(p *Permanent, attr Attr) int {
	if p == nil || !p.HasAttr(attr) {
		return 0
	}
	base := p.baseAttrs[attr]
	if p.printedAbilitiesSuppressed {
		base -= min(base, p.printedAbilityAttrs[attr])
	}
	return max(0, int(base)+int(p.grantedAttrs[attr]))
}

func blockCapacity(p *Permanent, attackers int) int {
	if p.HasKeyword(CanBlockAny) {
		return attackers
	}
	if p.HasKeyword(CanBlockAdditional) {
		return min(2, attackers)
	}
	return min(1, attackers)
}

func (g *Game) liveCombatAttacker(group *CombatGroup) *Permanent {
	p := g.FindPermanent(group.AttackerID)
	if p == nil || p.PhasedOut || !p.HasType(TypeCreature) || !g.combat.IsAttacking(p.ID()) {
		return nil
	}
	incarnation := g.combat.AttackerIncarnations[p.ID()]
	if incarnation != uuid.Nil && incarnation != p.incarnationID {
		return nil
	}
	return p
}

func (g *Game) combatDefender() uuid.UUID {
	for _, group := range g.combat.Groups {
		if g.GetPlayer(group.DefenderID) != nil {
			return group.DefenderID
		}
		if permanent := g.FindPermanent(group.DefenderID); permanent != nil {
			return permanent.ControllerID()
		}
		if attacker := g.liveCombatAttacker(group); attacker != nil {
			for _, player := range g.players {
				if player.PlayerID() != attacker.ControllerID() {
					return player.PlayerID()
				}
			}
		}
	}
	return g.NonActivePlayerObj().PlayerID()
}

func (g *Game) newBlockingPlan(defender uuid.UUID, proposed []BlockAssignment) *blockingPlan {
	plan := &blockingPlan{groups: g.combat.Groups}
	for _, group := range plan.groups {
		p := g.liveCombatAttacker(group)
		minimum := max(1, g.effects.MinBlockers(group.AttackerID))
		if p != nil && p.HasKeyword(Menace) {
			minimum = max(minimum, 2)
		}
		plan.minimum = append(plan.minimum, minimum)
		plan.single = append(plan.single, blockingAttributeCount(p, AttrMustBeBlockedIfAble))
		plan.counts = append(plan.counts, len(group.BlockerIDs))
		plan.directCounts = append(plan.directCounts, 0)
	}
	for _, blocker := range g.AllBattlefield() {
		if blocker.PhasedOut || !blocker.HasType(TypeCreature) || blocker.ControllerID() != defender || !blocker.CanDeclareAsBlocker(g) {
			continue
		}
		row := blockingRow{}
		usedBands := make(map[uuid.UUID]int)
		existing := make(map[uuid.UUID]bool)
		for _, group := range plan.groups {
			if slices.Contains(group.BlockerIDs, blocker.ID()) {
				key := group.AttackerID
				if band := g.combat.Bands[key]; len(band) > 0 {
					key = band[0]
				}
				existing[key] = true
			}
		}
		for i, group := range plan.groups {
			attacker := g.liveCombatAttacker(group)
			if attacker == nil || (group.DefenderID != uuid.Nil && group.DefenderID != defender && (g.FindPermanent(group.DefenderID) == nil || g.FindPermanent(group.DefenderID).ControllerID() != defender)) || !CanBlock(blocker, attacker, g) || HasLandwalkEvasion(attacker, defender, g) {
				continue
			}
			key := attacker.ID()
			if band := g.combat.Bands[key]; len(band) > 0 {
				key = band[0]
			}
			if existing[key] {
				continue
			}
			prior, seenBand := usedBands[key]
			if seenBand && plan.minimum[row.choices[prior].direct] <= plan.minimum[i] {
				continue
			}
			choice := blockingChoice{assignment: BlockAssignment{blocker.ID(), attacker.ID()}, direct: i}
			for j, member := range plan.groups {
				if j != i && !g.combat.IsBandedWith(member.AttackerID, attacker.ID()) {
					continue
				}
				choice.groups = append(choice.groups, j)
				if p := g.liveCombatAttacker(member); p != nil {
					choice.weight += g.effects.blockEachAttackerRules[blocker.ID()]
					choice.weight += blockingAttributeCount(p, MustBeBlocked)
					for _, filter := range g.effects.allCreaturesMustBlockRules[p.ID()] {
						if filter.Match(blocker, g) {
							choice.weight++
						}
					}
				}
			}
			if seenBand {
				row.choices[prior] = choice
			} else {
				usedBands[key] = len(row.choices)
				row.choices = append(row.choices, choice)
			}
		}
		limit := blockCapacity(blocker, len(plan.groups))
		if g.effects.blockAnyNumberRules[blocker.ID()] {
			limit = len(plan.groups)
		}
		capacity := max(0, limit-len(existing))
		if capacity == 0 || len(row.choices) == 0 {
			continue
		}
		selected := []int{}
		for _, assignment := range proposed {
			for i, choice := range row.choices {
				matches := assignment == choice.assignment
				if assignment.BlockerID == blocker.ID() && g.combat.IsBandedWith(assignment.AttackerID, choice.assignment.AttackerID) {
					attacker := g.FindPermanent(assignment.AttackerID)
					matches = attacker != nil && CanBlock(blocker, attacker, g) && !HasLandwalkEvasion(attacker, defender, g)
				}
				if matches && !slices.Contains(selected, i) && len(selected) < capacity {

					selected = append(selected, i)
				}
			}
		}
		row.preferred = slices.Clone(selected)
		row.capacity = capacity

		plan.rows = append(plan.rows, row)
	}
	plan.suffixWeight = make([]int, len(plan.rows)+1)
	for i := len(plan.rows) - 1; i >= 0; i-- {
		weights := make([]int, len(plan.rows[i].choices))
		for j, choice := range plan.rows[i].choices {
			weights[j] = choice.weight
		}
		slices.SortFunc(weights, func(a, b int) int { return b - a })
		upper := 0
		for _, weight := range weights[:min(plan.rows[i].capacity, len(weights))] {
			upper += weight
		}

		plan.suffixWeight[i] = plan.suffixWeight[i+1] + upper
	}
	return plan
}

func (p *blockingPlan) score(weight int) (int, bool) {
	for i, count := range p.counts {
		if p.directCounts[i] > 0 && count < p.minimum[i] {
			return 0, false
		}
		if count > 0 {
			weight += p.single[i]
		}
	}
	return weight, true
}

func (p *blockingPlan) preferredLegal() (assignments []BlockAssignment, requirements int) {
	selected := make([][]int, len(p.rows))
	for i, row := range p.rows {
		selected[i] = slices.Clone(row.preferred)
		for _, index := range selected[i] {
			p.directCounts[row.choices[index].direct]++
			for _, group := range row.choices[index].groups {
				p.counts[group]++
			}
		}
	}
	for {
		changed := false
		for i, row := range p.rows {
			kept := selected[i][:0]
			for _, index := range selected[i] {
				choice := row.choices[index]
				invalid := p.counts[choice.direct] < p.minimum[choice.direct]
				if invalid {
					p.directCounts[choice.direct]--
					for _, group := range choice.groups {
						p.counts[group]--
					}
					changed = true
				} else {
					kept = append(kept, index)
				}
			}
			selected[i] = kept
		}
		if !changed {
			break
		}
	}
	var result []BlockAssignment
	weight := 0
	for i, row := range p.rows {
		for _, index := range selected[i] {
			result = append(result, row.choices[index].assignment)
			weight += row.choices[index].weight
		}
	}
	score, _ := p.score(weight)
	for i, row := range p.rows {
		for _, index := range selected[i] {
			p.directCounts[row.choices[index].direct]--
			for _, group := range row.choices[index].groups {
				p.counts[group]--
			}
		}
	}
	return result, score
}

func (p *blockingPlan) upperBound(rowIndex, weight int) int {
	covered := 0
	reachable := make([]bool, len(p.groups))
	slots := 0
	for i, single := range p.single {
		if single > 0 && p.counts[i] > 0 {
			covered += single
		}
	}
	for _, row := range p.rows[rowIndex:] {
		gains := make([]int, len(row.choices))
		for i, choice := range row.choices {
			for _, group := range choice.groups {
				if p.single[group] > 0 && p.counts[group] == 0 {
					reachable[group] = true
					gains[i] += p.single[group]
				}
			}
		}
		slices.SortFunc(gains, func(a, b int) int { return b - a })
		for _, gain := range gains[:min(row.capacity, len(gains))] {
			slots += gain
		}
	}
	possible := 0
	for group, value := range reachable {
		if value {
			possible += p.single[group]
		}
	}
	return weight + p.suffixWeight[rowIndex] + covered + min(slots, possible)
}

func (p *blockingPlan) choose() []BlockAssignment {
	best, bestScore := p.preferredLegal()
	var current []BlockAssignment
	var search func(int, int)
	search = func(rowIndex, weight int) {
		if p.upperBound(rowIndex, weight) <= bestScore {
			return
		}
		if rowIndex == len(p.rows) {
			if score, ok := p.score(weight); ok && score > bestScore {
				bestScore = score
				best = slices.Clone(current)
			}
			return
		}
		row := p.rows[rowIndex]
		visit := func(option []int) bool {
			if p.upperBound(rowIndex, weight) <= bestScore {
				return false
			}
			start := len(current)
			added := 0
			for _, index := range option {
				choice := row.choices[index]
				current = append(current, choice.assignment)
				added += choice.weight
				p.directCounts[choice.direct]++
				for _, group := range choice.groups {
					p.counts[group]++
				}
			}
			search(rowIndex+1, weight+added)
			for _, index := range option {
				p.directCounts[row.choices[index].direct]--
				for _, group := range row.choices[index].groups {
					p.counts[group]--
				}
			}
			current = current[:start]
			return true
		}
		if !visit(row.preferred) {
			return
		}
		order := make([]int, len(row.choices))
		for i := range order {
			order[i] = i
		}
		choiceScore := func(choice blockingChoice) int {
			score := choice.weight
			for _, group := range choice.groups {
				if p.single[group] > 0 && p.counts[group] == 0 {
					score += p.single[group]
				}
			}
			return score
		}
		slices.SortStableFunc(order, func(i, j int) int { return choiceScore(row.choices[j]) - choiceScore(row.choices[i]) })
		if !visit(order[:min(row.capacity, len(order))]) {
			return
		}
		var subsets func(int, []int) bool
		subsets = func(start int, members []int) bool {
			if !visit(members) {
				return false
			}
			if len(members) == row.capacity {
				return true
			}
			for i := start; i < len(order); i++ {
				if !subsets(i+1, append(members, order[i])) {
					return false
				}
			}
			return true
		}
		subsets(0, nil)
	}
	search(0, 0)
	return best
}

// CompleteBlockAssignments keeps a legal proposal when it satisfies the maximum requirements.
// Otherwise it supplies a legal maximum. Existing combat pairs stay fixed.
// The query does not change combat, history, events, or player choices.
func (g *Game) CompleteBlockAssignments(defender uuid.UUID, proposed []BlockAssignment) []BlockAssignment {
	return g.newBlockingPlan(defender, proposed).choose()
}

func (g *Game) commitBlockAssignments(defender uuid.UUID, proposed []BlockAssignment) {
	before := g.blockerPairsBeforeDeclaration()
	assignments := g.CompleteBlockAssignments(defender, proposed)
	for _, assignment := range assignments {
		blocker := g.FindPermanent(assignment.BlockerID)
		g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, assignment.AttackerID)
		g.trackers.Turn.RecordBlocked(blocker.ID(), assignment.AttackerID)
	}
	for _, group := range g.combat.Groups {
		attacker := g.liveCombatAttacker(group)
		for i, id := range group.BlockerIDs {
			if i >= len(group.BlockerIncarnations) {
				continue
			}
			pair := combatBlockPair{group.AttackerID, id, group.BlockerIncarnations[i]}
			blocker := g.FindPermanent(id)
			if !before[pair] && blocker != nil && blocker.incarnationID == pair.object {
				g.recordBlockHistory(blocker, attacker)
			}
		}
	}
}
