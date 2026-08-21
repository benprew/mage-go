package mage

import (
	"container/heap"
	"fmt"
	"strconv"
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
	Color        Color
	Productions  []ManaProduction
	BonusColors  []Color
}

// ManaSolution is an ordered list of mana-ability activations.
type ManaSolution struct {
	SourcesToTap []ManaTap
}

type manaProductionChoice struct {
	Color       Color
	Productions []ManaProduction
	BonusColors []Color
}

type manaSearchNode struct {
	pool     *ManaPool
	used     []bool
	path     []ManaTap
	score    int
	taps     int
	cost     int
	priority int
	seq      int
}

type manaSearchQueue []*manaSearchNode

func (q manaSearchQueue) Len() int { return len(q) }
func (q manaSearchQueue) Less(i, j int) bool {
	if q[i].priority != q[j].priority {
		return q[i].priority < q[j].priority
	}
	if q[i].cost != q[j].cost {
		return q[i].cost < q[j].cost
	}
	if q[i].taps != q[j].taps {
		return q[i].taps > q[j].taps
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
		used: make([]bool, len(in.Sources)),
	}
	root.priority = manaSearchLowerBound(root, in)
	queue := manaSearchQueue{root}
	heap.Init(&queue)
	bestCost := map[string]int{manaSearchStateKey(rootPool, root.used, in.SpellContext): 0}
	productionChoices := make([][][]manaProductionChoice, len(in.Sources))
	for sourceIndex, source := range in.Sources {
		productionChoices[sourceIndex] = make([][]manaProductionChoice, len(source.Abilities))
		for abilityIndex, ability := range source.Abilities {
			productionChoices[sourceIndex][abilityIndex] = manaProductionChoices(ability.Productions, source.Bonuses)
		}
	}
	seq := 0

	for queue.Len() > 0 {
		node := heap.Pop(&queue).(*manaSearchNode)
		key := manaSearchStateKey(node.pool, node.used, in.SpellContext)
		if known, ok := bestCost[key]; ok && node.cost > known {
			continue
		}
		if node.pool.CanPay(in.Cost, in.SpellContext) {
			return append([]ManaTap(nil), node.path...), true
		}

		for sourceIndex, source := range in.Sources {
			if node.used[sourceIndex] {
				continue
			}
			for abilityIndex, ability := range source.Abilities {
				if !node.pool.CanPay(ability.ManaCost, nil) {
					continue
				}
				for _, choice := range productionChoices[sourceIndex][abilityIndex] {
					nextPool := cloneManaPoolForSearch(node.pool, in.Conversions)
					if err := nextPool.Pay(ability.ManaCost, nil); err != nil {
						continue
					}
					addConcreteMana(nextPool, choice.Productions, choice.BonusColors)

					nextUsed := append([]bool(nil), node.used...)
					nextUsed[sourceIndex] = true
					nextPath := append(append([]ManaTap(nil), node.path...), ManaTap{
						PermanentID:  source.PermanentID,
						AbilityIndex: ability.AbilityIndex,
						Color:        choice.Color,
						Productions:  append([]ManaProduction(nil), choice.Productions...),
						BonusColors:  append([]Color(nil), choice.BonusColors...),
					})
					nextScore := node.score + max(scoreAt(in.Scores, sourceIndex), 0)
					nextTaps := node.taps + 1
					nextCost := nextScore*weight + nextTaps
					nextKey := manaSearchStateKey(nextPool, nextUsed, in.SpellContext)
					if known, ok := bestCost[nextKey]; ok && known <= nextCost {
						continue
					}
					bestCost[nextKey] = nextCost
					seq++
					next := &manaSearchNode{
						pool:  nextPool,
						used:  nextUsed,
						path:  nextPath,
						score: nextScore,
						taps:  nextTaps,
						cost:  nextCost,
						seq:   seq,
					}
					next.priority = nextCost + manaSearchLowerBound(next, in)
					heap.Push(&queue, next)
				}
			}
		}
	}
	return nil, false
}

func cloneManaPoolForSearch(pool *ManaPool, conversions map[Color]Color) *ManaPool {
	clone := NewManaPool()
	clone.RestorePool(pool.SnapshotPool())
	clone.ManaConversions = conversions
	return clone
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
		if node.used[i] {
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

func manaSearchStateKey(pool *ManaPool, used []bool, spellCtx *SpellPaymentContext) string {
	var total [AnyColor + 1]int
	var abilityUsable [AnyColor + 1]int
	var spellUsable [AnyColor + 1]int
	for _, mana := range pool.pool {
		total[mana.Color]++
		if manaUsable(mana, nil) {
			abilityUsable[mana.Color]++
		}
		if manaUsable(mana, spellCtx) {
			spellUsable[mana.Color]++
		}
	}
	var b strings.Builder
	for _, color := range manaPoolColors {
		b.WriteString(strconv.Itoa(total[color]))
		b.WriteByte('/')
		b.WriteString(strconv.Itoa(abilityUsable[color]))
		b.WriteByte('/')
		b.WriteString(strconv.Itoa(spellUsable[color]))
		b.WriteByte(',')
	}
	b.WriteByte('|')
	for _, isUsed := range used {
		if isUsed {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
	}
	return b.String()
}

func manaProductionChoices(productions []ManaProduction, bonuses []ManaBonusColor) []manaProductionChoice {
	concrete := concreteManaProductions(productions)
	result := make([]manaProductionChoice, 0, len(concrete))
	for _, production := range concrete {
		producedColors := productionColors(production)
		bonusChoices := concreteBonusChoices(bonuses, producedColors)
		if len(bonusChoices) == 0 {
			bonusChoices = [][]Color{{}}
		}
		for _, bonusColors := range bonusChoices {
			result = append(result, manaProductionChoice{
				Color:       preferredProductionColor(productions, production),
				Productions: production,
				BonusColors: bonusColors,
			})
		}
	}
	return result
}

func concreteManaProductions(productions []ManaProduction) [][]ManaProduction {
	if len(productions) == 0 {
		return nil
	}
	var countChoices [][AnyColor + 1]int
	var expand func(int, [AnyColor + 1]int)
	expand = func(index int, current [AnyColor + 1]int) {
		if index == len(productions) {
			countChoices = append(countChoices, current)
			return
		}
		production := productions[index]
		amount := normalizedManaAmount(production.Amount)
		if production.Color != AnyColor {
			current[production.Color] += amount
			expand(index+1, current)
			return
		}
		if !production.AnyCombination {
			for color := White; color <= Green; color++ {
				next := current
				next[color] += amount
				expand(index+1, next)
			}
			return
		}
		for _, allocation := range coloredManaAllocations(amount) {
			next := current
			for color := White; color <= Green; color++ {
				next[color] += allocation[color]
			}
			expand(index+1, next)
		}
	}
	expand(0, [AnyColor + 1]int{})

	seen := make(map[[AnyColor + 1]int]bool, len(countChoices))
	result := make([][]ManaProduction, 0, len(countChoices))
	for _, choice := range countChoices {
		if seen[choice] {
			continue
		}
		seen[choice] = true
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

func coloredManaAllocations(amount int) [][AnyColor + 1]int {
	var result [][AnyColor + 1]int
	var distribute func(Color, int, [AnyColor + 1]int)
	distribute = func(color Color, remaining int, allocation [AnyColor + 1]int) {
		if color == Green {
			allocation[color] = remaining
			result = append(result, allocation)
			return
		}
		for n := 0; n <= remaining; n++ {
			next := allocation
			next[color] = n
			distribute(color+1, remaining-n, next)
		}
	}
	distribute(White, amount, [AnyColor + 1]int{})
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

func concreteBonusChoices(bonuses []ManaBonusColor, producedColors []Color) [][]Color {
	choices := [][]Color{{}}
	for _, bonus := range bonuses {
		if bonus != MatchProduced {
			for i := range choices {
				choices[i] = append(choices[i], Color(bonus))
			}
			continue
		}
		if len(producedColors) == 0 {
			continue
		}
		expanded := make([][]Color, 0, len(producedColors)*len(choices))
		for _, choice := range choices {
			for _, color := range producedColors {
				next := append([]Color(nil), choice...)
				next = append(next, color)
				expanded = append(expanded, next)
			}
		}
		choices = expanded
	}
	return choices
}

func preferredProductionColor(original, concrete []ManaProduction) Color {
	for _, production := range original {
		if production.Color != AnyColor {
			continue
		}
		for _, resolved := range concrete {
			if resolved.Color != Colorless && resolved.Amount > 0 {
				return resolved.Color
			}
		}
	}
	if len(concrete) == 1 {
		return concrete[0].Color
	}
	return Colorless
}

func scoreAt(scores []int, i int) int {
	if i < len(scores) {
		return scores[i]
	}
	return 0
}
