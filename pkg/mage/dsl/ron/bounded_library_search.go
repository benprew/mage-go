package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileOptionalBoundedLibrarySearch(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	steps, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(steps) != 4 {
		return nil, problem(n, "bounded search requires find, reveal, hand destination and shuffle")
	}
	for i, name := range []string{"RevealFoundCards", "PutFoundCardsIntoHand", "Shuffle"} {
		if steps[i+1].Kind != Symbol || steps[i+1].Name != name {
			return nil, unsupported(steps[i+1], "bounded search instruction")
		}
	}
	find := steps[0]
	if find.Name != "FindNumberCardsOfType" {
		return nil, unsupported(find, "bounded search instruction")
	}
	if err := arity(find, 2); err != nil {
		return nil, err
	}
	value := find.Args[0]
	if value.Name != "Integer" {
		return nil, unsupported(value, "fixed search maximum")
	}
	if err := arity(value, 1); err != nil {
		return nil, err
	}
	if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
		return nil, problem(value, "search maximum requires a nonnegative integer")
	}
	filter, err := compileBasicLandSearchFilter(find.Args[1])
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.MaySearchLibraryToHand(ctx.player.build(bindings), value.Args[0].Integer, filter)
	}, nil
}
