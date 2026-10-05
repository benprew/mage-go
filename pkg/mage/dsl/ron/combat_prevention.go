package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileGlobalCombatPrevention(n Node) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Kind != Symbol || event.Name != "CombatDamageWouldBeDealt" {
		return nil, unsupported(event, "global combat prevention event")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "global combat prevention requires exactly one operation")
	}
	operation := operations[0]
	if operation.Kind != Symbol || operation.Name != "PreventThatDamage" {
		return nil, unsupported(operation, "global combat prevention operation")
	}
	duration := n.Args[2]
	if duration.Kind != Symbol || duration.Name != "UntilEndOfTurn" {
		return nil, unsupported(duration, "global combat prevention duration")
	}
	return func(instanceBindings) mage.Effect { return mage.PreventAllCombatDamage() }, nil
}
