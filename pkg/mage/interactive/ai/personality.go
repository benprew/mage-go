package ai

import (
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

// SpellOrder controls the order in which the AI evaluates castable spells.
type SpellOrder int

const (
	MostExpensiveFirst SpellOrder = iota
	CheapestFirst
)

// Personality parameterises how an AIPlayer makes decisions.
// Deprecated: Use WeightedPersonality for new code.
type Personality struct {
	Name                string
	CastOrder           SpellOrder
	HoldInstants        bool
	AttackAll           bool
	BlockPowerThreshold int
	TargetFace          bool
}

// ToWeighted converts a boolean Personality into a continuous WeightedPersonality.
func (p Personality) ToWeighted() WeightedPersonality {
	wp := WeightedPersonality{Name: p.Name}

	if p.AttackAll {
		wp.Aggression = 1.0
	} else {
		wp.Aggression = 0.0
	}

	if p.BlockPowerThreshold >= 99 {
		wp.BlockThreshold = 0.95
	} else {
		wp.BlockThreshold = float64(p.BlockPowerThreshold) / 10.0
		if wp.BlockThreshold > 1.0 {
			wp.BlockThreshold = 1.0
		}
	}

	if p.HoldInstants {
		wp.HoldInstants = 1.0
	} else {
		wp.HoldInstants = 0.0
	}

	if p.TargetFace {
		wp.TargetFace = 1.0
	} else {
		wp.TargetFace = 0.0
	}

	if p.CastOrder == MostExpensiveFirst {
		wp.CurvePreference = 1.0
	} else {
		wp.CurvePreference = 0.0
	}

	wp.Weights = eval.Weights{
		Life:  2.0,
		Board: 2.0,
		Card:  2.0,
		Mana:  1.0,
		Tempo: 1.0,
	}

	return wp
}

// WeightedPersonality provides continuous-valued weights for AI decision-making.
type WeightedPersonality struct {
	Name string

	// Evaluation weights (flow into eval.WeightedEvaluator)
	eval.Weights

	// Decision weights (0.0 to 1.0 continuous)
	Aggression      float64
	BlockThreshold  float64
	HoldInstants    float64
	TargetFace      float64
	CurvePreference float64
}

// Preset weighted personalities.
var (
	AggroWeighted = WeightedPersonality{
		Name: "Aggro",
		Weights: eval.Weights{
			Life: 1.0, Board: 3.0, Card: 0.5, Mana: 0.5, Tempo: 0.5,
		},
		Aggression: 1.0, BlockThreshold: 0.7, HoldInstants: 0.0,
		TargetFace: 0.3, CurvePreference: 0.0,
	}
	ControlWeighted = WeightedPersonality{
		Name: "Control",
		Weights: eval.Weights{
			Life: 4.0, Board: 1.0, Card: 3.0, Mana: 1.0, Tempo: 2.0,
		},
		Aggression: 0.0, BlockThreshold: 0.0, HoldInstants: 1.0,
		TargetFace: 0.0, CurvePreference: 1.0,
	}
	MidrangeWeighted = WeightedPersonality{
		Name: "Midrange",
		Weights: eval.Weights{
			Life: 2.0, Board: 3.0, Card: 2.0, Mana: 1.0, Tempo: 1.0,
		},
		Aggression: 0.7, BlockThreshold: 0.3, HoldInstants: 0.0,
		TargetFace: 0.0, CurvePreference: 1.0,
	}
	TempoWeighted = WeightedPersonality{
		Name: "Tempo",
		Weights: eval.Weights{
			Life: 1.5, Board: 2.0, Card: 1.5, Mana: 2.0, Tempo: 3.0,
		},
		Aggression: 0.8, BlockThreshold: 0.3, HoldInstants: 0.7,
		TargetFace: 0.0, CurvePreference: 0.0,
	}
	BurnWeighted = WeightedPersonality{
		Name: "Burn",
		Weights: eval.Weights{
			Life: 0.5, Board: 1.0, Card: 0.5, Mana: 0.5, Tempo: 0.5,
		},
		Aggression: 1.0, BlockThreshold: 0.95, HoldInstants: 0.0,
		TargetFace: 1.0, CurvePreference: 0.0,
	}
)

// Preset personalities (boolean, deprecated).
var (
	AggroPersonality = Personality{
		Name: "Aggro", CastOrder: CheapestFirst,
		HoldInstants: false, AttackAll: true, BlockPowerThreshold: 3,
	}
	ControlPersonality = Personality{
		Name: "Control", CastOrder: MostExpensiveFirst,
		HoldInstants: true, AttackAll: false, BlockPowerThreshold: 0,
	}
	MidrangePersonality = Personality{
		Name: "Midrange", CastOrder: MostExpensiveFirst,
		HoldInstants: false, AttackAll: true, BlockPowerThreshold: 3,
	}
	TempoPersonality = Personality{
		Name: "Tempo", CastOrder: CheapestFirst,
		HoldInstants: true, AttackAll: true, BlockPowerThreshold: 2,
	}
	BurnPersonality = Personality{
		Name: "Burn", CastOrder: CheapestFirst,
		HoldInstants: false, AttackAll: true, BlockPowerThreshold: 99,
		TargetFace: true,
	}
)
