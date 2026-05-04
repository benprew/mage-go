package core

// Attr is a named attribute that can be granted to or revoked from a permanent.
// Keywords, capabilities, and battlefield type-identity are all expressed as Attr values.
// Storing counts (not booleans) enables additive/subtractive composition:
// base=1 + granted=-1 → net 0 → false; base=1 + granted=+1 → net 2 → true.
type Attr int

const (
	// Capability attrs — what a permanent can do / must do.
	AttrCanAttack Attr = iota + 1
	AttrCanBlock
	AttrHasPowerToughness  // has P/T; takes combat damage; subject to SBAs
	AttrSummonSick         // set on ETB for creatures; cleared at untap; Haste bypasses check
	AttrDoesNotUntap       // replaces DoesNotUntapKW
	AttrEntersTapped       // set on entry; cleared by PutOnBattlefield after tapping
	AttrMustAttack         // replaces MustAttack keyword
	AttrMustBeBlocked      // Lure semantics: all able blockers must block this
	AttrMustBeBlockedIfAble // CR 509.1c: must be blocked by at least one able blocker
	AttrMayNotUntap        // player may choose not to untap during untap step
	AttrCantBeEnchanted              // permanent can't have enchantments attached to it
	AttrCantBeTargetedByArtifacts    // permanent can't be targeted by abilities from artifact sources
	AttrCantChangeControl            // other players can't gain control (Guardian Beast)
	AttrCantActivateNonManaAbilities // permanent's non-mana activated abilities can't be activated (CR 605 mana abilities are unaffected)
	AttrAssignsDamageEqualToToughness // permanent assigns combat damage equal to its toughness rather than its power (Doran the Siege Tower / Assault Formation)

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
	Desertwalk
	CantRegenerate
	LegendaryLandwalk
	Flash

	// attrCount is a sentinel marking one past the last Attr value.
	// NumAttrs exposes this as a sized array bound for Permanent attr storage.
	attrCount
)

// NumAttrs is the array-bound size for per-permanent attr storage.
// Kept slightly above attrCount so future Attrs can be added without
// resizing array fields. Must stay >= int(attrCount).
const NumAttrs = 64

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
	case AttrMustBeBlockedIfAble:
		return "Must Be Blocked If Able"
	case AttrMayNotUntap:
		return "May Not Untap"
	case AttrCantBeEnchanted:
		return "Can't Be Enchanted"
	case AttrCantBeTargetedByArtifacts:
		return "Can't Be Targeted by Artifacts"
	case AttrCantChangeControl:
		return "Can't Change Control"
	case AttrCantActivateNonManaAbilities:
		return "Can't Activate Non-Mana Abilities"
	case AttrAssignsDamageEqualToToughness:
		return "Assigns Combat Damage Equal to Toughness"
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
	case Desertwalk:
		return "Desertwalk"
	case CantRegenerate:
		return "Can't Be Regenerated"
	case LegendaryLandwalk:
		return "Legendary Landwalk"
	case Flash:
		return "Flash"
	default:
		return "Unknown"
	}
}

// landwalkSubtypes maps each landwalk Attr to the land subtype string it cares about.
// To add new landwalk variants (e.g. Desertwalk for Arabian Nights), add an entry here
// rather than adding switch cases — no other code needs updating.
var landwalkSubtypes = map[Attr]string{
	Forestwalk:   "Forest",
	Islandwalk:   "Island",
	Swampwalk:    "Swamp",
	Mountainwalk: "Mountain",
	Plainswalk:   "Plains",
	Desertwalk:   "Desert",
}

// subtypeToLandwalk is the reverse of landwalkSubtypes, built once at init.
var subtypeToLandwalk map[string]Attr

func init() {
	if int(attrCount) > NumAttrs {
		panic("core: NumAttrs is smaller than attrCount; bump NumAttrs in attr.go")
	}
	subtypeToLandwalk = make(map[string]Attr, len(landwalkSubtypes))
	for attr, subtype := range landwalkSubtypes {
		subtypeToLandwalk[subtype] = attr
	}
}

// LandwalkSubtype returns the land subtype that this landwalk cares about.
// Returns "" if a is not a landwalk attr.
func (a Attr) LandwalkSubtype() string {
	return landwalkSubtypes[a]
}

// LandwalkAttrs returns the full landwalk attr → subtype map. Used by combat code
// to iterate over all registered landwalk variants.
func LandwalkAttrs() map[Attr]string {
	return landwalkSubtypes
}

// LandwalkAttr returns the Attr for a given land subtype (e.g. "Swamp" → Swampwalk).
// Returns 0 if no landwalk attr is registered for that subtype.
// Use this when registering new cards with landwalk from new sets.
func LandwalkAttr(subtype string) Attr {
	return subtypeToLandwalk[subtype]
}
