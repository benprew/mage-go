package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCollectionPermanentRule(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, unsupported(n.Args[1], "collection permanent rule operations")
	}
	operation := operations[0]
	if operation.Name == "CantAttackAPlayer" {
		return compileLastTurnAttackRule(n, operation, ctx)
	}
	if operation.Name == "CantAttack" {
		return compileCollectionAttackRule(n, operation)
	}
	if operation.Name != "DoesntUntapDuringControllersUntap" {
		return compileLandwalkBlockingRule(n)
	}
	if operation.Kind != Symbol {
		return nil, unsupported(operation, "collection untap operation")
	}
	filter, err := compileFilter(n.Args[0])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption { return mage.WithStaticAbility(mage.PreventUntapDuringStepForMatching(filter)) }, nil
}
