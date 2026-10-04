package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCounterRemovalCost(n Node) (costFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 1 {
		return nil, problem(n, "counter cost requires one source instruction")
	}
	instruction := parts[0]
	var counterNode, subjectNode Node
	amount := 1
	switch instruction.Name {
	case "ACounterOfTypeFromPermanent":
		if err := arity(instruction, 2); err != nil {
			return nil, err
		}
		counterNode, subjectNode = instruction.Args[0], instruction.Args[1]
	case "NumberCountersOfTypeFromPermanent":
		if err := arity(instruction, 3); err != nil {
			return nil, err
		}
		value := instruction.Args[0]
		if value.Name != "Integer" {
			return nil, unsupported(value, "fixed counter cost amount")
		}
		if err := arity(value, 1); err != nil {
			return nil, err
		}
		if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
			return nil, problem(value, "counter cost requires a nonnegative integer")
		}
		amount = value.Args[0].Integer
		counterNode, subjectNode = instruction.Args[1], instruction.Args[2]
	default:
		return nil, unsupported(instruction, "source counter cost instruction")
	}
	counter, err := compilePermanentCounterType(counterNode)
	if err != nil {
		return nil, err
	}
	if subjectNode.Kind != Symbol || subjectNode.Name != "ThisPermanent" {
		return nil, unsupported(subjectNode, "counter cost source")
	}
	return func() mage.Cost { return mage.RemoveCountersCost(counter, amount) }, nil
}
