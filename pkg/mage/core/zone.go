package core

//go:generate enumer -type=Zone -trimprefix=Zone -output=zone_enumer.go

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
	ZoneAnte
)
