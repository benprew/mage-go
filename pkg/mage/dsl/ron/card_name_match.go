package ron

import "github.com/benprew/mage-go/pkg/mage"

type cardNameMatcherFactory func(instanceBindings) mage.CardNameMatcher

func compileResolvingCardNameFilter(n Node, ctx scope) (cardNameMatcherFactory, error) {
	if n.Name != "IsNamed" {
		return nil, unsupported(n, "resolving card name filter")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	name, err := compileCardNameRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.CardNameMatcher {
		return mage.MatchesChosenCardName(name(bindings))
	}, nil
}
