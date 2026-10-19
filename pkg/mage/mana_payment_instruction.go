package mage

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ManaPaymentInstruction retains one optional mana payment and both branches.
// Amount is present only for a cost with X. An absent branch has no action.
type ManaPaymentInstruction struct {
	Player       PlayerRef
	Cost         core.ManaCost
	Amount       ResolutionValue
	Paid, Unpaid ResolvingEffect
}

// Effect builds the native payment with the shared cost transaction.
// Missing operands and malformed costs return nil.
func (instruction ManaPaymentInstruction) Effect() ResolvingEffect {
	cost := instruction.Cost
	if nilDefinition(instruction.Player) || instruction.Paid == nil && instruction.Unpaid == nil || instruction.Paid != nil && nilDefinition(instruction.Paid) || instruction.Unpaid != nil && nilDefinition(instruction.Unpaid) {
		return nil
	}
	if cost.Generic < 0 || cost.White < 0 || cost.Blue < 0 || cost.Black < 0 || cost.Red < 0 || cost.Green < 0 || cost.HasX != (cost.XCount > 0) || cost.XCount < 0 {
		return nil
	}
	for _, symbol := range cost.Hybrid {
		if symbol.A < core.White || symbol.A > core.Green || symbol.B < core.White || symbol.B > core.Green {
			return nil
		}
	}
	if cost.HasX && nilDefinition(instruction.Amount) || !cost.HasX && instruction.Amount != nil {
		return nil
	}
	cost.Hybrid = slices.Clone(cost.Hybrid)
	paid, unpaid := instruction.Paid, instruction.Unpaid
	if cost.HasX {
		if paid == nil {
			return UnlessManaPaid(instruction.Player, cost, instruction.Amount, unpaid)
		}
		if unpaid == nil {
			return IfManaPaid(instruction.Player, cost, instruction.Amount, paid)
		}
		return &resolutionManaPayment{player: instruction.Player, cost: cost, amount: instruction.Amount, effect: paid, otherwise: unpaid, ifPaid: true, twoBranches: true}
	}
	if paid == nil {
		return UnlessPaid(instruction.Player, Costs(&ManaCostPayment{MC: cost}), unpaid)
	}
	if unpaid == nil {
		return IfPaid(instruction.Player, Costs(&ManaCostPayment{MC: cost}), paid)
	}
	return PayOrElse(instruction.Player, Costs(&ManaCostPayment{MC: cost}), paid, unpaid)
}

// InspectManaPaymentInstruction reads operands without amounts or payment choices.
// Consumers must validate the player, amount and complete branches before use.
func InspectManaPaymentInstruction(effect ResolvingEffect) (ManaPaymentInstruction, bool) {
	var out ManaPaymentInstruction
	var body, otherwise ResolvingEffect
	var paid bool
	switch effect := effect.(type) {
	case *resolutionManaPayment:
		if effect == nil || effect.repeated {
			return out, false
		}
		out.Player, out.Cost, out.Amount = effect.player, effect.cost, effect.amount
		body, otherwise, paid = effect.effect, effect.otherwise, effect.ifPaid
		if nilDefinition(body) || effect.twoBranches && nilDefinition(otherwise) || !effect.twoBranches && otherwise != nil {
			return out, false
		}
	case *unlessPaidEffect:
		if effect == nil || effect.required || len(effect.costs.costs) != 1 {
			return out, false
		}
		cost, ok := effect.costs.costs[0].(*ManaCostPayment)
		if !ok || cost == nil || cost.MC.HasX {
			return out, false
		}
		out.Player, out.Cost = effect.player, cost.MC
		body, otherwise, paid = effect.effect, effect.otherwise, effect.ifPaid
		if nilDefinition(body) || effect.twoBranches && nilDefinition(otherwise) || !effect.twoBranches && otherwise != nil {
			return out, false
		}
	default:
		return out, false
	}
	if paid {
		out.Paid, out.Unpaid = body, otherwise
	} else {
		out.Unpaid, out.Paid = body, otherwise
	}
	out.Cost.Hybrid = slices.Clone(out.Cost.Hybrid)
	return out, out.Effect() != nil
}

func inspectOptionalCounterPayment(instruction ManaPaymentInstruction) (spellPermanentInstruction, bool) {
	payer, ok := instruction.Player.(spellControllerRef)
	if !ok || instruction.Cost.HasX && !pureFixedOrXAmount(instruction.Amount) {
		return spellPermanentInstruction{}, false
	}
	target, ok := payer.spell.(*SpellTargetBinding)
	if !ok {
		return spellPermanentInstruction{}, false
	}
	if _, ok := InspectSpellTarget(target); !ok {
		return spellPermanentInstruction{}, false
	}
	for _, branch := range []ResolvingEffect{instruction.Paid, instruction.Unpaid} {
		if branch != nil && !inspectOptionalCounterBranch(branch, target) {
			return spellPermanentInstruction{}, false
		}
	}
	return spellPermanentInstruction{optionalPayment: true, counterSubject: target}, true
}

func inspectOptionalCounterBranch(effect ResolvingEffect, target *SpellTargetBinding) bool {
	if saved, ok := InspectSavedPlayerInstruction(effect); ok {
		subject, ok := saved.Subject.(spellControllerRef)
		if !ok || !subject.withLKI || !sameOptionalCounterTarget(subject.spell, target) {
			return false
		}
		return inspectOptionalCounterPlayerBody(saved.Body, saved.Binding)
	}
	switch effect := effect.(type) {
	case *sequenceEffect:
		if effect == nil {
			return false
		}
		for _, child := range effect.effects {
			if !inspectOptionalCounterBranch(child, target) {
				return false
			}
		}
		return true
	case *compositeEffect:
		if effect == nil {
			return false
		}
		for _, child := range effect.effects {
			if !inspectOptionalCounterBranch(child, target) {
				return false
			}
		}
		return true
	}
	counters, ok := InspectCounterInstructions(effect)
	if !ok {
		return false
	}
	for _, counter := range counters {
		if !sameOptionalCounterTarget(counter.Subject, target) {
			return false
		}
	}
	return true
}

func sameOptionalCounterTarget(reference StackObjectRef, target *SpellTargetBinding) bool {
	binding, ok := reference.(*SpellTargetBinding)
	return ok && binding != nil && binding.targetDeclaration == target.targetDeclaration
}

func inspectOptionalCounterPlayerBody(effect ResolvingEffect, member *PlayerBinding) bool {
	switch effect := effect.(type) {
	case *sequenceEffect:
		if effect == nil {
			return false
		}
		for _, child := range effect.effects {
			if !inspectOptionalCounterPlayerBody(child, member) {
				return false
			}
		}
		return true
	case *compositeEffect:
		if effect == nil {
			return false
		}
		for _, child := range effect.effects {
			if !inspectOptionalCounterPlayerBody(child, member) {
				return false
			}
		}
		return true
	}
	if taps, ok := InspectTapInstructions(effect); ok {
		for _, tap := range taps {
			if tap.Operation != TapOperation || tap.Subject != nil {
				return false
			}
			root, ok := inspectContextualCollectionWithMember(tap.Subjects, member)
			if !ok || collectionReadsTargets(root) {
				return false
			}
		}
		return true
	}
	if losses, ok := InspectManaLossInstructions(effect); ok {
		for _, loss := range losses {
			subject, ok := loss.Subject.(savedPlayerRef)
			if !ok || subject.binding != member {
				return false
			}
		}
		return true
	}
	return false
}
