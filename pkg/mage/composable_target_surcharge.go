package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// TargetSurchargeOption permits the same cost rule in both spell constructors.
type TargetSurchargeOption interface {
	SpellOption
	ModalSpellOption
}

type targetSurcharge struct{ amount int }

// AdditionalManaPerTargetBeyondFirst uses the announced target count because
// target legality can change after the total cost is locked.
func AdditionalManaPerTargetBeyondFirst(amount int) TargetSurchargeOption {
	return targetSurcharge{amount: amount}
}

// WithAdditionalManaPerTargetBeyondFirst uses the shared announcement cost.
// Target changes after announcement cannot change the locked payment.
func WithAdditionalManaPerTargetBeyondFirst(amount int) ActionOption {
	return WithCost(targetSurcharge{amount: amount})
}

func (targetSurcharge) modalSpellOption() {}
func (o targetSurcharge) applyModalSpell(m *ModalSpellAbility) {
	o.applySpell(&m.SpellAbility)
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
