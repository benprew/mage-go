package mage

//go:generate enumer -type=Zone -trimprefix=Zone

// Zone represents a game zone where cards can exist.
type Zone int

const (
	ZoneLibrary Zone = iota
	ZoneHand
	ZoneBattlefield
	ZoneGraveyard
	ZoneStack
	ZoneExile
	ZoneCommand
)

