package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileTemporaryCollection(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "duration")
	}
	if n.Args[1].Kind == List && len(n.Args[1].Args) > 0 && n.Args[1].Args[0].Name == "AddAbility" {
		return compileTemporaryCollectionKeywords(n, ctx)
	}
	subjects, err := compileStaticPermanentFilter(n.Args[0])
	if err != nil {
		return nil, err
	}
	if subjects.excludeSource {
		return nil, unsupported(n.Args[0], "temporary collection source exclusion")
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(n.Args[1], "supply at least one temporary modification")
	}
	type adjustment struct{ power, toughness int }
	adjustments := make([]adjustment, len(parts))
	for i, part := range parts {
		if part.Name != "AdjustPT" {
			return nil, unsupported(part, "temporary collection modification")
		}
		power, toughness, err := compilePT(part)
		if err != nil {
			return nil, err
		}
		adjustments[i] = adjustment{power, toughness}
	}
	return func(_ instanceBindings) mage.Effect {
		modifications := make([]mage.ContinuousModification, len(adjustments))
		for i, value := range adjustments {
			modifications[i] = mage.ModifyEachPT(subjects.build(), mage.Fixed(value.power), mage.Fixed(value.toughness))
		}
		return mage.Until(core.EndOfTurn, modifications...)
	}, nil
}

func compileTemporaryCollectionKeywords(n Node, ctx scope) (effectFactory, error) {
	subjects, err := compileContextualCollection(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	var keywords []core.Attr
	for _, part := range n.Args[1].Args {
		if part.Name != "AddAbility" {
			return nil, unsupported(part, "temporary collection keyword modification")
		}
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		abilities, err := list(part.Args[0])
		if err != nil {
			return nil, err
		}
		if len(abilities) == 0 {
			return nil, problem(part, "supply at least one keyword")
		}
		for _, ability := range abilities {
			keyword, err := compileTemporaryKeyword(ability)
			if err != nil {
				return nil, err
			}
			keywords = append(keywords, keyword)
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		modifications := make([]mage.ContinuousModification, len(keywords))
		for i, keyword := range keywords {
			modifications[i] = mage.GrantEachKeyword(subjects(bindings), keyword)
		}
		return mage.Until(core.EndOfTurn, modifications...)
	}, nil
}
