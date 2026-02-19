package core

import "strings"

//go:generate enumer -type=Color -output=color_enumer.go

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
