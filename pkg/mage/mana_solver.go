package mage

import (
	"container/heap"
	"fmt"
	"strings"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ManaSolverInputs is the complete state needed to plan a mana payment. Each
// source contains the exact abilities available on one permanent; the solver
// chooses an ability, its mana-production choices, and an activation order.
type ManaSolverInputs struct {
	Pool         *ManaPool
	Cost         ManaCost
	Sources      []manaSourceInfo
	Scores       []int
	Conversions  map[Color]Color
	SpellContext *SpellPaymentContext
}

// ManaTap describes one exact mana-ability activation selected by the solver.
// Productions and BonusColors are concrete, so AnyColor and AnyCombination
// choices are preserved through execution.
type ManaTap struct {
	PermanentID  uuid.UUID
	AbilityIndex int
	Productions  []ManaProduction
	BonusColors  []Color
}

// ManaSolution is an ordered list of mana-ability activations.
type ManaSolution struct {
	SourcesToTap []ManaTap
}

type manaProductionChoice struct {
	Productions []ManaProduction
	BonusColors []Color
}

type manaSearchNode struct {
	pool     *ManaPool
	used     string
	parent   *manaSearchNode
	action   ManaTap
	score    int
	depth    int
	cost     int
	priority int
	seq      int
}

type manaSearchKey struct {
	total         [AnyColor + 1]int
	abilityUsable [AnyColor + 1]int
	spellUsable   [AnyColor + 1]int
	used          string
}

type manaChoiceCaps [AnyColor + 1]int

type concreteManaProductionState struct {
	amounts     [AnyColor + 1]int
	productions []ManaProduction
}

type concreteManaProductionStateKey struct {
	amounts    [AnyColor + 1]int
	restricted string
}

type manaSearchQueue []*manaSearchNode

type manaSearchStats struct {
	ExpandedNodes int
}

func (q manaSearchQueue) Len() int { return len(q) }
func (q manaSearchQueue) Less(i, j int) bool {
	if q[i].priority != q[j].priority {
		return q[i].priority < q[j].priority
	}
	if q[i].cost != q[j].cost {
		return q[i].cost < q[j].cost
	}
	return q[i].seq < q[j].seq
}
func (q manaSearchQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *manaSearchQueue) Push(x any)   { *q = append(*q, x.(*manaSearchNode)) }
func (q *manaSearchQueue) Pop() any {
	old := *q
	n := len(old)
	x := old[n-1]
	*q = old[:n-1]
	return x
}

// SolveMana finds the best executable activation plan for Cost. Search is
// ordered by preservation score, then number of taps. Because complete plans
// are searched, preservation preferences cannot make an affordable cost fail.
func SolveMana(in ManaSolverInputs) (*ManaSolution, error) {
	plan, ok := searchMana(in)
	if !ok {
		return nil, fmt.Errorf("cannot pay mana cost %s", in.Cost)
	}
	return &ManaSolution{SourcesToTap: plan}, nil
}

// CanSolveMana reports whether the same unified search used by SolveMana can
// produce a legal payment.
func CanSolveMana(in ManaSolverInputs) bool {
	_, ok := searchMana(in)
	return ok
}

func searchMana(in ManaSolverInputs) ([]ManaTap, bool) {
	return searchManaWithStats(in, nil)
}

func searchManaWithStats(in ManaSolverInputs, stats *manaSearchStats) ([]ManaTap, bool) {
	rootPool := NewManaPool()
	if in.Pool != nil {
		rootPool.RestorePool(in.Pool.SnapshotPool())
	}
	rootPool.ManaConversions = in.Conversions

	weight := len(in.Sources) + 1
	root := &manaSearchNode{
		pool: rootPool,
		used: emptyManaSourceSet(len(in.Sources)),
	}
	root.priority = manaSearchLowerBound(root, in)
	queue := manaSearchQueue{root}
	heap.Init(&queue)
	bestCost := map[manaSearchKey]int{manaSearchStateKey(rootPool, root.used, in.SpellContext): 0}
	choiceCaps := manaChoiceCapsForInputs(in)
	productionChoices := make([][][]manaProductionChoice, len(in.Sources))
	choicesComputed := make([][]bool, len(in.Sources))
	sourceGroups := equivalentManaSourceGroups(in)
	for sourceIndex, source := range in.Sources {
		productionChoices[sourceIndex] = make([][]manaProductionChoice, len(source.Abilities))
		choicesComputed[sourceIndex] = make([]bool, len(source.Abilities))
	}
	seq := 0
	groupGeneration := 0
	seenGroups := make([]int, len(in.Sources))

	for queue.Len() > 0 {
		node := heap.Pop(&queue).(*manaSearchNode)
		key := manaSearchStateKey(node.pool, node.used, in.SpellContext)
		if known, ok := bestCost[key]; ok && node.cost > known {
			continue
		}
		if stats != nil {
			stats.ExpandedNodes++
		}
		if node.pool.CanPay(in.Cost, in.SpellContext) {
			return reconstructManaPlan(node), true
		}
		if !manaSearchCanStillPay(node, in) {
			continue
		}

		groupGeneration++
		for sourceIndex, source := range in.Sources {
			if manaSourceSetContains(node.used, sourceIndex) {
				continue
			}
			group := sourceGroups[sourceIndex]
			if seenGroups[group] == groupGeneration {
				continue
			}
			seenGroups[group] = groupGeneration
			for abilityIndex, ability := range source.Abilities {
				if !node.pool.CanPay(ability.ManaCost, nil) {
					continue
				}
				if !choicesComputed[sourceIndex][abilityIndex] {
					productionChoices[sourceIndex][abilityIndex] = manaProductionChoicesForDemand(ability.Productions, source.Bonuses, choiceCaps)
					choicesComputed[sourceIndex][abilityIndex] = true
				}
				for _, choice := range productionChoices[sourceIndex][abilityIndex] {
					nextPool := cloneManaPoolForSearch(node.pool, in.Conversions)
					if !payManaForSearch(nextPool, ability.ManaCost) {
						continue
					}
					addConcreteManaForSearch(nextPool, choice.Productions, choice.BonusColors)

					nextUsed := manaSourceSetWith(node.used, sourceIndex)
					nextAction := ManaTap{
						PermanentID:  source.PermanentID,
						AbilityIndex: ability.AbilityIndex,
						Productions:  choice.Productions,
						BonusColors:  choice.BonusColors,
					}
					nextScore := node.score + max(scoreAt(in.Scores, sourceIndex), 0)
					nextDepth := node.depth + 1
					nextCost := nextScore*weight + nextDepth
					nextKey := manaSearchStateKey(nextPool, nextUsed, in.SpellContext)
					if known, ok := bestCost[nextKey]; ok && known <= nextCost {
						continue
					}
					bestCost[nextKey] = nextCost
					seq++
					next := &manaSearchNode{
						pool:   nextPool,
						used:   nextUsed,
						parent: node,
						action: nextAction,
						score:  nextScore,
						depth:  nextDepth,
						cost:   nextCost,
						seq:    seq,
					}
					next.priority = nextCost + manaSearchLowerBound(next, in)
					heap.Push(&queue, next)
				}
			}
		}
	}
	return nil, false
}

func manaSearchCanStillPay(node *manaSearchNode, in ManaSolverInputs) bool {
	maxUsableMana := usableManaCount(node.pool, in.SpellContext)
	for sourceIndex, source := range in.Sources {
		if manaSourceSetContains(node.used, sourceIndex) {
			continue
		}
		bestAbility := 0
		for _, ability := range source.Abilities {
			usable := 0
			for _, production := range ability.Productions {
				if productionRestrictionUsable(production.Restriction, in.SpellContext) {
					usable += normalizedManaAmount(production.Amount)
				}
			}
			bestAbility = max(bestAbility, usable+len(source.Bonuses))
		}
		maxUsableMana += bestAbility
	}
	if maxUsableMana < manaCostUnits(in.Cost) {
		return false
	}

	var available [AnyColor + 1]int
	for _, mana := range node.pool.pool {
		if manaUsable(mana, in.SpellContext) {
			available[mana.Color]++
		}
	}
	for sourceIndex, source := range in.Sources {
		if manaSourceSetContains(node.used, sourceIndex) {
			continue
		}
		for _, ability := range source.Abilities {
			for _, production := range ability.Productions {
				if !productionRestrictionUsable(production.Restriction, in.SpellContext) {
					continue
				}
				amount := normalizedManaAmount(production.Amount)
				if production.Color != AnyColor {
					available[production.Color] += amount
					continue
				}
				for color := White; color <= Green; color++ {
					available[color] += amount
				}
			}
		}
		for _, bonus := range source.Bonuses {
			if bonus == MatchProduced {
				for color := White; color <= Green; color++ {
					available[color]++
				}
				continue
			}
			available[Color(bonus)]++
		}
	}
	for _, requirement := range [...]struct {
		color Color
		count int
	}{
		{White, in.Cost.White},
		{Blue, in.Cost.Blue},
		{Black, in.Cost.Black},
		{Red, in.Cost.Red},
		{Green, in.Cost.Green},
	} {
		if relaxedManaAvailableForColor(available, requirement.color, in.Conversions) < requirement.count {
			return false
		}
	}
	for _, hybrid := range in.Cost.Hybrid {
		if relaxedManaAvailableForColor(available, hybrid.A, in.Conversions)+
			relaxedManaAvailableForColor(available, hybrid.B, in.Conversions) == 0 {
			return false
		}
	}
	return true
}

func productionRestrictionUsable(restriction ManaRestriction, spellCtx *SpellPaymentContext) bool {
	if restriction == nil {
		return true
	}
	return spellCtx != nil && restriction.IsSatisfiedBy(*spellCtx)
}

func relaxedManaAvailableForColor(available [AnyColor + 1]int, required Color, conversions map[Color]Color) int {
	total := available[required]
	for _, from := range manaPoolColors {
		if from != required && conversions[from] == required {
			total += available[from]
		}
	}
	return total
}

func equivalentManaSourceGroups(in ManaSolverInputs) []int {
	groups := make([]int, len(in.Sources))
	groupCount := 0
	for i := range in.Sources {
		groups[i] = -1
		for previous := range i {
			if manaSourcesEquivalentForSearch(in.Sources[i], in.Sources[previous], scoreAt(in.Scores, i), scoreAt(in.Scores, previous), in.SpellContext) {
				groups[i] = groups[previous]
				break
			}
		}
		if groups[i] < 0 {
			groups[i] = groupCount
			groupCount++
		}
	}
	return groups
}

func manaSourcesEquivalentForSearch(a, b manaSourceInfo, aScore, bScore int, spellCtx *SpellPaymentContext) bool {
	if max(aScore, 0) != max(bScore, 0) || len(a.Abilities) != len(b.Abilities) || len(a.Bonuses) != len(b.Bonuses) {
		return false
	}
	for i := range a.Bonuses {
		if a.Bonuses[i] != b.Bonuses[i] {
			return false
		}
	}
	for i := range a.Abilities {
		if !a.Abilities[i].Interchangeable || !b.Abilities[i].Interchangeable {
			return false
		}
		if !manaCostsEqual(a.Abilities[i].ManaCost, b.Abilities[i].ManaCost) || len(a.Abilities[i].Productions) != len(b.Abilities[i].Productions) {
			return false
		}
		for j := range a.Abilities[i].Productions {
			aProduction := a.Abilities[i].Productions[j]
			bProduction := b.Abilities[i].Productions[j]
			if aProduction.Color != bProduction.Color ||
				aProduction.Amount != bProduction.Amount ||
				aProduction.AnyCombination != bProduction.AnyCombination ||
				!manaRestrictionsEquivalentForSearch(aProduction.Restriction, bProduction.Restriction, spellCtx) {
				return false
			}
		}
	}
	return true
}

func manaCostsEqual(a, b ManaCost) bool {
	if a.Generic != b.Generic || a.White != b.White || a.Blue != b.Blue || a.Black != b.Black ||
		a.Red != b.Red || a.Green != b.Green || a.HasX != b.HasX || a.XCount != b.XCount || len(a.Hybrid) != len(b.Hybrid) {
		return false
	}
	for i := range a.Hybrid {
		if a.Hybrid[i] != b.Hybrid[i] {
			return false
		}
	}
	return true
}

func manaRestrictionsEquivalentForSearch(a, b ManaRestriction, spellCtx *SpellPaymentContext) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if spellCtx == nil {
		return true
	}
	return a.IsSatisfiedBy(*spellCtx) == b.IsSatisfiedBy(*spellCtx)
}

func reconstructManaPlan(node *manaSearchNode) []ManaTap {
	plan := make([]ManaTap, node.depth)
	for i := node.depth - 1; i >= 0; i-- {
		plan[i] = node.action
		node = node.parent
	}
	return plan
}

func emptyManaSourceSet(sourceCount int) string {
	return string(make([]byte, (sourceCount+7)/8))
}

func manaSourceSetContains(used string, sourceIndex int) bool {
	return used[sourceIndex/8]&(1<<uint(sourceIndex%8)) != 0
}

func manaSourceSetWith(used string, sourceIndex int) string {
	next := []byte(used)
	next[sourceIndex/8] |= 1 << uint(sourceIndex%8)
	return string(next)
}

func cloneManaPoolForSearch(pool *ManaPool, conversions map[Color]Color) *ManaPool {
	return &ManaPool{
		pool:            append([]Mana(nil), pool.pool...),
		ManaConversions: conversions,
	}
}

func payManaForSearch(pool *ManaPool, cost ManaCost) bool {
	plan, ok := pool.paymentPlan(cost, nil)
	if !ok {
		return false
	}
	remaining := plan
	kept := pool.pool[:0]
	for _, mana := range pool.pool {
		if remaining[mana.Color] > 0 && manaUsable(mana, nil) {
			remaining[mana.Color]--
			continue
		}
		kept = append(kept, mana)
	}
	pool.pool = kept
	return true
}

func addConcreteManaForSearch(pool *ManaPool, productions []ManaProduction, bonusColors []Color) {
	for _, production := range productions {
		for range normalizedManaAmount(production.Amount) {
			pool.pool = append(pool.pool, Mana{Color: production.Color, Restriction: production.Restriction})
		}
	}
	for _, color := range bonusColors {
		pool.pool = append(pool.pool, Mana{Color: color})
	}
}

func addConcreteMana(pool *ManaPool, productions []ManaProduction, bonusColors []Color) {
	for _, production := range productions {
		amount := normalizedManaAmount(production.Amount)
		if production.Restriction != nil {
			pool.AddRestricted(production.Color, amount, production.Restriction)
		} else {
			pool.Add(production.Color, amount)
		}
	}
	for _, color := range bonusColors {
		pool.Add(color, 1)
	}
}

func normalizedManaAmount(amount int) int {
	if amount <= 0 {
		return 1
	}
	return amount
}

func manaSearchLowerBound(node *manaSearchNode, in ManaSolverInputs) int {
	need := manaCostUnits(in.Cost) - usableManaCount(node.pool, in.SpellContext)
	if need <= 0 {
		return 0
	}
	maxOutput := 0
	for i, source := range in.Sources {
		if manaSourceSetContains(node.used, i) {
			continue
		}
		for _, ability := range source.Abilities {
			output := productionsTotalAmount(ability.Productions) + len(source.Bonuses)
			maxOutput = max(maxOutput, output)
		}
	}
	if maxOutput <= 0 {
		return 0
	}
	return (need + maxOutput - 1) / maxOutput
}

func manaCostUnits(cost ManaCost) int {
	return cost.Generic + cost.White + cost.Blue + cost.Black + cost.Red + cost.Green + len(cost.Hybrid)
}

func usableManaCount(pool *ManaPool, spellCtx *SpellPaymentContext) int {
	total := 0
	for _, mana := range pool.pool {
		if manaUsable(mana, spellCtx) {
			total++
		}
	}
	return total
}

func manaSearchStateKey(pool *ManaPool, used string, spellCtx *SpellPaymentContext) manaSearchKey {
	key := manaSearchKey{used: used}
	for _, mana := range pool.pool {
		key.total[mana.Color]++
		if manaUsable(mana, nil) {
			key.abilityUsable[mana.Color]++
		}
		if manaUsable(mana, spellCtx) {
			key.spellUsable[mana.Color]++
		}
	}
	return key
}

func manaChoiceCapsForInputs(in ManaSolverInputs) manaChoiceCaps {
	caps := manaChoiceCapsForCost(in.Cost, in.Conversions)
	for _, source := range in.Sources {
		var sourceCaps manaChoiceCaps
		for _, ability := range source.Abilities {
			abilityCaps := manaChoiceCapsForCost(ability.ManaCost, in.Conversions)
			for _, color := range manaPoolColors {
				sourceCaps[color] = max(sourceCaps[color], abilityCaps[color])
			}
		}
		for _, color := range manaPoolColors {
			caps[color] += sourceCaps[color]
		}
	}
	return caps
}

func manaChoiceCapsForCost(cost ManaCost, conversions map[Color]Color) manaChoiceCaps {
	var caps manaChoiceCaps
	addManaChoiceCaps(&caps, cost, conversions)
	return caps
}

func addManaChoiceCaps(caps *manaChoiceCaps, cost ManaCost, conversions map[Color]Color) {
	for _, requirement := range [...]struct {
		color Color
		count int
	}{
		{White, cost.White},
		{Blue, cost.Blue},
		{Black, cost.Black},
		{Red, cost.Red},
		{Green, cost.Green},
	} {
		addManaRequirementCaps(caps, requirement.color, requirement.count, conversions)
	}
	for _, hybrid := range cost.Hybrid {
		var candidates [AnyColor + 1]bool
		markManaRequirementCandidates(&candidates, hybrid.A, conversions)
		markManaRequirementCandidates(&candidates, hybrid.B, conversions)
		for _, color := range manaPoolColors {
			if candidates[color] {
				caps[color]++
			}
		}
	}
}

func addManaRequirementCaps(caps *manaChoiceCaps, required Color, count int, conversions map[Color]Color) {
	if count <= 0 {
		return
	}
	var candidates [AnyColor + 1]bool
	markManaRequirementCandidates(&candidates, required, conversions)
	for _, color := range manaPoolColors {
		if candidates[color] {
			caps[color] += count
		}
	}
}

func markManaRequirementCandidates(candidates *[AnyColor + 1]bool, required Color, conversions map[Color]Color) {
	candidates[required] = true
	for _, from := range manaPoolColors {
		if conversions[from] == required {
			candidates[from] = true
		}
	}
}

func manaProductionChoicesForDemand(productions []ManaProduction, bonuses []ManaBonusColor, caps manaChoiceCaps) []manaProductionChoice {
	concrete := concreteManaProductionsForDemand(productions, caps)
	result := make([]manaProductionChoice, 0, len(concrete))
	for _, production := range concrete {
		producedColors := productionColors(production)
		remainingCaps := caps
		for _, produced := range production {
			remainingCaps[produced.Color] = max(remainingCaps[produced.Color]-produced.Amount, 0)
		}
		bonusChoices := concreteBonusChoicesForDemand(bonuses, producedColors, remainingCaps)
		if len(bonusChoices) == 0 {
			bonusChoices = [][]Color{{}}
		}
		for _, bonusColors := range bonusChoices {
			result = append(result, manaProductionChoice{
				Productions: production,
				BonusColors: bonusColors,
			})
		}
	}
	return result
}

func concreteManaProductionsForDemand(productions []ManaProduction, caps manaChoiceCaps) [][]ManaProduction {
	if len(productions) == 0 {
		return nil
	}
	states := []concreteManaProductionState{{}}
	for _, production := range productions {
		nextStates := make([]concreteManaProductionState, 0, len(states))
		seen := make(map[concreteManaProductionStateKey]bool, len(states))
		appendState := func(candidate concreteManaProductionState) {
			signature := concreteManaStateKey(candidate, caps)
			if seen[signature] {
				return
			}
			seen[signature] = true
			nextStates = append(nextStates, candidate)
		}
		amount := normalizedManaAmount(production.Amount)
		for _, current := range states {
			if production.Color != AnyColor {
				current.amounts[production.Color] += amount
				current.productions = appendConcreteProduction(current.productions, production.Color, amount, production.Restriction)
				appendState(current)
				continue
			}
			if !production.AnyCombination {
				for color := White; color <= Green; color++ {
					next := cloneConcreteManaProductionState(current)
					next.amounts[color] += amount
					next.productions = appendConcreteProduction(next.productions, color, amount, production.Restriction)
					appendState(next)
				}
				continue
			}
			remainingCaps := caps
			for _, color := range manaPoolColors {
				remainingCaps[color] = max(remainingCaps[color]-current.amounts[color], 0)
			}
			for _, allocation := range boundedManaAllocations(amount, []Color{White, Blue, Black, Red, Green}, remainingCaps) {
				next := cloneConcreteManaProductionState(current)
				for color := White; color <= Green; color++ {
					if allocation[color] == 0 {
						continue
					}
					next.amounts[color] += allocation[color]
					next.productions = appendConcreteProduction(next.productions, color, allocation[color], production.Restriction)
				}
				appendState(next)
			}
		}
		states = nextStates
	}

	result := make([][]ManaProduction, 0, len(states))
	for _, state := range states {
		result = append(result, state.productions)
	}
	return result
}

func cloneConcreteManaProductionState(state concreteManaProductionState) concreteManaProductionState {
	state.productions = append([]ManaProduction(nil), state.productions...)
	return state
}

func appendConcreteProduction(productions []ManaProduction, color Color, amount int, restriction ManaRestriction) []ManaProduction {
	return append(productions, ManaProduction{Color: color, Amount: amount, Restriction: restriction})
}

func concreteManaStateKey(state concreteManaProductionState, caps manaChoiceCaps) concreteManaProductionStateKey {
	return concreteManaProductionStateKey{
		amounts:    boundedManaSignature(state.amounts, caps),
		restricted: restrictedManaProductionSignature(state.productions, caps),
	}
}

func restrictedManaProductionSignature(productions []ManaProduction, caps manaChoiceCaps) string {
	type restrictionAmounts struct {
		key     string
		total   int
		amounts [AnyColor + 1]int
	}
	var restrictions []restrictionAmounts
	for _, production := range productions {
		if production.Restriction == nil {
			continue
		}
		key := manaRestrictionIdentity(production.Restriction)
		index := -1
		for i := range restrictions {
			if restrictions[i].key == key {
				index = i
				break
			}
		}
		if index < 0 {
			restrictions = append(restrictions, restrictionAmounts{key: key})
			index = len(restrictions) - 1
		}
		restrictions[index].total += production.Amount
		restrictions[index].amounts[production.Color] += production.Amount
	}
	var signature strings.Builder
	for _, restriction := range restrictions {
		fmt.Fprintf(&signature, "%s:%d:%v;", restriction.key, restriction.total, boundedManaSignature(restriction.amounts, caps))
	}
	return signature.String()
}

func manaRestrictionIdentity(restriction ManaRestriction) string {
	if restriction == nil {
		return ""
	}
	return fmt.Sprintf("%T:%s", restriction, restriction.Description())
}

func boundedManaSignature(amounts [AnyColor + 1]int, caps manaChoiceCaps) [AnyColor + 1]int {
	var signature [AnyColor + 1]int
	for _, color := range manaPoolColors {
		signature[color] = min(amounts[color], caps[color])
	}
	return signature
}

func boundedManaAllocations(amount int, colors []Color, caps manaChoiceCaps) [][AnyColor + 1]int {
	if amount < 0 || len(colors) == 0 {
		return nil
	}
	var result [][AnyColor + 1]int
	var distribute func(int, int, [AnyColor + 1]int)
	distribute = func(index, remaining int, allocation [AnyColor + 1]int) {
		if index == len(colors) {
			if remaining == 0 {
				result = append(result, allocation)
				return
			}
			for _, color := range colors {
				if allocation[color] == caps[color] {
					allocation[color] += remaining
					result = append(result, allocation)
					return
				}
			}
			return
		}
		color := colors[index]
		for n := 0; n <= min(remaining, caps[color]); n++ {
			next := allocation
			next[color] = n
			distribute(index+1, remaining-n, next)
		}
	}
	distribute(0, amount, [AnyColor + 1]int{})
	return result
}

func productionColors(productions []ManaProduction) []Color {
	var colors []Color
	for _, production := range productions {
		if production.Amount != 0 {
			colors = append(colors, production.Color)
		}
	}
	return colors
}

func concreteBonusChoicesForDemand(bonuses []ManaBonusColor, producedColors []Color, caps manaChoiceCaps) [][]Color {
	var fixed []Color
	matchCount := 0
	for _, bonus := range bonuses {
		if bonus != MatchProduced {
			color := Color(bonus)
			fixed = append(fixed, color)
			caps[color] = max(caps[color]-1, 0)
			continue
		}
		matchCount++
	}
	if matchCount == 0 || len(producedColors) == 0 {
		return [][]Color{fixed}
	}
	producedColors = distinctColors(producedColors)
	allocations := boundedManaAllocations(matchCount, producedColors, caps)
	choices := make([][]Color, 0, len(allocations))
	for _, allocation := range allocations {
		choice := append([]Color(nil), fixed...)
		for _, color := range producedColors {
			for range allocation[color] {
				choice = append(choice, color)
			}
		}
		choices = append(choices, choice)
	}
	return choices
}

func distinctColors(colors []Color) []Color {
	var seen [AnyColor + 1]bool
	result := make([]Color, 0, len(colors))
	for _, color := range colors {
		if !seen[color] {
			seen[color] = true
			result = append(result, color)
		}
	}
	return result
}

func scoreAt(scores []int, i int) int {
	if i < len(scores) {
		return scores[i]
	}
	return 0
}
