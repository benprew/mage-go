package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"fmt"
)

// ManaPool tracks available mana for a player.
type ManaPool struct {
	pool            []Mana
	ManaConversions map[Color]Color // from → to (set by Sunglasses of Urza etc.)
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
	for i := 0; i < amount; i++ {
		mp.pool = append(mp.pool, Mana{Color: c})
	}
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
		mp.pool = mp.pool[:len(mp.pool)-1]
	}
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
			remaining += avail[r.color] - r.needed
		}
		remaining += avail[Colorless]
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
	remaining := 0
	for c, count := range avail {
		remaining += count - used[c]
	}
	return remaining >= mc.Generic
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
