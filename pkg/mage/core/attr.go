package core

// Attr is a named attribute that can be granted to or revoked from a permanent.
// Keywords, capabilities, and battlefield type-identity are all expressed as Attr values.
// Storing counts (not booleans) enables additive/subtractive composition:
// base=1 + granted=-1 → net 0 → false; base=1 + granted=+1 → net 2 → true.
type Attr int

const (
	// Capability attrs — what a permanent can do / must do.
	AttrCanAttack          Attr = iota + 1
	AttrCanBlock
	AttrHasPowerToughness  // has P/T; takes combat damage; subject to SBAs
	AttrSummonSick         // set on ETB for creatures; cleared at untap; Haste bypasses check
	AttrDoesNotUntap       // replaces DoesNotUntapKW
	AttrEntersTapped       // set on entry; cleared by PutOnBattlefield after tapping
	AttrMustAttack         // replaces MustAttack keyword
	AttrMustBeBlocked      // replaces MustBeBlocked keyword

	// Type-identity attrs (battlefield) — replaces TypesAdded []CardType on Permanent.
	AttrIsCreature
	AttrIsLand
	AttrIsArtifact
	AttrIsEnchantment

	// Keyword attrs — all existing keywords as Attr values.
	// IMPORTANT: Flying must remain the first keyword attr; code uses (a >= Flying) to
	// identify keyword-range attrs (e.g. for doppelganger copy).
	Flying
	Reach
	FirstStrike
	DoubleStrike
	Trample
	Vigilance
	Haste
	Menace
	Fear
	Deathtouch
	Lifelink
	Defender
	Banding
	Indestructible
	Hexproof
	Shroud
	Forestwalk
	Islandwalk
	Swampwalk
	Mountainwalk
	Plainswalk
	UnblockableKW
	CantBeBlockedByWalls
	CantBeBlockedExceptByWalls
	CanBlockAny
	CanBlockAdditional
	BasiliskTouch
)

// Backward-compat aliases: capability attrs that replaced old keyword constants.
// Existing card definitions using WithKeyword(DoesNotUntapKW) etc. compile unchanged.
const (
	DoesNotUntapKW = AttrDoesNotUntap
	EntersTapped   = AttrEntersTapped
	MustAttack     = AttrMustAttack
	MustBeBlocked  = AttrMustBeBlocked
)

// IsKeywordAttr returns true if a is in the keyword range (>= Flying).
func IsKeywordAttr(a Attr) bool {
	return a >= Flying
}

func (a Attr) String() string {
	switch a {
	case AttrCanAttack:
		return "CanAttack"
	case AttrCanBlock:
		return "CanBlock"
	case AttrHasPowerToughness:
		return "HasPowerToughness"
	case AttrSummonSick:
		return "SummonSick"
	case AttrDoesNotUntap:
		return "Does Not Untap"
	case AttrEntersTapped:
		return "Enters Tapped"
	case AttrMustAttack:
		return "Must Attack"
	case AttrMustBeBlocked:
		return "Must Be Blocked"
	case AttrIsCreature:
		return "IsCreature"
	case AttrIsLand:
		return "IsLand"
	case AttrIsArtifact:
		return "IsArtifact"
	case AttrIsEnchantment:
		return "IsEnchantment"
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
	case UnblockableKW:
		return "Can't Be Blocked"
	case CanBlockAny:
		return "Can Block Any Number"
	case BasiliskTouch:
		return "Basilisk Touch"
	default:
		return "Unknown"
	}
}

// IsLandwalk returns true if this attr is a landwalk ability.
func (a Attr) IsLandwalk() bool {
	switch a {
	case Forestwalk, Islandwalk, Swampwalk, Mountainwalk, Plainswalk:
		return true
	}
	return false
}

// LandwalkSubtype returns the land subtype that this landwalk cares about.
func (a Attr) LandwalkSubtype() string {
	switch a {
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
