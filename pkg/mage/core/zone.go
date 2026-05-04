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
	// ZoneAny is a sentinel for "any zone" used by zone-change matchers that
	// don't constrain a from or to (e.g. "when ~ leaves the battlefield" with
	// no destination filter). Not a real game zone — never store cards in it.
	ZoneAny
)
