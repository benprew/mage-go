package mage

import (
	"fmt"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithXSpendingColor limits mana spent on a spell's X to one color.
// Cost reductions can remove this part without changing announced X.
func WithXSpendingColor(color core.Color) ActionOption {
	return func(a *ActionDefinition) {
		a.xSpendingColor = color
		a.xSpendingOptions++
	}
}

func paymentCostForContext(cost core.ManaCost, ctx *core.SpellPaymentContext) (core.ManaCost, bool) {
	if ctx == nil || !ctx.XSpendingRestricted {
		return cost, true
	}
	amount := ctx.RestrictedGeneric
	if ctx.XSpendingColor < core.White || ctx.XSpendingColor > core.Green || amount < 0 || amount > cost.Generic {
		return cost, false
	}
	cost.Generic -= amount
	switch ctx.XSpendingColor {
	case core.White:
		cost.White += amount
	case core.Blue:
		cost.Blue += amount
	case core.Black:
		cost.Black += amount
	case core.Red:
		cost.Red += amount
	case core.Green:
		cost.Green += amount
	}
	return cost, true
}

func spellRestrictedXUnits(cost core.ManaCost, costs []Cost, x int) int {
	amount := 0
	if cost.HasX {
		amount = x * cost.XCount
	}
	for _, extra := range costs {
		switch c := extra.(type) {
		case *ManaCostPayment:
			if c.MC.HasX {
				amount += x * c.MC.XCount
			}
		case *xManaCost:
			amount += x
		}
	}
	return amount
}

func validateXSpendingCard(card Card) error {
	count := 0
	for _, ability := range card.Abilities() {
		action, ok := UnwrapAbility(ability).(*ActionDefinition)
		if !ok || action.xSpendingOptions == 0 {
			continue
		}
		count += action.xSpendingOptions
		if !card.ManaCost().HasX || card.ManaCost().XCount < 1 {
			return fmt.Errorf("X spending requires a printed X mana cost")
		}
	}
	if count > 1 {
		return fmt.Errorf("supply one X spending declaration")
	}
	return nil
}
