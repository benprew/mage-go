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

	// Fallen Empires counters
	Spore   // Thallid spore counters
	Tide    // Homarid / Tidal Influence tide counters
	Storage // Storage lands (Bottomless Vault, etc.)
	Credit  // Icatian Moneychanger credit counters
	Javelin // Icatian Javelineers javelin counters
	Net     // Merseine net counters
	Time    // Tourach's Gate time counters
	Cube    // Delif's Cube cube counters

	Corpse   // Scavenging Ghoul corpse counters
	Vitality // Living Artifact vitality counters

	// Generic P/T counters
	P1P2 // +1/+2 (Armor Thrull)
	M2M2 // -2/-2 (Ebon Praetor)
	M0M1 // -0/-1 (Takklemaggot, Lesser Werewolf)
	M0M2 // -0/-2 (Spirit Shackle)

	Stun // CR 122.1g — "If a permanent with a stun counter would become untapped, remove a stun counter from it instead. It doesn't untap." Removed in the untap step of doUntap (game.go).

	// Secrets of Strixhaven counters
	Page   // Diary of Dreams page counters
	Growth // Comforting Counsel growth counters

	// Astral counters
	Husk // Necropolis of Azar husk counters

	// The Dark counters
	P0P1   // +0/+1 (Living Armor, Necropolis)
	P2P0   // +2/+0 (Frankenstein's Monster)
	P0P2   // +0/+2 (Frankenstein's Monster)
	Hunger // Fasting

	// NumCounters must remain the last entry — it sizes the fixed-length
	// counter array on Permanent, so clone is a memcpy instead of a map copy.
	NumCounters
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
	case Spore:
		return "Spore"
	case Tide:
		return "Tide"
	case Storage:
		return "Storage"
	case Credit:
		return "Credit"
	case Javelin:
		return "Javelin"
	case Net:
		return "Net"
	case Time:
		return "Time"
	case Cube:
		return "Cube"
	case Corpse:
		return "Corpse"
	case Vitality:
		return "Vitality"
	case P1P2:
		return "+1/+2"
	case M2M2:
		return "-2/-2"
	case M0M1:
		return "-0/-1"
	case M0M2:
		return "-0/-2"
	case Stun:
		return "Stun"
	case Page:
		return "Page"
	case Growth:
		return "Growth"
	case Husk:
		return "Husk"
	case P0P1:
		return "+0/+1"
	case P2P0:
		return "+2/+0"
	case P0P2:
		return "+0/+2"
	case Hunger:
		return "Hunger"
	default:
		return "Unknown"
	}
}

// PowerBoost returns the power modification from this counter type.
func (ct CounterType) PowerBoost() int {
	switch ct {
	case P1P1, P1P0, P1P2:
		return 1
	case P2P0:
		return 2
	case M1M1:
		return -1
	case M2M2:
		return -2
	default:
		return 0
	}
}

// ToughnessBoost returns the toughness modification from this counter type.
func (ct CounterType) ToughnessBoost() int {
	switch ct {
	case P1P1, P0P1:
		return 1
	case P1P2, P0P2:
		return 2
	case M1M1:
		return -1
	case M2M2:
		return -2
	case M0M1:
		return -1
	case M0M2:
		return -2
	default:
		return 0
	}
}
