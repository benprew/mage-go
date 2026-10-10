package mage

import "fmt"

// WithSpellSacrificeManaValueContinuation permits a saved payment in fresh rules.
// The additional sacrifice cost and spell body must use the same binding.
func WithSpellSacrificeManaValueContinuation(paid *NumberBinding) ActionOption {
	return func(action *ActionDefinition) {
		action.spellTextContinuation = true
		action.spellSacrificeManaValue = paid
		action.spellSacrificeContinuationOptions++
	}
}

func spellSacrificeTextCost(base *BaseCard, action *ActionDefinition) (*sacrificeMatchingCost, error) {
	if action.spellSacrificeContinuationOptions != 1 || action.spellSacrificeManaValue == nil ||
		!action.supportsSpellTextContinuation() || len(base.additionalCosts) != 1 || len(base.alternateCosts) != 0 {
		return nil, fmt.Errorf("spell sacrifice text continuation requires one saved mana value and one additional cost")
	}
	cost, ok := base.additionalCosts[0].(*sacrificeMatchingCost)
	if !ok || cost == nil || cost.count != 1 || !cost.captureManaValue || cost.paidManaValue != action.spellSacrificeManaValue ||
		cost.captureStats || cost.captureSubtype || cost.captureCard || cost.capturePermanent || cost.captureTotalPower {
		return nil, fmt.Errorf("spell sacrifice text continuation requires one matching sacrifice mana-value producer")
	}
	return cost, nil
}

func transferSpellTextNumbers(obj *StackObject, old, fresh *ActionDefinition, definition Card) (map[*NumberBinding]int, error) {
	if len(obj.capturedPlayers)+len(obj.capturedCards)+len(obj.capturedCardSets)+len(obj.capturedPermanents) != 0 {
		return nil, fmt.Errorf("spell text continuation cannot transfer these casting bindings")
	}
	if old.spellSacrificeContinuationOptions == 0 {
		if fresh.spellSacrificeContinuationOptions != 0 || len(obj.capturedNumbers) != 0 {
			return nil, fmt.Errorf("spell text continuation changed its casting bindings")
		}
		return nil, nil
	}
	oldCost, err := spellSacrificeTextCost(obj.spellTextDefinition.(*BaseCard), old)
	if err != nil {
		return nil, err
	}
	freshCost, err := spellSacrificeTextCost(definition.(*BaseCard), fresh)
	if err != nil {
		return nil, err
	}
	if oldCost.includeSource != freshCost.includeSource || oldCost.text != freshCost.text {
		return nil, fmt.Errorf("spell sacrifice text continuation changed its payment context")
	}
	value, present := obj.capturedNumbers[old.spellSacrificeManaValue]
	if !present || len(obj.capturedNumbers) != 1 || value < 0 {
		return nil, fmt.Errorf("spell sacrifice text continuation requires its exact saved payment")
	}
	return map[*NumberBinding]int{fresh.spellSacrificeManaValue: value}, nil
}
