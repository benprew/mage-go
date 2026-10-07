package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileAttackCost(n, operation Node, ctx scope) (optionFactory, error) {
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if subject.kind != permanentSource && subject.kind != permanentHost {
		return nil, unsupported(n.Args[0], "static attack cost reference")
	}
	cost := operation.Args[0]
	if cost.Name != "PayMana" && cost.Name != "SacrificeNumberPermanents" {
		return nil, unsupported(cost, "static attack cost")
	}
	builds, hasX, err := compileCosts(cost)
	if err != nil {
		return nil, err
	}
	if hasX || len(builds) != 1 {
		return nil, problem(cost, "supply one fixed attack cost")
	}
	return func() mage.CardOption {
		payment := builds[0]()
		if subject.kind == permanentHost {
			return mage.WithStaticAbility(mage.AttachedCantAttackUnlessPays(payment))
		}
		return mage.WithStaticAbility(mage.SourceCantAttackUnlessPays(payment))
	}, nil
}
