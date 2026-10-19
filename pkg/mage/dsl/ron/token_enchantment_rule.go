package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileTokenEnchantmentRule(n Node) (func() mage.Ability, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	subject := n.Args[0]
	if subject.Kind != Symbol || subject.Name != "ThisPermanent" {
		return nil, unsupported(subject, "intrinsic token rule subject")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "CantBeEnchanted" {
		return nil, unsupported(n.Args[1], "intrinsic token rule operations")
	}
	return func() mage.Ability {
		return mage.Static(mage.CantBeEnchanted(mage.This))
	}, nil
}
