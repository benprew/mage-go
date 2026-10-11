package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileCapturedBasePT(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	subjects, err := compileContextualCollection(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) != 1 || parts[0].Kind != Tuple || parts[0].Name != "SetPT" {
		return nil, problem(n.Args[1], "supply one fixed base P/T modification")
	}
	part := parts[0]
	if err := arity(part, 1); err != nil {
		return nil, err
	}
	if part.Args[0].Kind != Tuple || part.Args[0].Name != "PT" {
		return nil, unsupported(part.Args[0], "fixed base P/T pair")
	}
	power, toughness, err := compilePT(part.Args[0])
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.Until(core.Indefinite, mage.SetEachBasePowerToughness(subjects(bindings), mage.Fixed(power), mage.Fixed(toughness)))
	}, nil
}
