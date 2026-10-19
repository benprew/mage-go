package ron

import "github.com/benprew/mage-go/pkg/mage"

type activationManaPlan struct{ used bool }

func compileActivationManaHandFilter(n Node, ctx scope) (mage.CardFilter, bool, error) {
	if !containsDeclarationName(n, "ManaCostIsSubsetOfManaPaidForThisAbility") {
		f, err := compileHandCardFilter(n)
		return f, false, err
	}
	if ctx.activationMana == nil || !ctx.activation || !ctx.x || ctx.delayedStep {
		return mage.CardFilter{}, false, problem(n, "mana subset requires its immediate activation body")
	}
	if n.Name == "ManaCostIsSubsetOfManaPaidForThisAbility" {
		if _, err := symbol(n); err != nil {
			return mage.CardFilter{}, false, err
		}
		ctx.activationMana.used = true
		return mage.CardFilter{}, true, nil
	}
	if n.Name != "And" {
		return mage.CardFilter{}, false, unsupported(n, "activation mana hand filter conjunction")
	}
	if err := arity(n, 1); err != nil {
		return mage.CardFilter{}, false, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return mage.CardFilter{}, false, err
	}
	filters := make([]mage.CardFilter, 0, len(parts))
	receipt := false
	for _, part := range parts {
		f, r, err := compileActivationManaHandFilter(part, ctx)
		if err != nil {
			return mage.CardFilter{}, false, err
		}
		filters = append(filters, f)
		receipt = receipt || r
	}
	return mage.NewCardFilter("activation mana hand filter", func(card mage.Card) bool {
		for _, filter := range filters {
			if !filter.Match(card) {
				return false
			}
		}
		return true
	}), receipt, nil
}
