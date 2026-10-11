package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileAcquiredStatic(n Node, ctx scope) (effectFactory, error) {
	if !ctx.aura || n.Args[0].Kind != Symbol || n.Args[0].Name != "ThisPermanent" {
		return nil, unsupported(n.Args[0], "acquired static recipient")
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) != 1 || parts[0].Name != "AddAbility" {
		return nil, unsupported(n.Args[1], "acquired static modification")
	}
	if err := arity(parts[0], 1); err != nil {
		return nil, err
	}
	abilities, err := list(parts[0].Args[0])
	if err != nil {
		return nil, err
	}
	if len(abilities) != 1 {
		return nil, problem(parts[0], "supply one acquired static ability")
	}
	rule := abilities[0]
	if rule.Name != "PermanentLayerEffect" {
		return nil, unsupported(rule, "acquired static ability")
	}
	if err := arity(rule, 2); err != nil {
		return nil, err
	}
	if rule.Args[0].Kind != Symbol || rule.Args[0].Name != "HostPermanent" {
		return nil, unsupported(rule.Args[0], "acquired static subject")
	}
	modifications, err := list(rule.Args[1])
	if err != nil {
		return nil, err
	}
	if len(modifications) != 1 || modifications[0].Name != "LosesAbility" {
		return nil, unsupported(rule.Args[1], "acquired static layer")
	}
	if err := arity(modifications[0], 1); err != nil {
		return nil, err
	}
	keyword, err := compileTemporaryKeyword(modifications[0].Args[0])
	if err != nil {
		return nil, err
	}
	return func(instanceBindings) mage.Effect {
		condition := mage.StateConditionFunc("acquired host rule", []mage.StateDependency{mage.StateBattlefield}, func(mage.StateReader, mage.StateContext) bool { return true })
		return mage.AcquireStaticAbility(mage.This, mage.While(condition, mage.RemoveKeyword(mage.AttachedPermanent, keyword)))
	}, nil
}
