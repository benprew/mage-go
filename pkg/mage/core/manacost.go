package core

import (
	"fmt"
	"strconv"
	"strings"
)

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
