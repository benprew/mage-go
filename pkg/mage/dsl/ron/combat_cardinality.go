package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCombatCardinality(n Node) (optionFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	value := n.Args[0]
	if value.Name != "Integer" {
		return nil, unsupported(value, "combat declaration limit")
	}
	if err := arity(value, 1); err != nil {
		return nil, err
	}
	amount := value.Args[0]
	if amount.Kind != IntegerNode || amount.Integer < 0 {
		return nil, problem(amount, "combat declaration limit requires a nonnegative integer")
	}
	return func() mage.CardOption {
		rule := mage.LimitAttackDeclarations(amount.Integer)
		if n.Name == "NoMoreThanNumberCreaturesCanBlock" {
			rule = mage.LimitBlockDeclarations(amount.Integer)
		}
		return mage.WithAbility(mage.Static(rule))
	}, nil
}
