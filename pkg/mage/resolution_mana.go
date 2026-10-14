package mage

import (
	"fmt"
	"slices"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type resolutionManaPayment struct {
	player      PlayerRef
	cost        ManaCost
	amount      ResolutionValue
	effect      ResolvingEffect
	ifPaid      bool
	repeated    bool
	otherwise   ResolvingEffect
	twoBranches bool
}

// UnlessManaPaid reads X at resolution and offers the complete mana payment.
// A declined or failed payment runs the effect.
func UnlessManaPaid(player PlayerRef, cost ManaCost, amount ResolutionValue, effect ResolvingEffect) ResolvingEffect {
	cost.Hybrid = slices.Clone(cost.Hybrid)
	return &resolutionManaPayment{player: player, cost: cost, amount: amount, effect: effect}
}

// IfManaPaid runs the effect after the player accepts and pays the mana cost.
func IfManaPaid(player PlayerRef, cost ManaCost, amount ResolutionValue, effect ResolvingEffect) ResolvingEffect {
	cost.Hybrid = slices.Clone(cost.Hybrid)
	return &resolutionManaPayment{player: player, cost: cost, amount: amount, effect: effect, ifPaid: true}
}

// IfRepeatedManaPaid reads the count once and offers the complete repeated cost.
func IfRepeatedManaPaid(player PlayerRef, cost ManaCost, count ResolutionValue, effect ResolvingEffect) ResolvingEffect {
	return &resolutionManaPayment{player: player, cost: cost, amount: count, effect: effect, ifPaid: true, repeated: true}
}

// UnlessRepeatedManaPaid runs the body after a declined or failed payment.
func UnlessRepeatedManaPaid(player PlayerRef, cost ManaCost, count ResolutionValue, effect ResolvingEffect) ResolvingEffect {
	return &resolutionManaPayment{player: player, cost: cost, amount: count, effect: effect, repeated: true}
}

// PayRepeatedManaOrElse selects one body after the complete repeated payment.
func PayRepeatedManaOrElse(player PlayerRef, cost ManaCost, count ResolutionValue, paid, unpaid ResolvingEffect) ResolvingEffect {
	return &resolutionManaPayment{player: player, cost: cost, amount: count, effect: paid, otherwise: unpaid, ifPaid: true, repeated: true, twoBranches: true}
}

func (e *resolutionManaPayment) Text() string { return "optional resolution mana payment" }
func (e *resolutionManaPayment) Properties() EffectProperties {
	if nilDefinition(e.effect) {
		return EffectProperties{}
	}
	props := e.effect.Properties()
	if !nilDefinition(e.otherwise) {
		mergeEffectProperties(&props, e.otherwise.Properties())
	}
	return props
}
func (e *resolutionManaPayment) Apply(ctx *EffectContext) error {
	cost := e.cost
	count := max(0, e.amount.resolveValue(ctx))
	if e.repeated {
		fields := []*int{&cost.Generic, &cost.White, &cost.Blue, &cost.Black, &cost.Red, &cost.Green}
		total := 0
		for _, field := range fields {
			if count > 0 && *field > int(^uint(0)>>1)/count {
				return fmt.Errorf("repeated mana cost exceeds integer range")
			}
			*field *= count
			if *field > int(^uint(0)>>1)-total {
				return fmt.Errorf("repeated mana total exceeds integer range")
			}
			total += *field
		}
	} else {
		cost.Generic += cost.XCount * count
	}
	cost.HasX, cost.XCount = false, 0
	// Fix the amount before mana abilities or payment choices can change it.
	return (&unlessPaidEffect{player: e.player, costs: Costs(&ManaCostPayment{MC: cost}), effect: e.effect, ifPaid: e.ifPaid, otherwise: e.otherwise, twoBranches: e.twoBranches}).Apply(ctx)
}
func (e *resolutionManaPayment) inspectDefinition() definitionNode {
	role, branch := "unless-paid", "unpaid"
	if e.ifPaid {
		role, branch = "if-paid", "paid"
	}
	n := definitionNode{Role: role, Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"amount", e.amount}, {branch, e.effect}}}
	if e.twoBranches {
		n.children = append(n.children, definitionEdge{"unpaid", e.otherwise})
	}
	if e.repeated {
		if e.cost.HasX || e.cost.XCount != 0 || len(e.cost.Hybrid) != 0 || e.cost.Generic < 0 || e.cost.White < 0 || e.cost.Blue < 0 || e.cost.Black < 0 || e.cost.Red < 0 || e.cost.Green < 0 {
			n.problem = "supply fixed nonnegative generic and colored mana"
		}
		return n
	}
	if !e.cost.HasX || e.cost.XCount <= 0 || e.cost.Generic < 0 || e.cost.White < 0 || e.cost.Blue < 0 || e.cost.Black < 0 || e.cost.Red < 0 || e.cost.Green < 0 {
		n.problem = "supply a nonnegative mana cost with X"
	}
	return n
}
