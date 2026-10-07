package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCounterPlacementPayment(n, payer Node, ctx scope) (costFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 1 {
		return nil, problem(n, "counter payment requires one instruction")
	}
	instruction := parts[0]
	if instruction.Name != "ACounterOfTypeOnAPermanent" {
		return nil, unsupported(instruction, "counter payment instruction")
	}
	if err := arity(instruction, 2); err != nil {
		return nil, err
	}
	counter, err := compilePermanentCounterType(instruction.Args[0])
	if err != nil {
		return nil, err
	}
	filter := instruction.Args[1]
	if filter.Name != "And" {
		return nil, unsupported(filter, "counter payment filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	filters, err := list(filter.Args[0])
	if err != nil {
		return nil, err
	}
	if len(filters) != 2 {
		return nil, problem(filter, "supply creature type and payer control filters")
	}
	creature, controlled := false, false
	for _, part := range filters {
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		switch part.Name {
		case "IsCardtype":
			if creature || part.Args[0].Kind != Symbol || part.Args[0].Name != "Creature" {
				return nil, unsupported(part, "counter payment type")
			}
			creature = true
		case "ControlledByAPlayer":
			if controlled {
				return nil, problem(part, "supply one payer control filter")
			}
			single := part.Args[0]
			if single.Name != "SinglePlayer" {
				return nil, unsupported(single, "counter payment controller")
			}
			if err := arity(single, 1); err != nil {
				return nil, err
			}
			if _, err := compilePlayerRef(single.Args[0], ctx); err != nil {
				return nil, err
			}
			if definitionKey(single.Args[0]) != definitionKey(payer) {
				return nil, problem(single, "counter payment controller must be the payer")
			}
			controlled = true
		default:
			return nil, unsupported(part, "counter payment filter")
		}
	}
	if !creature || !controlled {
		return nil, problem(filter, "supply creature type and payer control filters")
	}
	return func() mage.Cost { return mage.PutCounterOnMatchingCost(counter, mage.IsCreature) }, nil
}
