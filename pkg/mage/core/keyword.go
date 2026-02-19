package core

// Keyword represents a keyword ability.
type Keyword int

const (
	Flying Keyword = iota
	FirstStrike
	DoubleStrike
	Deathtouch
	Lifelink
	Trample
	Vigilance
	Haste
	Reach
	Menace
	Hexproof
	Shroud
	Indestructible
	Defender
	Fear
	Banding
	Forestwalk
	Islandwalk
	Swampwalk
	Mountainwalk
	Plainswalk
	CantBeBlockedByWalls
	CanBlockAdditional          // can block an additional creature each combat
	CantBeBlockedExceptByWalls  // can only be blocked by Walls
	BasiliskTouch                // deathtouch that doesn't kill Walls
	EntersTapped
	DoesNotUntapKW
	UnblockableKW
	CanBlockAny    // can block any number of creatures (Blaze of Glory)
	MustBeBlocked  // all creatures able to block this creature must do so (Lure)
	MustAttack     // this creature must attack this turn if able (Nettling Imp)
)

func (k Keyword) String() string {
	switch k {
	case Flying:
		return "Flying"
	case FirstStrike:
		return "First Strike"
	case DoubleStrike:
		return "Double Strike"
	case Deathtouch:
		return "Deathtouch"
	case Lifelink:
		return "Lifelink"
	case Trample:
		return "Trample"
	case Vigilance:
		return "Vigilance"
	case Haste:
		return "Haste"
	case Reach:
		return "Reach"
	case Menace:
		return "Menace"
	case Hexproof:
		return "Hexproof"
	case Shroud:
		return "Shroud"
	case Indestructible:
		return "Indestructible"
	case Defender:
		return "Defender"
	case Fear:
		return "Fear"
	case Banding:
		return "Banding"
	case Forestwalk:
		return "Forestwalk"
	case Islandwalk:
		return "Islandwalk"
	case Swampwalk:
		return "Swampwalk"
	case Mountainwalk:
		return "Mountainwalk"
	case Plainswalk:
		return "Plainswalk"
	case CantBeBlockedByWalls:
		return "Can't Be Blocked by Walls"
	case CanBlockAdditional:
		return "Can Block Additional Creature"
	case CantBeBlockedExceptByWalls:
		return "Can't Be Blocked Except by Walls"
	case EntersTapped:
		return "Enters Tapped"
	case DoesNotUntapKW:
		return "Does Not Untap"
	case UnblockableKW:
		return "Can't Be Blocked"
	case CanBlockAny:
		return "Can Block Any Number"
	case MustBeBlocked:
		return "Must Be Blocked"
	case MustAttack:
		return "Must Attack"
	default:
		return "Unknown"
	}
}

// IsLandwalk returns true if this keyword is a landwalk ability.
func (k Keyword) IsLandwalk() bool {
	switch k {
	case Forestwalk, Islandwalk, Swampwalk, Mountainwalk, Plainswalk:
		return true
	}
	return false
}

// LandwalkSubtype returns the land subtype that this landwalk cares about.
func (k Keyword) LandwalkSubtype() string {
	switch k {
	case Forestwalk:
		return "Forest"
	case Islandwalk:
		return "Island"
	case Swampwalk:
		return "Swamp"
	case Mountainwalk:
		return "Mountain"
	case Plainswalk:
		return "Plains"
	}
	return ""
}
