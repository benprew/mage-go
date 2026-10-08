package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCombatExit(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		if n.Name == "RemoveCreatureFromCombatAndUnblockBlockers" {
			return mage.RemovePermanentFromCombatAndUnblock(subject.build(bindings))
		}
		return mage.RemovePermanentFromCombat(subject.build(bindings))
	}, nil
}
