package mage

import (
	"fmt"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
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

// Empties every player's mana pool. Called at the end of every step and phase per
// CR 500.5.
func (g *Game) emptyManaPools() {
	for _, p := range g.players {
		p.ManaPool().Clear()
	}
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
func (mp *ManaPool) CanPay(mc ManaCost) bool {
	avail := map[Color]int{}
	for _, m := range mp.pool {
		avail[m.Color]++
	}

	type colorReq struct {
		color  Color
		needed int
	}
	reqs := []colorReq{
		{White, mc.White},
		{Blue, mc.Blue},
		{Black, mc.Black},
		{Red, mc.Red},
		{Green, mc.Green},
	}

	if len(mp.ManaConversions) == 0 {
		remaining := 0
		for _, r := range reqs {
			if avail[r.color] < r.needed {
				return false
			}
			avail[r.color] -= r.needed
			remaining += avail[r.color]
		}
		if !allocateHybrids(mc.Hybrid, avail) {
			return false
		}
		remaining = avail[Colorless]
		for _, r := range reqs {
			remaining += avail[r.color]
		}
		return remaining >= mc.Generic
	}

	used := map[Color]int{}
	for _, r := range reqs {
		need := r.needed
		exact := min(avail[r.color]-used[r.color], need)
		if exact > 0 {
			used[r.color] += exact
			need -= exact
		}
		if need > 0 {
			for from, to := range mp.ManaConversions {
				if to == r.color && from != r.color {
					conv := min(avail[from]-used[from], need)
					if conv > 0 {
						used[from] += conv
						need -= conv
					}
				}
			}
		}
		if need > 0 {
			return false
		}
	}
	remainingByColor := map[Color]int{}
	for c, count := range avail {
		remainingByColor[c] = count - used[c]
	}
	if !allocateHybrids(mc.Hybrid, remainingByColor) {
		return false
	}
	remaining := 0
	for _, count := range remainingByColor {
		remaining += count
	}
	return remaining >= mc.Generic
}

// allocateHybrids greedily assigns each hybrid symbol to one of its two colors
// from the available pool, mutating the map to reflect consumption. Returns
// false if any symbol cannot be paid. The allocation is deterministic: it
// prefers the more-abundant color, breaking ties by symbol order (color A
// first). This is sufficient for the simple cases the engine currently sees;
// it can fail to find a valid allocation only when colors share constraints
// across symbols, which doesn't arise for the hybrid costs in print.
func allocateHybrids(syms []HybridSymbol, avail map[Color]int) bool {
	for _, h := range syms {
		a, b := avail[h.A], avail[h.B]
		switch {
		case a >= b && a > 0:
			avail[h.A] = a - 1
		case b > 0:
			avail[h.B] = b - 1
		default:
			return false
		}
	}
	return true
}

// Surplus returns how much mana would remain after paying the given cost,
// or -1 if the cost cannot be paid.
func (mp *ManaPool) Surplus(mc ManaCost) int {
	if !mp.CanPay(mc) {
		return -1
	}
	return len(mp.pool) - mc.CMC()
}

// Pay removes mana from the pool to pay a cost. Returns error if insufficient.
func (mp *ManaPool) Pay(mc ManaCost) error {
	if !mp.CanPay(mc) {
		return fmt.Errorf("insufficient mana to pay %s", mc)
	}
	type colorReq struct {
		color  Color
		needed int
	}
	reqs := []colorReq{
		{White, mc.White},
		{Blue, mc.Blue},
		{Black, mc.Black},
		{Red, mc.Red},
		{Green, mc.Green},
	}
	for _, r := range reqs {
		need := r.needed
		removed := mp.removeUpTo(r.color, need)
		need = removed
		if need > 0 {
			for from, to := range mp.ManaConversions {
				if to == r.color && from != r.color {
					need = mp.removeUpTo(from, need)
					if need <= 0 {
						break
					}
				}
			}
		}
	}
	for _, h := range mc.Hybrid {
		a, b := mp.Count(h.A), mp.Count(h.B)
		if a >= b && a > 0 {
			mp.removeUpTo(h.A, 1)
		} else if b > 0 {
			mp.removeUpTo(h.B, 1)
		}
	}
	generic := mc.Generic
	generic = mp.removeUpTo(Colorless, generic)
	for _, c := range []Color{White, Blue, Black, Red, Green} {
		if generic <= 0 {
			break
		}
		generic = mp.removeUpTo(c, generic)
	}
	return nil
}

func (mp *ManaPool) removeUpTo(c Color, n int) int {
	for n > 0 {
		found := false
		for j, m := range mp.pool {
			if m.Color == c {
				mp.pool = append(mp.pool[:j], mp.pool[j+1:]...)
				n--
				found = true
				mp.recordDrain(c)
				break
			}
		}
		if !found {
			break
		}
	}
	return n
}

// Clear empties the mana pool.
func (mp *ManaPool) Clear() {
	mp.pool = mp.pool[:0]
}
