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
