package core

// CounterType represents a type of counter.
type CounterType int

const (
	P1P1 CounterType = iota // +1/+1
	M1M1                    // -1/-1
	P1P0                    // +1/+0
	Loyalty
	Charge
	Mire // Cyclopean Tomb mire counters
	Wind // Cyclone wind counters
	Age  // Cumulative upkeep age counters
	Doom // Armageddon Clock doom counters

	// Legends counters
	Dream        // Rasputin Dreamweaver
	Hatchling    // Triassic Egg
	Pin          // Voodoo Doll
	Pupa         // Cocoon
	Sleep        // Venarian Gold
	Scream       // All Hallow's Eve
	Intervention // Divine Intervention
	Carrion      // Carrion Ants variant tracking
	Glyph        // Glyph of Delusion
	Matrix       // Life Matrix

	// Generic P/T counters
	M0M1 // -0/-1 (Takklemaggot, Lesser Werewolf)
	M0M2 // -0/-2 (Spirit Shackle)
)

func (ct CounterType) String() string {
	switch ct {
	case P1P1:
		return "+1/+1"
	case M1M1:
		return "-1/-1"
	case P1P0:
		return "+1/+0"
	case Loyalty:
		return "Loyalty"
	case Charge:
		return "Charge"
	case Mire:
		return "Mire"
	case Wind:
		return "Wind"
	case Age:
		return "Age"
	case Doom:
		return "Doom"
	case Dream:
		return "Dream"
	case Hatchling:
		return "Hatchling"
	case Pin:
		return "Pin"
	case Pupa:
		return "Pupa"
	case Sleep:
		return "Sleep"
	case Scream:
		return "Scream"
	case Intervention:
		return "Intervention"
	case Carrion:
		return "Carrion"
	case Glyph:
		return "Glyph"
	case Matrix:
		return "Matrix"
	case M0M1:
		return "-0/-1"
	case M0M2:
		return "-0/-2"
	default:
		return "Unknown"
	}
}

// PowerBoost returns the power modification from this counter type.
func (ct CounterType) PowerBoost() int {
	switch ct {
	case P1P1, P1P0:
		return 1
	case M1M1:
		return -1
	default:
		return 0
	}
}

// ToughnessBoost returns the toughness modification from this counter type.
func (ct CounterType) ToughnessBoost() int {
	switch ct {
	case P1P1:
		return 1
	case M1M1:
		return -1
	case M0M1:
		return -1
	case M0M2:
		return -2
	default:
		return 0
	}
}
