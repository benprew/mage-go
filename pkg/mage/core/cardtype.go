package core

//go:generate enumer -type=CardType -trimprefix=Type -output=cardtype_enumer.go

// CardType represents a card's type.
type CardType int

const (
	TypeCreature CardType = iota
	TypeInstant
	TypeSorcery
	TypeLand
	TypeArtifact
	TypeEnchantment
)

// SuperType represents a card's supertype.
type SuperType int

const (
	SuperLegendary SuperType = iota + 1
	SuperBasic
	SuperSnow
	SuperWorld
)

func (s SuperType) String() string {
	switch s {
	case SuperLegendary:
		return "Legendary"
	case SuperBasic:
		return "Basic"
	case SuperSnow:
		return "Snow"
	case SuperWorld:
		return "World"
	}
	return "Unknown"
}
