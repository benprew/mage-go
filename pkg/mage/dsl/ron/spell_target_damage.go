package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSpellTargetDamagePrevention(n Node, ctx scope, condition mage.StateCondition) (optionFactory, error) {
	event := n.Args[0]
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	filter := event.Args[0]
	if filter.Name != "TargetsPermanent" {
		return nil, unsupported(filter, "spell damage target filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	target, err := compilePermanentRef(filter.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if target.kind != permanentSource {
		return nil, unsupported(filter.Args[0], "static spell damage target")
	}
	recipient := event.Args[1]
	if recipient.Name != "Permanent" {
		return nil, unsupported(recipient, "static spell damage recipient")
	}
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(recipient.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if subject.kind != permanentSource {
		return nil, unsupported(recipient.Args[0], "static spell damage recipient")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "PreventThatDamage" {
		return nil, problem(n.Args[1], "spell damage prevention requires the symbol PreventThatDamage")
	}
	return func() mage.CardOption {
		rule := mage.PreventDamageFromSpellsTargeting(subject.build(instanceBindings{}))
		if condition != nil {
			rule = mage.PreventDamageFromSpellsTargetingIf(subject.build(instanceBindings{}), condition)
		}
		return mage.WithAbility(mage.Static(rule))
	}, nil
}
