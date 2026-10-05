package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCombatPrevention(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	event := n.Args[0]
	var source *permanentReferencePlan
	if event.Name == "CombatDamageWouldBeDealtByCreature" {
		if err := arity(event, 1); err != nil {
			return nil, err
		}
		ref, err := compilePermanentRef(event.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		source = &ref
	} else if event.Kind != Symbol || event.Name != "CombatDamageWouldBeDealt" {
		return nil, unsupported(event, "combat prevention event")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "combat prevention requires exactly one operation")
	}
	operation := operations[0]
	if operation.Kind != Symbol || operation.Name != "PreventThatDamage" {
		return nil, unsupported(operation, "combat prevention operation")
	}
	duration := n.Args[2]
	if duration.Kind != Symbol || duration.Name != "UntilEndOfTurn" {
		return nil, unsupported(duration, "combat prevention duration")
	}
	if source != nil {
		return func(bindings instanceBindings) mage.Effect {
			return mage.PreventCombatDamageByThisTurn(source.build(bindings))
		}, nil
	}
	return func(instanceBindings) mage.Effect { return mage.PreventAllCombatDamage() }, nil
}
