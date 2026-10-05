package mage

import (
	"slices"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type resolutionManaPayment struct {
	player PlayerRef
	cost   ManaCost
	amount ResolutionValue
	effect ResolvingEffect
	ifPaid bool
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

func (e *resolutionManaPayment) Text() string { return "optional resolution mana payment" }
func (e *resolutionManaPayment) Properties() EffectProperties {
	if nilDefinition(e.effect) {
		return EffectProperties{}
	}
	return e.effect.Properties()
}
func (e *resolutionManaPayment) Apply(ctx *EffectContext) error {
	cost := e.cost
	cost.Generic += cost.XCount * max(0, e.amount.resolveValue(ctx))
	cost.HasX, cost.XCount = false, 0
	// Fix the amount before mana abilities or payment choices can change it.
	return (&unlessPaidEffect{player: e.player, costs: Costs(&ManaCostPayment{MC: cost}), effect: e.effect, ifPaid: e.ifPaid}).Apply(ctx)
}
func (e *resolutionManaPayment) inspectDefinition() definitionNode {
	role, branch := "unless-paid", "unpaid"
	if e.ifPaid {
		role, branch = "if-paid", "paid"
	}
	n := definitionNode{Role: role, Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"amount", e.amount}, {branch, e.effect}}}
	if !e.cost.HasX || e.cost.XCount <= 0 || e.cost.Generic < 0 || e.cost.White < 0 || e.cost.Blue < 0 || e.cost.Black < 0 || e.cost.Red < 0 || e.cost.Green < 0 {
		n.problem = "supply a nonnegative mana cost with X"
	}
	return n
}
