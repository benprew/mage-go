package mage

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

func (z Zone) String() string {
	switch z {
	case ZoneLibrary:
		return "Library"
	case ZoneHand:
		return "Hand"
	case ZoneBattlefield:
		return "Battlefield"
	case ZoneGraveyard:
		return "Graveyard"
	case ZoneStack:
		return "Stack"
	case ZoneExile:
		return "Exile"
	case ZoneCommand:
		return "Command"
	default:
		return "Unknown"
	}
}
