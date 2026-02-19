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
