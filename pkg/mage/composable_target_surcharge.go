package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type targetSurcharge struct{ amount int }

// AdditionalManaPerTargetBeyondFirst uses the announced target count because
// target legality can change after the total cost is locked.
func AdditionalManaPerTargetBeyondFirst(amount int) SpellOption {
	return targetSurcharge{amount: amount}
}

func (targetSurcharge) spellOption()                              {}
func (o targetSurcharge) applySpell(a *ActionDefinition)          { a.costs = append(a.costs, o) }
func (o targetSurcharge) CanPay(uuid.UUID, uuid.UUID, *Game) bool { return true }
func (o targetSurcharge) Pay(uuid.UUID, uuid.UUID, *Game) error {
	return fmt.Errorf("target surcharge requires spell announcement")
}
func (o targetSurcharge) Text() string {
	return fmt.Sprintf("{%d} for each target beyond the first", o.amount)
}
func (o targetSurcharge) resolveActionCost(ctx actionCostContext) Cost {
	return &ManaCostPayment{MC: ManaCost{Generic: o.amount * max(0, len(ctx.Targets)-1)}}
}
func (o targetSurcharge) inspectDefinition() definitionNode {
	n := definitionNode{Role: "additional-mana-per-target-beyond-first", Text: o.Text()}
	if o.amount <= 0 {
		n.problem = "supply a positive generic mana amount"
	}
	return n
}
