package core

import (
	"fmt"
	"strconv"
	"strings"
)

// HybridSymbol represents a two-color hybrid mana symbol like {W/U}.
// Either of the two colors satisfies the symbol (CR 107.4d, 117.7).
type HybridSymbol struct {
	A Color
	B Color
}

// String formats a hybrid symbol as "{X/Y}" using the Color shorthand.
func (h HybridSymbol) String() string {
	return fmt.Sprintf("{%s/%s}", colorSymbol(h.A), colorSymbol(h.B))
}

func colorSymbol(c Color) string {
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
	}
	return "?"
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
	// Hybrid is the list of two-color hybrid symbols in this cost. Each
	// entry can be paid with mana of either listed color (CR 107.4d).
	Hybrid []HybridSymbol
}

// CMC returns the converted mana cost (total mana value).
// Hybrid symbols contribute 1 each (CR 202.3f: each {X/Y} hybrid is mana value 1).
func (mc ManaCost) CMC() int {
	return mc.Generic + mc.White + mc.Blue + mc.Black + mc.Red + mc.Green + len(mc.Hybrid)
}

// Colors returns the set of colors in this mana cost.
// Hybrid symbols contribute both of their colors (CR 202.2c).
func (mc ManaCost) Colors() []Color {
	seen := map[Color]bool{}
	var colors []Color
	add := func(c Color) {
		if !seen[c] {
			seen[c] = true
			colors = append(colors, c)
		}
	}
	if mc.White > 0 {
		add(White)
	}
	if mc.Blue > 0 {
		add(Blue)
	}
	if mc.Black > 0 {
		add(Black)
	}
	if mc.Red > 0 {
		add(Red)
	}
	if mc.Green > 0 {
		add(Green)
	}
	for _, h := range mc.Hybrid {
		add(h.A)
		add(h.B)
	}
	return colors
}

// IsZero returns true if the mana cost is empty (e.g., for lands).
func (mc ManaCost) IsZero() bool {
	return mc.Generic == 0 && mc.White == 0 && mc.Blue == 0 && mc.Black == 0 && mc.Red == 0 && mc.Green == 0 && len(mc.Hybrid) == 0
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
	for _, h := range mc.Hybrid {
		parts = append(parts, h.String())
	}
	return strings.Join(parts, "")
}

// parseColorLetter maps a single mana-cost letter (W/U/B/R/G) to a Color.
// Returns Colorless and false if the letter is not a valid color.
func parseColorLetter(s string) (Color, bool) {
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
	}
	return Colorless, false
}

// ParseManaCost parses a mana cost string like "{1}{B}{G}" or "{1}{W/U}" into a ManaCost.
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
			} else if slash := strings.IndexByte(symbol, '/'); slash != -1 {
				a, okA := parseColorLetter(symbol[:slash])
				b, okB := parseColorLetter(symbol[slash+1:])
				if okA && okB {
					mc.Hybrid = append(mc.Hybrid, HybridSymbol{A: a, B: b})
				}
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
