package mage

import (
	"fmt"
	"slices"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ManaPool tracks available mana for a player.
type ManaPool struct {
	pool            []Mana
	ManaConversions map[Color]Color // from → to (set by Sunglasses of Urza etc.)

	// ProducedThisTurn records every unit of mana ever added to this pool
	// during the current turn, for observability when the live pool empties
	// between steps (CR 500.5). Reset at turn start.
	ProducedThisTurn []Mana

	// LastDrainedColors records, per color, how many mana of that color were
	// removed from the pool by the most recent payment operation (Pay or
	// DrainGeneric). Callers that need to know which colors were spent paying
	// for a particular spell or ability should call ResetLastDrained()
	// immediately before the payment and then inspect this map after.
	// Used by the cast pipeline to populate CastContext.ColorsSpent for
	// "for each color of mana spent to cast it" effects (Chamber Sentry).
	LastDrainedColors map[Color]int
}

func NewManaPool() *ManaPool {
	return &ManaPool{}
}

// SnapshotPool returns a copy of the internal mana pool for undo support.
func (mp *ManaPool) SnapshotPool() []Mana {
	s := make([]Mana, len(mp.pool))
	copy(s, mp.pool)
	return s
}

// RestorePool replaces the internal mana pool from a snapshot.
func (mp *ManaPool) RestorePool(snap []Mana) {
	mp.pool = snap
}

func (mp *ManaPool) Add(c Color, amount int) {
	for range amount {
		mp.pool = append(mp.pool, Mana{Color: c})
		mp.ProducedThisTurn = append(mp.ProducedThisTurn, Mana{Color: c})
	}
}

// SpellContextForCard builds a SpellPaymentContext for the spell being cast.
// Pass this into ManaPool.CanPay / ManaPool.Pay (and Game.CanAfford /
// Game.MaxXValue) at every spell-cast site so restricted mana (Mishra's
// Workshop, Metamorphosis) can be spent on eligible spells.
func SpellContextForCard(c Card) *SpellPaymentContext {
	if c == nil {
		return nil
	}
	return &SpellPaymentContext{
		IsArtifact: c.HasType(TypeArtifact),
		IsCreature: c.HasType(TypeCreature),
		CardName:   c.Name(),
	}
}

// AddRestricted adds mana that may only be spent on spells satisfying r
// (e.g. Mishra's Workshop's "Spend this mana only to cast artifact spells").
// Restriction lives on each Mana entry, so it survives in the pool but
// vanishes when the pool is cleared at end-of-step (CR 500.5).
func (mp *ManaPool) AddRestricted(c Color, amount int, r ManaRestriction) {
	for range amount {
		mp.pool = append(mp.pool, Mana{Color: c, Restriction: r})
		mp.ProducedThisTurn = append(mp.ProducedThisTurn, Mana{Color: c, Restriction: r})
	}
}

// usable reports whether a mana entry can be spent given the spell context.
// Unrestricted mana is always usable. Restricted mana requires a non-nil
// context that its restriction predicate accepts.
func manaUsable(m Mana, spellCtx *SpellPaymentContext) bool {
	if m.Restriction == nil {
		return true
	}
	if spellCtx == nil {
		return false
	}
	return m.Restriction.IsSatisfiedBy(*spellCtx)
}

// Empties every player's mana pool. Called at the end of every step and phase per
// CR 500.5.
func (g *Game) emptyManaPools() {
	g.mana.EmptyManaPools(g.players)
}

// CountProducedThisTurn returns how much mana of color c has been added to
// this pool during the current turn. Survives per-step pool emptying.
func (mp *ManaPool) CountProducedThisTurn(c Color) int {
	n := 0
	for _, m := range mp.ProducedThisTurn {
		if m.Color == c {
			n++
		}
	}
	return n
}

// TotalProducedThisTurn returns the total amount of mana added to this pool
// during the current turn.
func (mp *ManaPool) TotalProducedThisTurn() int {
	return len(mp.ProducedThisTurn)
}

// ResetProducedThisTurn clears the produced-this-turn tally. Called at the
// start of each turn by the driver.
func (mp *ManaPool) ResetProducedThisTurn() {
	mp.ProducedThisTurn = nil
}

func (mp *ManaPool) Count(c Color) int {
	n := 0
	for _, m := range mp.pool {
		if m.Color == c {
			n++
		}
	}
	return n
}

// TotalMana returns the total mana in the pool.
func (mp *ManaPool) TotalMana() int {
	return len(mp.pool)
}

// DrainGeneric removes n mana from the pool (any color).
func (mp *ManaPool) DrainGeneric(n int) {
	for i := 0; i < n && len(mp.pool) > 0; i++ {
		drained := mp.pool[len(mp.pool)-1]
		mp.pool = mp.pool[:len(mp.pool)-1]
		mp.recordDrain(drained.Color)
	}
}

// ResetLastDrained clears the last-drained-color tally so that a subsequent
// payment operation produces a clean snapshot. The cast pipeline calls this
// immediately before paying a spell's mana cost.
func (mp *ManaPool) ResetLastDrained() {
	mp.LastDrainedColors = nil
}

// recordDrain bumps the per-color tally for the most recent payment.
func (mp *ManaPool) recordDrain(c Color) {
	if mp.LastDrainedColors == nil {
		mp.LastDrainedColors = map[Color]int{}
	}
	mp.LastDrainedColors[c]++
}

// CanPay returns true if the pool can pay the given mana cost.
// spellCtx is non-nil when paying for a spell cast; pass nil for ability
// activations and other non-spell costs (restricted mana is excluded then).
func (mp *ManaPool) CanPay(mc ManaCost, spellCtx *SpellPaymentContext) bool {
	_, ok := mp.paymentPlan(mc, spellCtx)
	return ok
}

type manaPaymentRequirement struct {
	candidates []Color
}

type manaPaymentState struct {
	index     int
	available [AnyColor + 1]int
}

func (mp *ManaPool) paymentPlan(mc ManaCost, spellCtx *SpellPaymentContext) ([AnyColor + 1]int, bool) {
	var available [AnyColor + 1]int
	total := 0
	for _, mana := range mp.pool {
		if !manaUsable(mana, spellCtx) {
			continue
		}
		available[mana.Color]++
		total++
	}
	if total < mc.CMC() {
		return [AnyColor + 1]int{}, false
	}
	if len(mp.ManaConversions) == 0 && len(mc.Hybrid) == 0 {
		return directManaPaymentPlan(available, mc)
	}

	requirements := make([]manaPaymentRequirement, 0, mc.CMC()-mc.Generic)
	appendRequirements := func(color Color, count int) {
		candidates := mp.paymentCandidates(color)
		for range count {
			requirements = append(requirements, manaPaymentRequirement{candidates: candidates})
		}
	}
	appendRequirements(White, mc.White)
	appendRequirements(Blue, mc.Blue)
	appendRequirements(Black, mc.Black)
	appendRequirements(Red, mc.Red)
	appendRequirements(Green, mc.Green)
	for _, hybrid := range mc.Hybrid {
		candidates := append([]Color(nil), mp.paymentCandidates(hybrid.A)...)
		for _, candidate := range mp.paymentCandidates(hybrid.B) {
			if !containsPaymentColor(candidates, candidate) {
				candidates = append(candidates, candidate)
			}
		}
		requirements = append(requirements, manaPaymentRequirement{candidates: candidates})
	}

	remaining := available
	failed := make(map[manaPaymentState]bool)
	var allocate func(int) bool
	allocate = func(index int) bool {
		if index == len(requirements) {
			return true
		}
		state := manaPaymentState{index: index, available: remaining}
		if failed[state] {
			return false
		}
		for _, candidate := range requirements[index].candidates {
			if remaining[candidate] == 0 {
				continue
			}
			remaining[candidate]--
			if allocate(index + 1) {
				return true
			}
			remaining[candidate]++
		}
		failed[state] = true
		return false
	}
	if !allocate(0) {
		return [AnyColor + 1]int{}, false
	}

	used := [AnyColor + 1]int{}
	for _, color := range manaPoolColors {
		used[color] = available[color] - remaining[color]
	}
	generic := mc.Generic
	for _, color := range [...]Color{Colorless, White, Blue, Black, Red, Green} {
		amount := min(remaining[color], generic)
		used[color] += amount
		generic -= amount
		if generic == 0 {
			break
		}
	}
	if generic != 0 {
		return [AnyColor + 1]int{}, false
	}
	return used, true
}

func directManaPaymentPlan(available [AnyColor + 1]int, mc ManaCost) ([AnyColor + 1]int, bool) {
	used := [AnyColor + 1]int{}
	for _, requirement := range [...]struct {
		color Color
		count int
	}{
		{White, mc.White},
		{Blue, mc.Blue},
		{Black, mc.Black},
		{Red, mc.Red},
		{Green, mc.Green},
	} {
		if available[requirement.color] < requirement.count {
			return [AnyColor + 1]int{}, false
		}
		available[requirement.color] -= requirement.count
		used[requirement.color] = requirement.count
	}
	generic := mc.Generic
	for _, color := range [...]Color{Colorless, White, Blue, Black, Red, Green} {
		amount := min(available[color], generic)
		used[color] += amount
		generic -= amount
		if generic == 0 {
			break
		}
	}
	return used, generic == 0
}

func (mp *ManaPool) paymentCandidates(required Color) []Color {
	candidates := []Color{required}
	for _, from := range manaPoolColors {
		if from != required && mp.ManaConversions[from] == required {
			candidates = append(candidates, from)
		}
	}
	return candidates
}

func containsPaymentColor(colors []Color, color Color) bool {
	return slices.Contains(colors, color)
}

// Surplus returns how much mana would remain after paying the given cost,
// or -1 if the cost cannot be paid. spellCtx semantics match CanPay.
func (mp *ManaPool) Surplus(mc ManaCost, spellCtx *SpellPaymentContext) int {
	if !mp.CanPay(mc, spellCtx) {
		return -1
	}
	return len(mp.pool) - mc.CMC()
}

// Pay removes mana from the pool to pay a cost. Returns error if insufficient.
// spellCtx semantics match CanPay. When non-nil, eligible restricted mana is
// spent preferentially (it would otherwise be wasted at end-of-step per
// CR 500.5).
func (mp *ManaPool) Pay(mc ManaCost, spellCtx *SpellPaymentContext) error {
	plan, ok := mp.paymentPlan(mc, spellCtx)
	if !ok {
		return fmt.Errorf("insufficient mana to pay %s", mc)
	}
	for _, color := range [...]Color{Colorless, White, Blue, Black, Red, Green} {
		if remaining := mp.removeUpTo(color, plan[color], spellCtx); remaining != 0 {
			return fmt.Errorf("mana pool changed while paying %s", mc)
		}
	}
	return nil
}

// removeUpTo consumes up to n mana of color c, preferring restricted-but-
// eligible entries first (so they aren't wasted at end-of-step) and falling
// back to unrestricted. Returns the remaining unmet count.
func (mp *ManaPool) removeUpTo(c Color, n int, spellCtx *SpellPaymentContext) int {
	for _, restrictedFirst := range []bool{true, false} {
		for n > 0 {
			idx := -1
			for j, m := range mp.pool {
				if m.Color != c {
					continue
				}
				if !manaUsable(m, spellCtx) {
					continue
				}
				if restrictedFirst && m.Restriction == nil {
					continue
				}
				if !restrictedFirst && m.Restriction != nil {
					continue
				}
				idx = j
				break
			}
			if idx < 0 {
				break
			}
			mp.pool = append(mp.pool[:idx], mp.pool[idx+1:]...)
			n--
			mp.recordDrain(c)
		}
	}
	return n
}

// Clear empties the mana pool.
func (mp *ManaPool) Clear() {
	mp.pool = mp.pool[:0]
}
