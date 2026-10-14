package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileConditionalStaticPrevention(n Node, ctx scope, condition mage.StateCondition) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if n.Args[0].Name == "DamageWouldBeDealtByAPermanentToARecipient" {
		return compileStaticCollectionPrevention(n, ctx, condition)
	}
	if n.Args[0].Name != "DamageWouldBeDealtByAPermanentToRecipient" {
		return nil, unsupported(n.Args[0], "conditional static prevention event")
	}
	if err := arity(n.Args[0], 2); err != nil {
		return nil, err
	}
	if containsCombatTarget(n.Args[0].Args[0]) {
		return nil, unsupported(n.Args[0], "conditional combat-pair prevention")
	}
	return compileMatchingPermanentPrevention(n, ctx, condition)
}

func compileStaticPrevention(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name == "DamageWouldBeDealtByAPermanentToARecipient" {
		return compileStaticCollectionPrevention(n, ctx, nil)
	}
	if event.Name == "EachDamageWouldBeDealtToRecipient" {
		return compileCounterPointPrevention(n, ctx)
	}
	if event.Name == "DamageWouldBeDealtByASourceToRecipient" {
		return compileStaticSourceTypePrevention(n, ctx)
	}
	if event.Name == "Or" || event.Name == "CombatDamageWouldBeDealtByCreature" || event.Name == "CombatDamageWouldBeDealtToRecipient" || event.Name == "CombatDamageWouldBeDealtByACreatureToRecipient" {
		operations, err := list(n.Args[1])
		if err != nil {
			return nil, err
		}
		if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "PreventThatDamage" {
			return nil, problem(n.Args[1], "static prevention requires the symbol PreventThatDamage")
		}
		build, err := compileStaticCombatPrevention(event, ctx)
		if err != nil {
			return nil, err
		}
		return func() mage.CardOption { return mage.WithAbility(mage.Static(build()...)) }, nil
	}
	if event.Name != "DamageWouldBeDealtByAPermanentToRecipient" {
		return nil, unsupported(event, "static prevention event")
	}
	return compileMatchingPermanentPrevention(n, ctx, nil)
}

func compileMatchingPermanentPrevention(n Node, ctx scope, condition mage.StateCondition) (optionFactory, error) {
	event := n.Args[0]
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	if containsCombatTarget(event.Args[0]) {
		return compileBlockedDamagePrevention(n, ctx)
	}
	from, err := compileFilter(event.Args[0])
	if err != nil {
		return nil, err
	}
	recipient := event.Args[1]
	if recipient.Name != "Permanent" {
		return nil, unsupported(recipient, "static prevention recipient")
	}
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(recipient.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource {
		return nil, unsupported(recipient.Args[0], "static prevention reference")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "static prevention requires exactly one operation")
	}
	operation := operations[0]
	if operation.Kind != Symbol || operation.Name != "PreventThatDamage" {
		return nil, unsupported(operation, "static prevention operation")
	}
	return func() mage.CardOption {
		rule := mage.PreventDamageFromMatchingPermanents(ref.build(instanceBindings{}), from)
		if condition != nil {
			rule = mage.PreventDamageFromMatchingPermanentsIf(ref.build(instanceBindings{}), from, condition)
		}
		return mage.WithAbility(mage.Static(rule))
	}, nil
}
