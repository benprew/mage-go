package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileRevealResultMatcher(n Node, ctx scope) (func(instanceBindings) mage.RevealedCardMatcher, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	switch n.Name {
	case "IsCardtype", "IsNonCardtype":
		kind, err := cardType(n.Args[0])
		if err != nil {
			return nil, err
		}
		return func(instanceBindings) mage.RevealedCardMatcher {
			if n.Name == "IsNonCardtype" {
				return mage.RevealedCardLacksType(kind)
			}
			return mage.RevealedCardHasType(kind)
		}, nil
	case "IsNamed":
		matcher, err := compileResolvingCardNameFilter(n, ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.RevealedCardMatcher {
			return mage.RevealedCardNamed(matcher(bindings))
		}, nil
	default:
		return nil, unsupported(n, "reveal result filter")
	}
}

func compileRevealResultBranch(n Node, ctx scope) (effectFactory, error) {
	count := 2
	if n.Name == "IfElse" {
		count = 3
	}
	if err := arity(n, count); err != nil {
		return nil, err
	}
	condition := n.Args[0]
	if err := arity(condition, 1); err != nil {
		return nil, err
	}
	if ctx.handReveal == nil && ctx.libraryReveal == nil {
		return nil, problem(condition, "reveal predicate requires a bound reveal result")
	}
	if ctx.activation && containsManaAction(n) {
		return nil, problem(n, "reveal conditions do not supply mana solver planning")
	}
	matcher, err := compileRevealResultMatcher(condition.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	branches := make([][]effectFactory, 2)
	for i := 1; i < count; i++ {
		parts, err := list(n.Args[i])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n.Args[i], "reveal branch requires actions")
		}
		branches[i-1], err = compileActionSequence(parts, ctx)
		if err != nil {
			return nil, err
		}
	}
	plan := ctx.handReveal
	return func(bindings instanceBindings) mage.Effect {
		bodies := make([]mage.ResolvingEffect, 2)
		for i, branch := range branches {
			if len(branch) == 0 {
				continue
			}
			effects := make([]mage.Effect, len(branch))
			for j, build := range branch {
				effects[j] = build(bindings)
			}
			bodies[i] = mage.Sequence(effects...)
		}
		if n.Name == "Unless" {
			bodies[0], bodies[1] = bodies[1], bodies[0]
		}
		result := bindings.handReveals[plan]
		if ctx.libraryReveal != nil {
			result = ctx.libraryReveal.binding(bindings)
		}
		return mage.IfRevealedCardMatches(result, matcher(bindings), bodies[0], bodies[1])
	}, nil
}
