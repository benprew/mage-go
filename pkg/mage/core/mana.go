package core

// Mana represents a single unit of mana with a color.
// Restriction is nil for normal mana; non-nil for mana that may only be
// spent on spells satisfying the restriction (e.g. Mishra's Workshop).
type Mana struct {
	Color       Color
	Restriction ManaRestriction
}
