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
	default:
		return 0
	}
}
