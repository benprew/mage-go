package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSourcePreventionOperations(n Node, operations []Node) ([]mage.SourcePreventionOperation, error) {
	if len(operations) < 1 || len(operations) > 2 {
		return nil, problem(n, "exact-source prevention requires one prevention and at most one life gain operation")
	}
	operation := operations[0]
	var amount mage.ProposedDamageValue
	if operation.Kind == Symbol && operation.Name == "PreventThatDamage" {
		amount = mage.ProposedDamageAmount()
	} else if operation.Name == "PreventSomeOfThatDamage" {
		if err := arity(operation, 1); err != nil {
			return nil, err
		}
		var err error
		amount, err = compileProposedDamageValue(operation.Args[0])
		if err != nil {
			return nil, err
		}
	} else {
		return nil, unsupported(operation, "exact-source prevention operation")
	}
	result := []mage.SourcePreventionOperation{mage.PreventProposedDamage(amount)}
	if len(operations) == 2 {
		gain := operations[1]
		if gain.Name != "GainLife" {
			return nil, unsupported(gain, "post-prevention operation")
		}
		if err := arity(gain, 1); err != nil {
			return nil, err
		}
		value := gain.Args[0]
		if value.Kind != Symbol || value.Name != "TheAmountOfDamagePreventedThisWay" {
			return nil, unsupported(value, "actual prevention result")
		}
		result = append(result, mage.GainLifeFromPreventedDamage(mage.You))
	}
	return result, nil
}

func compileProposedDamageValue(n Node) (mage.ProposedDamageValue, error) {
	if n.Kind == Symbol && n.Name == "WouldDealDamage_ThatMuchDamage" {
		return mage.ProposedDamageAmount(), nil
	}
	if n.Name == "HalfRoundedDown" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		value := n.Args[0]
		if value.Kind != Symbol || value.Name != "WouldDealDamage_ThatMuchDamage" {
			return nil, unsupported(value, "proposed damage amount")
		}
		return mage.HalfProposedDamageRoundedDown(), nil
	}
	return nil, unsupported(n, "proposed damage value")
}
