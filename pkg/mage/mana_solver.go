package mage

import (
	"container/heap"
	"fmt"

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

type manaSearchQueue []*manaSearchNode

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
	for sourceIndex, source := range in.Sources {
		productionChoices[sourceIndex] = make([][]manaProductionChoice, len(source.Abilities))
		choicesComputed[sourceIndex] = make([]bool, len(source.Abilities))
	}
	seq := 0

	for queue.Len() > 0 {
		node := heap.Pop(&queue).(*manaSearchNode)
		key := manaSearchStateKey(node.pool, node.used, in.SpellContext)
		if known, ok := bestCost[key]; ok && node.cost > known {
			continue
		}
		if node.pool.CanPay(in.Cost, in.SpellContext) {
			return reconstructManaPlan(node), true
		}

		for sourceIndex, source := range in.Sources {
			if manaSourceSetContains(node.used, sourceIndex) {
				continue
			}
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
			pool.pool = append(pool.pool, Mana{Color: production.Color})
		}
	}
	for _, color := range bonusColors {
		pool.pool = append(pool.pool, Mana{Color: color})
	}
}

func addConcreteMana(pool *ManaPool, productions []ManaProduction, bonusColors []Color) {
	for _, production := range productions {
		pool.Add(production.Color, normalizedManaAmount(production.Amount))
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
	states := [][AnyColor + 1]int{{}}
	for _, production := range productions {
		nextStates := make([][AnyColor + 1]int, 0, len(states))
		seen := make(map[[AnyColor + 1]int]bool, len(states))
		appendState := func(candidate [AnyColor + 1]int) {
			signature := boundedManaSignature(candidate, caps)
			if seen[signature] {
				return
			}
			seen[signature] = true
			nextStates = append(nextStates, candidate)
		}
		amount := normalizedManaAmount(production.Amount)
		for _, current := range states {
			if production.Color != AnyColor {
				current[production.Color] += amount
				appendState(current)
				continue
			}
			if !production.AnyCombination {
				for color := White; color <= Green; color++ {
					next := current
					next[color] += amount
					appendState(next)
				}
				continue
			}
			remainingCaps := caps
			for _, color := range manaPoolColors {
				remainingCaps[color] = max(remainingCaps[color]-current[color], 0)
			}
			for _, allocation := range boundedManaAllocations(amount, []Color{White, Blue, Black, Red, Green}, remainingCaps) {
				next := current
				for color := White; color <= Green; color++ {
					next[color] += allocation[color]
				}
				appendState(next)
			}
		}
		states = nextStates
	}

	result := make([][]ManaProduction, 0, len(states))
	for _, choice := range states {
		var concrete []ManaProduction
		for _, color := range manaPoolColors {
			if choice[color] > 0 {
				concrete = append(concrete, ManaProduction{Color: color, Amount: choice[color]})
			}
		}
		result = append(result, concrete)
	}
	return result
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
