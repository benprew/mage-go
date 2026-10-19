package mage

import (
	"strconv"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ActivationManaBinding saves actual mana types for one activation payment.
type ActivationManaBinding struct {
	name    string
	amounts [core.AnyColor + 1]*NumberBinding
}

// BindActivationMana declares a receipt for one activated ability.
func BindActivationMana(name string) *ActivationManaBinding {
	b := &ActivationManaBinding{name: name}
	for i := range b.amounts {
		b.amounts[i] = BindNumber(name + " " + strconv.Itoa(i))
	}
	return b
}
func (b *ActivationManaBinding) bindingName() string { return b.name }
func (b *ActivationManaBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "activation-mana-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply an activation mana binding name"
	}
	for _, amount := range b.amounts {
		if amount == nil {
			n.problem = "use BindActivationMana to declare receipt amounts"
		}
	}
	return n
}

// WithActivationManaReceipt saves the committed total mana payment.
func WithActivationManaReceipt(binding *ActivationManaBinding) ActionOption {
	return func(a *ActionDefinition) { a.activationManaReceipt = binding; a.activationManaReceiptOptions++ }
}

// WithActivationManaXReceipt saves only the mana assigned to the X component.
// Fixed requirements receive mana before the X component.
func WithActivationManaXReceipt(binding *ActivationManaBinding) ActionOption {
	return func(a *ActionDefinition) { WithActivationManaReceipt(binding)(a); a.activationManaReceiptXOnly = true }
}

type activationManaDeclaration struct{ binding *ActivationManaBinding }

func (d activationManaDeclaration) inspectDefinition() definitionNode {
	return definitionNode{Role: "activation-mana-declaration", bindingWrite: d.binding, children: []definitionEdge{{"binding", d.binding}}}
}

// ChooseHandCardWithActivationManaOrElse requires a payable printed mana cost.
// The receipt supplies actual types. Current spending permissions do not apply.
func ChooseHandCardWithActivationManaOrElse(player PlayerRef, filter CardFilter, receipt *ActivationManaBinding, result *CardBinding, chosen, notChosen ResolvingEffect) ResolvingEffect {
	return &optionalHandChoice{player: player, filter: filter, result: result, chosen: chosen, notChosen: notChosen, receipt: receipt, requiresReceipt: true}
}

func (b *ActivationManaBinding) canPay(ctx *EffectContext, card Card) bool {
	if b == nil {
		return false
	}
	presence, ok := card.(interface{ HasManaCost() bool })
	if !ok || !presence.HasManaCost() {
		return false
	}
	pool := NewManaPool()
	for color, binding := range b.amounts {
		amount, present := ctx.Game.resolution.numberBindings[binding]
		if !present || amount < 0 {
			return false
		}
		pool.Add(core.Color(color), amount)
	}
	cost := card.ManaCost()
	cost.HasX = false
	cost.XCount = 0
	return pool.CanPay(cost, nil)
}

type activationManaRead struct{ binding *ActivationManaBinding }

func (r activationManaRead) inspectDefinition() definitionNode {
	return definitionNode{Role: "activation-mana-read", bindingRead: r.binding, children: []definitionEdge{{"receipt", r.binding}}}
}
func (e *optionalHandChoice) matches(ctx *EffectContext, card Card) bool {
	return e.filter.Match(card) && (!e.requiresReceipt || e.receipt.canPay(ctx, card))
}
