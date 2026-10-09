package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileRevealedManaValue(n Node, ctx scope) (valueFactory, error) {
	if n.Kind != Symbol || n.Name != "ManaValueOfTheCardRevealedThisWay" {
		return nil, problem(n, "saved reveal mana value requires a symbol")
	}
	if ctx.handReveal == nil && ctx.libraryReveal == nil {
		return nil, problem(n, "mana value requires a bound reveal result")
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		result := bindings.handReveals[ctx.handReveal]
		if ctx.libraryReveal != nil {
			result = ctx.libraryReveal.binding(bindings)
		}
		return mage.RevealedCardManaValue(result)
	}, nil
}
