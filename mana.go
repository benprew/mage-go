package mage

import (
	"fmt"
	"strconv"
	"strings"
)

// Color represents a Magic color.
type Color int

const (
	Colorless Color = iota
	White
	Blue
	Black
	Red
	Green
)

func (c Color) String() string {
	switch c {
	case Colorless:
		return "Colorless"
	case White:
		return "White"
	case Blue:
		return "Blue"
	case Black:
		return "Black"
	case Red:
		return "Red"
	case Green:
		return "Green"
	default:
		return "Unknown"
	}
}

func ColorFromSymbol(s string) (Color, bool) {
	switch strings.ToUpper(s) {
	case "W":
		return White, true
	case "U":
		return Blue, true
	case "B":
		return Black, true
	case "R":
		return Red, true
	case "G":
		return Green, true
	default:
		return Colorless, false
	}
}

func (c Color) Symbol() string {
	switch c {
	case White:
		return "W"
	case Blue:
		return "U"
	case Black:
		return "B"
	case Red:
		return "R"
	case Green:
		return "G"
	default:
		return "C"
	}
}

// ManaCost represents a parsed mana cost.
type ManaCost struct {
	Generic int
	White   int
	Blue    int
	Black   int
	Red     int
	Green   int
	HasX    bool // whether this cost includes {X}
	XCount  int  // number of X's (usually 1)
}

// CMC returns the converted mana cost (total mana value).
func (mc ManaCost) CMC() int {
	return mc.Generic + mc.White + mc.Blue + mc.Black + mc.Red + mc.Green
}

// Colors returns the set of colors in this mana cost.
func (mc ManaCost) Colors() []Color {
	var colors []Color
	if mc.White > 0 {
		colors = append(colors, White)
	}
	if mc.Blue > 0 {
		colors = append(colors, Blue)
	}
	if mc.Black > 0 {
		colors = append(colors, Black)
	}
	if mc.Red > 0 {
		colors = append(colors, Red)
	}
	if mc.Green > 0 {
		colors = append(colors, Green)
	}
	return colors
}

// IsZero returns true if the mana cost is empty (e.g., for lands).
func (mc ManaCost) IsZero() bool {
	return mc.Generic == 0 && mc.White == 0 && mc.Blue == 0 && mc.Black == 0 && mc.Red == 0 && mc.Green == 0
}

func (mc ManaCost) String() string {
	var parts []string
	for i := 0; i < mc.XCount; i++ {
		parts = append(parts, "{X}")
	}
	if mc.Generic > 0 || (mc.CMC() == 0 && !mc.HasX) {
		parts = append(parts, fmt.Sprintf("{%d}", mc.Generic))
	}
	for i := 0; i < mc.White; i++ {
		parts = append(parts, "{W}")
	}
	for i := 0; i < mc.Blue; i++ {
		parts = append(parts, "{U}")
	}
	for i := 0; i < mc.Black; i++ {
		parts = append(parts, "{B}")
	}
	for i := 0; i < mc.Red; i++ {
		parts = append(parts, "{R}")
	}
	for i := 0; i < mc.Green; i++ {
		parts = append(parts, "{G}")
	}
	return strings.Join(parts, "")
}

// ParseManaCost parses a mana cost string like "{1}{B}{G}" into a ManaCost.
func ParseManaCost(s string) ManaCost {
	mc := ManaCost{}
	if s == "" {
		return mc
	}
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) {
		if s[i] == '{' {
			end := strings.IndexByte(s[i:], '}')
			if end == -1 {
				break
			}
			symbol := s[i+1 : i+end]
			if n, err := strconv.Atoi(symbol); err == nil {
				mc.Generic += n
			} else {
				switch strings.ToUpper(symbol) {
				case "W":
					mc.White++
				case "U":
					mc.Blue++
				case "B":
					mc.Black++
				case "R":
					mc.Red++
				case "G":
					mc.Green++
				case "X":
					mc.HasX = true
					mc.XCount++
				}
			}
			i += end + 1
		} else {
			i++
		}
	}
	return mc
}

// Mana represents a single unit of mana with a color.
type Mana struct {
	Color Color
}

// ManaPool tracks available mana for a player.
type ManaPool struct {
	pool            []Mana
	ManaConversions map[Color]Color // from → to (set by Sunglasses of Urza etc.)
}

func NewManaPool() *ManaPool {
	return &ManaPool{}
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

func (mp *ManaPool) Total() int {
	return len(mp.pool)
}

// TotalMana returns the total mana in the pool (alias for Total).
func (mp *ManaPool) TotalMana() int {
	return len(mp.pool)
}

// DrainGeneric removes n mana from the pool (any color).
func (mp *ManaPool) DrainGeneric(n int) {
	for i := 0; i < n && len(mp.pool) > 0; i++ {
		mp.pool = mp.pool[:len(mp.pool)-1]
	}
}

// availableAs returns the count of mana available to pay as the given color,
// including mana that can be converted via ManaConversions.
func (mp *ManaPool) availableAs(avail map[Color]int, color Color) int {
	count := avail[color]
	// Check if any conversion allows another color to be used as this one
	for from, to := range mp.ManaConversions {
		if to == color && from != color {
			count += avail[from]
		}
	}
	return count
}

// CanPay returns true if the pool can pay the given mana cost.
func (mp *ManaPool) CanPay(mc ManaCost) bool {
	avail := map[Color]int{}
	for _, m := range mp.pool {
		avail[m.Color]++
	}

	// With mana conversion, we need to track which mana is "spent" from convertible sources
	// For simplicity: check each colored requirement can be met, then check generic
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
		// Fast path: no conversions
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

	// Slow path with conversions: consume from exact matches first, then conversions
	used := map[Color]int{}
	for _, r := range reqs {
		need := r.needed
		// First use exact color
		exact := min(avail[r.color]-used[r.color], need)
		if exact > 0 {
			used[r.color] += exact
			need -= exact
		}
		// Then use converted mana
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
	// Count remaining for generic
	remaining := 0
	for c, count := range avail {
		remaining += count - used[c]
	}
	return remaining >= mc.Generic
}

// Pay removes mana from the pool to pay a cost. Returns error if insufficient.
func (mp *ManaPool) Pay(mc ManaCost) error {
	if !mp.CanPay(mc) {
		return fmt.Errorf("insufficient mana to pay %s", mc)
	}
	// Pay each colored requirement, using conversions if needed
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
		// First remove exact color
		removed := mp.removeUpTo(r.color, need)
		need = removed // removeUpTo returns remaining
		// If still need more, use converted mana
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
	// Pay generic with any color (prefer colorless first)
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

func (mp *ManaPool) removeColor(c Color, n int) {
	for i := 0; i < n; i++ {
		for j, m := range mp.pool {
			if m.Color == c {
				mp.pool = append(mp.pool[:j], mp.pool[j+1:]...)
				break
			}
		}
	}
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
