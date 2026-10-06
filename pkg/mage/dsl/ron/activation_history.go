package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
)

type activationHistoryPlan struct{ used bool }

func compileActivationHistoryBranch(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if ctx.activationHistory == nil || ctx.delayedStep {
		return nil, problem(n, "activation history requires the resolving activated ability")
	}
	condition := n.Args[0]
	if condition.Name != "TimesThisAbilityHasBeenActivatedThisTurnIs" {
		return nil, unsupported(condition, "activation history condition")
	}
	if err := arity(condition, 1); err != nil {
		return nil, err
	}
	comparison := condition.Args[0]
	if comparison.Name != "GreaterThanOrEqualTo" {
		return nil, unsupported(comparison, "activation history comparison")
	}
	if err := arity(comparison, 1); err != nil {
		return nil, err
	}
	value := comparison.Args[0]
	if value.Name != "Integer" {
		return nil, unsupported(value, "activation history threshold")
	}
	if err := arity(value, 1); err != nil {
		return nil, err
	}
	if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
		return nil, problem(value, "activation threshold requires a nonnegative integer")
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(n.Args[1], "activation history branch requires actions")
	}
	effects, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	ctx.activationHistory.used = true
	return func(bindings instanceBindings) mage.Effect {
		body := make([]mage.Effect, len(effects))
		for i, build := range effects {
			body[i] = build(bindings)
		}
		return mage.IfActivationsAtLeast(bindings.activationHistory, value.Args[0].Integer, mage.Sequence(body...))
	}, nil
}

func containsManaAction(n Node) bool {
	if n.Name == "AddMana" || n.Name == "AddManaRepeated" {
		return true
	}
	return slices.ContainsFunc(n.Args, containsManaAction)
}
