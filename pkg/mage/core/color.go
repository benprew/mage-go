package core

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
	AnyColor
)

// Colors returns the five Magic colors in white, blue, black, red, green order.
// Each call returns a separate slice.
func Colors() []Color {
	return []Color{White, Blue, Black, Red, Green}
}
