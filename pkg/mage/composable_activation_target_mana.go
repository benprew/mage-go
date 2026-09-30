package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type activationTargetMana struct {
	color  Color
	amount int
}

// ActivationManaPerTarget uses the announced count because target legality can
// change after the total cost is locked.
func ActivationManaPerTarget(color Color, amount int) ActivationOption {
	return activationTargetMana{color: color, amount: amount}
}

func (activationTargetMana) activationOption() {}
func (o activationTargetMana) applyActivation(a *ActionDefinition) {
	a.costs = append(a.costs, o)
}
func (activationTargetMana) CanPay(uuid.UUID, uuid.UUID, *Game) bool { return true }
func (activationTargetMana) Pay(uuid.UUID, uuid.UUID, *Game) error {
	return fmt.Errorf("per-target mana requires activation announcement")
}
func (o activationTargetMana) manaCost() ManaCost {
	var cost ManaCost
	switch o.color {
	case White:
		cost.White = o.amount
	case Blue:
		cost.Blue = o.amount
	case Black:
		cost.Black = o.amount
	case Red:
		cost.Red = o.amount
	case Green:
		cost.Green = o.amount
	}
	return cost
}
func (o activationTargetMana) Text() string {
	return o.manaCost().String() + " for each target"
}
func (o activationTargetMana) resolveActionCost(ctx actionCostContext) Cost {
	return &ManaCostPayment{MC: multiplyManaCost(o.manaCost(), len(ctx.Targets))}
}
func (o activationTargetMana) inspectDefinition() definitionNode {
	n := definitionNode{Role: "activation-mana-per-target", Text: o.Text()}
	if o.amount <= 0 || (o.color != White && o.color != Blue && o.color != Black && o.color != Red && o.color != Green) {
		n.problem = "supply a positive amount of one colored mana type"
	}
	return n
}
