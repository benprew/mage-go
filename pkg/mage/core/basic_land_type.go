package core

// BasicLandType identifies one of the five basic land subtypes.
type BasicLandType uint8

const (
	NoBasicLandType BasicLandType = iota
	PlainsLandType
	IslandLandType
	SwampLandType
	MountainLandType
	ForestLandType
)

// String returns the subtype name, or an empty string for an invalid value.
func (t BasicLandType) String() string {
	if t < PlainsLandType || t > ForestLandType {
		return ""
	}
	return [...]string{"", "Plains", "Island", "Swamp", "Mountain", "Forest"}[t]
}
